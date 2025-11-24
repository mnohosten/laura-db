package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/mnohosten/laura-db/pkg/auth"
)

func main() {
	fmt.Println("LauraDB Authentication Demo")
	fmt.Println("============================")
	fmt.Println()

	// Demo 1: Basic User Management
	demo1_BasicUserManagement()

	// Demo 2: Authentication and Sessions
	demo2_AuthenticationAndSessions()

	// Demo 3: Role-Based Access Control
	demo3_RoleBasedAccessControl()

	// Demo 4: Session Management
	demo4_SessionManagement()

	// Demo 5: HTTP Server Integration
	demo5_HTTPServerIntegration()
}

func demo1_BasicUserManagement() {
	fmt.Println("Demo 1: Basic User Management")
	fmt.Println("------------------------------")

	// Create auth manager
	am := auth.NewAuthManager()

	// Create users with different roles
	fmt.Println("Creating users...")
	am.CreateUser("alice", "password123", auth.RoleAdmin)
	am.CreateUser("bob", "password456", auth.RoleReadWrite)
	am.CreateUser("charlie", "password789", auth.RoleRead)

	// List all users
	fmt.Println("\nAll users:")
	users := am.ListUsers()
	for _, u := range users {
		fmt.Printf("  - %s (%s)\n", u.Username, u.Role)
	}

	// Get specific user details
	fmt.Println("\nUser details for 'alice':")
	user, _ := am.GetUser("alice")
	fmt.Printf("  Username: %s\n", user.Username)
	fmt.Printf("  Role: %s\n", user.Role)
	fmt.Printf("  Created: %s\n", user.CreatedAt.Format(time.RFC3339))

	// Update user role
	fmt.Println("\nUpdating bob's role to admin...")
	am.UpdateUserRole("bob", auth.RoleAdmin)
	user, _ = am.GetUser("bob")
	fmt.Printf("  Bob's new role: %s\n", user.Role)

	// Delete user
	fmt.Println("\nDeleting charlie...")
	am.DeleteUser("charlie")
	users = am.ListUsers()
	fmt.Printf("  Remaining users: %d\n", len(users))

	fmt.Println()
}

func demo2_AuthenticationAndSessions() {
	fmt.Println("Demo 2: Authentication and Sessions")
	fmt.Println("------------------------------------")

	am := auth.NewAuthManager()
	am.CreateUser("testuser", "securepassword", auth.RoleReadWrite)

	// Successful authentication
	fmt.Println("Authenticating with correct password...")
	token, err := am.Authenticate("testuser", "securepassword")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("  ✓ Authentication successful\n")
	fmt.Printf("  Token: %s...\n", token[:20])

	// Validate session
	fmt.Println("\nValidating session...")
	session, err := am.ValidateSession(token)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("  ✓ Session valid\n")
	fmt.Printf("  Username: %s\n", session.Username)
	fmt.Printf("  Role: %s\n", session.Role)
	fmt.Printf("  Expires: %s\n", session.ExpiresAt.Format(time.RFC3339))

	// Failed authentication
	fmt.Println("\nAttempting authentication with wrong password...")
	_, err = am.Authenticate("testuser", "wrongpassword")
	if err != nil {
		fmt.Printf("  ✗ Authentication failed (expected): %v\n", err)
	}

	// Logout (invalidate session)
	fmt.Println("\nLogging out...")
	am.InvalidateSession(token)
	_, err = am.ValidateSession(token)
	if err != nil {
		fmt.Printf("  ✓ Session invalidated: %v\n", err)
	}

	fmt.Println()
}

