package admin

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"photostudio/internal/pkg/response"
)

type ManagementHandler struct {
	service *Service
}

func NewManagementHandler(service *Service) *ManagementHandler {
	return &ManagementHandler{service: service}
}

type CreateAdminRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name"`
	Role     string `json:"role"` // super_admin, support, moderator
}

type UpdateAdminRequest struct {
	Name     string `json:"name"`
	Role     string `json:"role"`
	IsActive *bool  `json:"is_active"`
	Password string `json:"password"` // optional
}

// ListAdmins
//
//	@Summary		List admins
//	@Description	Get a paginated list of administrators
//	@Tags			Admin
//	@Accept			json
//	@Produce		json
//	@Param			page	query		int	false	"Page number"	default(1)
//	@Param			limit	query		int	false	"Page size"		default(20)
//	@Success		200		{object}	map[string]interface{}
//	@Failure		500		{object}	map[string]interface{}
//	@Security		BearerAuth
//	@Router			/admin/management/admins [get]
func (h *ManagementHandler) ListAdmins(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page <= 0 {
		page = 1
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 {
		limit = 20
	}

	admins, total, err := h.service.ListAdmins(r.Context(), page, limit)
	if err != nil {
		response.CustomError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", err)
		return
	}

	response.Success(w, http.StatusOK, response.H{
		"admins": admins,
		"total":  total,
		"page":   page,
		"limit":  limit,
	})
}

// CreateAdmin
//
//	@Summary		Create admin
//	@Description	Create a new administrator account
//	@Tags			Admin
//	@Accept			json
//	@Produce		json
//	@Param			admin	body		CreateAdminRequest	true	"Admin fields"
//	@Success		201		{object}	map[string]interface{}
//	@Failure		400		{object}	map[string]interface{}
//	@Failure		500		{object}	map[string]interface{}
//	@Security		BearerAuth
//	@Router			/admin/management/admins [post]
func (h *ManagementHandler) CreateAdmin(w http.ResponseWriter, r *http.Request) {
	var req CreateAdminRequest
	if err := response.BindJSON(r, &req); err != nil {
		response.CustomError(w, r, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	admin, err := h.service.CreateAdmin(r.Context(), req.Email, req.Password, req.Name, req.Role)
	if err != nil {
		response.CustomError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", err)
		return
	}

	response.Success(w, http.StatusCreated, admin)
}

// UpdateAdmin
//
//	@Summary		Update admin
//	@Description	Update an existing administrator account
//	@Tags			Admin
//	@Accept			json
//	@Produce		json
//	@Param			id		path		int					true	"Admin ID"
//	@Param			admin	body		UpdateAdminRequest	true	"Admin fields"
//	@Success		200		{object}	map[string]interface{}
//	@Failure		400		{object}	map[string]interface{}
//	@Failure		500		{object}	map[string]interface{}
//	@Security		BearerAuth
//	@Router			/admin/management/admins/{id} [patch]
func (h *ManagementHandler) UpdateAdmin(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req UpdateAdminRequest
	if err := response.BindJSON(r, &req); err != nil {
		response.CustomError(w, r, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
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

	admin, err := h.service.UpdateAdmin(r.Context(), id, updates)
	if err != nil {
		response.CustomError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", err)
		return
	}

	response.Success(w, http.StatusOK, admin)
}
