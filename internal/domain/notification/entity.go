package notification

import (
	"database/sql"
	"encoding/json"
	"time"
)

// Type represents notification type
type Type string

const (
	// Booking notifications
	TypeBookingCreated   Type = "booking_created"   // Owner: новое бронирование
	TypeBookingConfirmed Type = "booking_confirmed" // Client: бронирование подтверждено
	TypeBookingCancelled Type = "booking_cancelled" // Client: бронирование отменено
	TypeBookingCompleted Type = "booking_completed" // Both: бронирование завершено

	// Verification notifications
	TypeVerificationApproved Type = "verification_approved" // Owner: верификация студии одобрена
	TypeVerificationRejected Type = "verification_rejected" // Owner: верификация студии отклонена

	// Review & Feedback
	TypeNewReview Type = "new_review" // Owner: новый отзыв

	// Communication
	TypeNewMessage Type = "new_message" // Both: новое сообщение в чате

	// Equipment & Rooms
	TypeEquipmentBooked Type = "equipment_booked" // Owner: оборудование забронировано

	// Studio updates
	TypeStudioUpdated Type = "studio_updated" // Followers: студия обновлена
)

// Notification represents a user notification
type Notification struct {
	ID        int64           `db:"id" gorm:"primaryKey;column:id" json:"id"`
	UserID    int64           `db:"user_id" gorm:"column:user_id;index:idx_notifications_user_unread" json:"user_id"`
	Type      Type            `db:"type" gorm:"column:type" json:"type"`
	Title     string          `db:"title" gorm:"column:title" json:"title"`
	Body      sql.NullString  `db:"body" gorm:"column:body" json:"body,omitempty"`
	Data      json.RawMessage `db:"data" gorm:"column:data;type:jsonb" json:"data,omitempty"`
	IsRead    bool            `db:"is_read" gorm:"column:is_read;index:idx_notifications_user_unread" json:"is_read"`
	ReadAt    sql.NullTime    `db:"read_at" gorm:"column:read_at" json:"read_at,omitempty"`
	CreatedAt time.Time       `db:"created_at" gorm:"column:created_at;index:idx_notifications_user_created,expression:user_id,created_at DESC" json:"created_at"`
}

// TableName specifies table name for GORM
func (Notification) TableName() string {
	return "notifications"
}

// NotificationData for linking to entities - structured data for notifications
type NotificationData struct {
	BookingID          *int64  `json:"booking_id,omitempty"`
	StudioID           *int64  `json:"studio_id,omitempty"`
	RoomID             *int64  `json:"room_id,omitempty"`
	ReviewID           *int64  `json:"review_id,omitempty"`
	EquipmentID        *int64  `json:"equipment_id,omitempty"`
	MessageID          *int64  `json:"message_id,omitempty"`
	ChatRoomID         *int64  `json:"chat_room_id,omitempty"`
	Rating             *int    `json:"rating,omitempty"`
	SenderName         *string `json:"sender_name,omitempty"`
	MessagePreview     *string `json:"message_preview,omitempty"`
	Reason             *string `json:"reason,omitempty"`
	StartTime          *string `json:"start_time,omitempty"` // ISO8601 format
	EndTime            *string `json:"end_time,omitempty"`   // ISO8601 format
	CancellationReason *string `json:"cancellation_reason,omitempty"`
}

// SetData encodes data to JSON
func (n *Notification) SetData(data *NotificationData) error {
	if data != nil {
		b, err := json.Marshal(data)
		if err != nil {
			return err
		}
		n.Data = b
	}
	return nil
}

// GetData decodes data from JSON
func (n *Notification) GetData() *NotificationData {
	if n.Data == nil || len(n.Data) == 0 {
		return &NotificationData{}
	}
	var data NotificationData
	_ = json.Unmarshal(n.Data, &data)
	return &data
}

// MarkAsRead marks notification as read with timestamp
func (n *Notification) MarkAsRead() {
	n.IsRead = true
	now := time.Now()
	n.ReadAt = sql.NullTime{Time: now, Valid: true}
}
