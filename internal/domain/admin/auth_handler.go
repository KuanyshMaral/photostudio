package admin

import (
	"net/http"
	"photostudio/internal/pkg/response"
	"strings"

	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	service *Service
}

func NewAuthHandler(service *Service) *AuthHandler {
	return &AuthHandler{service: service}
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// Login godoc
// @Summary Admin Login
// @Description Authenticate as admin and get JWT token
// @Tags Admin Auth
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body LoginRequest true "Login credentials"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Router /admin/auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.CustomError(c, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	token, admin, err := h.service.Login(c.Request.Context(), strings.ToLower(req.Email), req.Password)
	if err != nil {
		response.CustomError(c, http.StatusUnauthorized, "AUTH_FAILED", "Invalid email or password")
		return
	}

	response.Success(c, http.StatusOK, gin.H{
		"access_token": token,
		"admin":        admin,
	})
}

// GetMe godoc
// @Summary Get current admin
// @Description Get current authenticated admin profile
// @Tags Admin Auth
// @Security BearerAuth
// @Produce json
// @Success 200 {object} response.Response
// @Failure 401 {object} response.Response
// @Router /admin/auth/me [get]
func (h *AuthHandler) GetMe(c *gin.Context) {
	adminID := c.GetString("admin_id")
	if adminID == "" {
		response.CustomError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
		return
	}

	admin, err := h.service.GetAdminByID(c.Request.Context(), adminID)
	if err != nil {
		response.CustomError(c, http.StatusNotFound, "NOT_FOUND", "Admin not found")
		return
	}

	admin.PasswordHash = ""
	response.Success(c, http.StatusOK, admin)
}
