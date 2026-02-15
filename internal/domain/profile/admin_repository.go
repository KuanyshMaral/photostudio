package profile

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// AdminRepository handles admin profile data access
type AdminRepository struct {
	db *sqlx.DB
}

// NewAdminRepository creates admin profile repository
func NewAdminRepository(db *sqlx.DB) *AdminRepository {
	return &AdminRepository{db: db}
}

// GetByUserID retrieves admin profile by user ID
func (r *AdminRepository) GetByUserID(ctx context.Context, userID uuid.UUID) (*AdminProfile, error) {
	var profile AdminProfile
	query := `SELECT * FROM admin_profiles WHERE user_id = $1`
	err := r.db.GetContext(ctx, &profile, query, userID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &profile, err
}

// Create creates a new admin profile
func (r *AdminRepository) Create(ctx context.Context, profile *AdminProfile) error {
	query := `
		INSERT INTO admin_profiles (user_id, full_name, position, phone, is_active, created_by, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_at, updated_at
	`
	return r.db.QueryRowContext(
		ctx, query,
		profile.UserID, profile.FullName, profile.Position, profile.Phone,
		profile.IsActive, profile.CreatedBy, profile.CreatedAt, profile.UpdatedAt,
	).Scan(&profile.ID, &profile.CreatedAt, &profile.UpdatedAt)
}

// Update updates admin profile
func (r *AdminRepository) Update(ctx context.Context, profile *AdminProfile) error {
	query := `
		UPDATE admin_profiles
		SET full_name = $2, position = $3, phone = $4, updated_at = $5
		WHERE id = $1
	`
	_, err := r.db.ExecContext(
		ctx, query,
		profile.ID, profile.FullName, profile.Position, profile.Phone, time.Now(),
	)
	return err
}

// UpdateLastLogin updates last login timestamp and IP
func (r *AdminRepository) UpdateLastLogin(ctx context.Context, userID uuid.UUID, ip string) error {
	query := `
		UPDATE admin_profiles
		SET last_login_at = NOW(), last_login_ip = $2, updated_at = NOW()
		WHERE user_id = $1
	`
	_, err := r.db.ExecContext(ctx, query, userID, ip)
	return err
}
