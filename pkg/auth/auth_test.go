package auth

import (
	"strings"
	"testing"
	"time"
)

func TestNewAuthManager(t *testing.T) {
	am := NewAuthManager()
	if am == nil {
		t.Fatal("NewAuthManager returned nil")
	}

	// Should have default admin user
	user, err := am.GetUser("admin")
	if err != nil {
		t.Fatalf("Default admin user not created: %v", err)
	}

	if user.Role != RoleAdmin {
		t.Errorf("Expected admin role, got %v", user.Role)
	}
}

func TestCreateUser(t *testing.T) {
	am := NewAuthManager()

	// Create a new user
	err := am.CreateUser("testuser", "password123", RoleReadWrite)
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// Verify user was created
	user, err := am.GetUser("testuser")
	if err != nil {
		t.Fatalf("Failed to get user: %v", err)
	}

	if user.Username != "testuser" {
		t.Errorf("Expected username 'testuser', got %s", user.Username)
	}

	if user.Role != RoleReadWrite {
		t.Errorf("Expected role RoleReadWrite, got %v", user.Role)
	}

	// Try to create duplicate user
	err = am.CreateUser("testuser", "password456", RoleRead)
	if err != ErrUserExists {
		t.Errorf("Expected ErrUserExists, got %v", err)
	}
}

func TestDeleteUser(t *testing.T) {
	am := NewAuthManager()

	// Create and delete user
	_ = am.CreateUser("tempuser", "password", RoleRead)
	err := am.DeleteUser("tempuser")
	if err != nil {
		t.Fatalf("Failed to delete user: %v", err)
	}

	// Verify user was deleted
	_, err = am.GetUser("tempuser")
	if err != ErrUserNotFound {
		t.Errorf("Expected ErrUserNotFound, got %v", err)
	}

	// Try to delete non-existent user
	err = am.DeleteUser("nonexistent")
	if err != ErrUserNotFound {
		t.Errorf("Expected ErrUserNotFound, got %v", err)
	}
}

func TestUpdateUserPassword(t *testing.T) {
	am := NewAuthManager()
	_ = am.CreateUser("testuser", "oldpassword", RoleRead)

	// Update password
	err := am.UpdateUserPassword("testuser", "newpassword")
	if err != nil {
		t.Fatalf("Failed to update password: %v", err)
	}

	// Old password should not work
	_, err = am.Authenticate("testuser", "oldpassword")
	if err != ErrInvalidCredentials {
		t.Errorf("Old password should not work, got error: %v", err)
	}

	// New password should work
	token, err := am.Authenticate("testuser", "newpassword")
	if err != nil {
		t.Errorf("New password should work: %v", err)
	}
	if token == "" {
		t.Error("Expected non-empty token")
	}
}

func TestUpdateUserRole(t *testing.T) {
	am := NewAuthManager()
	_ = am.CreateUser("testuser", "password", RoleRead)

	// Update role
	err := am.UpdateUserRole("testuser", RoleAdmin)
	if err != nil {
		t.Fatalf("Failed to update role: %v", err)
	}

	// Verify role was updated
	user, _ := am.GetUser("testuser")
	if user.Role != RoleAdmin {
		t.Errorf("Expected role RoleAdmin, got %v", user.Role)
	}

	// Try to update non-existent user
	err = am.UpdateUserRole("nonexistent", RoleAdmin)
	if err != ErrUserNotFound {
		t.Errorf("Expected ErrUserNotFound, got %v", err)
	}
}

func TestListUsers(t *testing.T) {
	am := NewAuthManager()
	_ = am.CreateUser("user1", "password", RoleRead)
	_ = am.CreateUser("user2", "password", RoleReadWrite)

	users := am.ListUsers()
	if len(users) != 3 { // admin + user1 + user2
		t.Errorf("Expected 3 users, got %d", len(users))
	}

	// Verify user names
	userMap := make(map[string]Role)
	for _, u := range users {
		userMap[u.Username] = u.Role
	}

	if userMap["admin"] != RoleAdmin {
		t.Error("admin user not found or has wrong role")
	}
	if userMap["user1"] != RoleRead {
		t.Error("user1 not found or has wrong role")
	}
	if userMap["user2"] != RoleReadWrite {
		t.Error("user2 not found or has wrong role")
	}
}

func TestAuthenticate(t *testing.T) {
	am := NewAuthManager()
	_ = am.CreateUser("testuser", "correctpassword", RoleReadWrite)

	// Test correct password
	token, err := am.Authenticate("testuser", "correctpassword")
	if err != nil {
		t.Fatalf("Authentication failed: %v", err)
	}
	if token == "" {
		t.Error("Expected non-empty token")
	}

	// Test incorrect password
	_, err = am.Authenticate("testuser", "wrongpassword")
	if err != ErrInvalidCredentials {
		t.Errorf("Expected ErrInvalidCredentials, got %v", err)
	}

	// Test non-existent user
	_, err = am.Authenticate("nonexistent", "password")
	if err != ErrInvalidCredentials {
		t.Errorf("Expected ErrInvalidCredentials, got %v", err)
	}
}

