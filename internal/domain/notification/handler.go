package notification

import (
	"errors"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"net/http"
	"photostudio/internal/pkg/response"
	"strconv"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// GetNotifications получает список уведомлений текущего пользователя.
// @Summary		Получить уведомления
// @Description	Возвращает список последних уведомлений пользователя и количество непрочитанных. Поддерживает пагинацию через параметры limit и offset.
// @Tags		Уведомления
// @Security	BearerAuth
// @Param		limit	query	int	false	"Максимальное количество уведомлений (по умолчанию 20, макс 100)"
// @Param		offset	query	int	false	"Смещение для пагинации (по умолчанию 0)"
// @Success		200	{object}		NotificationListResponse "Список уведомлений и количество непрочитанных"
// @Failure		401	{object}		map[string]interface{} "Ошибка аутентификации: требуется токен"
// @Failure		500	{object}		map[string]interface{} "Ошибка сервера при получении уведомлений"
// @Router		/notifications [GET]
func (h *Handler) GetNotifications(c *gin.Context) {
	userID := c.GetInt64("user_id")
	if userID == 0 {
		response.CustomError(c, http.StatusUnauthorized, "UNAUTHORIZED", "User not authenticated")
		return
	}

	limit := 20
	if s := c.Query("limit"); s != "" {
		if v, err := strconv.Atoi(s); err == nil && v > 0 {
			limit = v
			if limit > 100 {
				limit = 100
			}
		}
	}

	offset := 0
	if s := c.Query("offset"); s != "" {
		if v, err := strconv.Atoi(s); err == nil && v >= 0 {
			offset = v
		}
	}

	notifications, unread, total, err := h.service.List(c.Request.Context(), userID, limit, offset)
	if err != nil {
		response.CustomError(c, http.StatusInternalServerError, "FETCH_FAILED", "Failed to get notifications")
		return
	}

	// Convert to response DTOs
	items := make([]*NotificationResponse, len(notifications))
	for i, n := range notifications {
		items[i] = NotificationResponseFromEntity(n)
	}

	response.Success(c, http.StatusOK, NotificationListResponse{
		Notifications: items,
		UnreadCount:   unread,
		Total:         total,
	})
}

// GetUnreadCount получает количество непрочитанных уведомлений.
// @Summary		Получить количество непрочитанных
// @Description	Возвращает количество непрочитанных уведомлений для текущего пользователя.
// @Tags		Уведомления
// @Security	BearerAuth
// @Success		200	{object}		UnreadCountResponse "Количество непрочитанных уведомлений"
// @Failure		401	{object}		map[string]interface{} "Ошибка аутентификации: требуется токен"
// @Failure		500	{object}		map[string]interface{} "Ошибка сервера"
// @Router		/notifications/unread-count [GET]
func (h *Handler) GetUnreadCount(c *gin.Context) {
	userID := c.GetInt64("user_id")
	if userID == 0 {
		response.CustomError(c, http.StatusUnauthorized, "UNAUTHORIZED", "User not authenticated")
		return
	}

	unread, err := h.service.GetUnreadCount(c.Request.Context(), userID)
	if err != nil {
		response.CustomError(c, http.StatusInternalServerError, "FETCH_FAILED", "Failed to get unread count")
		return
	}

	response.Success(c, http.StatusOK, UnreadCountResponse{UnreadCount: unread})
}

// MarkAsRead отмечает уведомление как прочитанное.
// @Summary		Отметить уведомление как прочитанное
// @Description	Отмечает конкретное уведомление как прочитанное. После этого оно больше не будет учитываться в счётчике непрочитанных.
// @Tags		Уведомления
// @Security	BearerAuth
// @Param		id	path	int	true	"ID уведомления"
// @Success		200	{object}		map[string]interface{} "Уведомление отмечено как прочитанное"
// @Failure		400	{object}		map[string]interface{} "Ошибка: неверный ID уведомления"
// @Failure		401	{object}		map[string]interface{} "Ошибка аутентификации: требуется токен"
// @Failure		404	{object}		map[string]interface{} "Ошибка: уведомление не найдено"
// @Failure		500	{object}		map[string]interface{} "Ошибка сервера при обновлении статуса"
// @Router		/notifications/{id}/read [PATCH]
func (h *Handler) MarkAsRead(c *gin.Context) {
	userID := c.GetInt64("user_id")
	if userID == 0 {
		response.CustomError(c, http.StatusUnauthorized, "UNAUTHORIZED", "User not authenticated")
		return
	}

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.CustomError(c, http.StatusBadRequest, "INVALID_ID", "Invalid notification ID")
		return
	}

	if err := h.service.MarkAsRead(c.Request.Context(), id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.CustomError(c, http.StatusNotFound, "NOT_FOUND", "Notification not found")
			return
		}
		response.CustomError(c, http.StatusInternalServerError, "UPDATE_FAILED", "Failed to mark as read")
		return
	}

	response.Success(c, http.StatusOK, gin.H{"status": "read"})
}

// MarkAllAsRead отмечает все уведомления пользователя как прочитанные.
// @Summary		Отметить все уведомления как прочитанные
// @Description	Отмечает все непрочитанные уведомления пользователя как прочитанные одним запросом.
// @Tags		Уведомления
// @Security	BearerAuth
// @Success		200	{object}		map[string]interface{} "Все уведомления отмечены как прочитанные"
// @Failure		401	{object}		map[string]interface{} "Ошибка аутентификации: требуется токен"
// @Failure		500	{object}		map[string]interface{} "Ошибка сервера при обновлении статуса"
// @Router		/notifications/read-all [PATCH]
func (h *Handler) MarkAllAsRead(c *gin.Context) {
	userID := c.GetInt64("user_id")
	if userID == 0 {
		response.CustomError(c, http.StatusUnauthorized, "UNAUTHORIZED", "User not authenticated")
		return
	}

	if err := h.service.MarkAllAsRead(c.Request.Context(), userID); err != nil {
		response.CustomError(c, http.StatusInternalServerError, "UPDATE_FAILED", "Failed to mark as read")
		return
	}

	response.Success(c, http.StatusOK, gin.H{"status": "all_read"})
}

// DeleteNotification удаляет конкретное уведомление.
// @Summary		Удалить уведомление
// @Description	Удаляет конкретное уведомление пользователя.
// @Tags		Уведомления
// @Security	BearerAuth
// @Param		id	path	int	true	"ID уведомления"
// @Success		200	{object}		map[string]interface{} "Уведомление удалено"
// @Failure		400	{object}		map[string]interface{} "Ошибка: неверный ID"
// @Failure		401	{object}		map[string]interface{} "Ошибка аутентификации"
// @Failure		500	{object}		map[string]interface{} "Ошибка сервера"
// @Router		/notifications/{id} [DELETE]
func (h *Handler) DeleteNotification(c *gin.Context) {
	userID := c.GetInt64("user_id")
	if userID == 0 {
		response.CustomError(c, http.StatusUnauthorized, "UNAUTHORIZED", "User not authenticated")
		return
	}

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.CustomError(c, http.StatusBadRequest, "INVALID_ID", "Invalid notification ID")
		return
	}

	if err := h.service.Delete(c.Request.Context(), id); err != nil {
		response.CustomError(c, http.StatusInternalServerError, "DELETE_FAILED", "Failed to delete notification")
		return
	}

	response.Success(c, http.StatusOK, gin.H{"status": "deleted"})

}