func demo3_RoleBasedAccessControl() {
	fmt.Println("Demo 3: Role-Based Access Control")
	fmt.Println("----------------------------------")

	am := auth.NewAuthManager()
	am.CreateUser("admin_user", "password", auth.RoleAdmin)
	am.CreateUser("writer", "password", auth.RoleReadWrite)
	am.CreateUser("reader", "password", auth.RoleRead)

	// Authenticate users
	adminToken, _ := am.Authenticate("admin_user", "password")
	writerToken, _ := am.Authenticate("writer", "password")
	readerToken, _ := am.Authenticate("reader", "password")

	fmt.Println("Testing permissions:")

	// Test read permission
	fmt.Println("\n1. Read permission:")
	testPermission(am, adminToken, auth.PermissionRead, "Admin")
	testPermission(am, writerToken, auth.PermissionRead, "ReadWrite")
	testPermission(am, readerToken, auth.PermissionRead, "Read")

	// Test write permission
	fmt.Println("\n2. Write permission:")
	testPermission(am, adminToken, auth.PermissionWrite, "Admin")
	testPermission(am, writerToken, auth.PermissionWrite, "ReadWrite")
	testPermission(am, readerToken, auth.PermissionWrite, "Read")

	// Test manage users permission
	fmt.Println("\n3. Manage users permission:")
	testPermission(am, adminToken, auth.PermissionManageUsers, "Admin")
	testPermission(am, writerToken, auth.PermissionManageUsers, "ReadWrite")
	testPermission(am, readerToken, auth.PermissionManageUsers, "Read")

	// Test create collection permission
	fmt.Println("\n4. Create collection permission:")
	testPermission(am, adminToken, auth.PermissionCreateCollection, "Admin")
	testPermission(am, writerToken, auth.PermissionCreateCollection, "ReadWrite")
	testPermission(am, readerToken, auth.PermissionCreateCollection, "Read")

	fmt.Println()
}

func testPermission(am *auth.AuthManager, token string, perm auth.Permission, roleName string) {
	err := am.CheckPermission(token, perm)
	if err == nil {
		fmt.Printf("  ✓ %s has %s permission\n", roleName, perm)
	} else {
		fmt.Printf("  ✗ %s does NOT have %s permission\n", roleName, perm)
	}
}

func demo4_SessionManagement() {
	fmt.Println("Demo 4: Session Management")
	fmt.Println("--------------------------")

	// Create auth manager with short TTL for demo
	am := auth.NewAuthManager()
	am.SetSessionTTL(2 * time.Second)

	am.CreateUser("testuser", "password", auth.RoleRead)

	// Authenticate
	fmt.Println("Creating session...")
	token, _ := am.Authenticate("testuser", "password")
	fmt.Printf("  Token: %s...\n", token[:20])

	// Validate immediately
	session, err := am.ValidateSession(token)
	if err == nil {
		fmt.Printf("  ✓ Session valid immediately\n")
		fmt.Printf("  Expires in: %v\n", time.Until(session.ExpiresAt).Round(time.Second))
	}

	// Wait for expiration
	fmt.Println("\nWaiting for session expiration (2 seconds)...")
	time.Sleep(2500 * time.Millisecond)

	// Try to validate expired session
	_, err = am.ValidateSession(token)
	if err != nil {
		fmt.Printf("  ✗ Session expired (expected): %v\n", err)
	}

	// Test cleanup routine
	fmt.Println("\nTesting automatic session cleanup...")
	am.SetSessionTTL(100 * time.Millisecond)

	// Create multiple sessions
	token1, _ := am.Authenticate("testuser", "password")
	token2, _ := am.Authenticate("testuser", "password")
	token3, _ := am.Authenticate("testuser", "password")

	fmt.Printf("  Created 3 sessions\n")

	// Start cleanup routine
	stop := am.StartCleanupRoutine(200 * time.Millisecond)
	defer close(stop)

	// Wait for cleanup
	time.Sleep(400 * time.Millisecond)

	// Check if sessions are cleaned up
	_, err1 := am.ValidateSession(token1)
	_, err2 := am.ValidateSession(token2)
	_, err3 := am.ValidateSession(token3)

	if err1 != nil && err2 != nil && err3 != nil {
		fmt.Printf("  ✓ All expired sessions cleaned up automatically\n")
	}

	fmt.Println()
}