func TestValidateSession(t *testing.T) {
	am := NewAuthManager()
	_ = am.CreateUser("testuser", "password", RoleReadWrite)

	// Authenticate and get token
	token, err := am.Authenticate("testuser", "password")
	if err != nil {
		t.Fatalf("Authentication failed: %v", err)
	}

	// Validate session
	session, err := am.ValidateSession(token)
	if err != nil {
		t.Fatalf("Session validation failed: %v", err)
	}

	if session.Username != "testuser" {
		t.Errorf("Expected username 'testuser', got %s", session.Username)
	}
	if session.Role != RoleReadWrite {
		t.Errorf("Expected role RoleReadWrite, got %v", session.Role)
	}

	// Try invalid token
	_, err = am.ValidateSession("invalidtoken")
	if err != ErrInvalidCredentials {
		t.Errorf("Expected ErrInvalidCredentials, got %v", err)
	}
}

func TestInvalidateSession(t *testing.T) {
	am := NewAuthManager()
	_ = am.CreateUser("testuser", "password", RoleRead)

	// Authenticate
	token, _ := am.Authenticate("testuser", "password")

	// Invalidate session
	err := am.InvalidateSession(token)
	if err != nil {
		t.Fatalf("Failed to invalidate session: %v", err)
	}

	// Session should no longer be valid
	_, err = am.ValidateSession(token)
	if err != ErrInvalidCredentials {
		t.Errorf("Expected ErrInvalidCredentials, got %v", err)
	}
}

func TestSessionExpiration(t *testing.T) {
	am := NewAuthManager()
	am.SetSessionTTL(100 * time.Millisecond) // Very short TTL for testing
	_ = am.CreateUser("testuser", "password", RoleRead)

	// Authenticate
	token, _ := am.Authenticate("testuser", "password")

	// Session should be valid immediately
	_, err := am.ValidateSession(token)
	if err != nil {
		t.Errorf("Session should be valid: %v", err)
	}

	// Wait for expiration
	time.Sleep(150 * time.Millisecond)

	// Session should be expired
	_, err = am.ValidateSession(token)
	if err != ErrInvalidCredentials {
		t.Errorf("Expected ErrInvalidCredentials for expired session, got %v", err)
	}
}

func TestCleanupExpiredSessions(t *testing.T) {
	am := NewAuthManager()
	am.SetSessionTTL(100 * time.Millisecond)
	_ = am.CreateUser("testuser", "password", RoleRead)

	// Create session
	token, _ := am.Authenticate("testuser", "password")

	// Verify session exists
	am.mu.RLock()
	sessionCount := len(am.sessions)
	am.mu.RUnlock()

	if sessionCount == 0 {
		t.Fatal("Session not created")
	}

	// Wait for expiration
	time.Sleep(150 * time.Millisecond)

	// Clean up
	am.CleanupExpiredSessions()

	// Verify session was removed
	am.mu.RLock()
	sessionCount = len(am.sessions)
	am.mu.RUnlock()

	// Session should be removed (we check token specifically)
	_, exists := am.sessions[token]
	if exists {
		t.Error("Expired session was not cleaned up")
	}
}

func TestStartCleanupRoutine(t *testing.T) {
	am := NewAuthManager()
	am.SetSessionTTL(50 * time.Millisecond)
	_ = am.CreateUser("testuser", "password", RoleRead)

	// Start cleanup routine
	stop := am.StartCleanupRoutine(100 * time.Millisecond)
	defer close(stop)

	// Create session
	token, _ := am.Authenticate("testuser", "password")

	// Wait for expiration and cleanup
	time.Sleep(200 * time.Millisecond)

	// Session should be cleaned up
	_, err := am.ValidateSession(token)
	if err != ErrInvalidCredentials {
		t.Errorf("Expected ErrInvalidCredentials after cleanup, got %v", err)
	}
}

func TestHasPermission(t *testing.T) {
	am := NewAuthManager()

	tests := []struct {
		role       Role
		permission Permission
		expected   bool
	}{
		{RoleAdmin, PermissionRead, true},
		{RoleAdmin, PermissionWrite, true},
		{RoleAdmin, PermissionManageUsers, true},
		{RoleReadWrite, PermissionRead, true},
		{RoleReadWrite, PermissionWrite, true},
		{RoleReadWrite, PermissionManageUsers, false},
		{RoleRead, PermissionRead, true},
		{RoleRead, PermissionWrite, false},
		{RoleRead, PermissionManageUsers, false},
	}

	for _, tt := range tests {
		result := am.HasPermission(tt.role, tt.permission)
		if result != tt.expected {
			t.Errorf("HasPermission(%v, %v) = %v, expected %v",
				tt.role, tt.permission, result, tt.expected)
		}
	}
}

