package profile

import (
	"context"
	"database/sql"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

// OwnerRepository handles owner profile data access
type OwnerRepository struct {
	db *sqlx.DB
}

// NewOwnerRepository creates owner profile repository
func NewOwnerRepository(db *sqlx.DB) *OwnerRepository {
	return &OwnerRepository{db: db}
}

// GetByUserID retrieves owner profile by user ID
func (r *OwnerRepository) GetByUserID(ctx context.Context, userID int64) (*OwnerProfile, error) {
	var profile OwnerProfile
	query := `SELECT * FROM owner_profiles WHERE user_id = $1`
	err := r.db.GetContext(ctx, &profile, query, userID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &profile, err
}

// Create creates a new owner profile
func (r *OwnerRepository) Create(ctx context.Context, profile *OwnerProfile) error {
	query := `
		INSERT INTO owner_profiles (
			user_id, company_name, bin, legal_address, contact_person, contact_position,
			phone, email, website, verification_status, verification_docs,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		RETURNING id, created_at, updated_at
	`
	return r.db.QueryRowContext(
		ctx, query,
		profile.UserID, profile.CompanyName, profile.Bin, profile.LegalAddress,
		profile.ContactPerson, profile.ContactPosition, profile.Phone, profile.Email,
		profile.Website, profile.VerificationStatus, pq.Array(profile.VerificationDocs),
		profile.CreatedAt, profile.UpdatedAt,
	).Scan(&profile.ID, &profile.CreatedAt, &profile.UpdatedAt)
}

// Update updates owner profile
func (r *OwnerRepository) Update(ctx context.Context, profile *OwnerProfile) error {
	query := `
		UPDATE owner_profiles
		SET company_name = $2, bin = $3, legal_address = $4, contact_person = $5,
			contact_position = $6, phone = $7, email = $8, website = $9, updated_at = $10
		WHERE id = $1
	`
	_, err := r.db.ExecContext(
		ctx, query,
		profile.ID, profile.CompanyName, profile.Bin, profile.LegalAddress,
		profile.ContactPerson, profile.ContactPosition, profile.Phone, profile.Email,
		profile.Website, time.Now(),
	)
	return err
}

// UpdateVerificationStatus updates verification status (admin only)
func (r *OwnerRepository) UpdateVerificationStatus(ctx context.Context, userID, adminID int64, status, reason, notes string) error {
	query := `
		UPDATE owner_profiles
		SET verification_status = $2, verified_by = $3, verified_at = CASE WHEN $2 = 'verified' THEN NOW() ELSE NULL END,
			rejected_reason = $4, admin_notes = $5, updated_at = NOW()
		WHERE user_id = $1
	`
	_, err := r.db.ExecContext(ctx, query, userID, status, adminID, nullString(reason), nullString(notes))
	return err
}
