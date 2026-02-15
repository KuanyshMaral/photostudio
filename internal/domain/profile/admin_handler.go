package profile

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"photostudio/internal/pkg/response"
	"photostudio/internal/pkg/validator"
)

// AdminHandler handles admin profile HTTP requests
type AdminHandler struct {
	service *Service
}

// NewAdminHandler creates admin profile handler
func NewAdminHandler(service *Service) *AdminHandler {
	return &AdminHandler{service: service}
}

// GetProfile handles GET /api/v1/profile/admin
// @Summary Get admin profile
// @Description Get authenticated admin's profile
// @Tags Admin Profile
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Response{data=AdminProfile}
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /profile/admin [get]
func (h *AdminHandler) GetProfile(c *gin.Context) {
	adminIDStr := c.GetString("admin_id")
	userID, err := uuid.Parse(adminIDStr)
	if err != nil {
		response.CustomError(c, http.StatusUnauthorized, "INVALID_TOKEN", "Invalid admin ID")
		return
	}

	profile, err := h.service.GetAdminProfile(c.Request.Context(), userID)
	if err != nil {
		if err == ErrProfileNotFound {
			response.CustomError(c, http.StatusNotFound, "PROFILE_NOT_FOUND", "Admin profile not found")
			return
		}
		response.CustomError(c, http.StatusInternalServerError, "INTERNAL_ERROR", err)
		return
	}

	response.Success(c, http.StatusOK, profile)
}

// UpdateProfile handles PUT /api/v1/profile/admin
// @Summary Update admin profile
// @Description Update authenticated admin's profile (admin only)
// @Tags Admin Profile
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body UpdateAdminProfileRequest true "Profile update"
// @Success 200 {object} response.Response{data=AdminProfile}
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 422 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /profile/admin [put]
func (h *AdminHandler) UpdateProfile(c *gin.Context) {
	adminIDStr := c.GetString("admin_id")
	userID, err := uuid.Parse(adminIDStr)
	if err != nil {
		response.CustomError(c, http.StatusUnauthorized, "INVALID_TOKEN", "Invalid admin ID")
		return
	}

	var req UpdateAdminProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.CustomError(c, http.StatusBadRequest, "INVALID_JSON", "Invalid JSON body")
		return
	}

	if errors := validator.Validate(&req); errors != nil {
		response.CustomError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", errors)
		return
	}

	profile, err := h.service.UpdateAdminProfile(c.Request.Context(), userID, &req)
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
