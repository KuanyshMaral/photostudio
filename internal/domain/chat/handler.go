package chat

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// Handler handles HTTP requests for the chat domain
type Handler struct {
	service *Service
	hub     *Hub
}

func NewHandler(service *Service, hub *Hub) *Handler {
	return &Handler{service: service, hub: hub}
}

// ---- Room endpoints ----

// CreateDirectRoom godoc
// @Summary Start or get a 1-on-1 room
// @Tags Chat
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param body body createDirectRequest true "Recipient"
// @Success 201 {object} map[string]interface{}
// @Router /rooms/direct [post]
func (h *Handler) CreateDirectRoom(c *gin.Context) {
	userID := mustUserID(c)
	if userID == 0 {
		return
	}
	var req createDirectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}
	room, err := h.service.GetOrCreateDirectRoom(c.Request.Context(), userID, req.RecipientID)
	if err != nil {
		handleRoomError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"success": true, "data": roomResponse(room)})
}

// CreateGroupRoom godoc
// @Summary Create a group room
// @Tags Chat
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param body body createGroupRequest true "Group details"
// @Success 201 {object} map[string]interface{}
// @Router /rooms/group [post]
func (h *Handler) CreateGroupRoom(c *gin.Context) {
	userID := mustUserID(c)
	if userID == 0 {
		return
	}
	var req createGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}
	room, err := h.service.CreateGroupRoom(c.Request.Context(), userID, req.Name, req.MemberIDs)
	if err != nil {
		handleRoomError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"success": true, "data": roomResponse(room)})
}

// ListRooms godoc
// @Summary List my rooms
// @Tags Chat
// @Security BearerAuth
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /rooms [get]
func (h *Handler) ListRooms(c *gin.Context) {
	userID := mustUserID(c)
	if userID == 0 {
		return
	}
	rooms, err := h.service.ListRooms(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "failed to list rooms"})
		return
	}
	items := make([]gin.H, 0, len(rooms))
	for _, r := range rooms {
		item := roomResponse(r.Room)
		item["unread_count"] = r.UnreadCount
		item["member_count"] = len(r.Members)
		items = append(items, item)
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": items})
}

// ---- Message endpoints ----

// GetMessages godoc
// @Summary Get messages in a room
// @Tags Chat
// @Security BearerAuth
// @Param id path string true "Room ID"
// @Param limit query int false "Limit (default 50)"
// @Param offset query int false "Offset"
// @Success 200 {object} map[string]interface{}
// @Router /rooms/{id}/messages [get]
func (h *Handler) GetMessages(c *gin.Context) {
	userID := mustUserID(c)
	if userID == 0 {
		return
	}
	roomID := c.Param("id")
	limit := 50
	offset := 0
	if l, err := strconv.Atoi(c.Query("limit")); err == nil && l > 0 && l <= 100 {
		limit = l
	}
	if o, err := strconv.Atoi(c.Query("offset")); err == nil && o >= 0 {
		offset = o
	}
	msgs, err := h.service.GetMessages(c.Request.Context(), userID, roomID, limit, offset)
	if err != nil {
		handleRoomError(c, err)
		return
	}
	items := make([]gin.H, 0, len(msgs))
	for _, m := range msgs {
		items = append(items, messageResponse(m))
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": items})
}

// SendMessage godoc
// @Summary Send a message
// @Tags Chat
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "Room ID"
// @Param body body sendMessageRequest true "Message"
// @Success 201 {object} map[string]interface{}
// @Router /rooms/{id}/messages [post]
func (h *Handler) SendMessage(c *gin.Context) {
	userID := mustUserID(c)
	if userID == 0 {
		return
	}
	roomID := c.Param("id")
	var req sendMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}
	msg, err := h.service.SendMessage(c.Request.Context(), userID, roomID, req.Content, req.UploadID)
	if err != nil {
		handleRoomError(c, err)
		return
	}

	// Broadcast via WebSocket
	h.hub.BroadcastToRoom(roomID, &WSEvent{
		Type:    EventNewMessage,
		RoomID:  roomID,
		Payload: messageResponse(msg),
	})

	c.JSON(http.StatusCreated, gin.H{"success": true, "data": messageResponse(msg)})
}

// MarkAsRead godoc
// @Summary Mark room as read
// @Tags Chat
// @Security BearerAuth
// @Param id path string true "Room ID"
// @Success 200 {object} map[string]interface{}
// @Router /rooms/{id}/read [post]
func (h *Handler) MarkAsRead(c *gin.Context) {
	userID := mustUserID(c)
	if userID == 0 {
		return
	}
	roomID := c.Param("id")
	if err := h.service.MarkAsRead(c.Request.Context(), userID, roomID); err != nil {
		handleRoomError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

// GetUnreadCount godoc
// @Summary Total unread messages count
// @Tags Chat
// @Security BearerAuth
// @Success 200 {object} map[string]interface{}
// @Router /rooms/unread [get]
func (h *Handler) GetUnreadCount(c *gin.Context) {
	userID := mustUserID(c)
	if userID == 0 {
		return
	}
	count, _ := h.service.GetUnreadCount(c.Request.Context(), userID)
	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"unread_count": count}})
}

// ---- Member management ----

