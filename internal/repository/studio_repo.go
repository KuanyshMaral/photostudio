package repository

import (
	"context"
	"photostudio/internal/pkg/utils"
	"time"

	"photostudio/internal/domain"

	"gorm.io/gorm"
)

type StudioFilters struct {
	City     string
	MinPrice float64
	MaxPrice float64
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

	// Filter by city
	if f.City != "" {
		q = q.Where("city = ?", f.City)
	}

	// Use subquery instead of JOIN for SQLite compatibility
	// This fixes Problem #B2: SQLite doesn't support complex JOINs well
	if f.MinPrice > 0 || f.MaxPrice > 0 || f.RoomType != "" {
		subQuery := r.db.Model(&domain.Room{}).
			Select("studio_id").
			Where("is_active = true")

		if f.MinPrice > 0 {
			subQuery = subQuery.Where("price_per_hour_min >= ?", f.MinPrice)
		}

		if f.MaxPrice > 0 {
			subQuery = subQuery.Where("price_per_hour_min <= ?", f.MaxPrice)
		}

		if f.RoomType != "" {
			subQuery = subQuery.Where("room_type = ?", f.RoomType)
		}

		q = q.Where("id IN (?)", subQuery)
	}

	// IMPORTANT: Clone query before counting to avoid Count modifying the query
	countQuery := q.Session(&gorm.Session{})
	countQuery.Count(&total)

	// Apply pagination and load relations to original query
	err := q.
		Preload("Rooms", "is_active = true").
		Preload("Rooms.Equipment").
		Limit(f.Limit).
		Offset(f.Offset).
		Find(&studios).Error

	return studios, total, err
}

// GetByID fetches a studio by its ID with all relations
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

// GetByOwnerID returns all studios belonging to a user
func (r *StudioRepository) GetByOwnerID(ctx context.Context, ownerID int64) ([]domain.Studio, error) {
	var studios []domain.Studio
	err := r.db.WithContext(ctx).
		Where("owner_id = ? AND deleted_at IS NULL", ownerID).
		Preload("Rooms", "is_active = true").
		Preload("Rooms.Equipment").
		Find(&studios).Error
	return studios, err
}

// Create creates a new studio
func (r *StudioRepository) Create(ctx context.Context, studio *domain.Studio) error {
	return r.db.WithContext(ctx).Create(studio).Error
}

// AddPhotos appends new URLs to the studio's photos array (works on SQLite & PostgreSQL)
func (r *StudioRepository) AddPhotos(ctx context.Context, id int64, newURLs []string) error {
	var studio domain.Studio

	// Load current
	if err := r.db.WithContext(ctx).
		Where("id = ?", id).
		First(&studio).Error; err != nil {
		return err
	}

	// Convert current DB string to slice
	current := utils.StringToPhotos(studio.Photos)

	// Append new URLs
	updated := append(current, newURLs...)

	// Convert back to JSON string
	studio.Photos = utils.PhotosToString(updated)

	// Save
	return r.db.WithContext(ctx).Save(&studio).Error
}

// Update updates an existing studio
func (r *StudioRepository) Update(ctx context.Context, studio *domain.Studio) error {
	return r.db.WithContext(ctx).Save(studio).Error
}

// Delete soft deletes a studio (sets deleted_at)
func (r *StudioRepository) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).
		Model(&domain.Studio{}).
		Where("id = ?", id).
		Update("deleted_at", time.Now()).Error
}

func (r *StudioRepository) DB() *gorm.DB {
	return r.db
}
