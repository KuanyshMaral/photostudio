package repository

import (
	"context"
	"gorm.io/gorm"
	"photostudio/internal/domain"
)

type OwnerRepository struct {
	db *gorm.DB
}

func NewStudioOwnerRepository(db *gorm.DB) *OwnerRepository {
	return &OwnerRepository{db: db}
}

func (r *OwnerRepository) Create(ctx context.Context, tx *gorm.DB, owner *domain.StudioOwner) error {
	if tx == nil {
		tx = r.db
	}
	return tx.WithContext(ctx).Create(owner).Error
}

func (r *OwnerRepository) AppendVerificationDocs(ctx context.Context, userID int64, urls []string) error {
	return r.db.WithContext(ctx).
		Model(&domain.StudioOwner{}).
		Where("user_id = ?", userID).
		UpdateColumn("verification_docs", gorm.Expr("array_cat(verification_docs, ?)", urls)).
		Error
}

func (r *OwnerRepository) DB() *gorm.DB {
	return r.db
}
