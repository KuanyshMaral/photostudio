package auth

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"photostudio/internal/pkg/chicontext"
	"photostudio/internal/pkg/response"
)

// Handler manages all HTTP interactions for authentication
type Handler struct {
	service        *Service
	profileService ProfileService
	bookingReader  BookingStatsReader
	cookieSecure   bool
	cookieSameSite string
	cookiePath     string
}

// NewHandler creates a new auth handler with injected service
func NewHandler(service *Service, profileService ProfileService, bookingReader BookingStatsReader, cookieSecure bool, cookieSameSite, cookiePath string) *Handler {
	return &Handler{
		service:        service,
		profileService: profileService,
		bookingReader:  bookingReader,
		cookieSecure:   cookieSecure,
		cookieSameSite: cookieSameSite,
		cookiePath:     cookiePath,
	}
}

// RequestEmailVerification
//
//	@Summary	Request email verification
//	@Tags		Auth
//
//	@Accept		json
//	@Produce	json
//	@Param		body	body		VerifyRequestDTO	true	"payload"
//	@Success	200		{object}	map[string]interface{}
//	@Router		/auth/verify/request [post]
func (h *Handler) RequestEmailVerification(w http.ResponseWriter, r *http.Request) {
	var req VerifyRequestDTO
	if err := response.BindJSON(r, &req); err != nil {
		response.CustomError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body")
		return
	}

	_, err := h.service.RequestEmailVerification(r.Context(), req.Email)
	if err != nil {
		if err == ErrRateLimitExceeded {
			response.CustomError(w, r, http.StatusTooManyRequests, "RATE_LIMIT_EXCEEDED", "Please wait before requesting a new code")
			return
		}
		response.CustomError(w, r, http.StatusInternalServerError, "VERIFICATION_REQUEST_FAILED", "Failed to process verification request")
		return
	}

	response.Success(w, http.StatusOK, response.H{"status": "accepted"})
}

// ConfirmEmailVerification
//
//	@Summary	Confirm email verification
//	@Tags		Auth
//
//	@Accept		json
//	@Produce	json
//	@Param		body	body		VerifyConfirmDTO	true	"payload"
//	@Success	200		{object}	map[string]interface{}
//	@Router		/auth/verify/confirm [post]
func (h *Handler) ConfirmEmailVerification(w http.ResponseWriter, r *http.Request) {
	var req VerifyConfirmDTO
	if err := response.BindJSON(r, &req); err != nil {
		response.CustomError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body")
		return
	}

	err := h.service.ConfirmEmailVerification(r.Context(), req.Email, req.Code)
	if err != nil {
		switch err {
		case ErrInvalidVerificationCodeFormat:
			response.CustomError(w, r, http.StatusBadRequest, "INVALID_CODE_FORMAT", "Verification code must be exactly 6 digits")
		case ErrTooManyAttempts:
			response.CustomError(w, r, http.StatusTooManyRequests, "TOO_MANY_ATTEMPTS", "Too many invalid verification attempts")
		case ErrInvalidVerificationCode:
			response.CustomError(w, r, http.StatusBadRequest, "INVALID_CODE", "Invalid or expired verification code")
		default:
			response.CustomError(w, r, http.StatusInternalServerError, "VERIFICATION_CONFIRM_FAILED", "Failed to confirm verification code")
		}
		return
	}

	response.Success(w, http.StatusOK, response.H{"status": "verified"})
}

// Refresh
//
//	@Summary	Refresh token
//	@Tags		Auth
//
//	@Produce	json
//	@Success	200	{object}	map[string]interface{}
//	@Router		/auth/refresh [post]
func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("refresh_token")
	if err != nil || strings.TrimSpace(cookie.Value) == "" {
		response.CustomError(w, r, http.StatusUnauthorized, "INVALID_REFRESH_TOKEN", "Refresh token is missing or invalid")
		return
	}

	clientIP := r.RemoteAddr
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		clientIP = strings.Split(xff, ",")[0]
	}

	result, err := h.service.RefreshSession(r.Context(), cookie.Value, r.UserAgent(), clientIP)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidRefreshToken):
			response.CustomError(w, r, http.StatusUnauthorized, "INVALID_REFRESH_TOKEN", "Refresh token is invalid or expired")
		case errors.Is(err, ErrRefreshTokenReused):
			response.CustomError(w, r, http.StatusUnauthorized, "REFRESH_TOKEN_REUSED", "Refresh token reuse detected")
		case errors.Is(err, ErrAccountBanned):
			response.CustomError(w, r, http.StatusForbidden, "ACCOUNT_BANNED", "Account is banned")
		case errors.Is(err, ErrEmailNotVerified):
			response.CustomError(w, r, http.StatusForbidden, "EMAIL_NOT_VERIFIED", "Email must be verified before login")
		default:
			response.CustomError(w, r, http.StatusInternalServerError, "REFRESH_FAILED", "Failed to refresh session")
		}
		return
	}

	h.setRefreshCookie(w, result.RefreshToken)
	response.Success(w, http.StatusOK, response.H{
		"tokens": response.H{"access_token": result.AccessToken},
	})
}

