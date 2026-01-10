package domain

import "time"

type NotificationType string

const (
	NotifBookingCreated       NotificationType = "booking_created"
	NotifBookingConfirmed     NotificationType = "booking_confirmed"
	NotifBookingCancelled     NotificationType = "booking_cancelled"
	NotifVerificationApproved NotificationType = "verification_approved"
	NotifVerificationRejected NotificationType = "verification_rejected"
	NotifNewReview            NotificationType = "new_review"
)

type Notification struct {
	ID        int64            `json:"id"`
	UserID    int64            `json:"user_id"`
	Type      NotificationType `json:"type"`
	Title     string           `json:"title"`
	Message   string           `json:"message,omitempty"`
	IsRead    bool             `json:"is_read"`
	Data      any              `json:"data,omitempty"`
	CreatedAt time.Time        `json:"created_at"`
}
