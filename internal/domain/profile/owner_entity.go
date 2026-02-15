package profile

import (
	"database/sql"
	"time"

	"github.com/lib/pq"
)

// OwnerProfile represents a studio owner's profile
type OwnerProfile struct {
	ID     int64 `db:"id" json:"id"`
	UserID int64 `db:"user_id" json:"user_id"`

	// Company info
	CompanyName     string         `db:"company_name" json:"company_name"`
	Bin             sql.NullString `db:"bin" json:"bin,omitempty"`
	LegalAddress    sql.NullString `db:"legal_address" json:"legal_address,omitempty"`
	ContactPerson   sql.NullString `db:"contact_person" json:"contact_person,omitempty"`
	ContactPosition sql.NullString `db:"contact_position" json:"contact_position,omitempty"`

	// Contact
	Phone   sql.NullString `db:"phone" json:"phone,omitempty"`
	Email   sql.NullString `db:"email" json:"email,omitempty"`
	Website sql.NullString `db:"website" json:"website,omitempty"`

	// Verification
	VerificationStatus string         `db:"verification_status" json:"verification_status"`
	VerificationDocs   pq.StringArray `db:"verification_docs" json:"verification_docs,omitempty"`
	VerifiedAt         sql.NullTime   `db:"verified_at" json:"verified_at,omitempty"`
	VerifiedBy         sql.NullInt64  `db:"verified_by" json:"verified_by,omitempty"`
	RejectedReason     sql.NullString `db:"rejected_reason" json:"rejected_reason,omitempty"`
	AdminNotes         sql.NullString `db:"admin_notes" json:"admin_notes,omitempty"`

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
