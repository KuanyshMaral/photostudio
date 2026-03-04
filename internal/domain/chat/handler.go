package chat

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/websocket"

	"photostudio/internal/pkg/chicontext"
	"photostudio/internal/pkg/response"
)

// Handler handles HTTP requests for the chat domain
type Handler struct {
	service *Service
	hub     *Hub
}

func NewHandler(service *Service, hub *Hub) *Handler {
	return &Handler{service: service, hub: hub}
}

// swaggerRoomData is a wrapper strictly for generating Swagger documentation.
type swaggerRoomData struct {
	ID          string `json:"id"`
	Type        string `json:"type"`
	Name        string `json:"name"`
	CreatorID   *int64 `json:"creator_id"`
	CreatedAt   string `json:"created_at"`
	UnreadCount int    `json:"unread_count,omitempty"`
	MemberCount int    `json:"member_count,omitempty"`
}

// swaggerRoomResponse is a wrapper strictly for generating Swagger documentation.
type swaggerRoomResponse struct {
	Success bool            `json:"success"`
	Data    swaggerRoomData `json:"data"`
}

// swaggerListRoomsResponse is a wrapper strictly for generating Swagger documentation.
type swaggerListRoomsResponse struct {
	Success bool              `json:"success"`
	Data    []swaggerRoomData `json:"data"`
}

// swaggerMessageData is a wrapper strictly for generating Swagger documentation.
type swaggerMessageData struct {
	ID             string `json:"id"`
	RoomID         string `json:"room_id"`
	SenderID       int64  `json:"sender_id"`
	Content        string `json:"content"`
	IsRead         bool   `json:"is_read"`
	CreatedAt      string `json:"created_at"`
	UploadID       string `json:"upload_id,omitempty"`
	AttachmentURL  string `json:"attachment_url,omitempty"`
	AttachmentName string `json:"attachment_name,omitempty"`
	AttachmentMime string `json:"attachment_mime,omitempty"`
}

// swaggerMessageResponse is a wrapper strictly for generating Swagger documentation.
type swaggerMessageResponse struct {
	Success bool               `json:"success"`
	Data    swaggerMessageData `json:"data"`
}

// swaggerListMessagesResponse is a wrapper strictly for generating Swagger documentation.
type swaggerListMessagesResponse struct {
	Success bool                 `json:"success"`
	Data    []swaggerMessageData `json:"data"`
}

// swaggerUnreadCountResponse is a wrapper strictly for generating Swagger documentation.
type swaggerUnreadCountResponse struct {
	Success bool `json:"success"`
	Data    struct {
		UnreadCount int `json:"unread_count"`
	} `json:"data"`
}

// swaggerMemberData is a wrapper strictly for generating Swagger documentation.
type swaggerMemberData struct {
	UserID   int64  `json:"user_id"`
	Role     string `json:"role"`
	JoinedAt string `json:"joined_at"`
}

// swaggerListMembersResponse is a wrapper strictly for generating Swagger documentation.
type swaggerListMembersResponse struct {
	Success bool                `json:"success"`
	Data    []swaggerMemberData `json:"data"`
}

// ---- Room endpoints ----

// CreateDirectRoom создание комнаты 1-на-1 (direct).
//
//	@Summary		Создать чат (direct)
//	@Description	Создает или возвращает существующий личный чат с пользователем.
//	@Tags			Chat
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			request	body		createDirectRequest		true	"ID получателя"
//	@Success		201		{object}	swaggerRoomResponse		"Чат"
//	@Failure		400		{object}	response.ErrorResponse	"Ошибка входных данных"
//	@Failure		401		{object}	response.ErrorResponse	"Не авторизован"
//	@Failure		403		{object}	response.ErrorResponse	"Пользователь заблокирован"
//	@Failure		500		{object}	response.ErrorResponse	"Ошибка сервера"
//	@Router			/chat/rooms/direct [post]
func (h *Handler) CreateDirectRoom(w http.ResponseWriter, r *http.Request) {
	userID := chicontext.UserIDFromCtx(r.Context())
	if userID == 0 {
		response.CustomError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "unauthorized")
		return
	}
	var req createDirectRequest
	if err := response.BindJSON(r, &req); err != nil {
		response.JSON(w, http.StatusBadRequest, response.H{"success": false, "error": err.Error()})
		return
	}
	room, err := h.service.GetOrCreateDirectRoom(r.Context(), userID, req.RecipientID)
	if err != nil {
		handleRoomError(w, err)
		return
	}
	response.JSON(w, http.StatusCreated, response.H{"success": true, "data": roomResponse(room)})
}

