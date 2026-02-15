package profile

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"photostudio/internal/pkg/response"
	"photostudio/internal/pkg/validator"
)

// OwnerHandler handles owner profile HTTP requests
type OwnerHandler struct {
	service *Service
}

// NewOwnerHandler creates owner profile handler
func NewOwnerHandler(service *Service) *OwnerHandler {
	return &OwnerHandler{service: service}
}

// GetProfile handles GET /api/v1/profile/owner
// @Summary Get owner profile
// @Description Get authenticated owner's profile
// @Tags Owner Profile
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Response{data=OwnerProfile}
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /profile/owner [get]
func (h *OwnerHandler) GetProfile(c *gin.Context) {
	userID := c.GetInt64("user_id")

	profile, err := h.service.GetOwnerProfile(c.Request.Context(), userID)
	if err != nil {
		if err == ErrProfileNotFound {
			response.CustomError(c, http.StatusNotFound, "PROFILE_NOT_FOUND", "Owner profile not found")
			return
		}
		response.CustomError(c, http.StatusInternalServerError, "INTERNAL_ERROR", err)
		return
	}

	response.Success(c, http.StatusOK, profile)
}

// UpdateProfile handles PUT /api/v1/profile/owner
// @Summary Update owner profile
// @Description Update authenticated owner's profile
// @Tags Owner Profile
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body UpdateOwnerProfileRequest true "Profile update"
// @Success 200 {object} response.Response{data=OwnerProfile}
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 422 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /profile/owner [put]
func (h *OwnerHandler) UpdateProfile(c *gin.Context) {
	userID := c.GetInt64("user_id")

	var req UpdateOwnerProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.CustomError(c, http.StatusBadRequest, "INVALID_JSON", "Invalid JSON body")
		return
	}

	if errors := validator.Validate(&req); errors != nil {
		response.CustomError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", errors)
		return
	}

	profile, err := h.service.UpdateOwnerProfile(c.Request.Context(), userID, &req)
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
