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
	Name      sql.NullString `db:"name" json:"name,omitempty"`
	Nickname  sql.NullString `db:"nickname" json:"nickname,omitempty"`
	Phone     sql.NullString `db:"phone" json:"phone,omitempty"`
	AvatarURL sql.NullString `db:"avatar_url" json:"avatar_url,omitempty"`

	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}

// GetDisplayName returns display name for ClientProfile
func (p *ClientProfile) GetDisplayName() string {
	if p.Name.Valid && p.Name.String != "" {
		return p.Name.String
	}
	if p.Nickname.Valid && p.Nickname.String != "" {
		return p.Nickname.String
	}
	return "Client"
}