// GetMembers godoc
// @Summary Get room members
// @Tags Chat
// @Security BearerAuth
// @Param id path string true "Room ID"
// @Success 200 {object} map[string]interface{}
// @Router /rooms/{id}/members [get]
func (h *Handler) GetMembers(c *gin.Context) {
	userID := mustUserID(c)
	if userID == 0 {
		return
	}
	roomID := c.Param("id")
	members, err := h.service.GetMembers(c.Request.Context(), userID, roomID)
	if err != nil {
		handleRoomError(c, err)
		return
	}
	items := make([]gin.H, 0, len(members))
	for _, m := range members {
		items = append(items, gin.H{
			"user_id":   m.UserID,
			"role":      m.Role,
			"joined_at": m.JoinedAt,
		})
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": items})
}

// AddMember godoc
// @Summary Add member to group room (admin only)
// @Tags Chat
// @Security BearerAuth
// @Param id path string true "Room ID"
// @Param body body addMemberRequest true "User to add"
// @Success 200 {object} map[string]interface{}
// @Router /rooms/{id}/members [post]
func (h *Handler) AddMember(c *gin.Context) {
	userID := mustUserID(c)
	if userID == 0 {
		return
	}
	roomID := c.Param("id")
	var req addMemberRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}
	if err := h.service.AddMember(c.Request.Context(), userID, roomID, req.UserID); err != nil {
		handleRoomError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "member added"})
}

// RemoveMember godoc
// @Summary Remove member from group room (admin only)
// @Tags Chat
// @Security BearerAuth
// @Param id path string true "Room ID"
// @Param user_id path int true "User ID to remove"
// @Success 200 {object} map[string]interface{}
// @Router /rooms/{id}/members/{user_id} [delete]
func (h *Handler) RemoveMember(c *gin.Context) {
	userID := mustUserID(c)
	if userID == 0 {
		return
	}
	roomID := c.Param("id")
	targetID, err := strconv.ParseInt(c.Param("user_id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid user_id"})
		return
	}
	if err := h.service.RemoveMember(c.Request.Context(), userID, roomID, targetID); err != nil {
		handleRoomError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "member removed"})
}

// LeaveRoom godoc
// @Summary Leave a room
// @Tags Chat
// @Security BearerAuth
// @Param id path string true "Room ID"
// @Success 200 {object} map[string]interface{}
// @Router /rooms/{id}/leave [post]
func (h *Handler) LeaveRoom(c *gin.Context) {
	userID := mustUserID(c)
	if userID == 0 {
		return
	}
	roomID := c.Param("id")
	if err := h.service.RemoveMember(c.Request.Context(), userID, roomID, userID); err != nil {
		handleRoomError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "left room"})
}

// ---- WebSocket ----

// WebSocket godoc
// @Summary Connect to WebSocket for real-time chat
// @Tags Chat
// @Security BearerAuth
// @Router /rooms/ws [get]
func (h *Handler) WebSocket(c *gin.Context) {
	userID := mustUserID(c)
	if userID == 0 {
		return
	}

	// Upgrade HTTP connection
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		// handle error
		return
	}

	// Auto-subscribe: Fetch user's current rooms
	rooms, err := h.service.ListRooms(c.Request.Context(), userID)
	var roomIDs []string
	if err == nil {
		for _, r := range rooms {
			roomIDs = append(roomIDs, r.ID)
		}
	}

	h.hub.ServeWS(conn, userID, roomIDs)
}

// ---- Helpers ----

func roomResponse(r *Room) gin.H {
	name := ""
	if r.Name.Valid {
		name = r.Name.String
	}
	var creatorID *int64
	if r.CreatorID.Valid {
		creatorID = &r.CreatorID.Int64
	}
	return gin.H{
		"id":         r.ID,
		"type":       r.Type,
		"name":       name,
		"creator_id": creatorID,
		"created_at": r.CreatedAt,
	}
}

func messageResponse(m *Message) gin.H {
	resp := gin.H{
		"id":         m.ID,
		"room_id":    m.RoomID,
		"sender_id":  m.SenderID,
		"content":    m.Content,
		"is_read":    m.IsRead,
		"created_at": m.CreatedAt,
	}
	if m.UploadID.Valid {
		resp["upload_id"] = m.UploadID.String
		resp["attachment_url"] = m.AttachmentURL
		resp["attachment_name"] = m.AttachmentName
		resp["attachment_mime"] = m.AttachmentMime
	}
	return resp
}

func handleRoomError(c *gin.Context, err error) {
	switch err {
	case ErrRoomNotFound:
		c.JSON(http.StatusNotFound, gin.H{"success": false, "error": err.Error()})
	case ErrNotRoomMember:
		c.JSON(http.StatusForbidden, gin.H{"success": false, "error": err.Error()})
	case ErrNotRoomAdmin:
		c.JSON(http.StatusForbidden, gin.H{"success": false, "error": err.Error()})
	case ErrAlreadyMember:
		c.JSON(http.StatusConflict, gin.H{"success": false, "error": err.Error()})
	case ErrCannotChatSelf:
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
	case ErrUserBlocked:
		c.JSON(http.StatusForbidden, gin.H{"success": false, "error": err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "internal error"})
	}
}

func mustUserID(c *gin.Context) int64 {
	id, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "unauthorized"})
		return 0
	}
	switch v := id.(type) {
	case int64:
		return v
	case float64:
		return int64(v)
	}
	c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "invalid user id"})
	return 0
}

// ---- Request types ----

type createDirectRequest struct {
	RecipientID int64 `json:"recipient_id" binding:"required"`
}

type createGroupRequest struct {
	Name      string  `json:"name" binding:"required"`
	MemberIDs []int64 `json:"member_ids"`
}

type sendMessageRequest struct {
	Content  string  `json:"content"`
	UploadID *string `json:"upload_id"` // optional
}

type addMemberRequest struct {
	UserID int64 `json:"user_id" binding:"required"`
}