// CreateGroupRoom создание групповой комнаты.
//
//	@Summary		Создать чат (group)
//	@Description	Создает групповой чат с названием и списком участников.
//	@Tags			Chat
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			request	body		createGroupRequest		true	"Данные группы"
//	@Success		201		{object}	swaggerRoomResponse		"Групповой чат"
//	@Failure		400		{object}	response.ErrorResponse	"Ошибка входных данных"
//	@Failure		401		{object}	response.ErrorResponse	"Не авторизован"
//	@Failure		500		{object}	response.ErrorResponse	"Ошибка сервера"
//	@Router			/chat/rooms/group [post]
func (h *Handler) CreateGroupRoom(w http.ResponseWriter, r *http.Request) {
	userID := chicontext.UserIDFromCtx(r.Context())
	if userID == 0 {
		response.CustomError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "unauthorized")
		return
	}
	var req createGroupRequest
	if err := response.BindJSON(r, &req); err != nil {
		response.JSON(w, http.StatusBadRequest, response.H{"success": false, "error": err.Error()})
		return
	}
	room, err := h.service.CreateGroupRoom(r.Context(), userID, req.Name, req.MemberIDs)
	if err != nil {
		handleRoomError(w, err)
		return
	}
	response.JSON(w, http.StatusCreated, response.H{"success": true, "data": roomResponse(room)})
}

// ListRooms получение списка комнат пользователя.
//
//	@Summary		Список чатов
//	@Description	Возвращает список всех чатов, в которых состоит пользователь.
//	@Tags			Chat
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	swaggerListRoomsResponse	"Список чатов"
//	@Failure		401	{object}	response.ErrorResponse		"Не авторизован"
//	@Failure		500	{object}	response.ErrorResponse		"Ошибка сервера"
//	@Router			/chat/rooms [get]
func (h *Handler) ListRooms(w http.ResponseWriter, r *http.Request) {
	userID := chicontext.UserIDFromCtx(r.Context())
	if userID == 0 {
		response.CustomError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "unauthorized")
		return
	}
	rooms, err := h.service.ListRooms(r.Context(), userID)
	if err != nil {
		response.JSON(w, http.StatusInternalServerError, response.H{"success": false, "error": "failed to list rooms"})
		return
	}
	items := make([]response.H, 0, len(rooms))
	for _, rm := range rooms {
		item := roomResponse(rm.Room)
		item["unread_count"] = rm.UnreadCount
		item["member_count"] = len(rm.Members)
		if rm.OtherUser != nil {
			item["other_user"] = response.H{
				"id":         rm.OtherUser.UserID,
				"name":       rm.OtherUser.Name,
				"avatar_url": rm.OtherUser.AvatarURL,
			}
		}
		items = append(items, item)
	}
	response.JSON(w, http.StatusOK, response.H{"success": true, "data": items})
}

// ---- Message endpoints ----

// GetMessages получение сообщений в комнате.
//
//	@Summary		Сообщения чата
//	@Description	Возвращает список сообщений в чате (с пагинацией).
//	@Tags			Chat
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		string						true	"ID чата"
//	@Param			limit	query		int							false	"Кол-во сообщений"
//	@Param			offset	query		int							false	"Отступ"
//	@Success		200		{object}	swaggerListMessagesResponse	"Список сообщений"
//	@Failure		401		{object}	response.ErrorResponse		"Не авторизован"
//	@Failure		403		{object}	response.ErrorResponse		"Вы не являетесь участником чата"
//	@Failure		404		{object}	response.ErrorResponse		"Чат не найден"
//	@Failure		500		{object}	response.ErrorResponse		"Ошибка сервера"
//	@Router			/chat/rooms/{id}/messages [get]
func (h *Handler) GetMessages(w http.ResponseWriter, r *http.Request) {
	userID := chicontext.UserIDFromCtx(r.Context())
	if userID == 0 {
		response.CustomError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "unauthorized")
		return
	}
	roomID := chi.URLParam(r, "id")
	limit := 50
	offset := 0
	q := r.URL.Query()
	if l, err := strconv.Atoi(q.Get("limit")); err == nil && l > 0 && l <= 100 {
		limit = l
	}
	if o, err := strconv.Atoi(q.Get("offset")); err == nil && o >= 0 {
		offset = o
	}
	msgs, err := h.service.GetMessages(r.Context(), userID, roomID, limit, offset)
	if err != nil {
		handleRoomError(w, err)
		return
	}
	items := make([]response.H, 0, len(msgs))
	for _, m := range msgs {
		items = append(items, messageResponse(m))
	}
	response.JSON(w, http.StatusOK, response.H{"success": true, "data": items})
}