func TestCheckPermission(t *testing.T) {
	am := NewAuthManager()
	_ = am.CreateUser("readonly", "password", RoleRead)
	_ = am.CreateUser("readwrite", "password", RoleReadWrite)
	_ = am.CreateUser("adminuser", "password", RoleAdmin)

	// Authenticate users
	readToken, _ := am.Authenticate("readonly", "password")
	rwToken, _ := am.Authenticate("readwrite", "password")
	adminToken, _ := am.Authenticate("adminuser", "password")

	// Test read-only user
	err := am.CheckPermission(readToken, PermissionRead)
	if err != nil {
		t.Errorf("Read-only user should have read permission: %v", err)
	}

	err = am.CheckPermission(readToken, PermissionWrite)
	if err != ErrPermissionDenied {
		t.Errorf("Read-only user should not have write permission")
	}

	// Test read-write user
	err = am.CheckPermission(rwToken, PermissionRead)
	if err != nil {
		t.Errorf("Read-write user should have read permission: %v", err)
	}

	err = am.CheckPermission(rwToken, PermissionWrite)
	if err != nil {
		t.Errorf("Read-write user should have write permission: %v", err)
	}

	err = am.CheckPermission(rwToken, PermissionManageUsers)
	if err != ErrPermissionDenied {
		t.Errorf("Read-write user should not have manage users permission")
	}

	// Test admin user
	err = am.CheckPermission(adminToken, PermissionManageUsers)
	if err != nil {
		t.Errorf("Admin user should have manage users permission: %v", err)
	}

	// Test invalid token
	err = am.CheckPermission("invalidtoken", PermissionRead)
	if err != ErrInvalidCredentials {
		t.Errorf("Invalid token should return ErrInvalidCredentials")
	}
}

func TestParseAuthHeader(t *testing.T) {
	tests := []struct {
		header      string
		expectToken string
		expectError bool
	}{
		{"Bearer abc123", "abc123", false},
		{"Bearer token-with-dashes", "token-with-dashes", false},
		{"Basic abc123", "", true},
		{"Bearer", "", true},
		{"abc123", "", true},
		{"", "", true},
	}

	for _, tt := range tests {
		token, err := ParseAuthHeader(tt.header)
		if tt.expectError {
			if err == nil {
				t.Errorf("ParseAuthHeader(%q) expected error, got nil", tt.header)
			}
		} else {
			if err != nil {
				t.Errorf("ParseAuthHeader(%q) unexpected error: %v", tt.header, err)
			}
			if token != tt.expectToken {
				t.Errorf("ParseAuthHeader(%q) = %q, expected %q", tt.header, token, tt.expectToken)
			}
		}
	}
}

func TestDeleteUserInvalidatesSessions(t *testing.T) {
	am := NewAuthManager()
	_ = am.CreateUser("testuser", "password", RoleRead)

	// Create session
	token, _ := am.Authenticate("testuser", "password")

	// Verify session is valid
	_, err := am.ValidateSession(token)
	if err != nil {
		t.Fatalf("Session should be valid: %v", err)
	}

	// Delete user
	_ = am.DeleteUser("testuser")

	// Session should be invalid
	_, err = am.ValidateSession(token)
	if err != ErrInvalidCredentials {
		t.Errorf("Session should be invalid after user deletion")
	}
}

func TestUpdatePasswordInvalidatesSessions(t *testing.T) {
	am := NewAuthManager()
	_ = am.CreateUser("testuser", "oldpassword", RoleRead)

	// Create session
	token, _ := am.Authenticate("testuser", "oldpassword")

	// Verify session is valid
	_, err := am.ValidateSession(token)
	if err != nil {
		t.Fatalf("Session should be valid: %v", err)
	}

	// Update password
	_ = am.UpdateUserPassword("testuser", "newpassword")

	// Old session should be invalid
	_, err = am.ValidateSession(token)
	if err != ErrInvalidCredentials {
		t.Errorf("Session should be invalid after password update")
	}
}

func TestUpdateRoleUpdatesActiveSessions(t *testing.T) {
	am := NewAuthManager()
	_ = am.CreateUser("testuser", "password", RoleRead)

	// Authenticate
	token, _ := am.Authenticate("testuser", "password")

	// Verify initial role
	session, _ := am.ValidateSession(token)
	if session.Role != RoleRead {
		t.Fatalf("Expected role RoleRead, got %v", session.Role)
	}

	// Update role
	_ = am.UpdateUserRole("testuser", RoleAdmin)

	// Verify role was updated in active session
	session, _ = am.ValidateSession(token)
	if session.Role != RoleAdmin {
		t.Errorf("Expected role RoleAdmin in active session, got %v", session.Role)
	}
}

