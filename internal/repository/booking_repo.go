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

type BusySlot struct {
	Start time.Time `gorm:"column:start"`
	End   time.Time `gorm:"column:end"`
}

func (r *BookingRepository) GetBusySlotsForRoom(ctx context.Context, roomID int64, from, to time.Time) ([]BusySlot, error) {
	var rows []BusySlot
	q := `
SELECT start_time AS start, end_time AS end
FROM bookings
WHERE room_id = ?
  AND status NOT IN ('cancelled')
  AND tstzrange(start_time, end_time, '[)') && tstzrange(?, ?, '[)')
ORDER BY start_time
`
	tx := r.db.WithContext(ctx).Raw(q, roomID, from, to).Scan(&rows)
	if tx.Error != nil {
		return nil, tx.Error
	}
	return rows, nil
}

type UserBookingDetails struct {
	ID         int64     `gorm:"column:id"`
	Status     string    `gorm:"column:status"`
	StartTime  time.Time `gorm:"column:start_time"`
	EndTime    time.Time `gorm:"column:end_time"`
	TotalPrice float64   `gorm:"column:total_price"`

	RoomID   int64  `gorm:"column:room_id"`
	RoomName string `gorm:"column:room_name"`

	StudioID   int64  `gorm:"column:studio_id"`
	StudioName string `gorm:"column:studio_name"`
}

func (r *BookingRepository) GetUserBookingsWithDetails(ctx context.Context, userID int64, limit, offset int) ([]UserBookingDetails, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	var rows []UserBookingDetails
	q := `
SELECT
  b.id,
  b.status,
  b.start_time,
  b.end_time,
  b.total_price,
  b.room_id,
  rm.name AS room_name,
  b.studio_id,
  s.name AS studio_name
FROM bookings b
JOIN rooms rm ON rm.id = b.room_id
JOIN studios s ON s.id = b.studio_id
WHERE b.user_id = ?
ORDER BY b.created_at DESC
LIMIT ? OFFSET ?
`
	tx := r.db.WithContext(ctx).Raw(q, userID, limit, offset).Scan(&rows)
	if tx.Error != nil {
		return nil, tx.Error
	}
	return rows, nil
}

func (r *BookingRepository) GetStudioOwnerForBooking(ctx context.Context, bookingID int64) (int64, string, error) {
	type row struct {
		OwnerID int64  `gorm:"column:owner_id"`
		Status  string `gorm:"column:status"`
	}
	var out row
	q := `
SELECT s.owner_id, b.status
FROM bookings b
JOIN studios s ON s.id = b.studio_id
WHERE b.id = ?
`
	tx := r.db.WithContext(ctx).Raw(q, bookingID).Scan(&out)
	if tx.Error != nil {
		return 0, "", tx.Error
	}
	if tx.RowsAffected == 0 {
		return 0, "", nil
	}
	return out.OwnerID, out.Status, nil
}

func (r *BookingRepository) UpdateStatus(ctx context.Context, bookingID int64, newStatus string) error {
	tx := r.db.WithContext(ctx).
		Table("bookings").
		Where("id = ?", bookingID).
		Updates(map[string]any{
			"status":     newStatus,
			"updated_at": time.Now().UTC(),
		})
	return tx.Error
}
func (r *BookingRepository) HasCompletedBookingForStudio(ctx context.Context, userID, studioID int64) (bool, error) {
	var cnt int64
	q := `
SELECT COUNT(1)
FROM bookings
WHERE user_id = ?
  AND studio_id = ?
  AND status = 'completed'
`
	tx := r.db.WithContext(ctx).Raw(q, userID, studioID).Scan(&cnt)
	if tx.Error != nil {
		return false, tx.Error
	}
	return cnt > 0, nil
}

func (r *BookingRepository) GetByStudioID(ctx context.Context, studioID int64) ([]domain.Booking, error) {
	var rows []bookingModel

	tx := r.db.WithContext(ctx).
		Where("studio_id = ?", studioID).
		Order("created_at DESC").
		Find(&rows)
	if tx.Error != nil {
		return nil, tx.Error
	}

	out := make([]domain.Booking, 0, len(rows))
	for _, m := range rows {
		out = append(out, *toDomainBooking(m))
	}
	return out, nil
}

func (r *BookingRepository) IsBookingOwnedByUser(ctx context.Context, bookingID, ownerID int64) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Table("bookings").
		Joins("JOIN studios ON studios.id = bookings.studio_id").
		Where("bookings.id = ? AND studios.owner_id = ? AND studios.deleted_at IS NULL", bookingID, ownerID).
		Count(&count).Error
	return count > 0, err
}
func (r *BookingRepository) UpdatePaymentStatus(ctx context.Context, bookingID int64, status domain.PaymentStatus) (*domain.Booking, error) {
	var m bookingModel
	if err := r.db.WithContext(ctx).First(&m, bookingID).Error; err != nil {
		return nil, err
	}

	// если в bookingModel PaymentStatus string
	m.PaymentStatus = string(status)

	if err := r.db.WithContext(ctx).Save(&m).Error; err != nil {
		return nil, err
	}

	return toDomainBooking(m), nil
}

func (r *BookingRepository) DB() *gorm.DB {
	return r.db
}
