package booking

import (
	"context"
	"photostudio/internal/domain/auth"
	"photostudio/internal/domain/catalog"
	"time"

	"gorm.io/gorm"
)

// BookingRepository defines the interface for booking operations
type BookingRepository interface {
	CheckAvailability(ctx context.Context, roomID int64, start, end time.Time) (bool, error)
	Create(ctx context.Context, b *Booking) error
	GetBusySlotsForRoom(ctx context.Context, roomID int64, start, end time.Time) ([]BusySlot, error)
	GetUserBookingsWithDetails(ctx context.Context, userID int64, limit, offset int) ([]UserBookingDetails, error)
	GetStudioOwnerForBooking(ctx context.Context, bookingID int64) (ownerID int64, status string, err error)
	UpdateStatus(ctx context.Context, bookingID int64, status string) error
	GetByID(ctx context.Context, id int64) (*Booking, error)
	GetByStudioID(ctx context.Context, studioID int64) ([]Booking, error)
	IsBookingOwnedByUser(ctx context.Context, bookingID, ownerID int64) (bool, error)
	UpdatePaymentStatus(ctx context.Context, bookingID int64, status PaymentStatus) (*Booking, error)
	UpdatePaymentStatusSystem(ctx context.Context, bookingID int64, status PaymentStatus) (*Booking, error)
	// Block 9: Cancel with reason
	CancelWithReason(ctx context.Context, bookingID int64, reason string) error
	// Block 10: Update deposit
	UpdateDeposit(ctx context.Context, bookingID int64, amount float64) error

	// Manager methods
	GetManagerBookings(ctx context.Context, ownerID int64, filters ManagerBookingFilters) ([]ManagerBookingRow, int64, error)
	GetBookingForManager(ctx context.Context, ownerID, bookingID int64) (*ManagerBookingRow, error)

	GetRecentByUserID(userID int64, limit int) ([]auth.RecentBookingRow, error)
	GetStatsByUserID(userID int64) (*auth.BookingStats, error)
	HasCompletedBookingForStudio(ctx context.Context, userID, studioID int64) (bool, error)
	DB() *gorm.DB
}

// RoomRepository defines the interface for room operations
type RoomRepository interface {
	GetPriceByID(ctx context.Context, id int64) (float64, error)
	GetStudioWorkingHoursByRoomID(ctx context.Context, roomID int64) ([]byte, error)
	// Добавляем метод GetByID
	GetByID(ctx context.Context, roomID int64) (*catalog.Room, error)
}

type NotificationSender interface {
	NotifyBookingCreated(ctx context.Context, ownerUserID, bookingID, studioID, roomID int64, start time.Time) error
	NotifyBookingConfirmed(ctx context.Context, clientUserID, bookingID, studioID int64) error
	NotifyBookingCancelled(ctx context.Context, clientUserID, bookingID, studioID int64, reason string) error
}