func demo5_HTTPServerIntegration() {
	fmt.Println("Demo 5: HTTP Server Integration")
	fmt.Println("--------------------------------")

	// Create auth manager
	am := auth.NewAuthManager()
	am.CreateUser("apiuser", "apipassword", auth.RoleReadWrite)

	// Start cleanup routine
	stop := am.StartCleanupRoutine(5 * time.Minute)
	defer close(stop)

	// Create router
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// Public routes
	r.Post("/auth/login", am.HandleLogin)
	r.Post("/auth/logout", am.HandleLogout)

	// Protected routes (require read permission)
	r.Route("/api", func(r chi.Router) {
		r.Use(am.Middleware(auth.PermissionRead))

		r.Get("/data", func(w http.ResponseWriter, r *http.Request) {
			session, _ := auth.GetSession(r)
			fmt.Fprintf(w, "Hello, %s! You have access to read data.\n", session.Username)
		})
	})

	// Protected routes (require write permission)
	r.Route("/api/write", func(r chi.Router) {
		r.Use(am.Middleware(auth.PermissionWrite))

		r.Post("/data", func(w http.ResponseWriter, r *http.Request) {
			session, _ := auth.GetSession(r)
			fmt.Fprintf(w, "Data created by %s\n", session.Username)
		})
	})

	// Admin routes (require manage users permission)
	r.Route("/admin", func(r chi.Router) {
		r.Use(am.Middleware(auth.PermissionManageUsers))

		r.Post("/users", am.HandleCreateUser)
		r.Get("/users", am.HandleListUsers)
		r.Get("/users/{username}", am.HandleGetUser)
		r.Put("/users/{username}/password", am.HandleUpdatePassword)
		r.Put("/users/{username}/role", am.HandleUpdateRole)
		r.Delete("/users/{username}", am.HandleDeleteUser)
	})

	// Optional authentication route
	r.Route("/public", func(r chi.Router) {
		r.Use(am.OptionalMiddleware())

		r.Get("/", func(w http.ResponseWriter, r *http.Request) {
			if session, ok := auth.GetSession(r); ok {
				fmt.Fprintf(w, "Hello, %s! (Authenticated)\n", session.Username)
			} else {
				fmt.Fprint(w, "Hello, guest! (Anonymous)\n")
			}
		})
	})

	fmt.Println("HTTP server configured with authentication")
	fmt.Println("\nEndpoints:")
	fmt.Println("  POST   /auth/login          - Public: Login")
	fmt.Println("  POST   /auth/logout         - Public: Logout")
	fmt.Println("  GET    /api/data            - Protected: Read data (requires read permission)")
	fmt.Println("  POST   /api/write/data      - Protected: Write data (requires write permission)")
	fmt.Println("  POST   /admin/users         - Protected: Create user (requires admin)")
	fmt.Println("  GET    /admin/users         - Protected: List users (requires admin)")
	fmt.Println("  GET    /admin/users/{name}  - Protected: Get user (requires admin)")
	fmt.Println("  PUT    /admin/users/{name}/password - Protected: Update password (requires admin)")
	fmt.Println("  PUT    /admin/users/{name}/role     - Protected: Update role (requires admin)")
	fmt.Println("  DELETE /admin/users/{name}  - Protected: Delete user (requires admin)")
	fmt.Println("  GET    /public              - Optional auth: Public endpoint")

	fmt.Println("\nExample usage:")
	fmt.Println("  # Login")
	fmt.Println("  curl -X POST http://localhost:8080/auth/login \\")
	fmt.Println("    -H 'Content-Type: application/json' \\")
	fmt.Println("    -d '{\"username\":\"apiuser\",\"password\":\"apipassword\"}'")
	fmt.Println()
	fmt.Println("  # Access protected endpoint")
	fmt.Println("  curl http://localhost:8080/api/data \\")
	fmt.Println("    -H 'Authorization: Bearer <token>'")
	fmt.Println()
	fmt.Println("  # Create user (admin only)")
	fmt.Println("  curl -X POST http://localhost:8080/admin/users \\")
	fmt.Println("    -H 'Authorization: Bearer <admin-token>' \\")
	fmt.Println("    -H 'Content-Type: application/json' \\")
	fmt.Println("    -d '{\"username\":\"newuser\",\"password\":\"password\",\"role\":\"read\"}'")

	fmt.Println("\nStarting server on http://localhost:8080...")
	fmt.Println("Press Ctrl+C to stop")
	fmt.Println()

	// Start server
	if err := http.ListenAndServe(":8080", r); err != nil {
		log.Fatal(err)
	}
}
