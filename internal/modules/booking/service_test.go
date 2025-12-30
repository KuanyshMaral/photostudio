package booking

import (
	"context"
	"encoding/json"
	"photostudio/internal/repository"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"photostudio/internal/domain"
)

// Mock repositories
type MockBookingRepository struct {
	mock.Mock
}

func (m *MockBookingRepository) Create(ctx context.Context, b *domain.Booking) error {
	args := m.Called(ctx, b)
	if b != nil {
		b.ID = 999 // simulate DB insert
	}
	return args.Error(0)
}

func (m *MockBookingRepository) GetBusySlotsForRoom(ctx context.Context, roomID int64, start, end time.Time) ([]repository.BusySlot, error) {
	args := m.Called(ctx, roomID, start, end)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]repository.BusySlot), args.Error(1)
}

func (m *MockBookingRepository) GetUserBookingsWithDetails(ctx context.Context, userID int64, limit, offset int) ([]repository.UserBookingDetails, error) {
	args := m.Called(ctx, userID, limit, offset)
	return args.Get(0).([]repository.UserBookingDetails), args.Error(1)
}

func (m *MockBookingRepository) GetStudioOwnerForBooking(ctx context.Context, bookingID int64) (int64, string, error) {
	args := m.Called(ctx, bookingID)
	return 0, args.String(1), args.Error(2)
}

func (m *MockBookingRepository) UpdateStatus(ctx context.Context, bookingID int64, status string) error {
	args := m.Called(ctx, bookingID, status)
	return args.Error(0)
}

func (m *MockBookingRepository) GetByID(ctx context.Context, id int64) (*domain.Booking, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Booking), args.Error(1)
}

type MockRoomRepository struct {
	mock.Mock
}

func (m *MockBookingRepository) CheckAvailability(ctx context.Context, roomID int64, start, end time.Time) (bool, error) {
	args := m.Called(ctx, roomID, start, end)
	return args.Bool(0), args.Error(1)
}

func (m *MockRoomRepository) GetPriceByID(ctx context.Context, id int64) (float64, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(float64), args.Error(1)
}

func (m *MockRoomRepository) GetStudioWorkingHoursByRoomID(ctx context.Context, roomID int64) ([]byte, error) {
	args := m.Called(ctx, roomID)
	return args.Get(0).([]byte), args.Error(1)
}

func TestService_CreateBooking_Success(t *testing.T) {
	mockBookings := new(MockBookingRepository)
	mockRooms := new(MockRoomRepository)

	mockRooms.On("GetPriceByID", mock.Anything, int64(10)).Return(15000.0, nil)

	start := time.Date(2025, 12, 31, 14, 0, 0, 0, time.UTC)
	end := start.Add(2 * time.Hour)
	mockBookings.On("CheckAvailability", mock.Anything, int64(10), start, end).Return(true, nil)
	mockBookings.On("Create", mock.Anything, mock.Anything).Return(nil)

	service := NewService(mockBookings, mockRooms)

	req := CreateBookingRequest{
		RoomID:    10,
		StudioID:  5,
		UserID:    999,
		StartTime: start,
		EndTime:   end,
		Notes:     "Test booking",
	}

	booking, err := service.CreateBooking(context.Background(), req)

	assert.NoError(t, err)
	assert.NotNil(t, booking)
	assert.Equal(t, 30000.0, booking.TotalPrice)
	assert.Equal(t, domain.BookingPending, booking.Status)
}

func TestService_CreateBooking_SlotUnavailable(t *testing.T) {
	mockBookings := new(MockBookingRepository)
	mockRooms := new(MockRoomRepository)

	mockRooms.On("GetPriceByID", mock.Anything, int64(10)).Return(15000.0, nil)
	mockBookings.On("CheckAvailability", mock.Anything, int64(10), mock.Anything, mock.Anything).Return(false, nil)

	service := NewService(mockBookings, mockRooms)

	req := CreateBookingRequest{
		RoomID:    10,
		StudioID:  5,
		UserID:    999,
		StartTime: time.Now().Add(1 * time.Hour),
		EndTime:   time.Now().Add(3 * time.Hour),
	}

	_, err := service.CreateBooking(context.Background(), req)
	assert.ErrorIs(t, err, ErrNotAvailable)
}

func TestService_GetRoomAvailability_WithBusySlots(t *testing.T) {
	mockBookings := new(MockBookingRepository)
	mockRooms := new(MockRoomRepository)

	// Working hours JSON
	wh := map[string]map[string]string{
		"wednesday": {
			"open":  "10:00",
			"close": "18:00",
		},
	}
	whBytes, _ := json.Marshal(wh)
	mockRooms.On("GetStudioWorkingHoursByRoomID", mock.Anything, int64(10)).Return(whBytes, nil)

	// Busy slot
	day := time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC)
	busy := []repository.BusySlot{
		{Start: day.Add(12 * time.Hour), End: day.Add(14 * time.Hour)},
	}
	mockBookings.On("GetBusySlotsForRoom", mock.Anything, int64(10), mock.Anything, mock.Anything).Return(busy, nil)

	service := NewService(mockBookings, mockRooms)

	slots, err := service.GetRoomAvailability(context.Background(), 10, "2025-12-31")

	assert.NoError(t, err)
	assert.Len(t, slots, 2)
	assert.Equal(t, "10:00", slots[0].Start.Format("15:04"))
	assert.Equal(t, "12:00", slots[0].End.Format("15:04"))
}
