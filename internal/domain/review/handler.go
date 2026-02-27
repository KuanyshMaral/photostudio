package review

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// Create создаёт новый полиморфный отзыв.
// @Summary		Написать отзыв
// @Description	Пользователь может написать отзыв о сущности (студия, профиль). Для студий требуется завершенное бронирование.
// @Tags		Отзывы
// @Security	BearerAuth
// @Param		request	body	CreateReviewRequest	true	"Данные отзыва (target_type, target_id, rating, context_type, context_id)"
// @Success		201	{object}		map[string]interface{} "Отзыв успешно сохранён"
// @Failure		400	{object}		map[string]interface{} "Ошибка валидации данных"
// @Failure		401	{object}		map[string]interface{} "Ошибка аутентификации"
// @Failure		403	{object}		map[string]interface{} "Запрещено писать отзыв"
// @Failure		409	{object}		map[string]interface{} "Ошибка: отзыв уже существует"
// @Failure		500	{object}		map[string]interface{} "Ошибка сервера при сохранении отзыва"
// @Router		/reviews [POST]
func (h *Handler) Create(c *gin.Context) {
	var req CreateReviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"code": "INVALID_REQUEST", "message": "Invalid request body"}})
		return
	}

	userID := c.GetInt64("user_id")
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": gin.H{"code": "UNAUTHORIZED", "message": "Authentication required"}})
		return
	}

	rv, err := h.svc.Create(c.Request.Context(), userID, req)
	if err != nil {
		switch err {
		case ErrInvalidRequest:
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"code": "INVALID_REQUEST", "message": "Invalid input"}})
		case ErrReviewNotAllowed:
			c.JSON(http.StatusForbidden, gin.H{"success": false, "error": gin.H{"code": "FORBIDDEN", "message": "Review not allowed"}})
		case ErrConflict:
			c.JSON(http.StatusConflict, gin.H{"success": false, "error": gin.H{"code": "CONFLICT", "message": "Only one review per entity context"}})
		default:
			if err.Error() == "you must have a completed booking to leave a review for a studio" {
				c.JSON(http.StatusForbidden, gin.H{"success": false, "error": gin.H{"code": "FORBIDDEN", "message": err.Error()}})
			} else if err.Error() == "you have already reviewed this entity in this context" {
				c.JSON(http.StatusConflict, gin.H{"success": false, "error": gin.H{"code": "CONFLICT", "message": err.Error()}})
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": gin.H{"code": "INTERNAL", "message": "Internal error"}})
			}
		}
		return
	}

	c.JSON(http.StatusCreated, gin.H{"success": true, "data": rv})
}

// GetByTarget получает список отзывов о конкретной сущности (например, студии).
// @Summary		Получить отзывы сущности
// @Description	Возвращает постраничный список отзывов о сущности.
// @Tags		Отзывы
// @Param		target_type query	string	true	"Тип сущности (например, studio)"
// @Param       target_id   query   int     true    "ID сущности"
// @Param		limit	    query	int	    false	"Максимум количество отзывов (дефолт: 20)"
// @Param		offset	    query	int	    false	"Офсет с какого рекорда начинать"
// @Success		200	{object}		map[string]interface{} "Список отзывов"
// @Failure		400	{object}		map[string]interface{} "Ошибка: неверный ID или тип"
// @Failure		500	{object}		map[string]interface{} "Ошибка сервера"
// @Router		/reviews [GET]
func (h *Handler) GetByTarget(c *gin.Context) {
	targetType := c.Query("target_type")
	targetIDStr := c.Query("target_id")
	targetID, err := strconv.ParseInt(targetIDStr, 10, 64)
	if err != nil || targetID <= 0 || targetType == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"code": "INVALID_ID", "message": "Invalid target type or ID"}})
		return
	}

	limit, _ := strconv.Atoi(c.Query("limit"))
	offset, _ := strconv.Atoi(c.Query("offset"))

	items, err := h.svc.GetByTarget(c.Request.Context(), targetType, targetID, limit, offset)
	if err != nil {
		if err == ErrInvalidRequest {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"code": "INVALID_REQUEST", "message": "Invalid input"}})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": gin.H{"code": "INTERNAL", "message": "Internal error"}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": items})
}

// GetByStudioLegacy fallback for previous architecture.
// @Summary		Получить отзывы студии (Legacy)
// @Router		/studios/:id/reviews [GET]
func (h *Handler) GetByStudioLegacy(c *gin.Context) {
	studioID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || studioID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"code": "INVALID_ID", "message": "Invalid studio ID"}})
		return
	}

	limit, _ := strconv.Atoi(c.Query("limit"))
	offset, _ := strconv.Atoi(c.Query("offset"))

	items, err := h.svc.GetByTarget(c.Request.Context(), "studio", studioID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": gin.H{"code": "INTERNAL", "message": "Internal error"}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": items})
}

// AddOwnerResponse добавляет ответ владельца на отзыв.
// @Summary		Ответить на отзыв
// @Description	Владелец сущности может добавить ответ на отзыв.
// @Tags		Отзывы
// @Security	BearerAuth
// @Param		id		path	int				true	"ID отзыва"
// @Param		request	body	OwnerResponseRequest	true	"Текст ответа"
// @Success		200	{object}		map[string]interface{} "Ответ успешно добавлен"
// @Failure		400	{object}		map[string]interface{} "Ошибка валидации данных"
// @Failure		401	{object}		map[string]interface{} "Ошибка аутентификации"
// @Failure		403	{object}		map[string]interface{} "Запрещено: вы не овнер сущности"
// @Failure		404	{object}		map[string]interface{} "Отзыв не найден"
// @Failure		500	{object}		map[string]interface{} "Ошибка сервера при добавлении ответа"
// @Router		/reviews/:id/response [POST]
func (h *Handler) AddOwnerResponse(c *gin.Context) {
	reviewID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || reviewID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"code": "INVALID_ID", "message": "Invalid review ID"}})
		return
	}

	var req OwnerResponseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"code": "INVALID_REQUEST", "message": "Invalid request body"}})
		return
	}

	userID := c.GetInt64("user_id")
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": gin.H{"code": "UNAUTHORIZED", "message": "Authentication required"}})
		return
	}

	rv, err := h.svc.AddOwnerResponse(c.Request.Context(), reviewID, userID, req.Response)
	if err != nil {
		switch err {
		case ErrInvalidRequest:
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"code": "INVALID_REQUEST", "message": "Invalid input"}})
		case ErrForbidden:
			c.JSON(http.StatusForbidden, gin.H{"success": false, "error": gin.H{"code": "FORBIDDEN", "message": "You don't own this entity"}})
		case ErrNotFound:
			c.JSON(http.StatusNotFound, gin.H{"success": false, "error": gin.H{"code": "NOT_FOUND", "message": "Review not found"}})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": gin.H{"code": "INTERNAL", "message": "Internal error"}})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": rv})
}
