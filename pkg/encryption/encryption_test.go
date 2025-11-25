package encryption

import (
	"bytes"
	"crypto/rand"
	"os"
	"path/filepath"
	"testing"

	"github.com/mnohosten/laura-db/pkg/storage"
)

func TestAlgorithmString(t *testing.T) {
	tests := []struct {
		algorithm Algorithm
		expected  string
	}{
		{AlgorithmAES256GCM, "AES-256-GCM"},
		{AlgorithmAES256CTR, "AES-256-CTR"},
		{AlgorithmNone, "None"},
		{Algorithm(99), "Unknown"},
	}

	for _, tt := range tests {
		if got := tt.algorithm.String(); got != tt.expected {
			t.Errorf("Algorithm.String() = %v, want %v", got, tt.expected)
		}
	}
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()
	if config.Algorithm != AlgorithmNone {
		t.Errorf("DefaultConfig() algorithm = %v, want %v", config.Algorithm, AlgorithmNone)
	}
}

func TestNewConfigFromPassword(t *testing.T) {
	tests := []struct {
		name      string
		password  string
		algorithm Algorithm
		wantErr   bool
	}{
		{"Valid password with GCM", "test-password-123", AlgorithmAES256GCM, false},
		{"Valid password with CTR", "another-password", AlgorithmAES256CTR, false},
		{"Empty password", "", AlgorithmAES256GCM, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, err := NewConfigFromPassword(tt.password, tt.algorithm)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewConfigFromPassword() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if config.Algorithm != tt.algorithm {
					t.Errorf("NewConfigFromPassword() algorithm = %v, want %v", config.Algorithm, tt.algorithm)
				}
				if len(config.Key) != 32 {
					t.Errorf("NewConfigFromPassword() key length = %d, want 32", len(config.Key))
				}
				if len(config.Salt) != 32 {
					t.Errorf("NewConfigFromPassword() salt length = %d, want 32", len(config.Salt))
				}
				if config.Password != tt.password {
					t.Errorf("NewConfigFromPassword() password = %v, want %v", config.Password, tt.password)
				}
			}
		})
	}
}

func TestNewConfigFromKey(t *testing.T) {
	validKey := make([]byte, 32)
	rand.Read(validKey)

	tests := []struct {
		name      string
		key       []byte
		algorithm Algorithm
		wantErr   bool
	}{
		{"Valid key with GCM", validKey, AlgorithmAES256GCM, false},
		{"Valid key with CTR", validKey, AlgorithmAES256CTR, false},
		{"Invalid key length", make([]byte, 16), AlgorithmAES256GCM, true},
		{"None algorithm with any key", nil, AlgorithmNone, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, err := NewConfigFromKey(tt.key, tt.algorithm)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewConfigFromKey() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if config.Algorithm != tt.algorithm {
					t.Errorf("NewConfigFromKey() algorithm = %v, want %v", config.Algorithm, tt.algorithm)
				}
			}
		})
	}
}

func TestNewEncryptor(t *testing.T) {
	validKey := make([]byte, 32)
	rand.Read(validKey)

	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{"Nil config", nil, false},
		{"Valid GCM config", &Config{Algorithm: AlgorithmAES256GCM, Key: validKey}, false},
		{"Valid CTR config", &Config{Algorithm: AlgorithmAES256CTR, Key: validKey}, false},
		{"None algorithm", &Config{Algorithm: AlgorithmNone}, false},
		{"Invalid key length", &Config{Algorithm: AlgorithmAES256GCM, Key: make([]byte, 16)}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encryptor, err := NewEncryptor(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewEncryptor() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && encryptor == nil {
				t.Error("NewEncryptor() returned nil encryptor")
			}
		})
	}
}

