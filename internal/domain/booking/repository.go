package booking

import (
	"context"
	"errors"
	"gorm.io/gorm"
	"photostudio/internal/domain/auth"
	"time"
)

type bookingRepository struct {
	db *gorm.DB
}

func NewBookingRepository(db *gorm.DB) BookingRepository {
	return &bookingRepository{db: db}
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

func toDomainBooking(m bookingModel) *Booking {
	var notes string
	if m.Notes != nil {
		notes = *m.Notes
	}

	return &Booking{
		ID:            m.ID,
		RoomID:        m.RoomID,
		StudioID:      m.StudioID,
		UserID:        m.UserID,
		StartTime:     m.StartTime,
		EndTime:       m.EndTime,
		TotalPrice:    m.TotalPrice,
		Status:        BookingStatus(m.Status),
		PaymentStatus: PaymentStatus(m.PaymentStatus),
		Notes:         notes,
		CreatedAt:     m.CreatedAt,
		UpdatedAt:     m.UpdatedAt,
		CancelledAt:   m.CancelledAt,
	}
}

func toBookingModel(b *Booking) bookingModel {
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

func (r *bookingRepository) Create(ctx context.Context, booking *Booking) error {
	// Проверяем пересечение времени (работает на обоих БД)
	var count int64
	err := r.db.WithContext(ctx).
		Model(&Booking{}).
		Where("room_id = ?", booking.RoomID).
		Where("status NOT IN ('cancelled', 'rejected')").
		Where("start_time < ? AND end_time > ?", booking.EndTime, booking.StartTime).
		Count(&count).Error

	if err != nil {
		return err
	}

	if count > 0 {
		return errors.New("time slot is already booked")
	}

	return r.db.WithContext(ctx).Create(booking).Error
}

func (r *bookingRepository) GetByID(ctx context.Context, id int64) (*Booking, error) {
	var m bookingModel
	tx := r.db.WithContext(ctx).First(&m, id)
	if tx.Error != nil {
		return nil, tx.Error
	}
	return toDomainBooking(m), nil
}

// CheckAvailability - Fixed for SQLite compatibility (Problem #B3)
// Uses standard time overlap check instead of PostgreSQL-specific tstzrange
func (r *bookingRepository) CheckAvailability(ctx context.Context, roomID int64, start, end time.Time) (bool, error) {
	var cnt int64

	// SQLite-compatible time overlap check
	// Two time ranges overlap if: start1 < end2 AND end1 > start2
	err := r.db.WithContext(ctx).
		Model(&bookingModel{}).
		Where("room_id = ?", roomID).
		Where("status NOT IN ('cancelled')").
		Where("start_time < ? AND end_time > ?", end, start).
		Count(&cnt).Error

	if err != nil {
		return false, err
	}
	return cnt == 0, nil
}

type BusySlot struct {
	Start time.Time `gorm:"column:start"`
	End   time.Time `gorm:"column:end"`
}

// GetBusySlotsForRoom - Fixed for SQLite compatibility (Problem #B3)
func (r *bookingRepository) GetBusySlotsForRoom(ctx context.Context, roomID int64, from, to time.Time) ([]BusySlot, error) {
	var rows []BusySlot

	// SQLite-compatible query
	err := r.db.WithContext(ctx).
		Model(&bookingModel{}).
		Select("start_time AS start, end_time AS end").
		Where("room_id = ?", roomID).
		Where("status NOT IN ('cancelled')").
		Where("start_time < ? AND end_time > ?", to, from).
		Order("start_time").
		Scan(&rows).Error

	if err != nil {
		return nil, err
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

func (r *bookingRepository) GetUserBookingsWithDetails(ctx context.Context, userID int64, limit, offset int) ([]UserBookingDetails, error) {
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

func (r *bookingRepository) GetStudioOwnerForBooking(ctx context.Context, bookingID int64) (int64, string, error) {
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

func (r *bookingRepository) UpdateStatus(ctx context.Context, bookingID int64, newStatus string) error {
	tx := r.db.WithContext(ctx).
		Table("bookings").
		Where("id = ?", bookingID).
		Updates(map[string]any{
			"status":     newStatus,
			"updated_at": time.Now().UTC(),
		})
	return tx.Error
}
func (r *bookingRepository) HasCompletedBookingForStudio(ctx context.Context, userID, studioID int64) (bool, error) {
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

func (r *bookingRepository) GetByStudioID(ctx context.Context, studioID int64) ([]Booking, error) {
	var rows []bookingModel

	tx := r.db.WithContext(ctx).
		Where("studio_id = ?", studioID).
		Order("created_at DESC").
		Find(&rows)
	if tx.Error != nil {
		return nil, tx.Error
	}

	out := make([]Booking, 0, len(rows))
	for _, m := range rows {
		out = append(out, *toDomainBooking(m))
	}
	return out, nil
}

func (r *bookingRepository) IsBookingOwnedByUser(ctx context.Context, bookingID, ownerID int64) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Table("bookings").
		Joins("JOIN studios ON studios.id = bookings.studio_id").
		Where("bookings.id = ? AND studios.owner_id = ? AND studios.deleted_at IS NULL", bookingID, ownerID).
		Count(&count).Error
	return count > 0, err
}
func (r *bookingRepository) UpdatePaymentStatus(ctx context.Context, bookingID int64, status PaymentStatus) (*Booking, error) {
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

func (r *bookingRepository) DB() *gorm.DB {
	return r.db
}

// GetStatsByUserID возвращает статистику бронирований пользователя
func (r *bookingRepository) GetStatsByUserID(userID int64) (*auth.BookingStats, error) {
	stats := &auth.BookingStats{}
	now := time.Now()

	if err := r.db.Model(&Booking{}).
		Where("user_id = ?", userID).
		Count(&stats.Total).Error; err != nil {
		return nil, err
	}

	if err := r.db.Model(&Booking{}).
		Where("user_id = ? AND status = ? AND start_time > ?", userID, BookingStatus("confirmed"), now).
		Count(&stats.Upcoming).Error; err != nil {
		return nil, err
	}

	if err := r.db.Model(&Booking{}).
		Where("user_id = ? AND status = ?", userID, BookingStatus("completed")).
		Count(&stats.Completed).Error; err != nil {
		return nil, err
	}

	if err := r.db.Model(&Booking{}).
		Where("user_id = ? AND status = ?", userID, BookingStatus("cancelled")).
		Count(&stats.Cancelled).Error; err != nil {
		return nil, err
	}

	return stats, nil
}

// GetRecentByUserID возвращает последние N бронирований пользователя (с JOIN названиями)
func (r *bookingRepository) GetRecentByUserID(userID int64, limit int) ([]auth.RecentBookingRow, error) {
	if limit <= 0 {
		limit = 3
	}

	rows := make([]auth.RecentBookingRow, 0, limit)

	// ВАЖНО: названия таблиц подстрой если у тебя они другие.
	// Обычно: bookings, rooms, studios
	err := r.db.Table("bookings b").
		Select(`
            b.id as id,
            s.name as studio_name,
            r.name as room_name,
            b.start_time as start_time,
            b.status as status
        `).
		Joins("JOIN rooms r ON r.id = b.room_id").
		Joins("JOIN studios s ON s.id = b.studio_id").
		Where("b.user_id = ?", userID).
		Order("b.created_at DESC").
		Limit(limit).
		Scan(&rows).Error

	if err != nil {
		return nil, err
	}

	return rows, nil
}

// CancelWithReason отменяет бронирование с сохранением причины
// Block 9: Обязательная причина отмены
func (r *bookingRepository) CancelWithReason(ctx context.Context, bookingID int64, reason string) error {
	now := time.Now()
	return r.db.WithContext(ctx).
		Model(&Booking{}).
		Where("id = ?", bookingID).
		Updates(map[string]interface{}{
			"status":              string(BookingCancelled),
			"cancellation_reason": reason,
			"cancelled_at":        &now,
		}).Error
}

// UpdateDeposit обновляет сумму предоплаты
// Block 10: Управление предоплатой
func (r *bookingRepository) UpdateDeposit(ctx context.Context, bookingID int64, amount float64) error {
	return r.db.WithContext(ctx).
		Model(&Booking{}).
		Where("id = ?", bookingID).
		Update("deposit_amount", amount).Error
}

// -------------------- Manager Bookings --------------------

type ManagerBookingFilters struct {
	StudioID   int64
	RoomID     int64
	Status     string
	DateFrom   time.Time
	DateTo     time.Time
	ClientName string
	Page       int
	PerPage    int
}

type ManagerBookingRow struct {
	ID                 int64     `json:"id"`
	RoomID             int64     `json:"room_id"`
	RoomName           string    `json:"room_name"`
	StudioID           int64     `json:"studio_id"`
	StudioName         string    `json:"studio_name"`
	ClientID           int64     `json:"client_id"`
	ClientName         string    `json:"client_name"`
	ClientPhone        string    `json:"client_phone"`
	ClientEmail        string    `json:"client_email"`
	StartTime          time.Time `json:"start_time"`
	EndTime            time.Time `json:"end_time"`
	Status             string    `json:"status"`
	TotalPrice         float64   `json:"total_price"`
	DepositAmount      float64   `json:"deposit_amount"`
	Balance            float64   `json:"balance"`
	Notes              string    `json:"notes,omitempty"`
	CancellationReason string    `json:"cancellation_reason,omitempty"`
	CreatedAt          time.Time `json:"created_at"`
}

func (r *bookingRepository) GetManagerBookings(
	ctx context.Context,
	ownerID int64,
	filters ManagerBookingFilters,
) ([]ManagerBookingRow, int64, error) {

	// 1) получаем студии владельца
	var studioIDs []int64
	if err := r.db.WithContext(ctx).
		Table("studios").
		Where("owner_id = ?", ownerID).
		Pluck("id", &studioIDs).Error; err != nil {
		return nil, 0, err
	}

	if len(studioIDs) == 0 {
		return []ManagerBookingRow{}, 0, nil
	}

	// 2) базовый запрос
	query := r.db.WithContext(ctx).
		Table("bookings b").
		Select(`
			b.id,
			b.room_id,
			r.name as room_name,
			b.studio_id,
			s.name as studio_name,
			b.user_id as client_id,
			u.name as client_name,
			u.phone as client_phone,
			u.email as client_email,
			b.start_time,
			b.end_time,
			b.status,
			b.total_price,
			COALESCE(b.deposit_amount, 0) as deposit_amount,
			b.total_price - COALESCE(b.deposit_amount, 0) as balance,
			b.notes,
			b.cancellation_reason,
			b.created_at
		`).
		Joins("JOIN rooms r ON r.id = b.room_id").
		Joins("JOIN studios s ON s.id = b.studio_id").
		Joins("JOIN users u ON u.id = b.user_id").
		Where("b.studio_id IN ?", studioIDs)

	// 3) фильтры
	if filters.StudioID > 0 {
		query = query.Where("b.studio_id = ?", filters.StudioID)
	}
	if filters.RoomID > 0 {
		query = query.Where("b.room_id = ?", filters.RoomID)
	}
	if filters.Status != "" && filters.Status != "all" {
		query = query.Where("b.status = ?", filters.Status)
	}
	if !filters.DateFrom.IsZero() {
		query = query.Where("b.start_time >= ?", filters.DateFrom)
	}
	if !filters.DateTo.IsZero() {
		query = query.Where("b.start_time <= ?", filters.DateTo)
	}

	// ⚠️ чтобы работало и в SQLite, и в Postgres:
	if filters.ClientName != "" {
		query = query.Where("LOWER(u.name) LIKE LOWER(?)", "%"+filters.ClientName+"%")
	}

	// 4) total count
	var total int64
	countQuery := query.Session(&gorm.Session{})
	if err := countQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 5) pagination
	if filters.PerPage == 0 {
		filters.PerPage = 20
	}
	if filters.Page == 0 {
		filters.Page = 1
	}
	offset := (filters.Page - 1) * filters.PerPage

	// 6) scan rows
	var rows []ManagerBookingRow
	if err := query.
		Order("b.start_time DESC").
		Limit(filters.PerPage).
		Offset(offset).
		Scan(&rows).Error; err != nil {
		return nil, 0, err
	}

	return rows, total, nil
}

func (r *bookingRepository) GetBookingForManager(ctx context.Context, ownerID, bookingID int64) (*ManagerBookingRow, error) {
	var row ManagerBookingRow

	err := r.db.WithContext(ctx).
		Table("bookings b").
		Select(`
			b.id,
			b.room_id,
			r.name as room_name,
			b.studio_id,
			s.name as studio_name,
			b.user_id as client_id,
			u.name as client_name,
			u.phone as client_phone,
			u.email as client_email,
			b.start_time,
			b.end_time,
			b.status,
			b.total_price,
			COALESCE(b.deposit_amount, 0) as deposit_amount,
			b.total_price - COALESCE(b.deposit_amount, 0) as balance,
			b.notes,
			b.cancellation_reason,
			b.created_at
		`).
		Joins("JOIN rooms r ON r.id = b.room_id").
		Joins("JOIN studios s ON s.id = b.studio_id").
		Joins("JOIN users u ON u.id = b.user_id").
		Where("b.id = ?", bookingID).
		Where("s.owner_id = ?", ownerID).
		First(&row).Error

	if err != nil {
		return nil, err
	}
	return &row, nil
}