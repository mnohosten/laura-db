package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/pbkdf2"
)

var (
	// ErrInvalidCredentials is returned when username or password is incorrect
	ErrInvalidCredentials = errors.New("invalid username or password")
	// ErrUserExists is returned when trying to create a user that already exists
	ErrUserExists = errors.New("user already exists")
	// ErrUserNotFound is returned when user is not found
	ErrUserNotFound = errors.New("user not found")
	// ErrInvalidNonce is returned when client nonce is invalid
	ErrInvalidNonce = errors.New("invalid nonce")
	// ErrInvalidProof is returned when client proof verification fails
	ErrInvalidProof = errors.New("invalid client proof")
	// ErrPermissionDenied is returned when user lacks required permission
	ErrPermissionDenied = errors.New("permission denied")
)

const (
	// SCRAM-SHA-256 parameters
	saltLength     = 16
	iterationCount = 4096
	keyLength      = 32
)

// Role represents a user role with associated permissions
type Role string

const (
	// RoleAdmin has full access to all operations
	RoleAdmin Role = "admin"
	// RoleReadWrite can read and write data
	RoleReadWrite Role = "readWrite"
	// RoleRead can only read data
	RoleRead Role = "read"
)

// Permission represents an operation permission
type Permission string

const (
	PermissionRead         Permission = "read"
	PermissionWrite        Permission = "write"
	PermissionCreateIndex  Permission = "createIndex"
	PermissionDropIndex    Permission = "dropIndex"
	PermissionCreateCollection Permission = "createCollection"
	PermissionDropCollection   Permission = "dropCollection"
	PermissionManageUsers  Permission = "manageUsers"
	PermissionViewStats    Permission = "viewStats"
)

// rolePermissions maps roles to their permissions
var rolePermissions = map[Role][]Permission{
	RoleAdmin: {
		PermissionRead,
		PermissionWrite,
		PermissionCreateIndex,
		PermissionDropIndex,
		PermissionCreateCollection,
		PermissionDropCollection,
		PermissionManageUsers,
		PermissionViewStats,
	},
	RoleReadWrite: {
		PermissionRead,
		PermissionWrite,
		PermissionCreateIndex,
		PermissionDropIndex,
		PermissionViewStats,
	},
	RoleRead: {
		PermissionRead,
		PermissionViewStats,
	},
}

// User represents a database user
type User struct {
	Username     string
	Salt         []byte
	StoredKey    []byte
	ServerKey    []byte
	Role         Role
	CreatedAt    time.Time
	LastModified time.Time
}

// Session represents an authenticated session
type Session struct {
	Username  string
	Role      Role
	ExpiresAt time.Time
	Token     string
}

// AuthManager manages users and authentication
type AuthManager struct {
	mu       sync.RWMutex
	users    map[string]*User
	sessions map[string]*Session

	// Session configuration
	sessionTTL time.Duration
}

// NewAuthManager creates a new authentication manager
func NewAuthManager() *AuthManager {
	am := &AuthManager{
		users:      make(map[string]*User),
		sessions:   make(map[string]*Session),
		sessionTTL: 24 * time.Hour, // Default 24 hour session
	}

	// Create default admin user (password: "admin")
	// In production, this should be changed immediately
	_ = am.CreateUser("admin", "admin", RoleAdmin)

	return am
}

// SetSessionTTL sets the session time-to-live duration
func (am *AuthManager) SetSessionTTL(ttl time.Duration) {
	am.mu.Lock()
	defer am.mu.Unlock()
	am.sessionTTL = ttl
}

// CreateUser creates a new user with the given username, password, and role
func (am *AuthManager) CreateUser(username, password string, role Role) error {
	am.mu.Lock()
	defer am.mu.Unlock()

	// Check if user already exists
	if _, exists := am.users[username]; exists {
		return ErrUserExists
	}

	// Generate random salt
	salt := make([]byte, saltLength)
	if _, err := rand.Read(salt); err != nil {
		return fmt.Errorf("failed to generate salt: %w", err)
	}

	// Compute SCRAM-SHA-256 keys
	saltedPassword := pbkdf2.Key([]byte(password), salt, iterationCount, keyLength, sha256.New)
	clientKey := hmacSHA256(saltedPassword, []byte("Client Key"))
	storedKey := sha256Hash(clientKey)
	serverKey := hmacSHA256(saltedPassword, []byte("Server Key"))

	// Create user
	user := &User{
		Username:     username,
		Salt:         salt,
		StoredKey:    storedKey,
		ServerKey:    serverKey,
		Role:         role,
		CreatedAt:    time.Now(),
		LastModified: time.Now(),
	}

	am.users[username] = user
	return nil
}

// DeleteUser deletes a user
func (am *AuthManager) DeleteUser(username string) error {
	am.mu.Lock()
	defer am.mu.Unlock()

	if _, exists := am.users[username]; !exists {
		return ErrUserNotFound
	}

	delete(am.users, username)

	// Invalidate all sessions for this user
	for token, session := range am.sessions {
		if session.Username == username {
			delete(am.sessions, token)
		}
	}

	return nil
}

