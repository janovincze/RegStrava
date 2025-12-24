package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/regstrava/regstrava/internal/domain"
	"github.com/regstrava/regstrava/internal/service"
)

// Context keys
type contextKey string

const (
	FunderIDKey contextKey = "funder_id"
	FunderKey   contextKey = "funder"
)

// AuthMiddleware provides authentication middleware
type AuthMiddleware struct {
	authService *service.AuthService
}

// NewAuthMiddleware creates a new auth middleware
func NewAuthMiddleware(authService *service.AuthService) *AuthMiddleware {
	return &AuthMiddleware{
		authService: authService,
	}
}

// Authenticate validates API key or JWT token
func (m *AuthMiddleware) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var funder *domain.Funder
		var funderID *uuid.UUID

		// Try API key first
		apiKey := r.Header.Get("X-API-Key")
		if apiKey != "" {
			f, err := m.authService.ValidateAPIKey(r.Context(), apiKey)
			if err == nil && f != nil {
				funder = f
				funderID = &f.ID
			}
		}

		// Try Bearer token if no API key
		if funder == nil {
			authHeader := r.Header.Get("Authorization")
			if strings.HasPrefix(authHeader, "Bearer ") {
				token := strings.TrimPrefix(authHeader, "Bearer ")
				id, err := m.authService.ValidateToken(token)
				if err == nil && id != nil {
					funderID = id
				}
			}
		}

		// If no valid auth found, reject
		if funderID == nil {
			http.Error(w, `{"error": "Unauthorized"}`, http.StatusUnauthorized)
			return
		}

		// Add funder info to context
		ctx := context.WithValue(r.Context(), FunderIDKey, funderID)
		if funder != nil {
			ctx = context.WithValue(ctx, FunderKey, funder)
		}

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetFunderID extracts funder ID from context
func GetFunderID(ctx context.Context) *uuid.UUID {
	if id, ok := ctx.Value(FunderIDKey).(*uuid.UUID); ok {
		return id
	}
	return nil
}

// GetFunder extracts funder from context
func GetFunder(ctx context.Context) *domain.Funder {
	if funder, ok := ctx.Value(FunderKey).(*domain.Funder); ok {
		return funder
	}
	return nil
}
