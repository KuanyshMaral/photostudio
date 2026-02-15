package profile

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"photostudio/internal/pkg/response"
	"photostudio/internal/pkg/validator"
)

// ClientHandler handles client profile HTTP requests
type ClientHandler struct {
	service *Service
}

// NewClientHandler creates client profile handler
func NewClientHandler(service *Service) *ClientHandler {
	return &ClientHandler{service: service}
}

// GetProfile handles GET /api/v1/profile/client
// @Summary Get client profile
// @Description Get authenticated client's profile
// @Tags Client Profile
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Response{data=ClientProfile}
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /profile/client [get]
func (h *ClientHandler) GetProfile(c *gin.Context) {
	userID := c.GetInt64("user_id")

	profile, err := h.service.GetClientProfile(c.Request.Context(), userID)
	if err != nil {
		if err == ErrProfileNotFound {
			// Auto-create if not exists (should happen on email verification)
			profile, err = h.service.EnsureClientProfile(c.Request.Context(), userID)
			if err != nil {
				response.CustomError(c, http.StatusInternalServerError, "INTERNAL_ERROR", err)
				return
			}
		} else {
			response.CustomError(c, http.StatusInternalServerError, "INTERNAL_ERROR", err)
			return
		}
	}

	response.Success(c, http.StatusOK, profile)
}

// UpdateProfile handles PUT /api/v1/profile/client
// @Summary Update client profile
// @Description Update authenticated client's profile
// @Tags Client Profile
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body UpdateClientProfileRequest true "Profile update"
// @Success 200 {object} response.Response{data=ClientProfile}
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 422 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /profile/client [put]
func (h *ClientHandler) UpdateProfile(c *gin.Context) {
	userID := c.GetInt64("user_id")

	var req UpdateClientProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.CustomError(c, http.StatusBadRequest, "INVALID_JSON", "Invalid JSON body")
		return
	}

	if errors := validator.Validate(&req); errors != nil {
		response.CustomError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", errors)
		return
	}

	profile, err := h.service.UpdateClientProfile(c.Request.Context(), userID, &req)
	if err != nil {
		if err == ErrProfileNotFound {
			response.CustomError(c, http.StatusNotFound, "PROFILE_NOT_FOUND", "Profile not found")
			return
		}
		response.CustomError(c, http.StatusInternalServerError, "INTERNAL_ERROR", err)
		return
	}

	response.Success(c, http.StatusOK, profile)
}
