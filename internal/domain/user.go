package domain

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
	ID            int64        `json:"id"`
	Email         string       `json:"email" validate:"required,email"`
	PasswordHash  string       `json:"-"`
	Role          UserRole     `json:"role"`
	Name          string       `json:"name"`
	Phone         string       `json:"phone,omitempty"`
	AvatarURL     string       `json:"avatar_url,omitempty"`
	EmailVerified bool         `json:"email_verified"`
	StudioStatus  StudioStatus `json:"studio_status,omitempty"`
	MworkUserID   string       `json:"-"`
	MworkRole     string       `json:"-"`
	CreatedAt     time.Time    `json:"created_at"`
	UpdatedAt     time.Time    `json:"updated_at"`
}

type StudioOwner struct {
	ID               int64      `json:"id"`
	UserID           int64      `json:"user_id"`
	CompanyName      string     `json:"company_name"`
	BIN              string     `json:"bin,omitempty"`
	LegalAddress     string     `json:"legal_address,omitempty"`
	ContactPerson    string     `json:"contact_person"`
	ContactPosition  string     `json:"contact_position,omitempty"`
	VerificationDocs []string   `json:"verification_docs,omitempty" gorm:"type:json"`
	VerifiedAt       *time.Time `json:"verified_at,omitempty"`
	VerifiedBy       *int64     `json:"verified_by,omitempty"`
	RejectedReason   string     `json:"rejected_reason,omitempty"`
	AdminNotes       string     `json:"admin_notes,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
}
