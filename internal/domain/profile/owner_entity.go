package profile

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// OwnerProfile represents a studio owner's profile
type OwnerProfile struct {
	ID     int64 `db:"id" json:"id"`
	UserID int64 `db:"user_id" json:"user_id"`

	// Company info
	CompanyName     string         `db:"company_name" json:"company_name"`
	Bin             sql.NullString `db:"bin" json:"bin,omitempty" swaggertype:"string"`
	LegalAddress    sql.NullString `db:"legal_address" json:"legal_address,omitempty" swaggertype:"string"`
	ContactPerson   sql.NullString `db:"contact_person" json:"contact_person,omitempty" swaggertype:"string"`
	ContactPosition sql.NullString `db:"contact_position" json:"contact_position,omitempty" swaggertype:"string"`

	// Contact
	Phone   sql.NullString `db:"phone" json:"phone,omitempty" swaggertype:"string"`
	Email   sql.NullString `db:"email" json:"email,omitempty" swaggertype:"string"`
	Website sql.NullString `db:"website" json:"website,omitempty" swaggertype:"string"`

	// Verification
	VerificationStatus string         `db:"verification_status" json:"verification_status"`
	VerificationDocs   pq.StringArray `db:"verification_docs" json:"verification_docs,omitempty" swaggertype:"array,string"`
	VerifiedAt         sql.NullTime   `db:"verified_at" json:"verified_at,omitempty" swaggertype:"string"`
	VerifiedBy         sql.NullInt64  `db:"verified_by" json:"verified_by,omitempty" swaggertype:"integer"`
	RejectedReason     sql.NullString `db:"rejected_reason" json:"rejected_reason,omitempty" swaggertype:"string"`
	AdminNotes         sql.NullString `db:"admin_notes" json:"admin_notes,omitempty" swaggertype:"string"`

	// Avatar
	AvatarURL      sql.NullString `db:"avatar_url" json:"avatar_url,omitempty" swaggertype:"string"`
	AvatarUploadID uuid.NullUUID  `db:"avatar_upload_id" json:"avatar_upload_id,omitempty" swaggertype:"string"`

	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}

// GetDisplayName returns display name for OwnerProfile
func (p *OwnerProfile) GetDisplayName() string {
	return p.CompanyName
}

// IsVerified returns true if owner is verified
func (p *OwnerProfile) IsVerified() bool {
	return p.VerificationStatus == "verified"
}

// PendingOwnerProfileRow Row DTO for queries on owner_profiles table for admin panel
type PendingOwnerProfileRow struct {
	ID          int64     `json:"id" db:"id"`
	UserID      int64     `json:"user_id" db:"user_id"`
	Bin         string    `json:"bin" db:"bin"`
	CompanyName string    `json:"company_name" db:"company_name"`
	Status      string    `json:"status" db:"verification_status"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
}