// SendMessage отправка сообщения в комнату.
//
//	@Summary		Отправить сообщение
//	@Description	Отправляет текстовое сообщение (с возможным вложением) в чат.
//	@Tags			Chat
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		string					true	"ID чата"
//	@Param			request	body		sendMessageRequest		true	"Сообщение"
//	@Success		201		{object}	swaggerMessageResponse	"Сообщение отправлено"
//	@Failure		400		{object}	response.ErrorResponse	"Ошибка запроса"
//	@Failure		401		{object}	response.ErrorResponse	"Не авторизован"
//	@Failure		403		{object}	response.ErrorResponse	"Нет прав отправлять в этот чат (пользователь заблокирован или вы не участник)"
//	@Failure		404		{object}	response.ErrorResponse	"Чат не найден"
//	@Failure		500		{object}	response.ErrorResponse	"Ошибка сервера"
//	@Router			/chat/rooms/{id}/messages [post]
func (h *Handler) SendMessage(w http.ResponseWriter, r *http.Request) {
	userID := chicontext.UserIDFromCtx(r.Context())
	if userID == 0 {
		response.CustomError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "unauthorized")
		return
	}
	roomID := chi.URLParam(r, "id")
	var req sendMessageRequest
	if err := response.BindJSON(r, &req); err != nil {
		response.JSON(w, http.StatusBadRequest, response.H{"success": false, "error": err.Error()})
		return
	}
	msg, err := h.service.SendMessage(r.Context(), userID, roomID, req.Content)
	if err != nil {
		handleRoomError(w, err)
		return
	}

	h.hub.BroadcastToRoom(roomID, &WSEvent{
		Type:    EventNewMessage,
		RoomID:  roomID,
		Payload: messageResponse(msg),
	})

	response.JSON(w, http.StatusCreated, response.H{"success": true, "data": messageResponse(msg)})
}

// MarkAsRead отметить сообщения как прочитанные.
//
//	@Summary		Отметить прочитанными
//	@Description	Помечает все сообщения в указанном чате как прочитанные для текущего юзера.
//	@Tags			Chat
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		string						true	"ID чата"
//	@Success		200	{object}	response.SuccessResponse	"Отмечено"
//	@Failure		401	{object}	response.ErrorResponse		"Не авторизован"
//	@Failure		403	{object}	response.ErrorResponse		"Не участник"
//	@Failure		404	{object}	response.ErrorResponse		"Чат не найден"
//	@Failure		500	{object}	response.ErrorResponse		"Ошибка сервера"
//	@Router			/chat/rooms/{id}/read [post]
func (h *Handler) MarkAsRead(w http.ResponseWriter, r *http.Request) {
	userID := chicontext.UserIDFromCtx(r.Context())
	if userID == 0 {
		response.CustomError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "unauthorized")
		return
	}
	roomID := chi.URLParam(r, "id")
	if err := h.service.MarkAsRead(r.Context(), userID, roomID); err != nil {
		handleRoomError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, response.H{"success": true})
}

// GetUnreadCount получение количества непрочитанных сообщений во всех чатах.
//
//	@Summary		Счетчик непрочитанных
//	@Description	Возвращает общее количество непрочитанных сообщений текущего пользователя во всех чатах.
//	@Tags			Chat
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	swaggerUnreadCountResponse	"Кол-во"
//	@Failure		401	{object}	response.ErrorResponse		"Не авторизован"
//	@Failure		500	{object}	response.ErrorResponse		"Ошибка сервера"
//	@Router			/chat/unread [get]
func (h *Handler) GetUnreadCount(w http.ResponseWriter, r *http.Request) {
	userID := chicontext.UserIDFromCtx(r.Context())
	if userID == 0 {
		response.CustomError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "unauthorized")
		return
	}
	count, _ := h.service.GetUnreadCount(r.Context(), userID)
	response.JSON(w, http.StatusOK, response.H{"success": true, "data": response.H{"unread_count": count}})
}

// ---- Member management ----

