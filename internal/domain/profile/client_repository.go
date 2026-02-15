package profile

import (
	"context"
	"database/sql"
	"time"

	"github.com/jmoiron/sqlx"
)

// ClientRepository handles client profile data access
type ClientRepository struct {
	db *sqlx.DB
}

// NewClientRepository creates client profile repository
func NewClientRepository(db *sqlx.DB) *ClientRepository {
	return &ClientRepository{db: db}
}

// GetByUserID retrieves client profile by user ID
func (r *ClientRepository) GetByUserID(ctx context.Context, userID int64) (*ClientProfile, error) {
	var profile ClientProfile
	query := `SELECT * FROM client_profiles WHERE user_id = $1`
	err := r.db.GetContext(ctx, &profile, query, userID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &profile, err
}

// Create creates a new client profile
func (r *ClientRepository) Create(ctx context.Context, profile *ClientProfile) error {
	query := `
		INSERT INTO client_profiles (user_id, name, nickname, phone, avatar_url, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at, updated_at
	`
	return r.db.QueryRowContext(
		ctx, query,
		profile.UserID, profile.Name, profile.Nickname, profile.Phone, profile.AvatarURL,
		profile.CreatedAt, profile.UpdatedAt,
	).Scan(&profile.ID, &profile.CreatedAt, &profile.UpdatedAt)
}

// Update updates client profile
func (r *ClientRepository) Update(ctx context.Context, profile *ClientProfile) error {
	query := `
		UPDATE client_profiles
		SET name = $2, nickname = $3, phone = $4, avatar_url = $5, updated_at = $6
		WHERE id = $1
	`
	_, err := r.db.ExecContext(
		ctx, query,
		profile.ID, profile.Name, profile.Nickname, profile.Phone, profile.AvatarURL, time.Now(),
	)
	return err
}
