package notification

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/websocket"
	"gorm.io/gorm"

	"photostudio/internal/domain/chat"
	"photostudio/internal/pkg/chicontext"
	"photostudio/internal/pkg/response"
)

type Handler struct {
	service *Service
	hub     *chat.Hub
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// SetRealtimeHub sets chat hub for notifications websocket channel.
func (h *Handler) SetRealtimeHub(hub *chat.Hub) {
	h.hub = hub
}

var notificationsUpgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

// WebSocket WS соединение для уведомлений.
//
//	@Summary		Сессия WS
//	@Description	Устанавливает WebSocket соединение для получения уведомлений в реальном времени.
//	@Tags			Notification
//	@Security		BearerAuth
//	@Success		101	"Switching Protocols"
//	@Failure		400	"Ошибка запроса"
//	@Failure		401	"Не авторизован"
//	@Failure		503	"Сервис недоступен"
//	@Router			/notifications/ws [get]
func (h *Handler) WebSocket(w http.ResponseWriter, r *http.Request) {
	userID := chicontext.UserIDFromCtx(r.Context())
	if userID == 0 {
		response.CustomError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "User not authenticated")
		return
	}
	if h.hub == nil {
		response.CustomError(w, r, http.StatusServiceUnavailable, "INTERNAL_ERROR", "WebSocket hub unavailable")
		return
	}

	conn, err := notificationsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		response.CustomError(w, r, http.StatusBadRequest, "INVALID_REQUEST", "Failed to upgrade websocket")
		return
	}

	h.hub.ServeWS(conn, userID, nil)
}

// GetNotifications список уведомлений.
//
//	@Summary		Список уведомлений
//	@Description	Возвращает страницу уведомлений пользователя.
//	@Tags			Notification
//	@Produce		json
//	@Security		BearerAuth
//	@Param			limit		query		int														false	"Кол-во на страницу"
//	@Param			offset		query		int														false	"Отступ"
//	@Param			only_unread	query		bool													false	"Только непрочитанные"
//	@Success		200			{object}	response.SuccessResponse{data=NotificationListResponse}	"Уведомления"
//	@Failure		400			{object}	response.ErrorResponse									"Неверные параметры"
//	@Failure		401			{object}	response.ErrorResponse									"Не авторизован"
//	@Failure		500			{object}	response.ErrorResponse									"Ошибка сервера"
//	@Router			/notifications [get]
func (h *Handler) GetNotifications(w http.ResponseWriter, r *http.Request) {
	userID := chicontext.UserIDFromCtx(r.Context())
	if userID == 0 {
		response.CustomError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "User not authenticated")
		return
	}

	limit := 20
	q := r.URL.Query()
	if s := q.Get("limit"); s != "" {
		if v, err := strconv.Atoi(s); err == nil && v > 0 {
			limit = v
			if limit > 100 {
				limit = 100
			}
		}
	}

	offset := 0
	if s := q.Get("offset"); s != "" {
		if v, err := strconv.Atoi(s); err == nil && v >= 0 {
			offset = v
		}
	}

	onlyUnread := false
	if s := q.Get("only_unread"); s != "" {
		v, err := strconv.ParseBool(s)
		if err != nil {
			response.CustomError(w, r, http.StatusBadRequest, "INVALID_REQUEST", "invalid only_unread value")
			return
		}
		onlyUnread = v
	}

	notifications, unread, total, err := h.service.List(r.Context(), userID, limit, offset, onlyUnread)
	if err != nil {
		response.CustomError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get notifications")
		return
	}

	items := make([]*NotificationResponse, len(notifications))
	for i, n := range notifications {
		items[i] = NotificationResponseFromEntity(n)
	}

	response.Success(w, http.StatusOK, NotificationListResponse{
		Notifications: items,
		UnreadCount:   unread,
		Total:         total,
	})
}

