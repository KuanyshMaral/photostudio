package booking

import (
	"context"
	"encoding/json"
	"photostudio/internal/repository"
	"testing"
	"time"

	"photostudio/internal/domain"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
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
	return args.Get(0).(int64), args.String(1), args.Error(2)
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

func (m *MockBookingRepository) GetByStudioID(ctx context.Context, studioID int64) ([]domain.Booking, error) {
	args := m.Called(ctx, studioID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.Booking), args.Error(1)
}

func (m *MockBookingRepository) IsBookingOwnedByUser(ctx context.Context, bookingID, ownerID int64) (bool, error) {
	args := m.Called(ctx, bookingID, ownerID)
	return args.Bool(0), args.Error(1)
}

func (m *MockBookingRepository) UpdatePaymentStatus(ctx context.Context, bookingID int64, status domain.PaymentStatus) (*domain.Booking, error) {
	args := m.Called(ctx, bookingID, status)
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

type MockNotificationSender struct {
	mock.Mock
}

func (m *MockNotificationSender) NotifyBookingCreated(ctx context.Context, ownerUserID, bookingID, studioID, roomID int64, start time.Time) error {
	args := m.Called(ctx, ownerUserID, bookingID, studioID, roomID, start)
	return args.Error(0)
}

func (m *MockNotificationSender) NotifyBookingConfirmed(ctx context.Context, clientUserID, bookingID, studioID int64) error {
	args := m.Called(ctx, clientUserID, bookingID, studioID)
	return args.Error(0)
}

func (m *MockNotificationSender) NotifyBookingCancelled(ctx context.Context, clientUserID, bookingID, studioID int64, reason string) error {
	args := m.Called(ctx, clientUserID, bookingID, studioID, reason)
	return args.Error(0)
}

func TestService_CreateBooking_Success(t *testing.T) {
	mockBookings := new(MockBookingRepository)
	mockRooms := new(MockRoomRepository)

	mockRooms.On("GetPriceByID", mock.Anything, int64(10)).Return(15000.0, nil)

	start := time.Date(2026, 12, 31, 14, 0, 0, 0, time.UTC)
	end := start.Add(2 * time.Hour)
	mockBookings.On("CheckAvailability", mock.Anything, int64(10), start, end).Return(true, nil)
	mockBookings.On("Create", mock.Anything, mock.Anything).Return(nil)
	// Mock for notification call
	mockBookings.On("GetStudioOwnerForBooking", mock.Anything, int64(999)).Return(int64(1), "pending", nil)

	mockNotifs := new(MockNotificationSender)
	mockNotifs.On("NotifyBookingCreated", mock.Anything, int64(1), int64(999), int64(5), int64(10), mock.Anything).Return(nil)
	service := NewService(mockBookings, mockRooms, mockNotifs)

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

	mockNotifs := new(MockNotificationSender)
	service := NewService(mockBookings, mockRooms, mockNotifs)

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
	day := time.Date(2026, 12, 30, 0, 0, 0, 0, time.UTC)
	busy := []repository.BusySlot{
		{Start: day.Add(12 * time.Hour), End: day.Add(14 * time.Hour)},
	}
	mockBookings.On("GetBusySlotsForRoom", mock.Anything, int64(10), mock.Anything, mock.Anything).Return(busy, nil)

	mockNotifs := new(MockNotificationSender)
	service := NewService(mockBookings, mockRooms, mockNotifs)

	slots, err := service.GetRoomAvailability(context.Background(), 10, "2026-12-30")

	assert.NoError(t, err)
	assert.Len(t, slots, 2)
	assert.Equal(t, "10:00", slots[0].Start.Format("15:04"))
	assert.Equal(t, "12:00", slots[0].End.Format("15:04"))
}

// ============================================================================
// Day 1 - Additional Unit Tests (Task 3.3)
// Backend Developer #3
// ============================================================================

// Test 1: Validation error when end_time is before start_time
func TestCreateBooking_ValidationError(t *testing.T) {
	mockBookings := new(MockBookingRepository)
	mockRooms := new(MockRoomRepository)
	mockNotifs := new(MockNotificationSender)
	service := NewService(mockBookings, mockRooms, mockNotifs)

	// end_time раньше start_time
	start := time.Date(2026, 12, 31, 14, 0, 0, 0, time.UTC)
	end := time.Date(2026, 12, 31, 12, 0, 0, 0, time.UTC)

	req := CreateBookingRequest{
		RoomID:    10,
		StudioID:  5,
		UserID:    999,
		StartTime: start,
		EndTime:   end,
		Notes:     "Invalid time range",
	}

	_, err := service.CreateBooking(context.Background(), req)

	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrValidation)
}

// Test 2: Attempt to book an already occupied time slot
func TestCreateBooking_Overbooking(t *testing.T) {
	mockBookings := new(MockBookingRepository)
	mockRooms := new(MockRoomRepository)

	mockRooms.On("GetPriceByID", mock.Anything, int64(10)).Return(15000.0, nil)

	start := time.Date(2026, 12, 31, 14, 0, 0, 0, time.UTC)
	end := start.Add(2 * time.Hour)

	// Room is NOT available (already booked)
	mockBookings.On("CheckAvailability", mock.Anything, int64(10), start, end).Return(false, nil)

	mockNotifs := new(MockNotificationSender)
	service := NewService(mockBookings, mockRooms, mockNotifs)

	req := CreateBookingRequest{
		RoomID:    10,
		StudioID:  5,
		UserID:    999,
		StartTime: start,
		EndTime:   end,
		Notes:     "Attempting to overbook",
	}

	_, err := service.CreateBooking(context.Background(), req)

	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrNotAvailable)
}

