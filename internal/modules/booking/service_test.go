package booking

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"photostudio/internal/domain"
)

type mockBookingRepo struct {
	mock.Mock
}

func (m *mockBookingRepo) CheckAvailability(ctx context.Context, roomID int64, start, end time.Time) (bool, error) {
	args := m.Called(ctx, roomID, start, end)
	return args.Bool(0), args.Error(1)
}

func (m *mockBookingRepo) Create(ctx context.Context, b *domain.Booking) error {
	args := m.Called(ctx, b)
	return args.Error(0)
}

type mockRoomRepo struct {
	mock.Mock
}

func (m *mockRoomRepo) GetByID(ctx context.Context, id int64) (*domain.Room, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Room), args.Error(1)
}

func TestService_CreateBooking_Success(t *testing.T) {
	bookingRepo := new(mockBookingRepo)
	roomRepo := new(mockRoomRepo)

	room := &domain.Room{
		ID:              10,
		StudioID:        5,
		PricePerHourMin: 15000,
	}

	roomRepo.On("GetByID", mock.Anything, int64(10)).Return(room, nil)
	bookingRepo.On("CheckAvailability", mock.Anything, int64(10),
		mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time")).Return(true, nil)
	bookingRepo.On("Create", mock.Anything, mock.Anything).Return(nil)

	service := NewService(bookingRepo, roomRepo)

	start := time.Date(2025, 12, 31, 14, 0, 0, 0, time.UTC)
	end := start.Add(2 * time.Hour)

	booking, err := service.CreateBooking(context.Background(), 999, CreateBookingRequest{
		RoomID:    10,
		StartTime: start,
		EndTime:   end,
		Notes:     "Photo session",
	})

	assert.NoError(t, err)
	assert.NotNil(t, booking)
	assert.Equal(t, 30000.0, booking.TotalPrice) // 2 hours * 15000
}

func TestService_CreateBooking_SlotUnavailable(t *testing.T) {
	bookingRepo := new(mockBookingRepo)
	roomRepo := new(mockRoomRepo)

	roomRepo.On("GetByID", mock.Anything, int64(10)).Return(&domain.Room{PricePerHourMin: 15000}, nil)
	bookingRepo.On("CheckAvailability", mock.Anything, int64(10), mock.Anything, mock.Anything).Return(false, nil)

	service := NewService(bookingRepo, roomRepo)

	_, err := service.CreateBooking(context.Background(), 999, CreateBookingRequest{
		RoomID:    10,
		StartTime: time.Now(),
		EndTime:   time.Now().Add(2 * time.Hour),
	})

	assert.Error(t, err)
	assert.Equal(t, ErrSlotUnavailable, err)
}
