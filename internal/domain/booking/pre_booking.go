package booking

import "time"

type PreBookingStatus string

const (
	PreBookingPending         PreBookingStatus = "pending"
	PreBookingConfirmedUnpaid PreBookingStatus = "confirmed_unpaid"
	PreBookingPaidConfirmed   PreBookingStatus = "paid_confirmed"
	PreBookingCancelled       PreBookingStatus = "cancelled"
	PreBookingExpired         PreBookingStatus = "expired"
)

type PreBooking struct {
	ID        int64            `json:"id"`
	UserID    int64            `json:"user_id"`
	StudioID  int64            `json:"studio_id"`
	StartTime time.Time        `json:"start_time"`
	EndTime   time.Time        `json:"end_time"`
	Status    PreBookingStatus `json:"status"`
	ExpiresAt time.Time        `json:"expires_at"`
	CreatedAt time.Time        `json:"created_at"`
	UpdatedAt time.Time        `json:"updated_at"`
}
