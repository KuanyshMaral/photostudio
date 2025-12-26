package repository

import (
	"context"
	"photostudio/internal/domain"

	"gorm.io/gorm"
)

type RoomRepository struct {
	db *gorm.DB
}

func NewRoomRepository(db *gorm.DB) *RoomRepository {
	return &RoomRepository{db: db}
}

func (r *RoomRepository) GetByID(ctx context.Context, id int64) (*domain.Room, error) {
	var room domain.Room
	err := r.db.WithContext(ctx).
		Table("rooms").
		Where("id = ?", id).
		First(&room).Error
	return &room, err
}

func (r *RoomRepository) GetByStudioID(ctx context.Context, studioID int64) ([]domain.Room, error) {
	var rooms []domain.Room
	err := r.db.WithContext(ctx).
		Table("rooms").
		Where("studio_id = ? AND is_active = true", studioID).
		Find(&rooms).Error
	return rooms, err
}

func (r *RoomRepository) Create(ctx context.Context, room *domain.Room) error {
	return r.db.WithContext(ctx).
		Table("rooms").
		Create(room).Error
}

func (r *RoomRepository) Update(ctx context.Context, room *domain.Room) error {
	return r.db.WithContext(ctx).
		Table("rooms").
		Where("id = ?", room.ID).
		Updates(room).Error
}

type EquipmentRepository struct {
	db *gorm.DB
}


func NewEquipmentRepository(db *gorm.DB) *EquipmentRepository {
	return &EquipmentRepository{db: db}
}

func (r *EquipmentRepository) Create(ctx context.Context, e *domain.Equipment) error {
	return r.db.WithContext(ctx).
		Table("equipment").
		Create(e).Error
}