// GetUnreadCount возвращает количество непрочитанных уведомлений.
//
//	@Summary		Счетчик непрочитанных
//	@Description	Возвращает количество непрочитанных уведомлений пользователя.
//	@Tags			Notification
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	response.SuccessResponse{data=UnreadCountResponse}	"Счетчик"
//	@Failure		401	{object}	response.ErrorResponse								"Не авторизован"
//	@Failure		500	{object}	response.ErrorResponse								"Ошибка сервера"
//	@Router			/notifications/unread [get]
func (h *Handler) GetUnreadCount(w http.ResponseWriter, r *http.Request) {
	userID := chicontext.UserIDFromCtx(r.Context())
	if userID == 0 {
		response.CustomError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "User not authenticated")
		return
	}

	unread, err := h.service.GetUnreadCount(r.Context(), userID)
	if err != nil {
		response.CustomError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get unread count")
		return
	}

	response.Success(w, http.StatusOK, UnreadCountResponse{UnreadCount: unread})
}

// MarkAsRead помечает уведомление прочитанным.
//
//	@Summary		Пометить как прочитанное
//	@Description	Помечает конкретное уведомление как прочитанное.
//	@Tags			Notification
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		int							true	"ID уведомления"
//	@Success		200	{object}	response.SuccessResponse	"Готово"
//	@Failure		400	{object}	response.ErrorResponse		"Неверный ID"
//	@Failure		401	{object}	response.ErrorResponse		"Не авторизован"
//	@Failure		404	{object}	response.ErrorResponse		"Уведомление не найдено"
//	@Failure		500	{object}	response.ErrorResponse		"Ошибка сервера"
//	@Router			/notifications/{id}/read [post]
func (h *Handler) MarkAsRead(w http.ResponseWriter, r *http.Request) {
	userID := chicontext.UserIDFromCtx(r.Context())
	if userID == 0 {
		response.CustomError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "User not authenticated")
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		response.CustomError(w, r, http.StatusBadRequest, "INVALID_ID", "Invalid notification ID")
		return
	}

	if err := h.service.MarkAsRead(r.Context(), id, userID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.CustomError(w, r, http.StatusNotFound, "NOT_FOUND", "Notification not found")
			return
		}
		response.CustomError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to mark as read")
		return
	}

	response.Success(w, http.StatusOK, response.H{"status": "read"})
}

// MarkAllAsRead помечает все уведомления прочитанными.
//
//	@Summary		Прочитать всё
//	@Description	Помечает все уведомления пользователя как прочитанные.
//	@Tags			Notification
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	response.SuccessResponse	"Готово"
//	@Failure		401	{object}	response.ErrorResponse		"Не авторизован"
//	@Failure		500	{object}	response.ErrorResponse		"Ошибка сервера"
//	@Router			/notifications/read/all [post]
func (h *Handler) MarkAllAsRead(w http.ResponseWriter, r *http.Request) {
	userID := chicontext.UserIDFromCtx(r.Context())
	if userID == 0 {
		response.CustomError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "User not authenticated")
		return
	}

	if err := h.service.MarkAllAsRead(r.Context(), userID); err != nil {
		response.CustomError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to mark as read")
		return
	}

	response.Success(w, http.StatusOK, response.H{"status": "all_read"})
}

// DeleteNotification удаляет уведомление.
//
//	@Summary		Удалить уведомление
//	@Description	Удаляет конкретное уведомление.
//	@Tags			Notification
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		int							true	"ID уведомления"
//	@Success		200	{object}	response.SuccessResponse	"Удалено"
//	@Failure		400	{object}	response.ErrorResponse		"Неверный ID"
//	@Failure		401	{object}	response.ErrorResponse		"Не авторизован"
//	@Failure		404	{object}	response.ErrorResponse		"Уведомление не найдено"
//	@Failure		500	{object}	response.ErrorResponse		"Ошибка сервера"
//	@Router			/notifications/{id} [delete]
func (h *Handler) DeleteNotification(w http.ResponseWriter, r *http.Request) {
	userID := chicontext.UserIDFromCtx(r.Context())
	if userID == 0 {
		response.CustomError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "User not authenticated")
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		response.CustomError(w, r, http.StatusBadRequest, "INVALID_ID", "Invalid notification ID")
		return
	}

	if err := h.service.Delete(r.Context(), id, userID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.CustomError(w, r, http.StatusNotFound, "NOT_FOUND", "Notification not found")
			return
		}
		response.CustomError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to delete notification")
		return
	}

	response.Success(w, http.StatusOK, response.H{"status": "deleted"})
}
