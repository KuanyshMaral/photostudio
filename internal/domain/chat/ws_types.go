package chat


type WSClientMessage struct {
	Type           string `json:"type"`
	ConversationID int64  `json:"conversation_id,omitempty"`
	Content        string `json:"content,omitempty"`
	IsTyping       bool   `json:"is_typing,omitempty"`
}

type WSServerMessage struct {
	Type           string          `json:"type"`
	ConversationID int64           `json:"conversation_id,omitempty"`
	Message        *Message `json:"message,omitempty"`
	UserID         int64           `json:"user_id,omitempty"`
	IsTyping       bool            `json:"is_typing,omitempty"`
	ErrorCode      string          `json:"code,omitempty"`
	ErrorMessage   string          `json:"message,omitempty"`
}

func NewMessageEvent(conversationID int64, msg *Message) *WSServerMessage {
	return &WSServerMessage{
		Type:           "new_message",
		ConversationID: conversationID,
		Message:        msg,
	}
}

func NewTypingEvent(conversationID, userID int64, isTyping bool) *WSServerMessage {
	return &WSServerMessage{
		Type:           "typing",
		ConversationID: conversationID,
		UserID:         userID,
		IsTyping:       isTyping,
	}
}

func NewReadEvent(conversationID, userID int64) *WSServerMessage {
	return &WSServerMessage{
		Type:           "read",
		ConversationID: conversationID,
		UserID:         userID,
	}
}

func NewPongEvent() *WSServerMessage {
	return &WSServerMessage{
		Type: "pong",
	}
}

func NewErrorEvent(code, message string) *WSServerMessage {
	return &WSServerMessage{
		Type:         "error",
		ErrorCode:    code,
		ErrorMessage: message,
	}
}
