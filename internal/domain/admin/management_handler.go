package admin

import (
	"net/http"
	"photostudio/internal/pkg/response"
	"strconv"

	"github.com/gin-gonic/gin"
)

type ManagementHandler struct {
	service *Service
}

func NewManagementHandler(service *Service) *ManagementHandler {
	return &ManagementHandler{service: service}
}

type CreateAdminRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
	Name     string `json:"name" binding:"required"`
	Role     string `json:"role" binding:"required"` // super_admin, support, moderator
}

type UpdateAdminRequest struct {
	Name     string `json:"name"`
	Role     string `json:"role"`
	IsActive *bool  `json:"is_active"`
	Password string `json:"password"` // optional
}

// ListAdmins godoc
// @Summary List admins
// @Description Get list of administrators
// @Tags Admin Management
// @Security BearerAuth
// @Param page query int false "Page number"
// @Param limit query int false "Items per page"
// @Produce json
// @Success 200 {object} response.Response
// @Failure 401 {object} response.Response
// @Router /admin/admins [get]
func (h *ManagementHandler) ListAdmins(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	admins, total, err := h.service.ListAdmins(c.Request.Context(), page, limit)
	if err != nil {
		response.CustomError(c, http.StatusInternalServerError, "INTERNAL_ERROR", err)
		return
	}

	response.Success(c, http.StatusOK, gin.H{
		"admins": admins,
		"total":  total,
		"page":   page,
		"limit":  limit,
	})
}

// CreateAdmin godoc
// @Summary Create admin
// @Description Create a new administrator account
// @Tags Admin Management
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body CreateAdminRequest true "Admin details"
// @Success 201 {object} response.Response
// @Failure 400 {object} response.Response
// @Router /admin/admins [post]
func (h *ManagementHandler) CreateAdmin(c *gin.Context) {
	var req CreateAdminRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.CustomError(c, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	admin, err := h.service.CreateAdmin(c.Request.Context(), req.Email, req.Password, req.Name, req.Role)
	if err != nil {
		response.CustomError(c, http.StatusInternalServerError, "INTERNAL_ERROR", err)
		return
	}

	response.Success(c, http.StatusCreated, admin)
}

// UpdateAdmin godoc
// @Summary Update admin
// @Description Update administrator details
// @Tags Admin Management
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "Admin UUID"
// @Param request body UpdateAdminRequest true "Update details"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /admin/admins/{id} [patch]
func (h *ManagementHandler) UpdateAdmin(c *gin.Context) {
	id := c.Param("id")

	var req UpdateAdminRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.CustomError(c, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	updates := make(map[string]interface{})
	if req.Name != "" {
		updates["name"] = req.Name
	}
	if req.Role != "" {
		updates["role"] = req.Role
	}
	if req.Password != "" {
		updates["password"] = req.Password
	}
	if req.IsActive != nil {
		updates["is_active"] = *req.IsActive
	}

	admin, err := h.service.UpdateAdmin(c.Request.Context(), id, updates)
	if err != nil {
		response.CustomError(c, http.StatusInternalServerError, "INTERNAL_ERROR", err)
		return
	}

	response.Success(c, http.StatusOK, admin)
}
