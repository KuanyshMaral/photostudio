package review

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"photostudio/internal/pkg/chicontext"
	"photostudio/internal/pkg/response"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// swaggerReviewResponse is a wrapper strictly for generating Swagger documentation.
type swaggerReviewResponse struct {
	Success bool    `json:"success"`
	Data    *Review `json:"data"`
}

// swaggerListReviewResponse is a wrapper strictly for generating Swagger documentation.
type swaggerListReviewResponse struct {
	Success bool      `json:"success"`
	Data    []*Review `json:"data"`
}

// Create создать новый отзыв.
//
//	@Summary		Написать отзыв
//	@Description	Пользователь может написать отзыв о сущности (студия, профиль). Для студий требуется завершенное бронирование.
//	@Tags			Reviews
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			request	body		CreateReviewRequest		true	"Данные отзыва"
//	@Success		201		{object}	swaggerReviewResponse	"Отыв создан"
//	@Failure		400		{object}	response.ErrorResponse	"Ошибка входных данных"
//	@Failure		401		{object}	response.ErrorResponse	"Не авторизован"
//	@Failure		403		{object}	response.ErrorResponse	"Оставление отзыва запрещено (нет брони и т.д.)"
//	@Failure		409		{object}	response.ErrorResponse	"Отзыв от этого пользователя уже существует"
//	@Failure		500		{object}	response.ErrorResponse	"Внутренняя ошибка сервера"
//	@Router			/reviews [post]
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateReviewRequest
	if err := response.BindJSON(r, &req); err != nil {
		response.JSON(w, http.StatusBadRequest, response.H{"success": false, "error": response.H{"code": "INVALID_REQUEST", "message": "Invalid request body"}})
		return
	}

	userID := chicontext.UserIDFromCtx(r.Context())
	if userID == 0 {
		response.JSON(w, http.StatusUnauthorized, response.H{"success": false, "error": response.H{"code": "UNAUTHORIZED", "message": "Authentication required"}})
		return
	}

	rv, err := h.svc.Create(r.Context(), userID, req)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidRequest):
			response.JSON(w, http.StatusBadRequest, response.H{"success": false, "error": response.H{"code": "INVALID_REQUEST", "message": "Invalid input"}})
		case errors.Is(err, ErrReviewNotAllowed):
			response.JSON(w, http.StatusForbidden, response.H{"success": false, "error": response.H{"code": "FORBIDDEN", "message": "Review not allowed"}})
		case errors.Is(err, ErrConflict):
			response.JSON(w, http.StatusConflict, response.H{"success": false, "error": response.H{"code": "CONFLICT", "message": "Only one review per entity context"}})
		default:
			msg := err.Error()
			if msg == "you must have a completed booking to leave a review for a studio" {
				response.JSON(w, http.StatusForbidden, response.H{"success": false, "error": response.H{"code": "FORBIDDEN", "message": msg}})
			} else if msg == "you have already reviewed this entity in this context" {
				response.JSON(w, http.StatusConflict, response.H{"success": false, "error": response.H{"code": "CONFLICT", "message": msg}})
			} else {
				response.JSON(w, http.StatusInternalServerError, response.H{"success": false, "error": response.H{"code": "INTERNAL", "message": "Internal error"}})
			}
		}
		return
	}

	response.JSON(w, http.StatusCreated, response.H{"success": true, "data": rv})
}

// GetByTarget получить отзывы сущности.
//
//	@Summary		Получить отзывы
//	@Description	Возвращает список отзывов для определенной сущности (например, студии).
//	@Tags			Reviews
//	@Produce		json
//	@Param			target_type	query		string						true	"Тип сущности (например, studio)"
//	@Param			target_id	query		int							true	"ID сущности"
//	@Param			limit		query		int							false	"Лимит (дефолт: 20)"
//	@Param			offset		query		int							false	"Отступ (дефолт: 0)"
//	@Success		200			{object}	swaggerListReviewResponse	"Список отзывов"
//	@Failure		400			{object}	response.ErrorResponse		"Неверные параметры запроса"
//	@Failure		500			{object}	response.ErrorResponse		"Внутренняя ошибка сервера"
//	@Router			/reviews [get]
func (h *Handler) GetByTarget(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	targetType := q.Get("target_type")
	targetID, err := strconv.ParseInt(q.Get("target_id"), 10, 64)
	if err != nil || targetID <= 0 || targetType == "" {
		response.JSON(w, http.StatusBadRequest, response.H{"success": false, "error": response.H{"code": "INVALID_ID", "message": "Invalid target type or ID"}})
		return
	}

	limit, _ := strconv.Atoi(q.Get("limit"))
	offset, _ := strconv.Atoi(q.Get("offset"))

	items, err := h.svc.GetByTarget(r.Context(), targetType, targetID, limit, offset)
	if err != nil {
		if errors.Is(err, ErrInvalidRequest) {
			response.JSON(w, http.StatusBadRequest, response.H{"success": false, "error": response.H{"code": "INVALID_REQUEST", "message": "Invalid input"}})
			return
		}
		response.JSON(w, http.StatusInternalServerError, response.H{"success": false, "error": response.H{"code": "INTERNAL", "message": "Internal error"}})
		return
	}

	response.JSON(w, http.StatusOK, response.H{"success": true, "data": items})
}

