package auth

import "time"

type UserRole string

const (
	RoleClient      UserRole = "client"
	RoleStudioOwner UserRole = "studio_owner"
	RoleAdmin       UserRole = "admin"
)

type StudioStatus string

const (
	StatusPending  StudioStatus = "pending"
	StatusVerified StudioStatus = "verified"
	StatusRejected StudioStatus = "rejected"
	StatusBlocked  StudioStatus = "blocked"
)

type User struct {
	ID                  int64        `json:"id"`
	Email               string       `json:"email" validate:"required,email"`
	PasswordHash        string       `json:"-"`
	Role                UserRole     `json:"role"`
	Name                string       `json:"name"`
	Phone               string       `json:"phone,omitempty"`
	AvatarURL           string       `json:"avatar_url,omitempty"`
	EmailVerified       bool         `json:"email_verified"`
	EmailVerifiedAt     *time.Time   `json:"email_verified_at,omitempty"`
	IsBanned            bool         `json:"is_banned"`
	BannedAt            *time.Time   `json:"banned_at,omitempty"`
	BanReason           string       `json:"ban_reason,omitempty"`
	FailedLoginAttempts int          `json:"failed_login_attempts"`
	LockedUntil         *time.Time   `json:"locked_until,omitempty"`
	StudioStatus        StudioStatus `json:"studio_status,omitempty"`
	MworkUserID         string       `json:"-"`
	MworkRole           string       `json:"-"`
	CreatedAt           time.Time    `json:"created_at"`
	UpdatedAt           time.Time    `json:"updated_at"`
}
