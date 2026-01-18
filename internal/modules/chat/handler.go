package chat

import (
	"net/http"
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
