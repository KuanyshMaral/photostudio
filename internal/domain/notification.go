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
	NotifNewMessage           NotificationType = "new_message"
)

type Notification struct {
	ID        int64            `json:"id"`
	UserID    int64            `json:"user_id"`
	Type      NotificationType `json:"type"`
	Title     string           `json:"title"`
	Message   string           `json:"message,omitempty"`
	IsRead    bool             `json:"is_read"`
	Data      any              `json:"data,omitempty" gorm:"serializer:json"`
	CreatedAt time.Time        `json:"created_at"`
}
