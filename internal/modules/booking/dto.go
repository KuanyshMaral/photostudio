package booking

import (
	"photostudio/internal/domain"
	"time"
)

type CreateBookingRequest struct {
	RoomID    int64     `json:"room_id" binding:"required"`
	StudioID  int64     `json:"studio_id" binding:"required"`
	UserID    int64     `json:"user_id" binding:"required"`
	StartTime time.Time `json:"start_time" binding:"required"`
	EndTime   time.Time `json:"end_time" binding:"required"`
	Notes     string    `json:"notes"`
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
