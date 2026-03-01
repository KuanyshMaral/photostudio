package attachment

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"photostudio/internal/pkg/chicontext"
	"photostudio/internal/pkg/response"
)

// Handler handles HTTP requests for the attachment domain.
type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// swaggerAttachmentResponse is a wrapper strictly for generating Swagger documentation.
type swaggerAttachmentResponse struct {
	Success bool          `json:"success"`
	Data    []*Attachment `json:"data"`
}

// Attach привязка загруженных файлов к сущности.
//
//	@Summary		Привязать файлы (вложения)
//	@Description	Связывает один или несколько загруженных файлов (upload_ids) с бизнес-сущностью (например, галерея студии, комнаты и тд).
//	@Tags			Attachments
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			request	body		attachRequest				true	"Данные для привязки"
//	@Success		201		{object}	swaggerAttachmentResponse	"Успешная привязка"
//	@Failure		400		{object}	response.ErrorResponse		"Неверный запрос"
//	@Failure		401		{object}	response.ErrorResponse		"Не авторизован"
//	@Failure		403		{object}	response.ErrorResponse		"Нет прав доступа"
//	@Failure		422		{object}	response.ErrorResponse		"Некорректная цель привязки"
//	@Failure		500		{object}	response.ErrorResponse		"Внутренняя ошибка сервера"
//	@Router			/attachments [post]
func (h *Handler) Attach(w http.ResponseWriter, r *http.Request) {
	callerID := chicontext.UserIDFromCtx(r.Context())
	if callerID == 0 {
		response.CustomError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "unauthorized")
		return
	}

	var req attachRequest
	if err := response.BindJSON(r, &req); err != nil {
		response.CustomError(w, r, http.StatusBadRequest, "INVALID_BODY", err.Error())
		return
	}
	if len(req.UploadIDs) == 0 {
		response.CustomError(w, r, http.StatusBadRequest, "MISSING_UPLOAD_IDS", "upload_ids required")
		return
	}

	results, err := h.service.Attach(
		r.Context(),
		req.UploadIDs,
		callerID,
		TargetType(req.TargetType),
		req.TargetID,
		Metadata{Caption: req.Caption},
	)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidTarget):
			response.CustomError(w, r, http.StatusUnprocessableEntity, "INVALID_TARGET", err.Error())
		case errors.Is(err, ErrNotOwner):
			response.CustomError(w, r, http.StatusForbidden, "FORBIDDEN", err.Error())
		default:
			response.CustomError(w, r, http.StatusInternalServerError, "ATTACH_FAILED", "attach failed")
		}
		return
	}

	response.JSON(w, http.StatusCreated, response.H{"success": true, "data": results})
}

// ListByTarget получение списка вложений сущности.
//
//	@Summary		Список вложений сущности
//	@Description	Возвращает все файлы (вложения), привязанные к определенной бизнес-сущности.
//	@Tags			Attachments
//	@Produce		json
//	@Param			target_type	query		string						true	"Тип сущности (studio_gallery, room_gallery, review_photos, chat_message)"
//	@Param			target_id	query		int							true	"ID сущности"
//	@Success		200			{object}	swaggerAttachmentResponse	"Список вложений"
//	@Failure		400			{object}	response.ErrorResponse		"Неверные параметры запроса"
//	@Failure		422			{object}	response.ErrorResponse		"Некорректная цель"
//	@Failure		500			{object}	response.ErrorResponse		"Внутренняя ошибка сервера"
//	@Router			/attachments [get]
func (h *Handler) ListByTarget(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	targetType := TargetType(q.Get("target_type"))
	targetID, err := strconv.ParseInt(q.Get("target_id"), 10, 64)
	if err != nil || targetID <= 0 {
		response.CustomError(w, r, http.StatusBadRequest, "INVALID_PARAMS", "valid target_id required")
		return
	}

	results, err := h.service.ListByTarget(r.Context(), targetType, targetID)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidTarget):
			response.CustomError(w, r, http.StatusUnprocessableEntity, "INVALID_TARGET", err.Error())
		default:
			response.CustomError(w, r, http.StatusInternalServerError, "FETCH_FAILED", "list failed")
		}
		return
	}

	response.JSON(w, http.StatusOK, response.H{"success": true, "data": results})
}