// Test 3: Get available time slots successfully
func TestGetRoomAvailability_Success(t *testing.T) {
	mockBookings := new(MockBookingRepository)
	mockRooms := new(MockRoomRepository)

	// Working hours for Wednesday
	wh := map[string]map[string]string{
		"wednesday": {
			"open":  "09:00",
			"close": "18:00",
		},
	}
	whBytes, _ := json.Marshal(wh)
	mockRooms.On("GetStudioWorkingHoursByRoomID", mock.Anything, int64(10)).Return(whBytes, nil)

	// No busy slots - fully available
	mockBookings.On("GetBusySlotsForRoom", mock.Anything, int64(10), mock.Anything, mock.Anything).Return([]repository.BusySlot{}, nil)

	mockNotifs := new(MockNotificationSender)
	service := NewService(mockBookings, mockRooms, mockNotifs)

	slots, err := service.GetRoomAvailability(context.Background(), 10, "2026-12-30")

	assert.NoError(t, err)
	assert.NotEmpty(t, slots)
	// Should have at least one available slot from 09:00 to 18:00
	assert.GreaterOrEqual(t, len(slots), 1)
}

// Test 4: Successfully update booking status
func TestUpdateBookingStatus_Success(t *testing.T) {
	mockBookings := new(MockBookingRepository)
	mockRooms := new(MockRoomRepository)

	bookingID := int64(123)
	ownerUserID := int64(999)
	clientUserID := int64(888)

	// Mock: Booking exists and belongs to this owner
	mockBookings.On("GetStudioOwnerForBooking", mock.Anything, bookingID).Return(ownerUserID, "pending", nil)
	// First GetByID for notification
	mockBookings.On("GetByID", mock.Anything, bookingID).Return(&domain.Booking{
		ID:       bookingID,
		UserID:   clientUserID,
		StudioID: int64(5),
		Status:   domain.BookingPending,
	}, nil).Once()
	mockBookings.On("UpdateStatus", mock.Anything, bookingID, "confirmed").Return(nil)
	// Second GetByID for final return
	mockBookings.On("GetByID", mock.Anything, bookingID).Return(&domain.Booking{
		ID:       bookingID,
		UserID:   clientUserID,
		StudioID: int64(5),
		Status:   domain.BookingConfirmed,
	}, nil).Once()

	mockNotifs := new(MockNotificationSender)
	mockNotifs.On("NotifyBookingConfirmed", mock.Anything, clientUserID, bookingID, int64(5)).Return(nil)
	service := NewService(mockBookings, mockRooms, mockNotifs)

	result, err := service.UpdateBookingStatus(context.Background(), bookingID, ownerUserID, string(domain.RoleStudioOwner), "confirmed")

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, domain.BookingConfirmed, result.Status)
	mockBookings.AssertExpectations(t)
}

// Test 5: Forbidden - user tries to update someone else's booking
func TestUpdateBookingStatus_Forbidden(t *testing.T) {
	mockBookings := new(MockBookingRepository)
	mockRooms := new(MockRoomRepository)

	bookingID := int64(123)
	realOwnerUserID := int64(999)
	unauthorizedUserID := int64(888)

	// Mock: Booking belongs to realOwnerUserID, not unauthorizedUserID
	mockBookings.On("GetStudioOwnerForBooking", mock.Anything, bookingID).Return(realOwnerUserID, "pending", nil)

	mockNotifs := new(MockNotificationSender)
	service := NewService(mockBookings, mockRooms, mockNotifs)

	// Unauthorized user tries to update
	_, err := service.UpdateBookingStatus(context.Background(), bookingID, unauthorizedUserID, string(domain.RoleStudioOwner), "confirmed")

	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrForbidden)
}
