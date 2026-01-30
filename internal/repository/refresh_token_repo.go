package repository

import (
	"context"
	"time"

	"photostudio/internal/domain"

	"gorm.io/gorm"
)

// RefreshTokenRepository provides DB access for refresh tokens.
type RefreshTokenRepository struct {
	db *gorm.DB
}

func NewRefreshTokenRepository(db *gorm.DB) *RefreshTokenRepository {
	return &RefreshTokenRepository{db: db}
}

func (r *RefreshTokenRepository) Create(ctx context.Context, t *domain.RefreshToken) error {
	return r.db.WithContext(ctx).Create(t).Error
}

func (r *RefreshTokenRepository) GetByHash(ctx context.Context, hash string) (*domain.RefreshToken, error) {
	var t domain.RefreshToken
	err := r.db.WithContext(ctx).Where("token_hash = ?", hash).First(&t).Error
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *RefreshTokenRepository) Revoke(ctx context.Context, id int64, replacedByID *int64) error {
	now := time.Now().UTC()
	updates := map[string]any{"revoked_at": now}
	if replacedByID != nil {
		updates["replaced_by_id"] = *replacedByID
	}
	return r.db.WithContext(ctx).Model(&domain.RefreshToken{}).
		Where("id = ? AND revoked_at IS NULL", id).
		Updates(updates).Error
}

func (r *RefreshTokenRepository) RevokeByUser(ctx context.Context, userID int64) error {
	now := time.Now().UTC()
	return r.db.WithContext(ctx).Model(&domain.RefreshToken{}).
		Where("user_id = ? AND revoked_at IS NULL", userID).
		Update("revoked_at", now).Error
}

func (r *RefreshTokenRepository) DeleteExpired(ctx context.Context) error {
	now := time.Now().UTC()
	return r.db.WithContext(ctx).
		Where("expires_at < ?", now).
		Delete(&domain.RefreshToken{}).Error
}
