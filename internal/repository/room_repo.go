package repository

import (
	"context"
	"encoding/json"
	"photostudio/internal/domain"

	"gorm.io/gorm"
)

type RoomRepository struct {
	db *gorm.DB
}

func NewRoomRepository(db *gorm.DB) *RoomRepository {
	return &RoomRepository{db: db}
}

func (r *RoomRepository) GetAll(ctx context.Context, studioID *int64) ([]domain.Room, error) {
	var rooms []domain.Room
	db := r.db.WithContext(ctx).Table("rooms").Where("is_active = true")
	if studioID != nil {
		db = db.Where("studio_id = ?", *studioID)
	}
	if err := db.Find(&rooms).Error; err != nil {
		return nil, err
	}
	return rooms, nil
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
func (r *RoomRepository) GetPriceByID(ctx context.Context, roomID int64) (float64, error) {
	var price float64
	tx := r.db.WithContext(ctx).
		Table("rooms").
		Select("price_per_hour_min").
		Where("id = ?", roomID).
		Scan(&price)
	if tx.Error != nil {
		return 0, tx.Error
	}
	return price, nil
}

func (r *RoomRepository) GetStudioWorkingHoursByRoomID(ctx context.Context, roomID int64) (json.RawMessage, error) {
	var raw []byte
	q := `
SELECT s.working_hours
FROM rooms rm
JOIN studios s ON s.id = rm.studio_id
WHERE rm.id = ?


`
	tx := r.db.WithContext(ctx).Raw(q, roomID).Scan(&raw)
	if tx.Error != nil {
		return nil, tx.Error
	}
	if tx.RowsAffected == 0 {
		return nil, nil
	}
	return json.RawMessage(raw), nil
}
