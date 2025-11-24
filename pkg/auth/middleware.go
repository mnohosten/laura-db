package auth

import (
	"context"
	"net/http"
)

// contextKey is a custom type for context keys to avoid collisions
type contextKey string

const (
	// ContextKeySession is the context key for the authenticated session
	ContextKeySession contextKey = "auth_session"
)

// Middleware returns an HTTP middleware that enforces authentication
func (am *AuthManager) Middleware(requiredPermission Permission) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract token from Authorization header
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, "Unauthorized: missing authorization header", http.StatusUnauthorized)
				return
			}

			token, err := ParseAuthHeader(authHeader)
			if err != nil {
				http.Error(w, "Unauthorized: invalid authorization header", http.StatusUnauthorized)
				return
			}

			// Validate session
			session, err := am.ValidateSession(token)
			if err != nil {
				http.Error(w, "Unauthorized: invalid or expired token", http.StatusUnauthorized)
				return
			}

			// Check permission
			if !am.HasPermission(session.Role, requiredPermission) {
				http.Error(w, "Forbidden: insufficient permissions", http.StatusForbidden)
				return
			}

			// Add session to context
			ctx := context.WithValue(r.Context(), ContextKeySession, session)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// OptionalMiddleware returns an HTTP middleware that adds session to context if present
// but doesn't require authentication
func (am *AuthManager) OptionalMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Try to extract token
			authHeader := r.Header.Get("Authorization")
			if authHeader != "" {
				token, err := ParseAuthHeader(authHeader)
				if err == nil {
					session, err := am.ValidateSession(token)
					if err == nil {
						// Add session to context
						ctx := context.WithValue(r.Context(), ContextKeySession, session)
						r = r.WithContext(ctx)
					}
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}

// GetSession extracts the session from the request context
func GetSession(r *http.Request) (*Session, bool) {
	session, ok := r.Context().Value(ContextKeySession).(*Session)
	return session, ok
}
