package booking

import (
	"context"
	"math"
	"time"

	"photostudio/internal/domain"
	"photostudio/internal/repository"

	"github.com/jackc/pgx/v5/pgconn"
)

type Service struct {
	bookings *repository.BookingRepository
	rooms    *repository.RoomRepository
}

func NewService(bookings *repository.BookingRepository, rooms *repository.RoomRepository) *Service {
	return &Service{bookings: bookings, rooms: rooms}
}

func (s *Service) CreateBooking(ctx context.Context, req CreateBookingRequest) (*domain.Booking, error) {
	if req.EndTime.Before(req.StartTime) || req.EndTime.Equal(req.StartTime) {
		return nil, ErrValidation
	}

	now := time.Now()
	if req.StartTime.Before(now) {
		return nil, ErrValidation
	}

	ok, err := s.bookings.CheckAvailability(ctx, req.RoomID, req.StartTime, req.EndTime)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrNotAvailable
	}

	pricePerHour, err := s.rooms.GetPriceByID(ctx, req.RoomID)
	if err != nil {
		return nil, err
	}

	durationHours := req.EndTime.Sub(req.StartTime).Hours()
	total := durationHours * pricePerHour
	total = math.Round(total*100) / 100

	b := &domain.Booking{
		RoomID:        req.RoomID,
		StudioID:      req.StudioID,
		UserID:        req.UserID,
		StartTime:     req.StartTime,
		EndTime:       req.EndTime,
		TotalPrice:    total,
		Status:        domain.BookingPending,
		PaymentStatus: domain.PaymentUnpaid,
		Notes:         req.Notes,
	}

	if err := s.bookings.Create(ctx, b); err != nil {
		if pgErr, ok := err.(*pgconn.PgError); ok {
			if pgErr.Code == "23505" && pgErr.ConstraintName == "idx_no_overbooking" {
				return nil, ErrOverbooking
			}
		}
		return nil, err
	}

	return b, nil
}