// GetByStudioLegacy получить отзывы студии (Legacy).
//
//	@Summary		Получить отзывы студии
//	@Description	Legacy эндпоинт для получения отзывов студии.
//	@Tags			Reviews
//	@Produce		json
//	@Param			id		path		int							true	"ID студии"
//	@Param			limit	query		int							false	"Лимит"
//	@Param			offset	query		int							false	"Отступ"
//	@Success		200		{object}	swaggerListReviewResponse	"Список отзывов"
//	@Failure		400		{object}	response.ErrorResponse		"Некорректный ID"
//	@Failure		500		{object}	response.ErrorResponse		"Внутренняя ошибка сервера"
//	@Router			/studios/{id}/reviews [get]
func (h *Handler) GetByStudioLegacy(w http.ResponseWriter, r *http.Request) {
	studioID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || studioID <= 0 {
		response.JSON(w, http.StatusBadRequest, response.H{"success": false, "error": response.H{"code": "INVALID_ID", "message": "Invalid studio ID"}})
		return
	}

	q := r.URL.Query()
	limit, _ := strconv.Atoi(q.Get("limit"))
	offset, _ := strconv.Atoi(q.Get("offset"))

	items, err := h.svc.GetByTarget(r.Context(), "studio", studioID, limit, offset)
	if err != nil {
		response.JSON(w, http.StatusInternalServerError, response.H{"success": false, "error": response.H{"code": "INTERNAL", "message": "Internal error"}})
		return
	}

	response.JSON(w, http.StatusOK, response.H{"success": true, "data": items})
}

// AddOwnerResponse ответить на отзыв.
//
//	@Summary		Ответить на отзыв
//	@Description	Владелец сущности может оставить официальный ответ на отзыв.
//	@Tags			Reviews
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		int						true	"ID отзыва"
//	@Param			request	body		OwnerResponseRequest	true	"Текст ответа"
//	@Success		200		{object}	swaggerReviewResponse	"Ответ добавлен"
//	@Failure		400		{object}	response.ErrorResponse	"Некорректный запрос"
//	@Failure		401		{object}	response.ErrorResponse	"Не авторизован"
//	@Failure		403		{object}	response.ErrorResponse	"Вложено запрещено оставлять ответ"
//	@Failure		404		{object}	response.ErrorResponse	"Отзыв не найден"
//	@Failure		500		{object}	response.ErrorResponse	"Внутренняя ошибка сервера"
//	@Router			/reviews/{id}/response [post]
func (h *Handler) AddOwnerResponse(w http.ResponseWriter, r *http.Request) {
	reviewID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || reviewID <= 0 {
		response.JSON(w, http.StatusBadRequest, response.H{"success": false, "error": response.H{"code": "INVALID_ID", "message": "Invalid review ID"}})
		return
	}

	var req OwnerResponseRequest
	if err := response.BindJSON(r, &req); err != nil {
		response.JSON(w, http.StatusBadRequest, response.H{"success": false, "error": response.H{"code": "INVALID_REQUEST", "message": "Invalid request body"}})
		return
	}

	userID := chicontext.UserIDFromCtx(r.Context())
	if userID == 0 {
		response.JSON(w, http.StatusUnauthorized, response.H{"success": false, "error": response.H{"code": "UNAUTHORIZED", "message": "Authentication required"}})
		return
	}

	rv, err := h.svc.AddOwnerResponse(r.Context(), reviewID, userID, req.Response)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidRequest):
			response.JSON(w, http.StatusBadRequest, response.H{"success": false, "error": response.H{"code": "INVALID_REQUEST", "message": "Invalid input"}})
		case errors.Is(err, ErrForbidden):
			response.JSON(w, http.StatusForbidden, response.H{"success": false, "error": response.H{"code": "FORBIDDEN", "message": "You don't own this entity"}})
		case errors.Is(err, ErrNotFound):
			response.JSON(w, http.StatusNotFound, response.H{"success": false, "error": response.H{"code": "NOT_FOUND", "message": "Review not found"}})
		default:
			response.JSON(w, http.StatusInternalServerError, response.H{"success": false, "error": response.H{"code": "INTERNAL", "message": "Internal error"}})
		}
		return
	}

	response.JSON(w, http.StatusOK, response.H{"success": true, "data": rv})
}
