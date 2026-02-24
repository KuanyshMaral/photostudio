package profile

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
)

// AdminProfile represents an admin user's profile
type AdminProfile struct {
	ID     int64     `db:"id" json:"id"`
	UserID uuid.UUID `db:"user_id" json:"user_id"`

	// Admin info
	FullName string         `db:"full_name" json:"full_name"`
	Position sql.NullString `db:"position" json:"position,omitempty" swaggertype:"string"`
	Phone    sql.NullString `db:"phone" json:"phone,omitempty" swaggertype:"string"`

	// Access
	IsActive    bool           `db:"is_active" json:"is_active"`
	LastLoginAt sql.NullTime   `db:"last_login_at" json:"last_login_at,omitempty" swaggertype:"string"`
	LastLoginIP sql.NullString `db:"last_login_ip" json:"last_login_ip,omitempty" swaggertype:"string"`

	CreatedAt time.Time     `db:"created_at" json:"created_at"`
	CreatedBy uuid.NullUUID `db:"created_by" json:"created_by,omitempty" swaggertype:"string"`
	UpdatedAt time.Time     `db:"updated_at" json:"updated_at"`
}

// GetDisplayName returns display name for AdminProfile
func (p *AdminProfile) GetDisplayName() string {
	return p.FullName
}