// Logout
//
//	@Summary	Logout
//	@Tags		Auth
//
//	@Success	204	"No Content"
//	@Router		/auth/logout [post]
func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie("refresh_token"); err == nil && strings.TrimSpace(cookie.Value) != "" {
		if logoutErr := h.service.Logout(r.Context(), cookie.Value); logoutErr != nil {
			response.CustomError(w, r, http.StatusInternalServerError, "LOGOUT_FAILED", "Failed to logout")
			return
		}
	}
	h.setRefreshCookie(w, "") // clear
	w.WriteHeader(http.StatusNoContent)
}

// RegisterClient
//
//	@Summary	Register client
//	@Tags		Auth
//
//	@Accept		json
//	@Produce	json
//	@Param		body	body		RegisterClientRequest	true	"payload"
//	@Success	201		{object}	RegisterClientResponseSwagger
//	@Router		/auth/register/client [post]
func (h *Handler) RegisterClient(w http.ResponseWriter, r *http.Request) {
	var req RegisterClientRequest
	if err := response.BindJSON(r, &req); err != nil {
		response.CustomError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body")
		return
	}

	user, verificationSent, err := h.service.RegisterClient(r.Context(), req)
	if err != nil {
		if err == ErrEmailAlreadyExists {
			response.CustomError(w, r, http.StatusConflict, "EMAIL_EXISTS", "This email is already registered")
			return
		}
		response.CustomError(w, r, http.StatusInternalServerError, "REGISTRATION_FAILED", "Failed to register client")
		return
	}

	response.Success(w, http.StatusCreated, response.H{
		"user": response.H{
			"id":            user.ID,
			"email":         user.Email,
			"name":          user.Name,
			"role":          user.Role,
			"phone":         user.Phone,
			"studio_status": user.StudioStatus,
		},
		"verification_sent": verificationSent,
	})
}

// Login
//
//	@Summary	Войти в аккаунт
//	@Tags		Auth
//
//	@Param		request	body		LoginRequest	true	"Учётные данные"
//	@Success	200		{object}	map[string]interface{}
//
//	@Router		/auth/login [post]
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := response.BindJSON(r, &req); err != nil {
		response.CustomError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body")
		return
	}

	clientIP := r.RemoteAddr
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		clientIP = strings.Split(xff, ",")[0]
	}

	loginResult, err := h.service.Login(r.Context(), req, r.UserAgent(), clientIP)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidCredentials):
			response.CustomError(w, r, http.StatusUnauthorized, "INVALID_CREDENTIALS", "Email or password is incorrect")
		case errors.Is(err, ErrEmailNotVerified):
			response.CustomError(w, r, http.StatusForbidden, "EMAIL_NOT_VERIFIED", "Email must be verified before login")
		case errors.Is(err, ErrAccountBanned):
			response.CustomError(w, r, http.StatusForbidden, "ACCOUNT_BANNED", "Account is banned")
		case errors.Is(err, ErrAccountLocked):
			response.CustomError(w, r, http.StatusForbidden, "ACCOUNT_LOCKED", "Account is temporarily locked")
		default:
			response.CustomError(w, r, http.StatusInternalServerError, "LOGIN_FAILED", "Failed to login")
		}
		return
	}

	h.setRefreshCookie(w, loginResult.RefreshToken)
	response.Success(w, http.StatusOK, response.H{
		"user": response.H{
			"id":            loginResult.User.ID,
			"email":         loginResult.User.Email,
			"name":          loginResult.User.Name,
			"role":          loginResult.User.Role,
			"phone":         loginResult.User.Phone,
			"studio_status": loginResult.User.StudioStatus,
		},
		"tokens": response.H{"access_token": loginResult.AccessToken},
	})
}

// GetMe
//
//	@Summary	Получить профиль пользователя
//	@Tags		Auth
//
//	@Security	BearerAuth
//	@Success	200	{object}	map[string]interface{}
//
//	@Router		/users/me [get]
func (h *Handler) GetMe(w http.ResponseWriter, r *http.Request) {
	userID := chicontext.UserIDFromCtx(r.Context())
	if userID == 0 {
		response.CustomError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
		return
	}

	user, err := h.service.GetCurrentUser(r.Context(), userID)
	if err != nil {
		response.CustomError(w, r, http.StatusNotFound, "NOT_FOUND", "User not found")
		return
	}

	var stats *UserStats
	if h.bookingReader != nil {
		if s, err := h.bookingReader.GetStatsByUserID(user.ID); err == nil && s != nil {
			stats = &UserStats{
				TotalBookings:     int(s.Total),
				UpcomingBookings:  int(s.Upcoming),
				CompletedBookings: int(s.Completed),
				CancelledBookings: int(s.Cancelled),
			}
		}
	}

	var profileData interface{}
	if h.profileService != nil {
		switch user.Role {
		case RoleClient:
			profileData, _ = h.profileService.EnsureClientProfile(r.Context(), user.ID)
		case RoleStudioOwner:
			profileData, _ = h.profileService.GetOwnerProfile(r.Context(), user.ID)
		}
	}

	response.JSON(w, http.StatusOK, response.H{
		"user":    user,
		"stats":   stats,
		"profile": profileData,
	})
}

