package booking

import (
	"context"
	"photostudio/internal/domain"
	"photostudio/internal/repository"
	"time"
)

// BookingRepository defines the interface for booking operations
type BookingRepository interface {
	CheckAvailability(ctx context.Context, roomID int64, start, end time.Time) (bool, error)
	Create(ctx context.Context, b *domain.Booking) error
	GetBusySlotsForRoom(ctx context.Context, roomID int64, start, end time.Time) ([]repository.BusySlot, error)
	GetUserBookingsWithDetails(ctx context.Context, userID int64, limit, offset int) ([]repository.UserBookingDetails, error)
	GetStudioOwnerForBooking(ctx context.Context, bookingID int64) (ownerID int64, status string, err error)
	UpdateStatus(ctx context.Context, bookingID int64, status string) error
	GetByID(ctx context.Context, id int64) (*domain.Booking, error)
	GetByStudioID(ctx context.Context, studioID int64) ([]domain.Booking, error)
	IsBookingOwnedByUser(ctx context.Context, bookingID, ownerID int64) (bool, error)
	UpdatePaymentStatus(ctx context.Context, bookingID int64, status domain.PaymentStatus) (*domain.Booking, error)
}

// RoomRepository defines the interface for room operations
type RoomRepository interface {
	GetPriceByID(ctx context.Context, id int64) (float64, error)
	GetStudioWorkingHoursByRoomID(ctx context.Context, roomID int64) ([]byte, error)
}