// GetMembers получить участников чата.
//
//	@Summary		Участники чата
//	@Description	Возвращает список пользователей, состоящих в этом чате.
//	@Tags			Chat
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		string						true	"ID чата"
//	@Success		200	{object}	swaggerListMembersResponse	"Участники"
//	@Failure		401	{object}	response.ErrorResponse		"Не авторизован"
//	@Failure		403	{object}	response.ErrorResponse		"Не участник"
//	@Failure		404	{object}	response.ErrorResponse		"Чат не найден"
//	@Failure		500	{object}	response.ErrorResponse		"Ошибка сервера"
//	@Router			/chat/rooms/{id}/members [get]
func (h *Handler) GetMembers(w http.ResponseWriter, r *http.Request) {
	userID := chicontext.UserIDFromCtx(r.Context())
	if userID == 0 {
		response.CustomError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "unauthorized")
		return
	}
	roomID := chi.URLParam(r, "id")
	members, err := h.service.GetMembers(r.Context(), userID, roomID)
	if err != nil {
		handleRoomError(w, err)
		return
	}
	items := make([]response.H, 0, len(members))
	for _, m := range members {
		items = append(items, response.H{
			"user_id":   m.UserID,
			"role":      m.Role,
			"joined_at": m.JoinedAt,
		})
	}
	response.JSON(w, http.StatusOK, response.H{"success": true, "data": items})
}

// AddMember добавить участника в групповой чат.
//
//	@Summary		Добавить участника
//	@Description	Админ чата или глобальный админ добавляет пользователя.
//	@Tags			Chat
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		string						true	"ID чата"
//	@Param			request	body		addMemberRequest			true	"Пользователь"
//	@Success		200		{object}	response.SuccessResponse	"Добавлен"
//	@Failure		400		{object}	response.ErrorResponse		"Ошибка: не группа и тд"
//	@Failure		401		{object}	response.ErrorResponse		"Не авторизован"
//	@Failure		403		{object}	response.ErrorResponse		"Нет прав (не админ)"
//	@Failure		404		{object}	response.ErrorResponse		"Чат не найден"
//	@Failure		409		{object}	response.ErrorResponse		"Уже участник"
//	@Failure		500		{object}	response.ErrorResponse		"Ошибка сервера"
//	@Router			/chat/rooms/{id}/members [post]
func (h *Handler) AddMember(w http.ResponseWriter, r *http.Request) {
	roomID := chi.URLParam(r, "id")
	var req addMemberRequest
	if err := response.BindJSON(r, &req); err != nil {
		response.JSON(w, http.StatusBadRequest, response.H{"success": false, "error": err.Error()})
		return
	}

	if chicontext.IsAdminTokenFromCtx(r.Context()) {
		if err := h.service.AddMemberByPlatformAdmin(r.Context(), roomID, req.UserID); err != nil {
			handleRoomError(w, err)
			return
		}
		response.JSON(w, http.StatusOK, response.H{"success": true, "message": "member added"})
		return
	}

	userID := chicontext.UserIDFromCtx(r.Context())
	if userID == 0 {
		response.CustomError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "unauthorized")
		return
	}
	if err := h.service.AddMember(r.Context(), userID, roomID, req.UserID); err != nil {
		handleRoomError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, response.H{"success": true, "message": "member added"})
}

// RemoveMember удалить участника из группового чата.
//
//	@Summary		Удалить участника
//	@Description	Админ чата или платформы удаляет пользователя.
//	@Tags			Chat
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		string						true	"ID чата"
//	@Param			user_id	path		int							true	"ID пользователя"
//	@Success		200		{object}	response.SuccessResponse	"Удален"
//	@Failure		400		{object}	response.ErrorResponse		"Ошибка запроса"
//	@Failure		401		{object}	response.ErrorResponse		"Не авторизован"
//	@Failure		403		{object}	response.ErrorResponse		"Нет прав"
//	@Failure		404		{object}	response.ErrorResponse		"Чат или участник не найден"
//	@Failure		500		{object}	response.ErrorResponse		"Ошибка сервера"
//	@Router			/chat/rooms/{id}/members/{user_id} [delete]
func (h *Handler) RemoveMember(w http.ResponseWriter, r *http.Request) {
	roomID := chi.URLParam(r, "id")
	targetID, err := strconv.ParseInt(chi.URLParam(r, "user_id"), 10, 64)
	if err != nil {
		response.JSON(w, http.StatusBadRequest, response.H{"success": false, "error": "invalid user_id"})
		return
	}

	if chicontext.IsAdminTokenFromCtx(r.Context()) {
		if err := h.service.RemoveMemberByPlatformAdmin(r.Context(), roomID, targetID); err != nil {
			handleRoomError(w, err)
			return
		}
		response.JSON(w, http.StatusOK, response.H{"success": true, "message": "member removed"})
		return
	}

	userID := chicontext.UserIDFromCtx(r.Context())
	if userID == 0 {
		response.CustomError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "unauthorized")
		return
	}
	if err := h.service.RemoveMember(r.Context(), userID, roomID, targetID); err != nil {
		handleRoomError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, response.H{"success": true, "message": "member removed"})
}

