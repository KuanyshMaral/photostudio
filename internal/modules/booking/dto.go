package booking

import (
	"photostudio/internal/domain"
	"time"
)

type CreateBookingRequest struct {
	RoomID    int64     `json:"room_id" binding:"required"`
	StudioID  int64     `json:"studio_id" binding:"required"`
	UserID    int64     `json:"user_id"`  // Optional - will be filled from context by middleware
	StartTime time.Time `json:"start_time" binding:"required"`
	EndTime   time.Time `json:"end_time" binding:"required"`
	Notes     string    `json:"notes,omitempty"`
}

type UpdatePaymentStatusRequest struct {
	PaymentStatus domain.PaymentStatus `json:"payment_status" binding:"required,oneof=unpaid paid refunded"`
}

type BookedSlot struct {
	Start  string `json:"start"`
	End    string `json:"end"`
	Status string `json:"status"`
}

type WorkingHours struct {
	Open  string `json:"open"`
	Close string `json:"close"`
}

type AvailabilityResponse struct {
	RoomID       int64        `json:"room_id"`
	Date         string       `json:"date"`
	WorkingHours WorkingHours `json:"working_hours"`
	BookedSlots  []BookedSlot `json:"booked_slots"`
}

// BookingResponse — ответ с информацией о бронировании
type BookingResponse struct {
	ID         int64   `json:"id"`
	RoomID     int64   `json:"room_id"`
	RoomName   string  `json:"room_name,omitempty"`
	StudioID   int64   `json:"studio_id,omitempty"`
	StudioName string  `json:"studio_name,omitempty"`
	StartTime  string  `json:"start_time"`
	EndTime    string  `json:"end_time"`
	Status     string  `json:"status"`
	TotalPrice float64 `json:"total_price"`
	Notes      string  `json:"notes,omitempty"`
	CreatedAt  string  `json:"created_at"`

	// Block 9: Только если отменено
	CancellationReason string `json:"cancellation_reason,omitempty"`

	// Block 10: Для менеджеров
	DepositAmount float64 `json:"deposit_amount,omitempty"`
	Balance       float64 `json:"balance,omitempty"` // TotalPrice - DepositAmount
}

// CancelBookingRequest — запрос на отмену бронирования
// Block 9: Причина обязательна!
type CancelBookingRequest struct {
	Reason string `json:"reason" binding:"required,min=10"`
}

// UpdateDepositRequest — запрос на обновление предоплаты (для менеджеров)
type UpdateDepositRequest struct {
	DepositAmount float64 `json:"deposit_amount" binding:"required,min=0"`
}

// ToBookingResponse конвертирует domain.Booking в DTO
func ToBookingResponse(b *domain.Booking, includeDeposit bool) BookingResponse {
	resp := BookingResponse{
		ID:         b.ID,
		RoomID:     b.RoomID,
		StartTime:  b.StartTime.Format(time.RFC3339),
		EndTime:    b.EndTime.Format(time.RFC3339),
		Status:     string(b.Status),
		TotalPrice: b.TotalPrice,
		Notes:      b.Notes,
		CreatedAt:  b.CreatedAt.Format(time.RFC3339),
	}

	// Room info
	if b.Room != nil {
		resp.RoomName = b.Room.Name
	}

	// Studio info (from Booking's StudioID)
	resp.StudioID = b.StudioID

	// Причина отмены (если есть)
	if b.Status == domain.BookingCancelled {
		resp.CancellationReason = b.CancellationReason
	}

	// Deposit info (только для менеджеров)
	if includeDeposit {
		resp.DepositAmount = b.DepositAmount
		resp.Balance = b.TotalPrice - b.DepositAmount
	}

	return resp
}
