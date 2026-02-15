package notification

import (
	"time"
)

// NotificationResponse for API responses
type NotificationResponse struct {
	ID        int64                  `json:"id"`
	UserID    int64                  `json:"user_id"`
	Type      string                 `json:"type"`
	Title     string                 `json:"title"`
	Body      *string                `json:"body,omitempty"`
	Data      *NotificationData      `json:"data,omitempty"`
	IsRead    bool                   `json:"is_read"`
	ReadAt    *string                `json:"read_at,omitempty"`
	CreatedAt string                 `json:"created_at"`
}

// NotificationResponseFromEntity converts entity to response DTO
func NotificationResponseFromEntity(n *Notification) *NotificationResponse {
	resp := &NotificationResponse{
		ID:        n.ID,
		UserID:    n.UserID,
		Type:      string(n.Type),
		Title:     n.Title,
		IsRead:    n.IsRead,
		CreatedAt: n.CreatedAt.Format(time.RFC3339),
	}

	if n.Body.Valid {
		resp.Body = &n.Body.String
	}

	if n.Data != nil && len(n.Data) > 0 {
		resp.Data = n.GetData()
	}

	if n.ReadAt.Valid {
		readAt := n.ReadAt.Time.Format(time.RFC3339)
		resp.ReadAt = &readAt
	}

	return resp
}

// NotificationListResponse for list endpoint
type NotificationListResponse struct {
	Notifications []*NotificationResponse `json:"notifications"`
	UnreadCount   int64                   `json:"unread_count"`
	Total         int64                   `json:"total"`
}

// UnreadCountResponse for unread count endpoint
type UnreadCountResponse struct {
	UnreadCount int64 `json:"unread_count"`
}

// CreateNotificationRequest for creating notifications via API
type CreateNotificationRequest struct {
	UserID int64              `json:"user_id" validate:"required"`
	Type   string             `json:"type" validate:"required"`
	Title  string             `json:"title" validate:"required,max=255"`
	Body   *string            `json:"body,omitempty"`
	Data   *NotificationData  `json:"data,omitempty"`
}

// MarkAsReadRequest for marking notifications as read
type MarkAsReadRequest struct {
	ID int64 `json:"id" validate:"required"`
}

// PreferencesResponse for notification preferences endpoint
type PreferencesResponse struct {
	ID              int64                       `json:"id"`
	UserID          int64                       `json:"user_id"`
	EmailEnabled    bool                        `json:"email_enabled"`
	PushEnabled     bool                        `json:"push_enabled"`
	InAppEnabled    bool                        `json:"in_app_enabled"`
	DigestEnabled   bool                        `json:"digest_enabled"`
	DigestFrequency string                      `json:"digest_frequency"`
	PerTypeSettings map[string]ChannelSettings `json:"per_type_settings,omitempty"`
	CreatedAt       string                      `json:"created_at"`
	UpdatedAt       string                      `json:"updated_at"`
}

// ChannelSettings represents which channels a notification type should use
type ChannelSettings struct {
	InApp bool `json:"in_app"`
	Email bool `json:"email"`
	Push  bool `json:"push"`
}

// UpdatePreferencesRequest for updating notification preferences
type UpdatePreferencesRequest struct {
	EmailEnabled    *bool                       `json:"email_enabled,omitempty"`
	PushEnabled     *bool                       `json:"push_enabled,omitempty"`
	InAppEnabled    *bool                       `json:"in_app_enabled,omitempty"`
	DigestEnabled   *bool                       `json:"digest_enabled,omitempty"`
	DigestFrequency *string                     `json:"digest_frequency,omitempty"`
	PerTypeSettings map[string]ChannelSettings `json:"per_type_settings,omitempty"`
}

// DeviceTokenResponse for device tokens endpoint
type DeviceTokenResponse struct {
	ID         int64  `json:"id"`
	UserID     int64  `json:"user_id"`
	Token      string `json:"token"`
	Platform   string `json:"platform"`
	DeviceName string `json:"device_name,omitempty"`
	IsActive   bool   `json:"is_active"`
	CreatedAt  string `json:"created_at"`
	LastUsedAt string `json:"last_used_at,omitempty"`
}

// RegisterDeviceTokenRequest for registering a new device token
type RegisterDeviceTokenRequest struct {
	Token      string `json:"token" validate:"required"`
	Platform   string `json:"platform" validate:"required,oneof=web ios android"`
	DeviceName string `json:"device_name,omitempty"`
}
