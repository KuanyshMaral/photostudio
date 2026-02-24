package upload

import (
	"context"

	"gorm.io/gorm"
)

type Repository interface {
	Create(ctx context.Context, u *Upload) error
	GetByID(ctx context.Context, id string) (*Upload, error)
	Delete(ctx context.Context, id string) error
	ListByUserID(ctx context.Context, userID int64) ([]*Upload, error)
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

func (r *repository) Create(ctx context.Context, u *Upload) error {
	return r.db.WithContext(ctx).Create(u).Error
}

func (r *repository) GetByID(ctx context.Context, id string) (*Upload, error) {
	var u Upload
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&u).Error
	if err == gorm.ErrRecordNotFound {
		return nil, ErrUploadNotFound
	}
	return &u, err
}

func (r *repository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Where("id = ?", id).Delete(&Upload{}).Error
}

func (r *repository) ListByUserID(ctx context.Context, userID int64) ([]*Upload, error) {
	var uploads []*Upload
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).Order("created_at DESC").Find(&uploads).Error
	return uploads, err
}
