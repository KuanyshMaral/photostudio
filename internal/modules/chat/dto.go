package chat

import (
	"photostudio/internal/domain"
	"time"
)

// ============================================================
// REQUEST DTOs — что приходит от клиента
// ============================================================

// CreateConversationRequest — запрос на создание диалога
type CreateConversationRequest struct {
	RecipientID    int64  `json:"recipient_id" binding:"required"`
	StudioID       *int64 `json:"studio_id"`
	BookingID      *int64 `json:"booking_id"`
	InitialMessage string `json:"initial_message"`
}

// SendMessageRequest — запрос на отправку сообщения
type SendMessageRequest struct {
	Content string `json:"content" binding:"required,max=4000"`
}

// BlockUserRequest — запрос на блокировку пользователя
type BlockUserRequest struct {
	Reason string `json:"reason"`
}

// ============================================================
// RESPONSE DTOs — что отправляем клиенту
// ============================================================

type UserBrief struct {
	ID     int64  `json:"id"`
	Name   string `json:"name"`
	Avatar string `json:"avatar,omitempty"`
	Role   string `json:"role,omitempty"`
}

type StudioBrief struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type BookingBrief struct {
	ID        int64  `json:"id"`
	StartTime string `json:"start_time"`
	Status    string `json:"status"`
}

type MessageBrief struct {
	ID        int64  `json:"id"`
	Content   string `json:"content"`
	IsMine    bool   `json:"is_mine"`
	CreatedAt string `json:"created_at"`
}

type ConversationResponse struct {
	ID        int64         `json:"id"`
	OtherUser *UserBrief    `json:"other_user"`
	Studio    *StudioBrief  `json:"studio,omitempty"`
	Booking   *BookingBrief `json:"booking,omitempty"`

	LastMessage   *MessageBrief `json:"last_message,omitempty"`
	UnreadCount   int           `json:"unread_count"`
	LastMessageAt string        `json:"last_message_at"`
	CreatedAt     string        `json:"created_at"`
}

type MessageResponse struct {
	ID             int64              `json:"id"`
	ConversationID int64              `json:"conversation_id"`
	SenderID       int64              `json:"sender_id"`
	Content        string             `json:"content"`
	MessageType    domain.MessageType `json:"message_type"`
	AttachmentURL  *string            `json:"attachment_url,omitempty"`
	IsRead         bool               `json:"is_read"`
	ReadAt         *string            `json:"read_at,omitempty"`
	CreatedAt      string             `json:"created_at"`
	Sender         *UserBrief         `json:"sender,omitempty"`
}

// ============================================================
// CONVERTERS — преобразование Domain → Response
// ============================================================

func ToUserBrief(u *domain.User) *UserBrief {
	if u == nil {
		return nil
	}
	return &UserBrief{
		ID:     u.ID,
		Name:   u.Name,
		Avatar: u.AvatarURL, // важно: в твоём domain.User поле AvatarURL
		Role:   string(u.Role),
	}
}

func ToMessageResponse(m *domain.Message) *MessageResponse {
	if m == nil {
		return nil
	}

	resp := &MessageResponse{
		ID:             m.ID,
		ConversationID: m.ConversationID,
		SenderID:       m.SenderID,
		Content:        m.Content,
		MessageType:    m.MessageType,
		AttachmentURL:  m.AttachmentURL,
		IsRead:         m.IsRead,
		CreatedAt:      m.CreatedAt.Format(time.RFC3339),
		Sender:         ToUserBrief(m.Sender),
	}

	if m.ReadAt != nil {
		ra := m.ReadAt.Format(time.RFC3339)
		resp.ReadAt = &ra
	}

	return resp
}

func ToConversationResponse(c *domain.Conversation, currentUserID int64) *ConversationResponse {
	if c == nil {
		return nil
	}

	resp := &ConversationResponse{
		ID:            c.ID,
		OtherUser:     ToUserBrief(c.OtherUser),
		UnreadCount:   c.UnreadCount,
		LastMessageAt: c.LastMessageAt.Format(time.RFC3339),
		CreatedAt:     c.CreatedAt.Format(time.RFC3339),
	}

	if c.Studio != nil {
		resp.Studio = &StudioBrief{
			ID:   c.Studio.ID,
			Name: c.Studio.Name,
		}
	}

	if c.Booking != nil {
		resp.Booking = &BookingBrief{
			ID:        c.Booking.ID,
			StartTime: c.Booking.StartTime.Format(time.RFC3339),
			Status:    string(c.Booking.Status),
		}
	}

	if c.LastMessage != nil {
		resp.LastMessage = &MessageBrief{
			ID:        c.LastMessage.ID,
			Content:   c.LastMessage.Content,
			IsMine:    c.LastMessage.SenderID == currentUserID,
			CreatedAt: c.LastMessage.CreatedAt.Format(time.RFC3339),
		}
	}

	return resp
}


