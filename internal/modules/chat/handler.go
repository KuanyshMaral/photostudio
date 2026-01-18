package chat

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"strconv"

	"photostudio/internal/pkg/response"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers chat routes under protected group (JWT required).
// Base path is /api/v1/chat
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	chatGroup := rg.Group("/chat")
	{
		chatGroup.POST("/conversations", h.CreateConversation)
		chatGroup.GET("/conversations", h.ListConversations)

		chatGroup.GET("/conversations/:id/messages", h.GetMessages)
		chatGroup.POST("/conversations/:id/messages", h.SendMessage)
		chatGroup.POST("/conversations/:id/messages/upload", h.UploadImage) // <-- NEW
		chatGroup.POST("/conversations/:id/read", h.MarkAsRead)

		chatGroup.POST("/users/:id/block", h.BlockUser)
		chatGroup.DELETE("/users/:id/block", h.UnblockUser)
	}
}

func (h *Handler) CreateConversation(c *gin.Context) {
	userID := c.GetInt64("user_id")

	var req CreateConversationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}

	conv, initialMsg, err := h.service.GetOrCreateConversation(c.Request.Context(), userID, req)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "CHAT_ERROR", err.Error())
		return
	}

	resp := ToConversationResponse(conv, userID)
	out := gin.H{"conversation": resp}
	if initialMsg != nil {
		out["initial_message"] = ToMessageResponse(initialMsg)
	}

	response.Success(c, http.StatusCreated, out)
}

func (h *Handler) ListConversations(c *gin.Context) {
	userID := c.GetInt64("user_id")

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	convs, err := h.service.GetUserConversations(c.Request.Context(), userID, limit, offset)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "FETCH_ERROR", err.Error())
		return
	}

	items := make([]*ConversationResponse, 0, len(convs))
	for i := range convs {
		items = append(items, ToConversationResponse(&convs[i], userID))
	}

	response.Success(c, http.StatusOK, gin.H{"conversations": items})
}

func (h *Handler) GetMessages(c *gin.Context) {
	userID := c.GetInt64("user_id")

	conversationID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "INVALID_ID", "Invalid conversation ID")
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))

	var beforeID *int64
	if v := c.Query("before_id"); v != "" {
		id, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", "before_id must be integer")
			return
		}
		beforeID = &id
	}

	msgs, hasMore, err := h.service.GetMessages(c.Request.Context(), userID, conversationID, limit, beforeID)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "CHAT_ERROR", err.Error())
		return
	}

	out := make([]*MessageResponse, 0, len(msgs))
	for i := range msgs {
		out = append(out, ToMessageResponse(&msgs[i]))
	}

	response.Success(c, http.StatusOK, gin.H{
		"messages": out,
		"has_more": hasMore,
	})
}

func (h *Handler) SendMessage(c *gin.Context) {
	userID := c.GetInt64("user_id")

	conversationID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "INVALID_ID", "Invalid conversation ID")
		return
	}

	var req SendMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}

	msg, err := h.service.SendMessage(c.Request.Context(), userID, conversationID, req)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "CHAT_ERROR", err.Error())
		return
	}

	response.Success(c, http.StatusCreated, gin.H{"message": ToMessageResponse(msg)})
}

func (h *Handler) MarkAsRead(c *gin.Context) {
	userID := c.GetInt64("user_id")

	conversationID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "INVALID_ID", "Invalid conversation ID")
		return
	}

	updated, err := h.service.MarkAsRead(c.Request.Context(), userID, conversationID)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "CHAT_ERROR", err.Error())
		return
	}

	response.Success(c, http.StatusOK, gin.H{"updated": updated})
}

func (h *Handler) BlockUser(c *gin.Context) {
	userID := c.GetInt64("user_id")

	targetID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "INVALID_ID", "Invalid user ID")
		return
	}

	var req BlockUserRequest
	_ = c.ShouldBindJSON(&req)

	if err := h.service.BlockUser(c.Request.Context(), userID, targetID, req.Reason); err != nil {
		response.Error(c, http.StatusBadRequest, "CHAT_ERROR", err.Error())
		return
	}

	response.Success(c, http.StatusOK, gin.H{"message": "User blocked"})
}

