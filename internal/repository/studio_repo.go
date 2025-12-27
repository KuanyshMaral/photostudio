package repository

import (
	"context"

	"photostudio/internal/domain"

	"gorm.io/gorm"
)

type StudioFilters struct {
	City     string
	MinPrice float64
	RoomType string
	Limit    int
	Offset   int
}

type StudioRepository struct {
	db *gorm.DB
}

func NewStudioRepository(db *gorm.DB) *StudioRepository {
	return &StudioRepository{db: db}
}

// GetAll returns studios with optional filters
func (r *StudioRepository) GetAll(
	ctx context.Context,
	f StudioFilters,
) ([]domain.Studio, int64, error) {

	var studios []domain.Studio
	var total int64

	q := r.db.WithContext(ctx).
		Model(&domain.Studio{}).
		Where("deleted_at IS NULL")

	if f.City != "" {
		q = q.Where("city = ?", f.City)
	}

	if f.MinPrice > 0 || f.RoomType != "" {
		q = q.Joins("JOIN rooms ON rooms.studio_id = studios.id AND rooms.is_active = true")
	}

	if f.MinPrice > 0 {
		q = q.Where("rooms.price_per_hour_min >= ?", f.MinPrice)
	}

	if f.RoomType != "" {
		q = q.Where("rooms.room_type = ?", f.RoomType)
	}

	q.Count(&total)

	err := q.
		Preload("Rooms", "is_active = true").
		Preload("Rooms.Equipment").
		Limit(f.Limit).
		Offset(f.Offset).
		Find(&studios).Error

	return studios, total, err
}

// GetByID fetches a studio by its ID
func (r *StudioRepository) GetByID(
	ctx context.Context,
	id int64,
) (*domain.Studio, error) {

	var studio domain.Studio

	err := r.db.WithContext(ctx).
		Where("studios.id = ? AND deleted_at IS NULL", id).
		Preload("Rooms", "is_active = true").
		Preload("Rooms.Equipment").
		First(&studio).Error

	if err != nil {
		return nil, err
	}

	return &studio, nil
}

// Create a new studio
func (r *StudioRepository) Create(ctx context.Context, studio *domain.Studio) error {
	return r.db.WithContext(ctx).Create(studio).Error
}

// Update an existing studio
func (r *StudioRepository) Update(ctx context.Context, studio *domain.Studio) error {
	return r.db.WithContext(ctx).Save(studio).Error
}
