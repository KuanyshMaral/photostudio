package repository

import (
	"context"
	"time"

	"photostudio/internal/domain"

	"gorm.io/gorm"
)

type BookingRepository struct {
	db *gorm.DB
}

func NewBookingRepository(db *gorm.DB) *BookingRepository {
	return &BookingRepository{db: db}
}

type bookingModel struct {
	ID            int64      `gorm:"column:id;primaryKey"`
	RoomID        int64      `gorm:"column:room_id"`
	StudioID      int64      `gorm:"column:studio_id"`
	UserID        int64      `gorm:"column:user_id"`
	StartTime     time.Time  `gorm:"column:start_time"`
	EndTime       time.Time  `gorm:"column:end_time"`
	TotalPrice    float64    `gorm:"column:total_price"`
	Status        string     `gorm:"column:status"`
	PaymentStatus string     `gorm:"column:payment_status"`
	Notes         *string    `gorm:"column:notes"`
	CreatedAt     time.Time  `gorm:"column:created_at"`
	UpdatedAt     time.Time  `gorm:"column:updated_at"`
	CancelledAt   *time.Time `gorm:"column:cancelled_at"`
}

func (bookingModel) TableName() string { return "bookings" }

func toDomainBooking(m bookingModel) *domain.Booking {
	var notes string
	if m.Notes != nil {
		notes = *m.Notes
	}

	return &domain.Booking{
		ID:            m.ID,
		RoomID:        m.RoomID,
		StudioID:      m.StudioID,
		UserID:        m.UserID,
		StartTime:     m.StartTime,
		EndTime:       m.EndTime,
		TotalPrice:    m.TotalPrice,
		Status:        domain.BookingStatus(m.Status),
		PaymentStatus: domain.PaymentStatus(m.PaymentStatus),
		Notes:         notes,
		CreatedAt:     m.CreatedAt,
		UpdatedAt:     m.UpdatedAt,
		CancelledAt:   m.CancelledAt,
	}
}

func toBookingModel(b *domain.Booking) bookingModel {
	var notes *string
	if b.Notes != "" {
		v := b.Notes
		notes = &v
	}

	return bookingModel{
		ID:            b.ID,
		RoomID:        b.RoomID,
		StudioID:      b.StudioID,
		UserID:        b.UserID,
		StartTime:     b.StartTime,
		EndTime:       b.EndTime,
		TotalPrice:    b.TotalPrice,
		Status:        string(b.Status),
		PaymentStatus: string(b.PaymentStatus),
		Notes:         notes,
		CreatedAt:     b.CreatedAt,
		UpdatedAt:     b.UpdatedAt,
		CancelledAt:   b.CancelledAt,
	}
}

func (r *BookingRepository) Create(ctx context.Context, b *domain.Booking) error {
	m := toBookingModel(b)
	tx := r.db.WithContext(ctx).Create(&m)
	if tx.Error != nil {
		return tx.Error
	}
	*b = *toDomainBooking(m)
	return nil
}

func (r *BookingRepository) GetByID(ctx context.Context, id int64) (*domain.Booking, error) {
	var m bookingModel
	tx := r.db.WithContext(ctx).First(&m, id)
	if tx.Error != nil {
		return nil, tx.Error
	}
	return toDomainBooking(m), nil
}

func (r *BookingRepository) CheckAvailability(ctx context.Context, roomID int64, start, end time.Time) (bool, error) {
	var cnt int64
	q := `
SELECT COUNT(1)
FROM bookings
WHERE room_id = ?
  AND status NOT IN ('cancelled')
  AND tstzrange(start_time, end_time, '[)') && tstzrange(?, ?, '[)')
`
	tx := r.db.WithContext(ctx).Raw(q, roomID, start, end).Scan(&cnt)
	if tx.Error != nil {
		return false, tx.Error
	}
	return cnt == 0, nil
}
