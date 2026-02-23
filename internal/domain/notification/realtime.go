package notification

import "photostudio/internal/domain/chat"

const EventNotificationCreated = "notification.created"

// RealtimePublisher publishes notification events to connected users.
type RealtimePublisher interface {
	PublishToUser(userID int64, event *chat.WSEvent)
}

// RealtimeNotificationPayload is minimized payload for websocket events.
type RealtimeNotificationPayload struct {
	ID        int64             `json:"id"`
	Type      string            `json:"type"`
	Title     string            `json:"title"`
	Body      *string           `json:"body,omitempty"`
	Data      *NotificationData `json:"data,omitempty"`
	Link      *string           `json:"link,omitempty"`
	IsRead    bool              `json:"is_read"`
	CreatedAt string            `json:"created_at"`
}

func RealtimePayloadFromNotificationResponse(n *NotificationResponse) *RealtimeNotificationPayload {
	if n == nil {
		return nil
	}
	return &RealtimeNotificationPayload{
		ID:        n.ID,
		Type:      n.Type,
		Title:     n.Title,
		Body:      n.Body,
		Data:      n.Data,
		Link:      nil,
		IsRead:    n.IsRead,
		CreatedAt: n.CreatedAt,
	}
}