func TestConcurrentAuthentication(t *testing.T) {
	am := NewAuthManager()
	_ = am.CreateUser("testuser", "password", RoleRead)

	// Run concurrent authentications
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			token, err := am.Authenticate("testuser", "password")
			if err != nil {
				t.Errorf("Authentication failed: %v", err)
			}
			if token == "" {
				t.Error("Expected non-empty token")
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestSCRAMSHA256KeyGeneration(t *testing.T) {
	am := NewAuthManager()
	_ = am.CreateUser("user1", "password123", RoleRead)
	_ = am.CreateUser("user2", "password123", RoleRead)

	// Even with same password, salt should make keys different
	am.mu.RLock()
	user1 := am.users["user1"]
	user2 := am.users["user2"]
	am.mu.RUnlock()

	// Salts should be different
	if string(user1.Salt) == string(user2.Salt) {
		t.Error("Salts should be different for different users")
	}

	// Stored keys should be different
	if string(user1.StoredKey) == string(user2.StoredKey) {
		t.Error("Stored keys should be different even with same password")
	}

	// Keys should have correct length
	if len(user1.Salt) != saltLength {
		t.Errorf("Expected salt length %d, got %d", saltLength, len(user1.Salt))
	}
	if len(user1.StoredKey) != 32 { // SHA-256 output
		t.Errorf("Expected stored key length 32, got %d", len(user1.StoredKey))
	}
	if len(user1.ServerKey) != 32 {
		t.Errorf("Expected server key length 32, got %d", len(user1.ServerKey))
	}
}

func TestGetUserDoesNotExposeSensitiveData(t *testing.T) {
	am := NewAuthManager()
	_ = am.CreateUser("testuser", "password", RoleRead)

	user, _ := am.GetUser("testuser")

	// Sensitive data should not be exposed
	if user.Salt != nil {
		t.Error("Salt should not be exposed in GetUser")
	}
	if user.StoredKey != nil {
		t.Error("StoredKey should not be exposed in GetUser")
	}
	if user.ServerKey != nil {
		t.Error("ServerKey should not be exposed in GetUser")
	}

	// Public data should be present
	if user.Username != "testuser" {
		t.Error("Username should be present")
	}
	if user.Role != RoleRead {
		t.Error("Role should be present")
	}
}

func TestRolePermissions(t *testing.T) {
	// Verify role permission mappings are correct
	adminPerms := rolePermissions[RoleAdmin]
	if len(adminPerms) != 8 {
		t.Errorf("Expected 8 admin permissions, got %d", len(adminPerms))
	}

	rwPerms := rolePermissions[RoleReadWrite]
	if len(rwPerms) != 5 {
		t.Errorf("Expected 5 read-write permissions, got %d", len(rwPerms))
	}

	readPerms := rolePermissions[RoleRead]
	if len(readPerms) != 2 {
		t.Errorf("Expected 2 read permissions, got %d", len(readPerms))
	}

	// Verify admin has all permissions
	hasManageUsers := false
	for _, p := range adminPerms {
		if p == PermissionManageUsers {
			hasManageUsers = true
		}
	}
	if !hasManageUsers {
		t.Error("Admin should have manage users permission")
	}
}

func TestParseAuthHeaderEdgeCases(t *testing.T) {
	// Test with multiple spaces - this will actually parse " token" as the token
	token, err := ParseAuthHeader("Bearer  token")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if token != " token" {
		t.Errorf("Expected ' token', got '%s'", token)
	}

	// Test with lowercase bearer
	_, err = ParseAuthHeader("bearer token")
	if err == nil {
		t.Error("Should fail with lowercase 'bearer'")
	}

	// Test with empty token
	token, err = ParseAuthHeader("Bearer ")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if token != "" {
		t.Errorf("Expected empty token, got '%s'", token)
	}
}

func BenchmarkCreateUser(b *testing.B) {
	am := NewAuthManager()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		username := "user" + strings.Repeat("a", i%100)
		_ = am.CreateUser(username, "password", RoleRead)
	}
}

func BenchmarkAuthenticate(b *testing.B) {
	am := NewAuthManager()
	_ = am.CreateUser("testuser", "password", RoleRead)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = am.Authenticate("testuser", "password")
	}
}

func BenchmarkValidateSession(b *testing.B) {
	am := NewAuthManager()
	_ = am.CreateUser("testuser", "password", RoleRead)
	token, _ := am.Authenticate("testuser", "password")
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = am.ValidateSession(token)
	}
}

func BenchmarkCheckPermission(b *testing.B) {
	am := NewAuthManager()
	_ = am.CreateUser("testuser", "password", RoleReadWrite)
	token, _ := am.Authenticate("testuser", "password")
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = am.CheckPermission(token, PermissionRead)
	}
}