// UpdateProfile
//
//	@Summary	Обновить профиль пользователя
//	@Tags		Auth
//
//	@Security	BearerAuth
//	@Param		request	body		UpdateProfileRequest	true	"Данные для обновления"
//	@Success	200		{object}	map[string]interface{}
//
//	@Router		/users/me [put]
func (h *Handler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	userID := chicontext.UserIDFromCtx(r.Context())

	var req UpdateProfileRequest
	if err := response.BindJSON(r, &req); err != nil {
		response.CustomError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body")
		return
	}

	user, err := h.service.UpdateProfile(r.Context(), userID, req)
	if err != nil {
		response.CustomError(w, r, http.StatusInternalServerError, "UPDATE_FAILED", "Could not update profile")
		return
	}

	response.Success(w, http.StatusOK, response.H{
		"user": response.H{
			"id":    user.ID,
			"name":  user.Name,
			"phone": user.Phone,
			"email": user.Email,
			"role":  user.Role,
		},
	})
}

// UploadVerificationDocuments
//
//	@Summary	Загрузить документы верификации
//	@Tags		Auth
//
//	@Security	BearerAuth
//	@Accept		multipart/form-data
//	@Param		documents	formData	file	true	"Файлы документов"
//	@Success	200			{object}	map[string]interface{}
//
//	@Router		/users/verification/documents [post]
func (h *Handler) UploadVerificationDocuments(w http.ResponseWriter, r *http.Request) {
	userID := chicontext.UserIDFromCtx(r.Context())

	if err := r.ParseMultipartForm(10 << 20); err != nil {
		response.CustomError(w, r, http.StatusBadRequest, "INVALID_FORM", "Failed to parse form")
		return
	}

	files := r.MultipartForm.File["documents"]
	if len(files) == 0 {
		response.CustomError(w, r, http.StatusBadRequest, "NO_FILES", "No files uploaded")
		return
	}

	uploadDir := "./uploads/verification"
	if err := os.MkdirAll(uploadDir, os.ModePerm); err != nil {
		response.CustomError(w, r, http.StatusInternalServerError, "STORAGE_ERROR", "Failed to create upload directory")
		return
	}

	var uploadedURLs []string
	for _, fh := range files {
		filename := fmt.Sprintf("%d_%s", time.Now().UnixNano(), fh.Filename)
		savePath := filepath.Join(uploadDir, filename)

		f, err := fh.Open()
		if err != nil {
			response.CustomError(w, r, http.StatusInternalServerError, "SAVE_FAILED", "Failed to open file")
			return
		}
		out, err := os.Create(savePath)
		if err != nil {
			f.Close()
			response.CustomError(w, r, http.StatusInternalServerError, "SAVE_FAILED", "Failed to save file")
			return
		}
		_, _ = io.Copy(out, f)
		out.Close()
		f.Close()

		uploadedURLs = append(uploadedURLs, "/static/verification/"+filename)
	}

	if err := h.service.AppendVerificationDocs(r.Context(), userID, uploadedURLs); err != nil {
		response.CustomError(w, r, http.StatusInternalServerError, "DB_ERROR", "Failed to save document references")
		return
	}

	response.Success(w, http.StatusOK, response.H{
		"message":       "Documents uploaded successfully",
		"uploaded_urls": uploadedURLs,
	})
}

// ─── Cookie helpers ────────────────────────────────────────────────────────────

func (h *Handler) setRefreshCookie(w http.ResponseWriter, value string) {
	maxAge := 604800
	if value == "" {
		maxAge = 0
	}
	cookie := &http.Cookie{
		Name:     "refresh_token",
		Value:    value,
		MaxAge:   maxAge,
		Path:     h.cookiePath,
		HttpOnly: true,
		Secure:   h.cookieSecure,
		SameSite: parseSameSite(h.cookieSameSite),
	}
	http.SetCookie(w, cookie)
}

func parseSameSite(mode string) http.SameSite {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "strict":
		return http.SameSiteStrictMode
	case "none":
		return http.SameSiteNoneMode
	default:
		return http.SameSiteLaxMode
	}
}