// Delete удаление привязки файла.
//
//	@Summary		Удалить вложение
//	@Description	Удаляет связь файла с сущностью (саму запись файла upload не удаляет).
//	@Tags			Attachments
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		int							true	"ID вложения"
//	@Success		200	{object}	response.SuccessResponse	"Вложение удалено"
//	@Failure		400	{object}	response.ErrorResponse		"Неверный ID"
//	@Failure		401	{object}	response.ErrorResponse		"Не авторизован"
//	@Failure		403	{object}	response.ErrorResponse		"Нет прав на удаление этого вложения"
//	@Failure		404	{object}	response.ErrorResponse		"Вложение не найдено"
//	@Failure		500	{object}	response.ErrorResponse		"Внутренняя ошибка сервера"
//	@Router			/attachments/{id} [delete]
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	callerID := chicontext.UserIDFromCtx(r.Context())
	if callerID == 0 {
		response.CustomError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "unauthorized")
		return
	}

	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		response.CustomError(w, r, http.StatusBadRequest, "INVALID_ID", "invalid attachment id")
		return
	}

	if err := h.service.Delete(r.Context(), id, callerID); err != nil {
		switch {
		case errors.Is(err, ErrNotFound):
			response.CustomError(w, r, http.StatusNotFound, "NOT_FOUND", "attachment not found")
		case errors.Is(err, ErrNotOwner):
			response.CustomError(w, r, http.StatusForbidden, "FORBIDDEN", "you do not own this attachment")
		default:
			response.CustomError(w, r, http.StatusInternalServerError, "DELETE_FAILED", "delete failed")
		}
		return
	}

	response.JSON(w, http.StatusOK, response.H{"success": true, "message": "attachment removed"})
}

// Reorder изменение порядка вложений.
//
//	@Summary		Переупорядочить вложения
//	@Description	Изменяет порядок вложений для сущности. Отправьте массив ID вложений в нужном порядке.
//	@Tags			Attachments
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			request	body		reorderRequest				true	"Новый порядок ID"
//	@Success		200		{object}	response.SuccessResponse	"Порядок обновлен"
//	@Failure		400		{object}	response.ErrorResponse		"Неверный запрос"
//	@Failure		401		{object}	response.ErrorResponse		"Не авторизован"
//	@Failure		500		{object}	response.ErrorResponse		"Внутренняя ошибка сервера"
//	@Router			/attachments/reorder [patch]
func (h *Handler) Reorder(w http.ResponseWriter, r *http.Request) {
	callerID := chicontext.UserIDFromCtx(r.Context())
	if callerID == 0 {
		response.CustomError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "unauthorized")
		return
	}

	var req reorderRequest
	if err := response.BindJSON(r, &req); err != nil {
		response.CustomError(w, r, http.StatusBadRequest, "INVALID_BODY", err.Error())
		return
	}

	if err := h.service.Reorder(r.Context(), req.IDs); err != nil {
		response.CustomError(w, r, http.StatusInternalServerError, "REORDER_FAILED", "reorder failed")
		return
	}

	response.JSON(w, http.StatusOK, response.H{"success": true, "message": "reordered"})
}

// ─── Request DTOs ─────────────────────────────────────────────────────────────

type attachRequest struct {
	UploadIDs  []string `json:"upload_ids"`
	TargetType string   `json:"target_type"`
	TargetID   int64    `json:"target_id"`
	Caption    string   `json:"caption"`
}

type reorderRequest struct {
	IDs []int64 `json:"ids"`
}