func TestEncryptDecryptGCM(t *testing.T) {
	config, err := NewConfigFromPassword("test-password", AlgorithmAES256GCM)
	if err != nil {
		t.Fatalf("Failed to create config: %v", err)
	}

	encryptor, err := NewEncryptor(config)
	if err != nil {
		t.Fatalf("Failed to create encryptor: %v", err)
	}

	tests := []struct {
		name      string
		plaintext []byte
	}{
		{"Empty data", []byte{}},
		{"Small data", []byte("Hello, World!")},
		{"Medium data", bytes.Repeat([]byte("A"), 1000)},
		{"Large data", bytes.Repeat([]byte("B"), 10000)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Encrypt
			ciphertext, err := encryptor.Encrypt(tt.plaintext)
			if err != nil {
				t.Fatalf("Encrypt() error = %v", err)
			}

			// Verify ciphertext is different (unless empty)
			if len(tt.plaintext) > 0 && bytes.Equal(ciphertext, tt.plaintext) {
				t.Error("Ciphertext should be different from plaintext")
			}

			// Decrypt
			decrypted, err := encryptor.Decrypt(ciphertext)
			if err != nil {
				t.Fatalf("Decrypt() error = %v", err)
			}

			// Verify decrypted matches original
			if !bytes.Equal(decrypted, tt.plaintext) {
				t.Errorf("Decrypted data does not match original")
			}
		})
	}
}

func TestEncryptDecryptCTR(t *testing.T) {
	config, err := NewConfigFromPassword("test-password", AlgorithmAES256CTR)
	if err != nil {
		t.Fatalf("Failed to create config: %v", err)
	}

	encryptor, err := NewEncryptor(config)
	if err != nil {
		t.Fatalf("Failed to create encryptor: %v", err)
	}

	tests := []struct {
		name      string
		plaintext []byte
	}{
		{"Empty data", []byte{}},
		{"Small data", []byte("Hello, World!")},
		{"Medium data", bytes.Repeat([]byte("A"), 1000)},
		{"Large data", bytes.Repeat([]byte("B"), 10000)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Encrypt
			ciphertext, err := encryptor.Encrypt(tt.plaintext)
			if err != nil {
				t.Fatalf("Encrypt() error = %v", err)
			}

			// Verify ciphertext is different (unless empty)
			if len(tt.plaintext) > 0 && bytes.Equal(ciphertext, tt.plaintext) {
				t.Error("Ciphertext should be different from plaintext")
			}

			// Decrypt
			decrypted, err := encryptor.Decrypt(ciphertext)
			if err != nil {
				t.Fatalf("Decrypt() error = %v", err)
			}

			// Verify decrypted matches original
			if !bytes.Equal(decrypted, tt.plaintext) {
				t.Errorf("Decrypted data does not match original")
			}
		})
	}
}

func TestEncryptDecryptNone(t *testing.T) {
	config := DefaultConfig()
	encryptor, err := NewEncryptor(config)
	if err != nil {
		t.Fatalf("Failed to create encryptor: %v", err)
	}

	plaintext := []byte("This should not be encrypted")

	// Encrypt (should be a no-op)
	ciphertext, err := encryptor.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}

	// Should be the same
	if !bytes.Equal(ciphertext, plaintext) {
		t.Error("AlgorithmNone should not modify data")
	}

	// Decrypt (should be a no-op)
	decrypted, err := encryptor.Decrypt(ciphertext)
	if err != nil {
		t.Fatalf("Decrypt() error = %v", err)
	}

	// Should be the same
	if !bytes.Equal(decrypted, plaintext) {
		t.Error("AlgorithmNone should not modify data")
	}
}

func TestGCMAuthentication(t *testing.T) {
	config, err := NewConfigFromPassword("test-password", AlgorithmAES256GCM)
	if err != nil {
		t.Fatalf("Failed to create config: %v", err)
	}

	encryptor, err := NewEncryptor(config)
	if err != nil {
		t.Fatalf("Failed to create encryptor: %v", err)
	}

	plaintext := []byte("Authenticated data")

	// Encrypt
	ciphertext, err := encryptor.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}

	// Tamper with ciphertext
	if len(ciphertext) > 20 {
		ciphertext[20] ^= 0xFF
	}

	// Decrypt should fail due to authentication
	_, err = encryptor.Decrypt(ciphertext)
	if err == nil {
		t.Error("Decrypt() should fail with tampered ciphertext")
	}
}

