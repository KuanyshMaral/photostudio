package notification

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"

	"photostudio/internal/pkg/chicontext"
	"photostudio/internal/pkg/response"
)

// DeviceTokensHandler handles device tokens API endpoints
type DeviceTokensHandler struct {
	service *Service
}

// NewDeviceTokensHandler creates device tokens handler
func NewDeviceTokensHandler(service *Service) *DeviceTokensHandler {
	return &DeviceTokensHandler{service: service}
}

// RegisterDeviceToken регистрирует новый device token для push-уведомлений.
//
//	@Summary	Зарегистрировать device token
//	@Tags		Уведомления - Device Tokens
//	@Security	BearerAuth
//	@Router		/notifications/device-tokens [POST]
func (h *DeviceTokensHandler) RegisterDeviceToken(w http.ResponseWriter, r *http.Request) {
	userID := chicontext.UserIDFromCtx(r.Context())
	if userID == 0 {
		response.CustomError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "User not authenticated")
		return
	}

	var req RegisterDeviceTokenRequest
	if err := response.BindJSON(r, &req); err != nil {
		response.CustomError(w, r, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}

	dt, err := h.service.RegisterDeviceToken(r.Context(), userID, req.Token, req.Platform, req.DeviceName)
	if err != nil {
		response.CustomError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to register device token")
		return
	}

	response.Success(w, http.StatusCreated, h.deviceTokenToResponse(dt))
}

// ListDeviceTokens возвращает список активных device tokens.
//
//	@Summary	Получить device tokens
//	@Tags		Уведомления - Device Tokens
//	@Security	BearerAuth
//	@Router		/notifications/device-tokens [GET]
func (h *DeviceTokensHandler) ListDeviceTokens(w http.ResponseWriter, r *http.Request) {
	userID := chicontext.UserIDFromCtx(r.Context())
	if userID == 0 {
		response.CustomError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "User not authenticated")
		return
	}

	tokens, err := h.service.ListDeviceTokens(r.Context(), userID)
	if err != nil {
		response.CustomError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to list device tokens")
		return
	}

	respTokens := make([]*DeviceTokenResponse, len(tokens))
	for i, dt := range tokens {
		respTokens[i] = h.deviceTokenToResponse(dt)
	}

	response.Success(w, http.StatusOK, respTokens)
}

// DeactivateDeviceToken деактивирует device token.
//
//	@Summary	Деактивировать device token
//	@Tags		Уведомления - Device Tokens
//	@Security	BearerAuth
//	@Router		/notifications/device-tokens/{id} [DELETE]
func (h *DeviceTokensHandler) DeactivateDeviceToken(w http.ResponseWriter, r *http.Request) {
	userID := chicontext.UserIDFromCtx(r.Context())
	if userID == 0 {
		response.CustomError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "User not authenticated")
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		response.CustomError(w, r, http.StatusBadRequest, "INVALID_ID", "Invalid device token ID")
		return
	}

	if err := h.service.DeactivateDeviceToken(r.Context(), id, userID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.CustomError(w, r, http.StatusNotFound, "NOT_FOUND", "Device token not found")
			return
		}
		response.CustomError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to deactivate device token")
		return
	}

	response.Success(w, http.StatusOK, response.H{"status": "deactivated"})
}

func (h *DeviceTokensHandler) deviceTokenToResponse(dt *DeviceToken) *DeviceTokenResponse {
	resp := &DeviceTokenResponse{
		ID:         dt.ID,
		UserID:     dt.UserID,
		Token:      dt.Token,
		Platform:   dt.Platform,
		DeviceName: dt.DeviceName,
		IsActive:   dt.IsActive,
		CreatedAt:  dt.CreatedAt.Format(time.RFC3339),
	}
	if !dt.LastUsedAt.IsZero() {
		resp.LastUsedAt = dt.LastUsedAt.Format(time.RFC3339)
	}
	return resp
}
