package notification

import (
	"errors"
	"net/http"
	"strconv"

	"photostudio/internal/pkg/response"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(protected *gin.RouterGroup) {
	g := protected.Group("/notifications")
	{
		g.GET("", h.GetNotifications)
		g.PATCH("/:id/read", h.MarkAsRead)
		g.PATCH("/read-all", h.MarkAllAsRead)
	}
}

// GetNotifications получает список уведомлений текущего пользователя.
// @Summary		Получить уведомления
// @Description	Возвращает список последних уведомлений пользователя и количество непрочитанных. Поддерживает пагинацию через параметр limit.
// @Tags		Уведомления
// @Security	BearerAuth
// @Param		limit	query	int	false	"Максимальное количество уведомлений (по умолчанию 20, макс 100)"
// @Success		200	{object}	gin.H{notifications=[]interface{},unread_count=int} "Список уведомлений и количество непрочитанных"
// @Failure		401	{object}	gin.H "Ошибка аутентификации: требуется токен"
// @Failure		500	{object}	gin.H "Ошибка сервера при получении уведомлений"
// @Router		/notifications [GET]
func (h *Handler) GetNotifications(c *gin.Context) {
	userID := c.GetInt64("user_id")
	if userID == 0 {
		response.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "User not authenticated")
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

	list, unread, err := h.service.GetUserNotifications(c.Request.Context(), userID, limit)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "FETCH_FAILED", "Failed to get notifications")
		return
	}

	response.Success(c, http.StatusOK, gin.H{
		"notifications": list,
		"unread_count":  unread,
	})
}

// MarkAsRead отмечает уведомление как прочитанное.
// @Summary		Отметить уведомление как прочитанное
// @Description	Отмечает конкретное уведомление как прочитанное. После этого оно больше не будет учитываться в счётчике непрочитанных.
// @Tags		Уведомления
// @Security	BearerAuth
// @Param		id	path	int	true	"ID уведомления"
// @Success		200	{object}	gin.H{status=string} "Уведомление отмечено как прочитанное"
// @Failure		400	{object}	gin.H "Ошибка: неверный ID уведомления"
// @Failure		401	{object}	gin.H "Ошибка аутентификации: требуется токен"
// @Failure		404	{object}	gin.H "Ошибка: уведомление не найдено"
// @Failure		500	{object}	gin.H "Ошибка сервера при обновлении статуса"
// @Router		/notifications/:id/read [PATCH]
func (h *Handler) MarkAsRead(c *gin.Context) {
	userID := c.GetInt64("user_id")
	if userID == 0 {
		response.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "User not authenticated")
		return
	}

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.Error(c, http.StatusBadRequest, "INVALID_ID", "Invalid notification ID")
		return
	}

	if err := h.service.MarkAsRead(c.Request.Context(), id, userID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, http.StatusNotFound, "NOT_FOUND", "Notification not found")
			return
		}
		response.Error(c, http.StatusInternalServerError, "UPDATE_FAILED", "Failed to mark as read")
		return
	}

	response.Success(c, http.StatusOK, gin.H{"status": "read"})
}

// MarkAllAsRead отмечает все уведомления пользователя как прочитанные.
// @Summary		Отметить все уведомления как прочитанные
// @Description	Отмечает все непрочитанные уведомления пользователя как прочитанные одним запросом.
// @Tags		Уведомления
// @Security	BearerAuth
// @Success		200	{object}	gin.H{status=string} "Все уведомления отмечены как прочитанные"
// @Failure		401	{object}	gin.H "Ошибка аутентификации: требуется токен"
// @Failure		500	{object}	gin.H "Ошибка сервера при обновлении статуса"
// @Router		/notifications/read-all [PATCH]
func (h *Handler) MarkAllAsRead(c *gin.Context) {
	userID := c.GetInt64("user_id")
	if userID == 0 {
		response.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "User not authenticated")
		return
	}

	if err := h.service.MarkAllAsRead(c.Request.Context(), userID); err != nil {
		response.Error(c, http.StatusInternalServerError, "UPDATE_FAILED", "Failed to mark as read")
		return
	}

	response.Success(c, http.StatusOK, gin.H{"status": "all_read"})
}
