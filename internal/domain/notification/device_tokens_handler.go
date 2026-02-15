package notification

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"photostudio/internal/pkg/response"
	"strconv"
	"time"
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
// @Summary		Зарегистрировать device token
// @Description	Регистрирует новый device token (для мобильных или веб-приложений) для получения push-уведомлений.
// @Tags		Уведомления - Device Tokens
// @Security	BearerAuth
// @Param		body	body		RegisterDeviceTokenRequest	true	"Device token информация"
// @Success		201	{object}		DeviceTokenResponse "Device token зарегистрирован"
// @Failure		400	{object}		map[string]interface{} "Ошибка валидации"
// @Failure		401	{object}		map[string]interface{} "Ошибка аутентификации"
// @Failure		500	{object}		map[string]interface{} "Ошибка сервера"
// @Router		/notifications/device-tokens [POST]
func (h *DeviceTokensHandler) RegisterDeviceToken(c *gin.Context) {
	userID := c.GetInt64("user_id")
	if userID == 0 {
		response.CustomError(c, http.StatusUnauthorized, "UNAUTHORIZED", "User not authenticated")
		return
	}

	var req RegisterDeviceTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.CustomError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}

	dt, err := h.service.RegisterDeviceToken(c.Request.Context(), userID, req.Token, req.Platform, req.DeviceName)
	if err != nil {
		response.CustomError(c, http.StatusInternalServerError, "REGISTER_FAILED", "Failed to register device token")
		return
	}

	respToken := h.deviceTokenToResponse(dt)
	response.Success(c, http.StatusCreated, respToken)
}

// ListDeviceTokens возвращает список активных device tokens.
// @Summary		Получить device tokens
// @Description	Возвращает список всех активных device tokens для текущего пользователя.
// @Tags		Уведомления - Device Tokens
// @Security	BearerAuth
// @Success		200	{object}		[]DeviceTokenResponse "Список device tokens"
// @Failure		401	{object}		map[string]interface{} "Ошибка аутентификации"
// @Failure		500	{object}		map[string]interface{} "Ошибка сервера"
// @Router		/notifications/device-tokens [GET]
func (h *DeviceTokensHandler) ListDeviceTokens(c *gin.Context) {
	userID := c.GetInt64("user_id")
	if userID == 0 {
		response.CustomError(c, http.StatusUnauthorized, "UNAUTHORIZED", "User not authenticated")
		return
	}

	tokens, err := h.service.ListDeviceTokens(c.Request.Context(), userID)
	if err != nil {
		response.CustomError(c, http.StatusInternalServerError, "FETCH_FAILED", "Failed to list device tokens")
		return
	}

	respTokens := make([]*DeviceTokenResponse, len(tokens))
	for i, dt := range tokens {
		respTokens[i] = h.deviceTokenToResponse(dt)
	}

	response.Success(c, http.StatusOK, respTokens)
}

// DeactivateDeviceToken деактивирует device token.
// @Summary		Деактивировать device token
// @Description	Деактивирует device token, устройство больше не будет получать push-уведомления.
// @Tags		Уведомления - Device Tokens
// @Security	BearerAuth
// @Param		id	path	int	true	"ID device token"
// @Success		200	{object}		map[string]interface{} "Device token деактивирован"
// @Failure		400	{object}		map[string]interface{} "Ошибка: неверный ID"
// @Failure		401	{object}		map[string]interface{} "Ошибка аутентификации"
// @Failure		500	{object}		map[string]interface{} "Ошибка сервера"
// @Router		/notifications/device-tokens/{id} [DELETE]
func (h *DeviceTokensHandler) DeactivateDeviceToken(c *gin.Context) {
	userID := c.GetInt64("user_id")
	if userID == 0 {
		response.CustomError(c, http.StatusUnauthorized, "UNAUTHORIZED", "User not authenticated")
		return
	}

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.CustomError(c, http.StatusBadRequest, "INVALID_ID", "Invalid device token ID")
		return
	}

	if err := h.service.DeactivateDeviceToken(c.Request.Context(), id); err != nil {
		response.CustomError(c, http.StatusInternalServerError, "DEACTIVATE_FAILED", "Failed to deactivate device token")
		return
	}

	response.Success(c, http.StatusOK, gin.H{"status": "deactivated"})
}

// deviceTokenToResponse converts DeviceToken to DeviceTokenResponse
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
