package repository

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

func (r *RoomRepository) GetPriceByID(ctx context.Context, roomID int64) (float64, error) {
	var price float64
	tx := r.db.WithContext(ctx).
		Table("rooms").
		Select("price_per_hour").
		Where("id = ?", roomID).
		Scan(&price)
	if tx.Error != nil {
		return 0, tx.Error
	}
	return price, nil
}
