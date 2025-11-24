package auth

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestHandleLogin(t *testing.T) {
	am := NewAuthManager()
	_ = am.CreateUser("testuser", "password", RoleRead)

	// Valid login
	body := `{"username":"testuser","password":"password"}`
	req := httptest.NewRequest("POST", "/auth/login", bytes.NewBufferString(body))
	w := httptest.NewRecorder()

	am.HandleLogin(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response LoginResponse
	json.NewDecoder(w.Body).Decode(&response)

	if response.Token == "" {
		t.Error("Expected non-empty token")
	}
	if response.Role != RoleRead {
		t.Errorf("Expected role RoleRead, got %v", response.Role)
	}

	// Invalid password
	body = `{"username":"testuser","password":"wrongpassword"}`
	req = httptest.NewRequest("POST", "/auth/login", bytes.NewBufferString(body))
	w = httptest.NewRecorder()

	am.HandleLogin(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}

	// Missing username
	body = `{"password":"password"}`
	req = httptest.NewRequest("POST", "/auth/login", bytes.NewBufferString(body))
	w = httptest.NewRecorder()

	am.HandleLogin(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}

	// Invalid JSON
	req = httptest.NewRequest("POST", "/auth/login", bytes.NewBufferString("invalid json"))
	w = httptest.NewRecorder()

	am.HandleLogin(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestHandleLogout(t *testing.T) {
	am := NewAuthManager()
	_ = am.CreateUser("testuser", "password", RoleRead)
	token, _ := am.Authenticate("testuser", "password")

	// Valid logout
	req := httptest.NewRequest("POST", "/auth/logout", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	am.HandleLogout(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Token should be invalidated
	_, err := am.ValidateSession(token)
	if err != ErrInvalidCredentials {
		t.Error("Token should be invalidated")
	}

	// Missing authorization header
	req = httptest.NewRequest("POST", "/auth/logout", nil)
	w = httptest.NewRecorder()

	am.HandleLogout(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestHandleCreateUser(t *testing.T) {
	am := NewAuthManager()

	// Valid creation
	body := `{"username":"newuser","password":"password","role":"readWrite"}`
	req := httptest.NewRequest("POST", "/users", bytes.NewBufferString(body))
	w := httptest.NewRecorder()

	am.HandleCreateUser(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d", w.Code)
	}

	// Verify user was created
	user, err := am.GetUser("newuser")
	if err != nil {
		t.Fatalf("User was not created: %v", err)
	}
	if user.Role != RoleReadWrite {
		t.Errorf("Expected role RoleReadWrite, got %v", user.Role)
	}

	// Duplicate user
	req = httptest.NewRequest("POST", "/users", bytes.NewBufferString(body))
	w = httptest.NewRecorder()

	am.HandleCreateUser(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("Expected status 409, got %d", w.Code)
	}

	// Invalid role
	body = `{"username":"baduser","password":"password","role":"invalid"}`
	req = httptest.NewRequest("POST", "/users", bytes.NewBufferString(body))
	w = httptest.NewRecorder()

	am.HandleCreateUser(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}

	// Missing password
	body = `{"username":"baduser","role":"read"}`
	req = httptest.NewRequest("POST", "/users", bytes.NewBufferString(body))
	w = httptest.NewRecorder()

	am.HandleCreateUser(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestHandleGetUser(t *testing.T) {
	am := NewAuthManager()
	_ = am.CreateUser("testuser", "password", RoleReadWrite)

	// Create router with chi to test URL params
	r := chi.NewRouter()
	r.Get("/users/{username}", am.HandleGetUser)

	// Valid request
	req := httptest.NewRequest("GET", "/users/testuser", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response UserResponse
	json.NewDecoder(w.Body).Decode(&response)

	if response.Username != "testuser" {
		t.Errorf("Expected username 'testuser', got %s", response.Username)
	}
	if response.Role != RoleReadWrite {
		t.Errorf("Expected role RoleReadWrite, got %v", response.Role)
	}

	// Non-existent user
	req = httptest.NewRequest("GET", "/users/nonexistent", nil)
	w = httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

func TestHandleListUsers(t *testing.T) {
	am := NewAuthManager()
	_ = am.CreateUser("user1", "password", RoleRead)
	_ = am.CreateUser("user2", "password", RoleReadWrite)

	req := httptest.NewRequest("GET", "/users", nil)
	w := httptest.NewRecorder()

	am.HandleListUsers(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var users []map[string]interface{}
	json.NewDecoder(w.Body).Decode(&users)

	if len(users) != 3 { // admin + user1 + user2
		t.Errorf("Expected 3 users, got %d", len(users))
	}
}

func TestHandleDeleteUser(t *testing.T) {
	am := NewAuthManager()
	_ = am.CreateUser("tempuser", "password", RoleRead)

	// Create router with chi
	r := chi.NewRouter()
	r.Delete("/users/{username}", am.HandleDeleteUser)

	// Valid deletion
	req := httptest.NewRequest("DELETE", "/users/tempuser", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Verify user was deleted
	_, err := am.GetUser("tempuser")
	if err != ErrUserNotFound {
		t.Error("User should have been deleted")
	}

	// Delete non-existent user
	req = httptest.NewRequest("DELETE", "/users/nonexistent", nil)
	w = httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

func TestHandleUpdatePassword(t *testing.T) {
	am := NewAuthManager()
	_ = am.CreateUser("testuser", "oldpassword", RoleRead)

	// Create router with chi
	r := chi.NewRouter()
	r.Put("/users/{username}/password", am.HandleUpdatePassword)

	// Valid update
	body := `{"newPassword":"newpassword"}`
	req := httptest.NewRequest("PUT", "/users/testuser/password", bytes.NewBufferString(body))
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Verify new password works
	_, err := am.Authenticate("testuser", "newpassword")
	if err != nil {
		t.Error("New password should work")
	}

	// Old password should not work
	_, err = am.Authenticate("testuser", "oldpassword")
	if err != ErrInvalidCredentials {
		t.Error("Old password should not work")
	}

	// Missing new password
	body = `{}`
	req = httptest.NewRequest("PUT", "/users/testuser/password", bytes.NewBufferString(body))
	w = httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}

	// Non-existent user
	body = `{"newPassword":"password"}`
	req = httptest.NewRequest("PUT", "/users/nonexistent/password", bytes.NewBufferString(body))
	w = httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

func TestHandleUpdateRole(t *testing.T) {
	am := NewAuthManager()
	_ = am.CreateUser("testuser", "password", RoleRead)

	// Create router with chi
	r := chi.NewRouter()
	r.Put("/users/{username}/role", am.HandleUpdateRole)

	// Valid update
	body := `{"role":"admin"}`
	req := httptest.NewRequest("PUT", "/users/testuser/role", bytes.NewBufferString(body))
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Verify role was updated
	user, _ := am.GetUser("testuser")
	if user.Role != RoleAdmin {
		t.Errorf("Expected role RoleAdmin, got %v", user.Role)
	}

	// Invalid role
	body = `{"role":"invalid"}`
	req = httptest.NewRequest("PUT", "/users/testuser/role", bytes.NewBufferString(body))
	w = httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}

	// Non-existent user
	body = `{"role":"read"}`
	req = httptest.NewRequest("PUT", "/users/nonexistent/role", bytes.NewBufferString(body))
	w = httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

func BenchmarkHandleLogin(b *testing.B) {
	am := NewAuthManager()
	_ = am.CreateUser("testuser", "password", RoleRead)

	body := `{"username":"testuser","password":"password"}`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("POST", "/auth/login", bytes.NewBufferString(body))
		w := httptest.NewRecorder()
		am.HandleLogin(w, req)
	}
}

func BenchmarkHandleListUsers(b *testing.B) {
	am := NewAuthManager()
	for i := 0; i < 100; i++ {
		_ = am.CreateUser("user"+string(rune(i)), "password", RoleRead)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/users", nil)
		w := httptest.NewRecorder()
		am.HandleListUsers(w, req)
	}
}