// UpdateUserPassword updates a user's password
func (am *AuthManager) UpdateUserPassword(username, newPassword string) error {
	am.mu.Lock()
	defer am.mu.Unlock()

	user, exists := am.users[username]
	if !exists {
		return ErrUserNotFound
	}

	// Generate new salt
	salt := make([]byte, saltLength)
	if _, err := rand.Read(salt); err != nil {
		return fmt.Errorf("failed to generate salt: %w", err)
	}

	// Compute new SCRAM-SHA-256 keys
	saltedPassword := pbkdf2.Key([]byte(newPassword), salt, iterationCount, keyLength, sha256.New)
	clientKey := hmacSHA256(saltedPassword, []byte("Client Key"))
	storedKey := sha256Hash(clientKey)
	serverKey := hmacSHA256(saltedPassword, []byte("Server Key"))

	// Update user
	user.Salt = salt
	user.StoredKey = storedKey
	user.ServerKey = serverKey
	user.LastModified = time.Now()

	// Invalidate all sessions for this user
	for token, session := range am.sessions {
		if session.Username == username {
			delete(am.sessions, token)
		}
	}

	return nil
}

// UpdateUserRole updates a user's role
func (am *AuthManager) UpdateUserRole(username string, role Role) error {
	am.mu.Lock()
	defer am.mu.Unlock()

	user, exists := am.users[username]
	if !exists {
		return ErrUserNotFound
	}

	user.Role = role
	user.LastModified = time.Now()

	// Update role in active sessions
	for _, session := range am.sessions {
		if session.Username == username {
			session.Role = role
		}
	}

	return nil
}

// GetUser retrieves a user (without sensitive data)
func (am *AuthManager) GetUser(username string) (*User, error) {
	am.mu.RLock()
	defer am.mu.RUnlock()

	user, exists := am.users[username]
	if !exists {
		return nil, ErrUserNotFound
	}

	// Return a copy without sensitive data
	return &User{
		Username:     user.Username,
		Role:         user.Role,
		CreatedAt:    user.CreatedAt,
		LastModified: user.LastModified,
	}, nil
}

// ListUsers returns all usernames and roles
func (am *AuthManager) ListUsers() []struct {
	Username string
	Role     Role
} {
	am.mu.RLock()
	defer am.mu.RUnlock()

	users := make([]struct {
		Username string
		Role     Role
	}, 0, len(am.users))

	for _, user := range am.users {
		users = append(users, struct {
			Username string
			Role     Role
		}{
			Username: user.Username,
			Role:     user.Role,
		})
	}

	return users
}

// Authenticate performs SCRAM-SHA-256 authentication and returns a session token
// This is a simplified version for basic auth; full SCRAM requires challenge-response
func (am *AuthManager) Authenticate(username, password string) (string, error) {
	am.mu.Lock()
	defer am.mu.Unlock()

	user, exists := am.users[username]
	if !exists {
		return "", ErrInvalidCredentials
	}

	// Compute salted password
	saltedPassword := pbkdf2.Key([]byte(password), user.Salt, iterationCount, keyLength, sha256.New)
	clientKey := hmacSHA256(saltedPassword, []byte("Client Key"))
	storedKey := sha256Hash(clientKey)

	// Compare stored key
	if !hmac.Equal(storedKey, user.StoredKey) {
		return "", ErrInvalidCredentials
	}

	// Generate session token
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", fmt.Errorf("failed to generate token: %w", err)
	}
	token := base64.URLEncoding.EncodeToString(tokenBytes)

	// Create session
	session := &Session{
		Username:  username,
		Role:      user.Role,
		ExpiresAt: time.Now().Add(am.sessionTTL),
		Token:     token,
	}

	am.sessions[token] = session
	return token, nil
}

// ValidateSession validates a session token and returns the session
func (am *AuthManager) ValidateSession(token string) (*Session, error) {
	am.mu.RLock()
	defer am.mu.RUnlock()

	session, exists := am.sessions[token]
	if !exists {
		return nil, ErrInvalidCredentials
	}

	// Check expiration
	if time.Now().After(session.ExpiresAt) {
		return nil, ErrInvalidCredentials
	}

	return session, nil
}

// InvalidateSession invalidates a session token (logout)
func (am *AuthManager) InvalidateSession(token string) error {
	am.mu.Lock()
	defer am.mu.Unlock()

	delete(am.sessions, token)
	return nil
}

// HasPermission checks if a role has a specific permission
func (am *AuthManager) HasPermission(role Role, permission Permission) bool {
	permissions, exists := rolePermissions[role]
	if !exists {
		return false
	}

	for _, p := range permissions {
		if p == permission {
			return true
		}
	}
	return false
}

// CheckPermission checks if a session has a specific permission
func (am *AuthManager) CheckPermission(token string, permission Permission) error {
	session, err := am.ValidateSession(token)
	if err != nil {
		return err
	}

	if !am.HasPermission(session.Role, permission) {
		return ErrPermissionDenied
	}

	return nil
}

// CleanupExpiredSessions removes expired sessions
func (am *AuthManager) CleanupExpiredSessions() {
	am.mu.Lock()
	defer am.mu.Unlock()

	now := time.Now()
	for token, session := range am.sessions {
		if now.After(session.ExpiresAt) {
			delete(am.sessions, token)
		}
	}
}

// StartCleanupRoutine starts a background goroutine to clean up expired sessions
func (am *AuthManager) StartCleanupRoutine(interval time.Duration) chan struct{} {
	stop := make(chan struct{})

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				am.CleanupExpiredSessions()
			case <-stop:
				return
			}
		}
	}()

	return stop
}

// ParseAuthHeader parses an Authorization header (Bearer token)
func ParseAuthHeader(header string) (string, error) {
	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 || parts[0] != "Bearer" {
		return "", errors.New("invalid authorization header")
	}
	return parts[1], nil
}

// Helper functions

func hmacSHA256(key, data []byte) []byte {
	h := hmac.New(sha256.New, key)
	h.Write(data)
	return h.Sum(nil)
}

func sha256Hash(data []byte) []byte {
	h := sha256.Sum256(data)
	return h[:]
}
