package encryption

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"io"

	"golang.org/x/crypto/pbkdf2"
)

// Algorithm represents an encryption algorithm
type Algorithm uint8

const (
	// AlgorithmAES256GCM uses AES-256 in GCM mode (recommended)
	AlgorithmAES256GCM Algorithm = iota
	// AlgorithmAES256CTR uses AES-256 in CTR mode
	AlgorithmAES256CTR
	// AlgorithmNone disables encryption
	AlgorithmNone
)

// String returns the string representation of the algorithm
func (a Algorithm) String() string {
	switch a {
	case AlgorithmAES256GCM:
		return "AES-256-GCM"
	case AlgorithmAES256CTR:
		return "AES-256-CTR"
	case AlgorithmNone:
		return "None"
	default:
		return "Unknown"
	}
}

// Config holds encryption configuration
type Config struct {
	Algorithm Algorithm
	Key       []byte // Encryption key (32 bytes for AES-256)
	// For key derivation from password
	Password string
	Salt     []byte
}

// DefaultConfig returns a default encryption configuration (no encryption)
func DefaultConfig() *Config {
	return &Config{
		Algorithm: AlgorithmNone,
	}
}

// NewConfigFromPassword creates a config with key derived from password
func NewConfigFromPassword(password string, algorithm Algorithm) (*Config, error) {
	if password == "" {
		return nil, fmt.Errorf("password cannot be empty")
	}

	// Generate a random salt
	salt := make([]byte, 32)
	if _, err := rand.Read(salt); err != nil {
		return nil, fmt.Errorf("failed to generate salt: %w", err)
	}

	// Derive key using PBKDF2
	key := pbkdf2.Key([]byte(password), salt, 100000, 32, sha256.New)

	return &Config{
		Algorithm: algorithm,
		Key:       key,
		Password:  password,
		Salt:      salt,
	}, nil
}

// NewConfigFromKey creates a config with an explicit encryption key
func NewConfigFromKey(key []byte, algorithm Algorithm) (*Config, error) {
	if algorithm != AlgorithmNone && len(key) != 32 {
		return nil, fmt.Errorf("key must be 32 bytes for AES-256, got %d", len(key))
	}

	return &Config{
		Algorithm: algorithm,
		Key:       key,
	}, nil
}

// Encryptor handles data encryption and decryption
type Encryptor struct {
	config *Config
	block  cipher.Block
}

// NewEncryptor creates a new encryptor
func NewEncryptor(config *Config) (*Encryptor, error) {
	if config == nil {
		config = DefaultConfig()
	}

	e := &Encryptor{
		config: config,
	}

	// Initialize cipher block if encryption is enabled
	if config.Algorithm != AlgorithmNone {
		if len(config.Key) != 32 {
			return nil, fmt.Errorf("encryption key must be 32 bytes, got %d", len(config.Key))
		}

		block, err := aes.NewCipher(config.Key)
		if err != nil {
			return nil, fmt.Errorf("failed to create cipher: %w", err)
		}
		e.block = block
	}

	return e, nil
}

// Encrypt encrypts data using the configured algorithm
// Returns: encrypted data with algorithm-specific metadata prepended
func (e *Encryptor) Encrypt(plaintext []byte) ([]byte, error) {
	if e.config.Algorithm == AlgorithmNone {
		return plaintext, nil
	}

	switch e.config.Algorithm {
	case AlgorithmAES256GCM:
		return e.encryptGCM(plaintext)
	case AlgorithmAES256CTR:
		return e.encryptCTR(plaintext)
	default:
		return nil, fmt.Errorf("unsupported encryption algorithm: %v", e.config.Algorithm)
	}
}

// Decrypt decrypts data using the configured algorithm
func (e *Encryptor) Decrypt(ciphertext []byte) ([]byte, error) {
	if e.config.Algorithm == AlgorithmNone {
		return ciphertext, nil
	}

	switch e.config.Algorithm {
	case AlgorithmAES256GCM:
		return e.decryptGCM(ciphertext)
	case AlgorithmAES256CTR:
		return e.decryptCTR(ciphertext)
	default:
		return nil, fmt.Errorf("unsupported encryption algorithm: %v", e.config.Algorithm)
	}
}

// encryptGCM encrypts using AES-256-GCM (provides authentication)
// Format: [12-byte nonce][ciphertext+tag]
func (e *Encryptor) encryptGCM(plaintext []byte) ([]byte, error) {
	gcm, err := cipher.NewGCM(e.block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	// Generate random nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Encrypt and authenticate
	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

// decryptGCM decrypts using AES-256-GCM
func (e *Encryptor) decryptGCM(ciphertext []byte) ([]byte, error) {
	gcm, err := cipher.NewGCM(e.block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	// Extract nonce and ciphertext
	nonce := ciphertext[:nonceSize]
	ciphertext = ciphertext[nonceSize:]

	// Decrypt and verify
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("decryption failed: %w", err)
	}

	return plaintext, nil
}

// encryptCTR encrypts using AES-256-CTR (no authentication)
// Format: [16-byte IV][ciphertext]
func (e *Encryptor) encryptCTR(plaintext []byte) ([]byte, error) {
	// Generate random IV
	iv := make([]byte, aes.BlockSize)
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return nil, fmt.Errorf("failed to generate IV: %w", err)
	}

	// Create CTR stream
	stream := cipher.NewCTR(e.block, iv)

	// Encrypt
	ciphertext := make([]byte, len(plaintext))
	stream.XORKeyStream(ciphertext, plaintext)

	// Prepend IV
	result := make([]byte, aes.BlockSize+len(ciphertext))
	copy(result[:aes.BlockSize], iv)
	copy(result[aes.BlockSize:], ciphertext)

	return result, nil
}

// decryptCTR decrypts using AES-256-CTR
func (e *Encryptor) decryptCTR(ciphertext []byte) ([]byte, error) {
	if len(ciphertext) < aes.BlockSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	// Extract IV and ciphertext
	iv := ciphertext[:aes.BlockSize]
	ciphertext = ciphertext[aes.BlockSize:]

	// Create CTR stream
	stream := cipher.NewCTR(e.block, iv)

	// Decrypt
	plaintext := make([]byte, len(ciphertext))
	stream.XORKeyStream(plaintext, ciphertext)

	return plaintext, nil
}

// GetConfig returns the encryptor's configuration
func (e *Encryptor) GetConfig() *Config {
	return e.config
}
