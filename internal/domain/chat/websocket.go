package chat

import (
	"context"
	"encoding/json"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"log"
	"net/http"
	"photostudio/internal/pkg/jwt"
	"time"
)

// WebSocket upgrader с настройками
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,

	// Разрешаем подключения с любого origin (для dev)
	// В production заменить на проверку origin
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// WSHandler обрабатывает WebSocket соединения
type WSHandler struct {
	hub         *Hub
	jwtService  *jwt.Service
	chatService *Service
}

// NewWSHandler создаёт новый WebSocket handler
func NewWSHandler(hub *Hub, jwtService *jwt.Service, chatService *Service) *WSHandler {
	return &WSHandler{
		hub:         hub,
		jwtService:  jwtService,
		chatService: chatService,
	}
}

// HandleWebSocket обрабатывает WebSocket подключение
//
// Endpoint: GET /ws/chat?token=JWT_TOKEN
//
// Аутентификация через query parameter (WebSocket не поддерживает headers)
func (h *WSHandler) HandleWebSocket(c *gin.Context) {
	// 1. Получаем токен из query
	token := c.Query("token")
	if token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Token is required. Use ?token=YOUR_JWT_TOKEN",
		})
		return
	}

	// 2. Валидируем токен
	claims, err := h.jwtService.ValidateToken(token)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Invalid or expired token",
		})
		return
	}

	userID := claims.UserID // Или claims.UserId / claims.ID / claims.Sub — см. Ниже

	// 3. Upgrade HTTP → WebSocket
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}

	// 4. Регистрируем соединение
	h.hub.Register(userID, conn)
	log.Printf("User %d connected via WebSocket", userID)

	// 5. Cleanup при закрытии
	defer func() {
		h.hub.Unregister(userID)
		conn.Close()
		log.Printf("User %d disconnected from WebSocket", userID)
	}()

	// 6. Настраиваем ping/pong для keep-alive
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	// 7. Запускаем ping ticker в отдельной goroutine
	go h.pingLoop(conn)

	// 8. Читаем сообщения
	h.readLoop(conn, userID)
}

// pingLoop отправляет ping каждые 30 секунд
func (h *WSHandler) pingLoop(conn *websocket.Conn) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
			return
		}
	}
}

// readLoop читает сообщения от клиента
func (h *WSHandler) readLoop(conn *websocket.Conn, userID int64) {
	for {
		// Читаем сообщение
		_, rawMsg, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err,
				websocket.CloseGoingAway,
				websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error for user %d: %v", userID, err)
			}
			return
		}

		// Парсим JSON
		var msg WSClientMessage
		if err := json.Unmarshal(rawMsg, &msg); err != nil {
			h.sendError(conn, "INVALID_JSON", "Failed to parse message")
			continue
		}

		// Обрабатываем по типу
		switch msg.Type {
		case "message":
			h.handleMessage(conn, userID, msg)
		case "typing":
			h.handleTyping(userID, msg)
		case "read":
			h.handleRead(userID, msg)
		case "ping":
			conn.WriteJSON(NewPongEvent())
		default:
			h.sendError(conn, "UNKNOWN_TYPE", "Unknown message type: "+msg.Type)
		}
	}
}

// handleMessage обрабатывает отправку сообщения
func (h *WSHandler) handleMessage(conn *websocket.Conn, senderID int64, msg WSClientMessage) {
	ctx := context.Background()

	// Валидация
	if msg.ConversationID <= 0 {
		h.sendError(conn, "INVALID_CONVERSATION", "conversation_id is required")
		return
	}
	if msg.Content == "" {
		h.sendError(conn, "EMPTY_CONTENT", "content is required")
		return
	}

	// Отправляем через service
	newMsg, err := h.chatService.SendMessage(ctx, senderID, msg.ConversationID,
		SendMessageRequest{Content: msg.Content})
	if err != nil {
		h.sendError(conn, "SEND_FAILED", err.Error())
		return
	}

	// Получаем conversation для определения получателя
	conv, err := h.chatService.chatRepo.GetConversationByID(ctx, msg.ConversationID)
	if err != nil {
		return
	}

	recipientID := h.chatService.GetRecipientID(conv, senderID)

	// Создаём событие
	event := NewMessageEvent(msg.ConversationID, newMsg)

	// Отправляем обоим участникам
	h.hub.SendToUser(senderID, event)                 // Отправителю (подтверждение)
	delivered := h.hub.SendToUser(recipientID, event) // Получателю

	// Если не доставлено — создаём notification
	if !delivered {
		h.chatService.NotifyIfOffline(ctx, recipientID, conv, newMsg)
	}
}

// handleTyping обрабатывает индикатор "печатает"
func (h *WSHandler) handleTyping(userID int64, msg WSClientMessage) {
	ctx := context.Background()

	if msg.ConversationID <= 0 {
		return
	}

	// Проверяем участие
	if !h.chatService.IsParticipant(ctx, userID, msg.ConversationID) {
		return
	}

	// Получаем conversation
	conv, err := h.chatService.chatRepo.GetConversationByID(ctx, msg.ConversationID)
	if err != nil {
		return
	}

	recipientID := h.chatService.GetRecipientID(conv, userID)

	// Отправляем индикатор получателю
	event := NewTypingEvent(msg.ConversationID, userID, msg.IsTyping)
	h.hub.SendToUser(recipientID, event)
}

// handleRead обрабатывает "прочитано"
func (h *WSHandler) handleRead(userID int64, msg WSClientMessage) {
	ctx := context.Background()

	if msg.ConversationID <= 0 {
		return
	}

	// Помечаем как прочитанные в БД
	_, err := h.chatService.MarkAsRead(ctx, userID, msg.ConversationID)
	if err != nil {
		return
	}

	// Получаем conversation
	conv, err := h.chatService.chatRepo.GetConversationByID(ctx, msg.ConversationID)
	if err != nil {
		return
	}

	recipientID := h.chatService.GetRecipientID(conv, userID)

	// Отправляем событие отправителю сообщений
	event := NewReadEvent(msg.ConversationID, userID)
	h.hub.SendToUser(recipientID, event)
}

// sendError отправляет ошибку клиенту
func (h *WSHandler) sendError(conn *websocket.Conn, code, message string) {
	conn.WriteJSON(NewErrorEvent(code, message))
}
