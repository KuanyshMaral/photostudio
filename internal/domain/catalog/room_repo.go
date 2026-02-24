package catalog

import (
	"context"

	"gorm.io/gorm"
)

type RoomRepository struct {
	db *gorm.DB
}

func NewRoomRepository(db *gorm.DB) *RoomRepository {
	return &RoomRepository{db: db}
}

func (r *RoomRepository) GetAll(ctx context.Context, studioID *int64) ([]Room, error) {
	var rooms []Room
	db := r.db.WithContext(ctx).Table("rooms").Where("is_active = true")
	if studioID != nil {
		db = db.Where("studio_id = ?", *studioID)
	}
	if err := db.Find(&rooms).Error; err != nil {
		return nil, err
	}
	return rooms, nil
}

func (r *RoomRepository) GetByID(ctx context.Context, id int64) (*Room, error) {
	var room Room
	err := r.db.WithContext(ctx).
		Table("rooms").
		Where("id = ?", id).
		First(&room).Error
	return &room, err
}

func (r *RoomRepository) GetByStudioID(ctx context.Context, studioID int64) ([]Room, error) {
	var rooms []Room
	err := r.db.WithContext(ctx).
		Table("rooms").
		Where("studio_id = ? AND is_active = true", studioID).
		Find(&rooms).Error
	return rooms, err
}

func (r *RoomRepository) Create(ctx context.Context, room *Room) error {
	return r.db.WithContext(ctx).
		Table("rooms").
		Create(room).Error
}

func (r *RoomRepository) Update(ctx context.Context, room *Room) error {
	return r.db.WithContext(ctx).
		Table("rooms").
		Where("id = ?", room.ID).
		Updates(room).Error
}

// CountRoomsByOwnerID counts how many rooms a studio owner has across all their studios.
// Used by the subscription service to enforce the MaxRooms plan limit.
func (r *RoomRepository) CountRoomsByOwnerID(ctx context.Context, ownerID int64) (int, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Table("rooms").
		Joins("JOIN studios ON studios.id = rooms.studio_id").
		Where("studios.owner_id = ? AND rooms.is_active = true", ownerID).
		Count(&count).Error
	return int(count), err
}

type EquipmentRepository struct {
	db *gorm.DB
}

func NewEquipmentRepository(db *gorm.DB) *EquipmentRepository {
	return &EquipmentRepository{db: db}
}

func (r *EquipmentRepository) Create(ctx context.Context, e *Equipment) error {
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
func (r *RoomRepository) SetActive(ctx context.Context, id int64, active bool) error {
	return r.db.WithContext(ctx).
		Table("rooms").
		Where("id = ?", id).
		Update("is_active", active).Error
}

func (r *RoomRepository) GetStudioWorkingHoursByRoomID(ctx context.Context, roomID int64) ([]byte, error) {
	// TODO: working_hours field doesn't exist in current schema
	// For now, return empty/nil to unblock testing
	// Original: SELECT s.working_hours FROM rooms rm JOIN studios s ON s.id = rm.studio_id WHERE rm.id = ?
	return nil, nil
}

func (r *RoomRepository) GetRoomWorkingHoursRaw(ctx context.Context, roomID int64) ([]byte, error) {
	// TODO: working_hours field doesn't exist in current schema
	// Return empty for now
	return nil, nil
}

func (r *RoomRepository) GetRoomWorkingHoursRawOld(ctx context.Context, roomID int64) ([]byte, error) {
	// Deprecated - use GetStudioWorkingHoursByRoomID instead
	return nil, nil
}
