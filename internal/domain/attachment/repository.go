package attachment

import (
	"context"

	"gorm.io/gorm"
)

type Repository interface {
	Create(ctx context.Context, a *Attachment) error
	GetByID(ctx context.Context, id int64) (*Attachment, error)
	ListByTarget(ctx context.Context, targetType TargetType, targetID int64) ([]*Attachment, error)
	CountByTarget(ctx context.Context, targetType TargetType, targetID int64) (int, error)
	Delete(ctx context.Context, id int64) error
	UpdateSortOrder(ctx context.Context, id int64, sortOrder int) error
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

func (r *repository) Create(ctx context.Context, a *Attachment) error {
	return r.db.WithContext(ctx).Create(a).Error
}

func (r *repository) GetByID(ctx context.Context, id int64) (*Attachment, error) {
	var a Attachment
	if err := r.db.WithContext(ctx).First(&a, id).Error; err != nil {
		return nil, err
	}
	return &a, nil
}

func (r *repository) ListByTarget(ctx context.Context, targetType TargetType, targetID int64) ([]*Attachment, error) {
	var out []*Attachment
	err := r.db.WithContext(ctx).
		Where("target_type = ? AND target_id = ?", targetType, targetID).
		Order("sort_order ASC, created_at ASC").
		Find(&out).Error
	return out, err
}

func (r *repository) CountByTarget(ctx context.Context, targetType TargetType, targetID int64) (int, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&Attachment{}).
		Where("target_type = ? AND target_id = ?", targetType, targetID).
		Count(&count).Error
	return int(count), err
}

func (r *repository) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&Attachment{}, id).Error
}

func (r *repository) UpdateSortOrder(ctx context.Context, id int64, sortOrder int) error {
	return r.db.WithContext(ctx).
		Model(&Attachment{}).
		Where("id = ?", id).
		Update("sort_order", sortOrder).Error
}
