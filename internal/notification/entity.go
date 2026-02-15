package notification

import (
	"database/sql"
	"encoding/json"
	"time"
)

// Notification type constants
const (
	TypeBookingCreated          = "booking.created"
	TypeBookingConfirmed        = "booking.confirmed"
	TypeBookingCancelled        = "booking.cancelled"
	TypeReviewPosted            = "review.posted"
	TypeVerificationApproved    = "verification.approved"
	TypeVerificationRejected    = "verification.rejected"
	TypeNewMessage              = "chat.message"
	TypeStudioInfoUpdated       = "studio.info_updated"
	TypeDeviceOffline           = "device.offline"
	TypeMaintenanceNotification = "maintenance.notification"
)

// Channel constants for delivery methods
const (
	ChannelInApp  = "in_app"
	ChannelEmail  = "email"
	ChannelPush   = "push"
	ChannelSMS    = "sms"
)

// NotificationData holds structured notification metadata
type NotificationData struct {
	BookingID    *int64  `json:"booking_id,omitempty"`
	RoomID       *int64  `json:"room_id,omitempty"`
	StudioID     *int64  `json:"studio_id,omitempty"`
	ReviewID     *int64  `json:"review_id,omitempty"`
	UserID       *int64  `json:"user_id,omitempty"`
	VerificationID *int64 `json:"verification_id,omitempty"`
	ChatID       *int64  `json:"chat_id,omitempty"`
	MessageID    *int64  `json:"message_id,omitempty"`
	Extra        map[string]interface{} `json:"extra,omitempty"`
}

// SetData encodes notification data to JSON
func (nd *NotificationData) SetData() string {
	data, _ := json.Marshal(nd)
	return string(data)
}

// GetData decodes notification data from JSON
func (nd *NotificationData) GetData() map[string]interface{} {
	result := make(map[string]interface{})
	bytes := []byte(nd.SetData())
	json.Unmarshal(bytes, &result)
	return result
}

// Notification represents a user notification record
type Notification struct {
	ID        int64          `gorm:"primaryKey" json:"id"`
	UserID    int64          `gorm:"index" json:"user_id"`
	Type      string         `gorm:"index" json:"type"`
	Title     string         `json:"title"`
	Body      string         `json:"body"`
	Data      string         `gorm:"type:jsonb" json:"data"`
	ReadAt    sql.NullTime   `json:"read_at"`
	CreatedAt time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
}

// MarkAsRead marks a notification as read
func (n *Notification) MarkAsRead() {
	n.ReadAt = sql.NullTime{
		Time:  time.Now(),
		Valid: true,
	}
}

// IsRead returns true if notification has been read
func (n *Notification) IsRead() bool {
	return n.ReadAt.Valid
}

// TableName specifies the table name for GORM
func (n *Notification) TableName() string {
	return "notifications"
}

// UserPreferences stores user notification settings
type UserPreferences struct {
	ID              int64                   `gorm:"primaryKey" json:"id"`
	UserID          int64                   `gorm:"uniqueIndex" json:"user_id"`
	GlobalEnabled   bool                    `gorm:"default:true" json:"global_enabled"`
	EmailEnabled    bool                    `gorm:"default:true" json:"email_enabled"`
	PushEnabled     bool                    `gorm:"default:true" json:"push_enabled"`
	PerTypeSettings string                  `gorm:"type:jsonb;default:'{}'" json:"per_type_settings"`
	CreatedAt       time.Time               `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt       time.Time               `gorm:"autoUpdateTime" json:"updated_at"`
}

// GetPerTypeSettings returns parsed per-type settings
func (up *UserPreferences) GetPerTypeSettings() map[string]map[string]bool {
	result := make(map[string]map[string]bool)
	json.Unmarshal([]byte(up.PerTypeSettings), &result)
	return result
}

// SetPerTypeSettings sets per-type settings from map
func (up *UserPreferences) SetPerTypeSettings(settings map[string]map[string]bool) {
	data, _ := json.Marshal(settings)
	up.PerTypeSettings = string(data)
}

// TableName specifies the table name for GORM
func (up *UserPreferences) TableName() string {
	return "user_notification_preferences"
}

// DeviceToken represents a push notification device token
type DeviceToken struct {
	ID        int64     `gorm:"primaryKey" json:"id"`
	UserID    int64     `gorm:"index" json:"user_id"`
	Token     string    `gorm:"uniqueIndex" json:"token"`
	Platform  string    `json:"platform"` // ios, android, web
	Active    bool      `gorm:"default:true" json:"active"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

// TableName specifies the table name for GORM
func (dt *DeviceToken) TableName() string {
	return "device_tokens"
}

// GetDefaultPreferences returns default user preferences
func GetDefaultPreferences(userID int64) *UserPreferences {
	return &UserPreferences{
		UserID:          userID,
		GlobalEnabled:   true,
		EmailEnabled:    true,
		PushEnabled:     true,
		PerTypeSettings: "{}",
	}
}
