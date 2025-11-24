package auth

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
)

// LoginRequest represents a login request
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// LoginResponse represents a login response
type LoginResponse struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expiresAt"`
	Role      Role      `json:"role"`
}

// CreateUserRequest represents a request to create a user
type CreateUserRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Role     Role   `json:"role"`
}

// UpdatePasswordRequest represents a request to update a password
type UpdatePasswordRequest struct {
	NewPassword string `json:"newPassword"`
}

// UpdateRoleRequest represents a request to update a user's role
type UpdateRoleRequest struct {
	Role Role `json:"role"`
}

// UserResponse represents a user in the response
type UserResponse struct {
	Username     string    `json:"username"`
	Role         Role      `json:"role"`
	CreatedAt    time.Time `json:"createdAt"`
	LastModified time.Time `json:"lastModified"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error string `json:"error"`
}

// SuccessResponse represents a success response
type SuccessResponse struct {
	Message string `json:"message"`
}

// HandleLogin handles user login
func (am *AuthManager) HandleLogin(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Username == "" || req.Password == "" {
		writeError(w, "Username and password are required", http.StatusBadRequest)
		return
	}

	token, err := am.Authenticate(req.Username, req.Password)
	if err != nil {
		writeError(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	session, _ := am.ValidateSession(token)

	response := LoginResponse{
		Token:     token,
		ExpiresAt: session.ExpiresAt,
		Role:      session.Role,
	}

	writeJSON(w, response, http.StatusOK)
}

// HandleLogout handles user logout
func (am *AuthManager) HandleLogout(w http.ResponseWriter, r *http.Request) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		writeError(w, "Missing authorization header", http.StatusBadRequest)
		return
	}

	token, err := ParseAuthHeader(authHeader)
	if err != nil {
		writeError(w, "Invalid authorization header", http.StatusBadRequest)
		return
	}

	_ = am.InvalidateSession(token)

	writeJSON(w, SuccessResponse{Message: "Logged out successfully"}, http.StatusOK)
}

// HandleCreateUser handles user creation
func (am *AuthManager) HandleCreateUser(w http.ResponseWriter, r *http.Request) {
	var req CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Username == "" || req.Password == "" {
		writeError(w, "Username and password are required", http.StatusBadRequest)
		return
	}

	// Validate role
	if req.Role != RoleAdmin && req.Role != RoleReadWrite && req.Role != RoleRead {
		writeError(w, "Invalid role. Must be 'admin', 'readWrite', or 'read'", http.StatusBadRequest)
		return
	}

	err := am.CreateUser(req.Username, req.Password, req.Role)
	if err != nil {
		if err == ErrUserExists {
			writeError(w, "User already exists", http.StatusConflict)
		} else {
			writeError(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	writeJSON(w, SuccessResponse{Message: "User created successfully"}, http.StatusCreated)
}

// HandleGetUser handles getting a user
func (am *AuthManager) HandleGetUser(w http.ResponseWriter, r *http.Request) {
	username := chi.URLParam(r, "username")

	user, err := am.GetUser(username)
	if err != nil {
		if err == ErrUserNotFound {
			writeError(w, "User not found", http.StatusNotFound)
		} else {
			writeError(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	response := UserResponse{
		Username:     user.Username,
		Role:         user.Role,
		CreatedAt:    user.CreatedAt,
		LastModified: user.LastModified,
	}

	writeJSON(w, response, http.StatusOK)
}

// HandleListUsers handles listing all users
func (am *AuthManager) HandleListUsers(w http.ResponseWriter, r *http.Request) {
	users := am.ListUsers()

	response := make([]map[string]interface{}, len(users))
	for i, u := range users {
		response[i] = map[string]interface{}{
			"username": u.Username,
			"role":     u.Role,
		}
	}

	writeJSON(w, response, http.StatusOK)
}

// HandleDeleteUser handles user deletion
func (am *AuthManager) HandleDeleteUser(w http.ResponseWriter, r *http.Request) {
	username := chi.URLParam(r, "username")

	err := am.DeleteUser(username)
	if err != nil {
		if err == ErrUserNotFound {
			writeError(w, "User not found", http.StatusNotFound)
		} else {
			writeError(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	writeJSON(w, SuccessResponse{Message: "User deleted successfully"}, http.StatusOK)
}

// HandleUpdatePassword handles password updates
func (am *AuthManager) HandleUpdatePassword(w http.ResponseWriter, r *http.Request) {
	username := chi.URLParam(r, "username")

	var req UpdatePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.NewPassword == "" {
		writeError(w, "New password is required", http.StatusBadRequest)
		return
	}

	err := am.UpdateUserPassword(username, req.NewPassword)
	if err != nil {
		if err == ErrUserNotFound {
			writeError(w, "User not found", http.StatusNotFound)
		} else {
			writeError(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	writeJSON(w, SuccessResponse{Message: "Password updated successfully"}, http.StatusOK)
}

// HandleUpdateRole handles role updates
func (am *AuthManager) HandleUpdateRole(w http.ResponseWriter, r *http.Request) {
	username := chi.URLParam(r, "username")

	var req UpdateRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate role
	if req.Role != RoleAdmin && req.Role != RoleReadWrite && req.Role != RoleRead {
		writeError(w, "Invalid role. Must be 'admin', 'readWrite', or 'read'", http.StatusBadRequest)
		return
	}

	err := am.UpdateUserRole(username, req.Role)
	if err != nil {
		if err == ErrUserNotFound {
			writeError(w, "User not found", http.StatusNotFound)
		} else {
			writeError(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	writeJSON(w, SuccessResponse{Message: "Role updated successfully"}, http.StatusOK)
}

// Helper functions

func writeJSON(w http.ResponseWriter, data interface{}, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, message string, status int) {
	writeJSON(w, ErrorResponse{Error: message}, status)
}