func TestDecryptInvalidData(t *testing.T) {
	config, err := NewConfigFromPassword("test-password", AlgorithmAES256GCM)
	if err != nil {
		t.Fatalf("Failed to create config: %v", err)
	}

	encryptor, err := NewEncryptor(config)
	if err != nil {
		t.Fatalf("Failed to create encryptor: %v", err)
	}

	tests := []struct {
		name       string
		ciphertext []byte
	}{
		{"Too short", []byte("short")},
		{"Empty", []byte{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := encryptor.Decrypt(tt.ciphertext)
			if err == nil {
				t.Error("Decrypt() should fail with invalid ciphertext")
			}
		})
	}
}

func TestGetConfig(t *testing.T) {
	config, err := NewConfigFromPassword("test-password", AlgorithmAES256GCM)
	if err != nil {
		t.Fatalf("Failed to create config: %v", err)
	}

	encryptor, err := NewEncryptor(config)
	if err != nil {
		t.Fatalf("Failed to create encryptor: %v", err)
	}

	retrievedConfig := encryptor.GetConfig()
	if retrievedConfig.Algorithm != config.Algorithm {
		t.Errorf("GetConfig() algorithm = %v, want %v", retrievedConfig.Algorithm, config.Algorithm)
	}
}

func TestDifferentKeys(t *testing.T) {
	plaintext := []byte("Secret message")

	// Create first encryptor
	config1, _ := NewConfigFromPassword("password1", AlgorithmAES256GCM)
	encryptor1, _ := NewEncryptor(config1)

	// Encrypt with first key
	ciphertext, err := encryptor1.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}

	// Create second encryptor with different key
	config2, _ := NewConfigFromPassword("password2", AlgorithmAES256GCM)
	encryptor2, _ := NewEncryptor(config2)

	// Try to decrypt with wrong key
	_, err = encryptor2.Decrypt(ciphertext)
	if err == nil {
		t.Error("Decrypt() should fail with different key")
	}
}

func BenchmarkEncryptGCM(b *testing.B) {
	config, _ := NewConfigFromPassword("test-password", AlgorithmAES256GCM)
	encryptor, _ := NewEncryptor(config)
	data := bytes.Repeat([]byte("A"), 4096) // 4KB page

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = encryptor.Encrypt(data)
	}
}

func BenchmarkDecryptGCM(b *testing.B) {
	config, _ := NewConfigFromPassword("test-password", AlgorithmAES256GCM)
	encryptor, _ := NewEncryptor(config)
	data := bytes.Repeat([]byte("A"), 4096)
	ciphertext, _ := encryptor.Encrypt(data)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = encryptor.Decrypt(ciphertext)
	}
}

func BenchmarkEncryptCTR(b *testing.B) {
	config, _ := NewConfigFromPassword("test-password", AlgorithmAES256CTR)
	encryptor, _ := NewEncryptor(config)
	data := bytes.Repeat([]byte("A"), 4096)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = encryptor.Encrypt(data)
	}
}

func BenchmarkDecryptCTR(b *testing.B) {
	config, _ := NewConfigFromPassword("test-password", AlgorithmAES256CTR)
	encryptor, _ := NewEncryptor(config)
	data := bytes.Repeat([]byte("A"), 4096)
	ciphertext, _ := encryptor.Encrypt(data)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = encryptor.Decrypt(ciphertext)
	}
}

// TestDecryptCTR_ShortCiphertext tests error handling in decryptCTR
func TestDecryptCTR_ShortCiphertext(t *testing.T) {
	config, _ := NewConfigFromPassword("test-password", AlgorithmAES256CTR)
	encryptor, _ := NewEncryptor(config)

	// Try to decrypt data shorter than IV size
	shortData := []byte("short")
	_, err := encryptor.Decrypt(shortData)
	if err == nil {
		t.Error("Decrypt() should fail with short ciphertext for CTR mode")
	}
}

