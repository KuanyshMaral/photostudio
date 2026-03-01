// Package chicontext provides typed context keys and helper functions for
// reading Chi middleware-injected values (user_id, role, etc.) from
// net/http request contexts.
//
// It is intentionally kept dependency-free to avoid import cycles between
// internal/middleware and domain packages.
package chicontext

import "context"

// contextKey is an unexported type to prevent collisions with other packages.
type contextKey int

const (
	KeyUserID       contextKey = iota // int64 — authenticated user's database ID
	KeyRole                           // string — user's role (client, studio_owner, admin, ...)
	KeyAdminID                        // int64 — admin's database ID (for admin-token flows)
	KeyIsAdminToken                   // bool
	KeyMWorkUserID                    // string — MWork UUID
	KeyMWorkRole                      // string — MWork role
)

// SetUserID returns a new context with the authenticated user's int64 ID stored.
func SetUserID(ctx context.Context, id int64) context.Context {
	return context.WithValue(ctx, KeyUserID, id)
}

// SetRole returns a new context with the role string stored.
func SetRole(ctx context.Context, role string) context.Context {
	return context.WithValue(ctx, KeyRole, role)
}

// SetAdminID returns a new context with the admin's string UUID stored.
func SetAdminID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, KeyAdminID, id)
}

// SetIsAdminToken returns a new context flagging this as an admin-token request.
func SetIsAdminToken(ctx context.Context, v bool) context.Context {
	return context.WithValue(ctx, KeyIsAdminToken, v)
}

// SetMWorkUserID returns a new context with the MWork UUID stored.
func SetMWorkUserID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, KeyMWorkUserID, id)
}

// SetMWorkRole returns a new context with the MWork role stored.
func SetMWorkRole(ctx context.Context, role string) context.Context {
	return context.WithValue(ctx, KeyMWorkRole, role)
}

// UserIDFromCtx returns the authenticated user's int64 ID, or 0 if absent.
func UserIDFromCtx(ctx context.Context) int64 {
	v, _ := ctx.Value(KeyUserID).(int64)
	return v
}

// RoleFromCtx returns the role string, or "" if absent.
func RoleFromCtx(ctx context.Context) string {
	v, _ := ctx.Value(KeyRole).(string)
	return v
}

// AdminIDFromCtx returns the admin's string UUID, or "" if absent.
func AdminIDFromCtx(ctx context.Context) string {
	v, _ := ctx.Value(KeyAdminID).(string)
	return v
}

// IsAdminTokenFromCtx returns true when current request used an admin JWT.
func IsAdminTokenFromCtx(ctx context.Context) bool {
	v, _ := ctx.Value(KeyIsAdminToken).(bool)
	return v
}

// MWorkUserIDFromCtx returns the MWork UUID, or "" if absent.
func MWorkUserIDFromCtx(ctx context.Context) string {
	v, _ := ctx.Value(KeyMWorkUserID).(string)
	return v
}

// MWorkRoleFromCtx returns the MWork role, or "" if absent.
func MWorkRoleFromCtx(ctx context.Context) string {
	v, _ := ctx.Value(KeyMWorkRole).(string)
	return v
}
