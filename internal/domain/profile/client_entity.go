package profile

import (
	"database/sql"
	"time"
)

// ClientProfile represents a client's profile
type ClientProfile struct {
	ID     int64 `db:"id" json:"id"`
	UserID int64 `db:"user_id" json:"user_id"`

	// Basic info
	FullName       sql.NullString `db:"full_name" json:"full_name,omitempty" swaggertype:"string"`
	Nickname       sql.NullString `db:"nickname" json:"nickname,omitempty" swaggertype:"string"`
	Phone          sql.NullString `db:"phone" json:"phone,omitempty" swaggertype:"string"`
	AvatarURL      sql.NullString `db:"avatar_url" json:"avatar_url,omitempty" swaggertype:"string"`
	AvatarUploadID sql.NullString `db:"avatar_upload_id" json:"avatar_upload_id,omitempty" swaggertype:"string"`

	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}

// GetDisplayName returns display name for ClientProfile
func (p *ClientProfile) GetDisplayName() string {
	if p.FullName.Valid && p.FullName.String != "" {
		return p.FullName.String
	}
	if p.Nickname.Valid && p.Nickname.String != "" {
		return p.Nickname.String
	}
	return "Client"
}