// TestEncrypt_UnknownAlgorithm tests error handling for unknown algorithm
func TestEncrypt_UnknownAlgorithm(t *testing.T) {
	// Create encryptor with unknown algorithm
	encryptor := &Encryptor{
		config: &Config{Algorithm: Algorithm(99)},
	}

	data := []byte("test data")
	_, err := encryptor.Encrypt(data)
	if err == nil {
		t.Error("Encrypt() should fail with unknown algorithm")
	}
}

// TestDecrypt_UnknownAlgorithm tests error handling for unknown algorithm
func TestDecrypt_UnknownAlgorithm(t *testing.T) {
	// Create encryptor with unknown algorithm
	encryptor := &Encryptor{
		config: &Config{Algorithm: Algorithm(99)},
	}

	data := []byte("test data")
	_, err := encryptor.Decrypt(data)
	if err == nil {
		t.Error("Decrypt() should fail with unknown algorithm")
	}
}

// TestNewConfigFromPassword_SaltGeneration tests salt generation
func TestNewConfigFromPassword_SaltGeneration(t *testing.T) {
	password := "test-password"

	// Create two configs with same password
	config1, err1 := NewConfigFromPassword(password, AlgorithmAES256GCM)
	config2, err2 := NewConfigFromPassword(password, AlgorithmAES256GCM)

	if err1 != nil || err2 != nil {
		t.Fatalf("Failed to create configs: %v, %v", err1, err2)
	}

	// Salts should be different (randomly generated)
	if bytes.Equal(config1.Salt, config2.Salt) {
		t.Error("Two configs with same password should have different salts")
	}

	// Keys should be different (derived from different salts)
	if bytes.Equal(config1.Key, config2.Key) {
		t.Error("Two configs with same password should have different keys (due to different salts)")
	}
}

// TestNewEncryptor_NilConfig tests NewEncryptor with nil config
func TestNewEncryptor_NilConfig(t *testing.T) {
	encryptor, err := NewEncryptor(nil)
	if err != nil {
		t.Errorf("NewEncryptor(nil) should not return error, got: %v", err)
	}

	if encryptor == nil {
		t.Error("NewEncryptor(nil) should return valid encryptor")
	}

	// Should use AlgorithmNone
	if encryptor.config.Algorithm != AlgorithmNone {
		t.Errorf("NewEncryptor(nil) algorithm = %v, want %v", encryptor.config.Algorithm, AlgorithmNone)
	}
}

// TestReadPage_SuccessfulRoundTrip tests successful write and read with encryption
func TestReadPage_SuccessfulRoundTrip(t *testing.T) {
	dataDir := filepath.Join(os.TempDir(), "test-round-trip")
	defer os.RemoveAll(dataDir)
	os.MkdirAll(dataDir, 0755)

	dataPath := filepath.Join(dataDir, "test.db")

	config, _ := NewConfigFromPassword("test-password", AlgorithmAES256GCM)
	edm, _ := NewEncryptedDiskManager(dataPath, config)

	// Normal operation should always have matching sizes
	pageID, _ := edm.AllocatePage()
	page := storage.NewPage(pageID, storage.PageTypeData)
	testData := []byte("test data for round trip")

	// Use a safe size that leaves room for encryption overhead
	maxDataSize := len(page.Data) - EncryptionOverhead - EncryptedPageHeaderSize
	if len(testData) < maxDataSize {
		copy(page.Data[:len(testData)], testData)
		page.Data = page.Data[:maxDataSize]
	}

	err := edm.WritePage(page)
	if err != nil {
		t.Fatalf("WritePage() error = %v", err)
	}

	edm.Sync()
	edm.Close()

	// Reopen and read
	edm2, _ := NewEncryptedDiskManager(dataPath, config)
	defer edm2.Close()

	// Reading should succeed with matching size
	readPage, err := edm2.ReadPage(pageID)
	if err != nil {
		t.Fatalf("ReadPage() error = %v", err)
	}

	if len(readPage.Data) < len(testData) {
		t.Error("ReadPage() returned page with truncated data")
	}

	// Verify data matches
	if string(readPage.Data[:len(testData)]) != string(testData) {
		t.Error("ReadPage() data mismatch after round trip")
	}
}
