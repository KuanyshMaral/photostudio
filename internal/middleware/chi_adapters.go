package middleware

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"photostudio/internal/pkg/chicontext"
	jwtpkg "photostudio/internal/pkg/jwt"

	"photostudio/internal/domain/auth"
)

// ---------------------------------------------------------------------------
// ChiJWTAuth — Chi-native equivalent of JWTAuth(jwtService).
// ---------------------------------------------------------------------------
func ChiJWTAuth(jwtService *jwtpkg.Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")

			if authHeader == "" {
				upgrade := strings.EqualFold(r.Header.Get("Upgrade"), "websocket")
				connUpgrade := strings.Contains(strings.ToLower(r.Header.Get("Connection")), "upgrade")
				path := r.URL.Path
				isWS := strings.HasSuffix(path, "/notifications/ws") || strings.HasSuffix(path, "/chats/ws")

				if upgrade && connUpgrade && isWS {
					qToken := r.URL.Query().Get("token")
					if len(qToken) > maxWebSocketTokenLength {
						chiWriteError(w, http.StatusUnauthorized, "INVALID_TOKEN", "Token is too long")
						return
					}
					if qToken != "" {
						authHeader = "Bearer " + qToken
					}
				}
			}

			if authHeader == "" {
				chiWriteError(w, http.StatusUnauthorized, "AUTH_HEADER_MISSING", "Authorization header is required")
				return
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
				chiWriteError(w, http.StatusUnauthorized, "INVALID_AUTH_FORMAT", "Authorization header must be 'Bearer <token>'")
				return
			}

			claims, err := jwtService.ValidateToken(parts[1])
			if err != nil {
				chiWriteError(w, http.StatusUnauthorized, "INVALID_TOKEN", "Invalid or expired token")
				return
			}

			ctx := r.Context()

			if claims.TokenType == "access_admin" {
				if !chiAdminTokenAllowed(r) {
					chiWriteError(w, http.StatusForbidden, "FORBIDDEN", "Admin token is not allowed for this endpoint")
					return
				}
				ctx = chicontext.SetAdminID(ctx, claims.AdminID)
				ctx = chicontext.SetRole(ctx, claims.Role)
				ctx = chicontext.SetIsAdminToken(ctx, true)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			if claims.UserID <= 0 {
				chiWriteError(w, http.StatusUnauthorized, "INVALID_TOKEN", "Token subject is invalid")
				return
			}

			ctx = chicontext.SetUserID(ctx, claims.UserID)
			ctx = chicontext.SetRole(ctx, claims.Role)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// ---------------------------------------------------------------------------
// ChiRequireRole — Chi-native equivalent of RequireRole(role).
// ---------------------------------------------------------------------------
func ChiRequireRole(requiredRoles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			role := chicontext.RoleFromCtx(r.Context())
			if role == "" {
				chiWriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Role not found in token")
				return
			}
			for _, allowed := range requiredRoles {
				if role == allowed {
					next.ServeHTTP(w, r)
					return
				}
			}
			chiWriteError(w, http.StatusForbidden, "FORBIDDEN", "Access denied: insufficient permissions")
		})
	}
}

// syncEnabled checks if MWork sync is configured
func syncEnabled() bool {
	return os.Getenv("MWORK_SYNC_TOKEN") != ""
}

// ---------------------------------------------------------------------------
// ChiInternalTokenAuth — Chi-native equivalent of InternalTokenAuth().
// ---------------------------------------------------------------------------
func ChiInternalTokenAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !syncEnabled() {
			chiWriteError(w, http.StatusForbidden, "AUTH_INVALID", "MWork sync disabled")
			return
		}

		allowed := strings.TrimSpace(os.Getenv("MWORK_SYNC_ALLOWED_IPS"))
		if allowed != "" {
			clientIP := r.RemoteAddr
			if idx := strings.LastIndex(clientIP, ":"); idx != -1 {
				clientIP = clientIP[:idx]
			}
			clientIP = strings.TrimPrefix(clientIP, "[")
			clientIP = strings.TrimSuffix(clientIP, "]")
			found := false
			for _, ip := range strings.Split(allowed, ",") {
				if strings.TrimSpace(ip) == clientIP {
					found = true
					break
				}
			}
			if !found {
				chiWriteError(w, http.StatusForbidden, "AUTH_INVALID", "IP not allowed")
				return
			}
		}

		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			chiWriteError(w, http.StatusUnauthorized, "AUTH_MISSING", "Authorization header is required")
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			chiWriteError(w, http.StatusUnauthorized, "AUTH_INVALID", "Authorization header must be 'Bearer <token>'")
			return
		}

		expected := os.Getenv("MWORK_SYNC_TOKEN")
		if expected == "" {
			expected = os.Getenv("PHOTO_STUDIO_INTERNAL_TOKEN")
		}
		if expected == "" || parts[1] != expected {
			chiWriteError(w, http.StatusForbidden, "AUTH_INVALID", "Invalid internal token")
			return
		}

		next.ServeHTTP(w, r)
	})
}

// ---------------------------------------------------------------------------
// ChiMWorkUserAuth — Chi-native equivalent of MWorkUserAuth(userRepo).
// ---------------------------------------------------------------------------
func ChiMWorkUserAuth(userRepo *auth.UserRepository) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			parts := strings.SplitN(authHeader, " ", 2)
			expected := os.Getenv("MWORK_SYNC_TOKEN")
			if expected == "" {
				expected = os.Getenv("PHOTO_STUDIO_INTERNAL_TOKEN")
			}
			if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" || parts[1] != expected || expected == "" {
				chiWriteError(w, http.StatusUnauthorized, "AUTH_INVALID", "Invalid or missing internal token")
				return
			}

			mworkUserID := r.Header.Get("X-MWork-User-ID")
			if mworkUserID == "" {
				chiWriteError(w, http.StatusBadRequest, "MWORK_USER_ID_MISSING", "X-MWork-User-ID header is required")
				return
			}
			if _, err := uuid.Parse(mworkUserID); err != nil {
				chiWriteError(w, http.StatusBadRequest, "INVALID_USER_ID", "X-MWork-User-ID must be a valid UUID")
				return
			}

			user, err := userRepo.GetByMworkUserID(r.Context(), mworkUserID)
			if err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					chiWriteError(w, http.StatusUnauthorized, "USER_NOT_SYNCED", "User not found in PhotoStudio. Please contact support if this persists.")
					return
				}
				chiWriteError(w, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to lookup user")
				return
			}

			ctx := r.Context()
			ctx = chicontext.SetUserID(ctx, user.ID)
			ctx = chicontext.SetRole(ctx, string(user.Role))
			ctx = chicontext.SetMWorkUserID(ctx, mworkUserID)
			ctx = chicontext.SetMWorkRole(ctx, user.MworkRole)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

func chiWriteError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"success": false,
		"error": map[string]any{
			"code":    code,
			"message": message,
		},
	})
}

func chiAdminTokenAllowed(r *http.Request) bool {
	routePattern := chi.RouteContext(r.Context()).RoutePattern()
	method := r.Method
	if routePattern == "/api/v1/chats/{id}/members" && method == http.MethodPost {
		return true
	}
	if routePattern == "/api/v1/chats/{id}/members/{user_id}" && method == http.MethodDelete {
		return true
	}
	path := r.URL.Path
	if method == http.MethodPost {
		return strings.HasPrefix(path, "/api/v1/chats/") && strings.HasSuffix(path, "/members")
	}
	if method == http.MethodDelete {
		return strings.HasPrefix(path, "/api/v1/chats/") && strings.Contains(path, "/members/")
	}
	return false
}
