package admin

import (
	"net/http"
	"strings"

	"photostudio/internal/pkg/chicontext"
	"photostudio/internal/pkg/response"
)

type AuthHandler struct {
	service *Service
}

func NewAuthHandler(service *Service) *AuthHandler {
	return &AuthHandler{service: service}
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// Login
//
//	@Summary		Admin login
//	@Description	Authenticate as an administrator
//	@Tags			Admin
//	@Accept			json
//	@Produce		json
//	@Param			credentials	body		LoginRequest	true	"Admin credentials"
//	@Success		200			{object}	map[string]interface{}
//	@Failure		400			{object}	map[string]interface{}
//	@Failure		401			{object}	map[string]interface{}
//	@Router			/admin/auth/login [post]
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := response.BindJSON(r, &req); err != nil {
		response.CustomError(w, r, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	token, admin, err := h.service.Login(r.Context(), strings.ToLower(req.Email), req.Password)
	if err != nil {
		response.CustomError(w, r, http.StatusUnauthorized, "AUTH_FAILED", "Invalid email or password")
		return
	}

	response.Success(w, http.StatusOK, response.H{
		"access_token": token,
		"admin":        admin,
	})
}

// GetMe
//
//	@Summary		Get current admin
//	@Description	Get info about the currently logged in admin
//	@Tags			Admin
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	map[string]interface{}
//	@Failure		401	{object}	map[string]interface{}
//	@Failure		404	{object}	map[string]interface{}
//	@Security		BearerAuth
//	@Router			/admin/auth/me [get]
func (h *AuthHandler) GetMe(w http.ResponseWriter, r *http.Request) {
	adminID := chicontext.AdminIDFromCtx(r.Context())
	if adminID == "" {
		response.CustomError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
		return
	}

	admin, err := h.service.GetAdminByID(r.Context(), adminID)
	if err != nil {
		response.CustomError(w, r, http.StatusNotFound, "NOT_FOUND", "Admin not found")
		return
	}

	admin.PasswordHash = ""
	response.Success(w, http.StatusOK, admin)
}
