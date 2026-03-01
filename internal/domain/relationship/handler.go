package relationship

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"photostudio/internal/pkg/chicontext"
	"photostudio/internal/pkg/response"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

type blockRequest struct {
	UserID int64 `json:"user_id"`
}

// swaggerBlockedUser is a wrapper strictly for generating Swagger documentation.
type swaggerBlockedUser struct {
	UserID    int64  `json:"user_id"`
	BlockedAt string `json:"blocked_at"` // using string for time representation
}

// swaggerListBlockedResponse is a wrapper strictly for generating Swagger documentation.
type swaggerListBlockedResponse struct {
	Success bool                 `json:"success"`
	Data    []swaggerBlockedUser `json:"data"`
}

// Block блокирует пользователя.
//
//	@Summary		Заблокировать пользователя
//	@Description	Добавляет указанного пользователя в черный список текущего авторизованного пользователя. Блокированный пользователь не сможет писать сообщения или совершать другие взаимодействия.
//	@Tags			Relationships
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			request	body		blockRequest				true	"ID пользователя для блокировки"
//	@Success		200		{object}	response.SuccessResponse	"Пользователь заблокирован"
//	@Failure		400		{object}	response.ErrorResponse		"Ошибка запроса или попытка заблокировать себя"
//	@Failure		401		{object}	response.ErrorResponse		"Не авторизован"
//	@Failure		409		{object}	response.ErrorResponse		"Пользователь уже заблокирован"
//	@Failure		500		{object}	response.ErrorResponse		"Внутренняя ошибка сервера"
//	@Router			/relationships/block [post]
func (h *Handler) Block(w http.ResponseWriter, r *http.Request) {
	userID := chicontext.UserIDFromCtx(r.Context())
	if userID == 0 {
		response.CustomError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "unauthorized")
		return
	}
	var req blockRequest
	if err := response.BindJSON(r, &req); err != nil {
		response.CustomError(w, r, http.StatusBadRequest, "INVALID_BODY", err.Error())
		return
	}
	if err := h.service.Block(r.Context(), userID, req.UserID); err != nil {
		switch {
		case errors.Is(err, ErrCannotBlockSelf):
			response.CustomError(w, r, http.StatusBadRequest, "CANNOT_BLOCK_SELF", err.Error())
		case errors.Is(err, ErrAlreadyBlocked):
			response.CustomError(w, r, http.StatusConflict, "ALREADY_BLOCKED", err.Error())
		default:
			response.CustomError(w, r, http.StatusInternalServerError, "BLOCK_FAILED", "failed to block user")
		}
		return
	}
	response.JSON(w, http.StatusOK, response.H{"success": true, "message": "user blocked"})
}

// Unblock разблокирует пользователя.
//
//	@Summary		Разблокировать пользователя
//	@Description	Удаляет пользователя из черного списка.
//	@Tags			Relationships
//	@Produce		json
//	@Security		BearerAuth
//	@Param			user_id	path		int							true	"ID пользователя"
//	@Success		200		{object}	response.SuccessResponse	"Пользователь разблокирован"
//	@Failure		400		{object}	response.ErrorResponse		"Некорректный ID"
//	@Failure		401		{object}	response.ErrorResponse		"Не авторизован"
//	@Failure		404		{object}	response.ErrorResponse		"Пользователь не был заблокирован"
//	@Failure		500		{object}	response.ErrorResponse		"Внутренняя ошибка сервера"
//	@Router			/relationships/block/{user_id} [delete]
func (h *Handler) Unblock(w http.ResponseWriter, r *http.Request) {
	userID := chicontext.UserIDFromCtx(r.Context())
	if userID == 0 {
		response.CustomError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "unauthorized")
		return
	}
	targetID, err := strconv.ParseInt(chi.URLParam(r, "user_id"), 10, 64)
	if err != nil {
		response.CustomError(w, r, http.StatusBadRequest, "INVALID_PARAMS", "invalid user_id")
		return
	}
	if err := h.service.Unblock(r.Context(), userID, targetID); err != nil {
		if errors.Is(err, ErrNotBlocked) {
			response.CustomError(w, r, http.StatusNotFound, "NOT_FOUND", err.Error())
		} else {
			response.CustomError(w, r, http.StatusInternalServerError, "UNBLOCK_FAILED", "failed to unblock user")
		}
		return
	}
	response.JSON(w, http.StatusOK, response.H{"success": true, "message": "user unblocked"})
}

// ListBlocked список заблокированных пользователей.
//
//	@Summary		Список заблокированных пользователей
//	@Description	Возвращает список ID пользователей, которых заблокировал текущий пользователь, и время блокировки.
//	@Tags			Relationships
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	swaggerListBlockedResponse	"Успех"
//	@Failure		401	{object}	response.ErrorResponse		"Не авторизован"
//	@Failure		500	{object}	response.ErrorResponse		"Внутренняя ошибка сервера"
//	@Router			/relationships/blocked [get]
func (h *Handler) ListBlocked(w http.ResponseWriter, r *http.Request) {
	userID := chicontext.UserIDFromCtx(r.Context())
	if userID == 0 {
		response.CustomError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "unauthorized")
		return
	}
	blocked, err := h.service.ListBlocked(r.Context(), userID)
	if err != nil {
		response.CustomError(w, r, http.StatusInternalServerError, "FETCH_FAILED", "failed to list blocked users")
		return
	}
	items := make([]response.H, 0, len(blocked))
	for _, b := range blocked {
		items = append(items, response.H{"user_id": b.BlockedID, "blocked_at": b.CreatedAt})
	}
	response.JSON(w, http.StatusOK, response.H{"success": true, "data": items})
}