// LeaveRoom покинуть чат.
//
//	@Summary		Покинуть чат
//	@Description	Текущий пользователь добровольно выходит из группового чата.
//	@Tags			Chat
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		string						true	"ID чата"
//	@Success		200	{object}	response.SuccessResponse	"Вышел"
//	@Failure		401	{object}	response.ErrorResponse		"Не авторизован"
//	@Failure		403	{object}	response.ErrorResponse		"Не в чате"
//	@Failure		404	{object}	response.ErrorResponse		"Чат не найден"
//	@Failure		500	{object}	response.ErrorResponse		"Ошибка сервера"
//	@Router			/chat/rooms/{id}/leave [post]
func (h *Handler) LeaveRoom(w http.ResponseWriter, r *http.Request) {
	userID := chicontext.UserIDFromCtx(r.Context())
	if userID == 0 {
		response.CustomError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "unauthorized")
		return
	}
	roomID := chi.URLParam(r, "id")
	if err := h.service.RemoveMember(r.Context(), userID, roomID, userID); err != nil {
		handleRoomError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, response.H{"success": true, "message": "left room"})
}

// ---- WebSocket ----

func (h *Handler) WebSocket(w http.ResponseWriter, r *http.Request) {
	userID := chicontext.UserIDFromCtx(r.Context())
	if userID == 0 {
		response.CustomError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "unauthorized")
		return
	}

	if !websocket.IsWebSocketUpgrade(r) {
		response.JSON(w, http.StatusUpgradeRequired, response.H{
			"success": false,
			"error":   "websocket upgrade required",
		})
		return
	}

	if strings.TrimSpace(r.Header.Get("Sec-WebSocket-Key")) == "" {
		response.JSON(w, http.StatusBadRequest, response.H{
			"success": false,
			"error":   "invalid websocket handshake: missing Sec-WebSocket-Key",
		})
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		response.JSON(w, http.StatusBadRequest, response.H{"success": false, "error": "failed to upgrade connection"})
		return
	}

	rooms, err := h.service.ListRooms(r.Context(), userID)
	var roomIDs []string
	if err == nil {
		for _, rm := range rooms {
			roomIDs = append(roomIDs, rm.ID)
		}
	}

	h.hub.ServeWS(conn, userID, roomIDs)
}

// ---- Helpers ----

func roomResponse(r *Room) response.H {
	name := ""
	if r.Name.Valid {
		name = r.Name.String
	}
	var creatorID *int64
	if r.CreatorID.Valid {
		creatorID = &r.CreatorID.Int64
	}
	return response.H{
		"id":         r.ID,
		"type":       r.Type,
		"name":       name,
		"creator_id": creatorID,
		"created_at": r.CreatedAt,
	}
}

func messageResponse(m *Message) response.H {
	resp := response.H{
		"id":         m.ID,
		"room_id":    m.RoomID,
		"sender_id":  m.SenderID,
		"content":    m.Content,
		"is_read":    m.IsRead,
		"created_at": m.CreatedAt,
	}
	if len(m.Attachments) > 0 {
		resp["attachments"] = m.Attachments
	}
	return resp
}

func handleRoomError(w http.ResponseWriter, err error) {
	switch err {
	case ErrRoomNotFound:
		response.JSON(w, http.StatusNotFound, response.H{"success": false, "error": err.Error()})
	case ErrNotRoomMember:
		response.JSON(w, http.StatusForbidden, response.H{"success": false, "error": err.Error()})
	case ErrNotRoomAdmin:
		response.JSON(w, http.StatusForbidden, response.H{"success": false, "error": err.Error()})
	case ErrAlreadyMember:
		response.JSON(w, http.StatusConflict, response.H{"success": false, "error": err.Error()})
	case ErrCannotChatSelf:
		response.JSON(w, http.StatusBadRequest, response.H{"success": false, "error": err.Error()})
	case ErrUserBlocked:
		response.JSON(w, http.StatusForbidden, response.H{"success": false, "error": err.Error()})
	default:
		response.JSON(w, http.StatusInternalServerError, response.H{"success": false, "error": "internal error"})
	}
}

// ---- Request types ----

type createDirectRequest struct {
	RecipientID int64 `json:"recipient_id"`
}

type createGroupRequest struct {
	Name      string  `json:"name"`
	MemberIDs []int64 `json:"member_ids"`
}

type sendMessageRequest struct {
	Content string `json:"content"`
}

type addMemberRequest struct {
	UserID int64 `json:"user_id"`
}
