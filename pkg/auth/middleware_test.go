package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMiddleware(t *testing.T) {
	am := NewAuthManager()
	_ = am.CreateUser("testuser", "password", RoleReadWrite)
	token, _ := am.Authenticate("testuser", "password")

	// Create a test handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session, ok := GetSession(r)
		if !ok {
			t.Error("Session not found in context")
		}
		if session.Username != "testuser" {
			t.Errorf("Expected username 'testuser', got %s", session.Username)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Wrap with middleware
	protected := am.Middleware(PermissionRead)(handler)

	// Test with valid token
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	protected.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestMiddleware_MissingAuthHeader(t *testing.T) {
	am := NewAuthManager()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Handler should not be called")
	})

	protected := am.Middleware(PermissionRead)(handler)

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	protected.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
}

func TestMiddleware_InvalidAuthHeader(t *testing.T) {
	am := NewAuthManager()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Handler should not be called")
	})

	protected := am.Middleware(PermissionRead)(handler)

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "InvalidFormat token")
	w := httptest.NewRecorder()

	protected.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
}

func TestMiddleware_InvalidToken(t *testing.T) {
	am := NewAuthManager()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Handler should not be called")
	})

	protected := am.Middleware(PermissionRead)(handler)

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer invalidtoken")
	w := httptest.NewRecorder()

	protected.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
}

func TestMiddleware_InsufficientPermission(t *testing.T) {
	am := NewAuthManager()
	_ = am.CreateUser("readonly", "password", RoleRead)
	token, _ := am.Authenticate("readonly", "password")

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Handler should not be called")
	})

	// Require write permission (read-only user doesn't have this)
	protected := am.Middleware(PermissionWrite)(handler)

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	protected.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("Expected status 403, got %d", w.Code)
	}
}

func TestOptionalMiddleware(t *testing.T) {
	am := NewAuthManager()
	_ = am.CreateUser("testuser", "password", RoleRead)
	token, _ := am.Authenticate("testuser", "password")

	// Create a test handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session, ok := GetSession(r)
		if ok {
			w.Write([]byte("Authenticated: " + session.Username))
		} else {
			w.Write([]byte("Anonymous"))
		}
	})

	// Wrap with optional middleware
	wrapped := am.OptionalMiddleware()(handler)

	// Test with valid token
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	wrapped.ServeHTTP(w, req)

	if w.Body.String() != "Authenticated: testuser" {
		t.Errorf("Expected 'Authenticated: testuser', got %s", w.Body.String())
	}

	// Test without token
	req = httptest.NewRequest("GET", "/test", nil)
	w = httptest.NewRecorder()

	wrapped.ServeHTTP(w, req)

	if w.Body.String() != "Anonymous" {
		t.Errorf("Expected 'Anonymous', got %s", w.Body.String())
	}
}

func TestOptionalMiddleware_InvalidToken(t *testing.T) {
	am := NewAuthManager()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session, ok := GetSession(r)
		if ok {
			t.Error("Session should not be present with invalid token")
		}
		if session != nil {
			t.Error("Session should be nil")
		}
		w.WriteHeader(http.StatusOK)
	})

	wrapped := am.OptionalMiddleware()(handler)

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer invalidtoken")
	w := httptest.NewRecorder()

	wrapped.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestGetSession(t *testing.T) {
	am := NewAuthManager()
	_ = am.CreateUser("testuser", "password", RoleRead)
	token, _ := am.Authenticate("testuser", "password")

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session, ok := GetSession(r)
		if !ok {
			t.Error("Expected session to be present")
		}
		if session.Username != "testuser" {
			t.Errorf("Expected username 'testuser', got %s", session.Username)
		}
	})

	protected := am.Middleware(PermissionRead)(handler)

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	protected.ServeHTTP(w, req)
}

func TestGetSession_NotPresent(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session, ok := GetSession(r)
		if ok {
			t.Error("Expected session to not be present")
		}
		if session != nil {
			t.Error("Expected session to be nil")
		}
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)
}

func BenchmarkMiddleware(b *testing.B) {
	am := NewAuthManager()
	_ = am.CreateUser("testuser", "password", RoleRead)
	token, _ := am.Authenticate("testuser", "password")

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	protected := am.Middleware(PermissionRead)(handler)

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		protected.ServeHTTP(w, req)
	}
}

func BenchmarkOptionalMiddleware(b *testing.B) {
	am := NewAuthManager()
	_ = am.CreateUser("testuser", "password", RoleRead)
	token, _ := am.Authenticate("testuser", "password")

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrapped := am.OptionalMiddleware()(handler)

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		wrapped.ServeHTTP(w, req)
	}
}