func (h *Handler) UnblockUser(c *gin.Context) {
	userID := c.GetInt64("user_id")

	targetID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "INVALID_ID", "Invalid user ID")
		return
	}

	if err := h.service.UnblockUser(c.Request.Context(), userID, targetID); err != nil {
		response.Error(c, http.StatusBadRequest, "CHAT_ERROR", err.Error())
		return
	}

	response.Success(c, http.StatusOK, gin.H{"message": "User unblocked"})
}

// UploadImage загружает изображение в чат
//
// Method: POST /api/v1/conversations/:id/messages/upload
// Content-Type: multipart/form-data
// Body: image (file)
//
// Валидация:
// - Размер: максимум 5 MB
// - Формат: jpg, jpeg, png, webp
// - Права: только участник диалога
func (h *Handler) UploadImage(c *gin.Context) {
	// 1. Получаем user_id из JWT
	userID := c.GetInt64("user_id")
	if userID == 0 {
		response.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "User not authenticated")
		return
	}

	// 2. Получаем conversation_id из URL
	conversationID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || conversationID <= 0 {
		response.Error(c, http.StatusBadRequest, "INVALID_ID", "Invalid conversation ID")
		return
	}

	// 3. Проверяем что пользователь — участник диалога
	if !h.service.IsParticipant(c.Request.Context(), userID, conversationID) {
		response.Error(c, http.StatusForbidden, "NOT_PARTICIPANT", "You are not a participant of this conversation")
		return
	}

	// 4. Получаем файл из multipart form
	file, err := c.FormFile("image")
	if err != nil {
		response.Error(c, http.StatusBadRequest, "NO_FILE", "Image file is required")
		return
	}

	// 5. Валидация размера (5 MB = 5 * 1024 * 1024 bytes)
	const maxSize = 5 * 1024 * 1024
	if file.Size > maxSize {
		response.Error(c, http.StatusBadRequest, "FILE_TOO_LARGE",
			fmt.Sprintf("File size exceeds %d MB limit", maxSize/(1024*1024)))
		return
	}

	// 6. Валидация расширения
	ext := strings.ToLower(filepath.Ext(file.Filename))
	allowedExts := map[string]bool{
		".jpg":  true,
		".jpeg": true,
		".png":  true,
		".webp": true,
	}
	if !allowedExts[ext] {
		response.Error(c, http.StatusBadRequest, "INVALID_FORMAT",
			"Only jpg, jpeg, png, webp files are allowed")
		return
	}

	// 7. Создаём директорию для файлов диалога
	uploadDir := fmt.Sprintf("./uploads/chat/%d", conversationID)
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		response.Error(c, http.StatusInternalServerError, "MKDIR_FAILED",
			"Failed to create upload directory")
		return
	}

	// 8. Генерируем уникальное имя файла
	// Формат: {timestamp_ns}{ext} — гарантирует уникальность
	filename := fmt.Sprintf("%d%s", time.Now().UnixNano(), ext)
	savePath := filepath.Join(uploadDir, filename)

	// 9. Сохраняем файл
	if err := c.SaveUploadedFile(file, savePath); err != nil {
		response.Error(c, http.StatusInternalServerError, "SAVE_FAILED",
			"Failed to save uploaded file")
		return
	}

	// 10. Формируем URL для доступа к файлу
	// /static уже настроен в main.go: r.Static("/static", "./uploads")
	imageURL := fmt.Sprintf("/static/chat/%d/%s", conversationID, filename)

	// 11. Создаём сообщение с типом image
	msg, err := h.service.SendImageMessage(
		c.Request.Context(),
		userID,
		conversationID,
		imageURL,
	)
	if err != nil {
		// Удаляем файл если не удалось создать сообщение
		_ = os.Remove(savePath)

		switch {
		case errors.Is(err, ErrNotParticipant):
			response.Error(c, http.StatusForbidden, "NOT_PARTICIPANT", err.Error())
		case errors.Is(err, ErrBlocked):
			response.Error(c, http.StatusForbidden, "BLOCKED", err.Error())
		default:
			response.Error(c, http.StatusInternalServerError, "MESSAGE_FAILED", err.Error())
		}
		return
	}

	// 12. Отправляем через WebSocket (если Hub подключен)
	// TODO: h.hub.BroadcastToConversation(...)

	response.Success(c, http.StatusCreated, gin.H{
		"message": msg,
	})
}
