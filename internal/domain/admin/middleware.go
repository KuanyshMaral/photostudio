package admin

import (
	"net/http"
	"strings"

	jwtsvc "photostudio/internal/pkg/jwt"

	"photostudio/internal/pkg/chicontext"
	"photostudio/internal/pkg/response"
)

// ChiAdminJWTAuth is the chi-native equivalent of AdminJWTAuth.
func ChiAdminJWTAuth(jwtService *jwtsvc.Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				response.CustomError(w, r, http.StatusUnauthorized, "AUTH_HEADER_MISSING", "Authorization header is required")
				return
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
				response.CustomError(w, r, http.StatusUnauthorized, "INVALID_AUTH_FORMAT", "Authorization header must be 'Bearer <token>'")
				return
			}

			claims, err := jwtService.ValidateToken(parts[1])
			if err != nil {
				response.CustomError(w, r, http.StatusUnauthorized, "INVALID_TOKEN", "Invalid or expired token")
				return
			}

			if claims.AdminID == "" && claims.Role != "admin" && claims.Role != "super_admin" {
				response.CustomError(w, r, http.StatusForbidden, "FORBIDDEN", "Admin access required")
				return
			}

			ctx := r.Context()
			ctx = chicontext.SetAdminID(ctx, claims.AdminID)
			if claims.UserID > 0 {
				ctx = chicontext.SetUserID(ctx, claims.UserID)
			}
			ctx = chicontext.SetRole(ctx, claims.Role)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// AdminJWTAuth is the gin-compatible version kept for backward compatibility during migration.
// It delegates to the same JWT logic.
// NOTE: This is only used during the Gin transition period. Remove once admin is fully on chi.
// Keep the gin version in middleware package if needed by remaining gin routes.
