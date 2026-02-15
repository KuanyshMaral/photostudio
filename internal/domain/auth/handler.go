package auth

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"photostudio/internal/pkg/response"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
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

// RequestEmailVerification запрашивает код подтверждения.
// @Summary		Request email verification
// @Description	Sends a verification code to the specified email.
// @Tags		Auth
// @Accept		json
// @Produce		json
// @Param		body	body	VerifyRequestDTO	true	"payload"
// @Success		200	{object}		map[string]interface{}
// @Router		/auth/verify/request [post]
func (h *Handler) RequestEmailVerification(c *gin.Context) {
	var req VerifyRequestDTO
	if err := c.ShouldBindJSON(&req); err != nil {
		response.CustomError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body")
		return
	}

	_, err := h.service.RequestEmailVerification(c.Request.Context(), req.Email)
	if err != nil {
		if err == ErrRateLimitExceeded {
			response.CustomError(c, http.StatusTooManyRequests, "RATE_LIMIT_EXCEEDED", "Please wait before requesting a new code")
			return
		}
		response.CustomError(c, http.StatusInternalServerError, "VERIFICATION_REQUEST_FAILED", "Failed to process verification request")
		return
	}

	response.Success(c, http.StatusOK, gin.H{
		"status": "accepted",
	})
}

// ConfirmEmailVerification подтверждает email по коду.
// @Summary		Confirm email verification
// @Description	Verifies the email using the provided 6-digit code.
// @Tags		Auth
// @Accept		json
// @Produce		json
// @Param		body	body	VerifyConfirmDTO	true	"payload"
// @Success		200	{object}		map[string]interface{}
// @Router		/auth/verify/confirm [post]
func (h *Handler) ConfirmEmailVerification(c *gin.Context) {
	var req VerifyConfirmDTO
	if err := c.ShouldBindJSON(&req); err != nil {
		response.CustomError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body")
		return
	}

	err := h.service.ConfirmEmailVerification(c.Request.Context(), req.Email, req.Code)
	if err != nil {
		switch err {
		case ErrInvalidVerificationCodeFormat:
			response.CustomError(c, http.StatusBadRequest, "INVALID_CODE_FORMAT", "Verification code must be exactly 6 digits")
		case ErrTooManyAttempts:
			response.CustomError(c, http.StatusTooManyRequests, "TOO_MANY_ATTEMPTS", "Too many invalid verification attempts")
		case ErrInvalidVerificationCode:
			response.CustomError(c, http.StatusBadRequest, "INVALID_CODE", "Invalid or expired verification code")
		default:
			response.CustomError(c, http.StatusInternalServerError, "VERIFICATION_CONFIRM_FAILED", "Failed to confirm verification code")
		}
		return
	}

	response.Success(c, http.StatusOK, gin.H{
		"status": "verified",
	})
}

// Refresh обновляет access токен по refresh токену из куки.
// @Summary		Refresh token
// @Description	Refreshes the session using the refresh token stored in the cookie.
// @Tags		Auth
// @Produce		json
// @Success		200	{object}		map[string]interface{}
// @Router		/auth/refresh [post]
func (h *Handler) Refresh(c *gin.Context) {
	refreshRaw, err := c.Cookie("refresh_token")
	if err != nil || strings.TrimSpace(refreshRaw) == "" {
		response.CustomError(c, http.StatusUnauthorized, "INVALID_REFRESH_TOKEN", "Refresh token is missing or invalid")
		return
	}

	result, err := h.service.RefreshSession(c.Request.Context(), refreshRaw, c.Request.UserAgent(), c.ClientIP())
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidRefreshToken):
			response.CustomError(c, http.StatusUnauthorized, "INVALID_REFRESH_TOKEN", "Refresh token is invalid or expired")
		case errors.Is(err, ErrRefreshTokenReused):
			response.CustomError(c, http.StatusUnauthorized, "REFRESH_TOKEN_REUSED", "Refresh token reuse detected")
		case errors.Is(err, ErrAccountBanned):
			response.CustomError(c, http.StatusForbidden, "ACCOUNT_BANNED", "Account is banned")
		case errors.Is(err, ErrEmailNotVerified):
			response.CustomError(c, http.StatusForbidden, "EMAIL_NOT_VERIFIED", "Email must be verified before login")
		default:
			response.CustomError(c, http.StatusInternalServerError, "REFRESH_FAILED", "Failed to refresh session")
		}
		return
	}

	c.SetSameSite(parseSameSite(h.cookieSameSite))
	c.SetCookie("refresh_token", result.RefreshToken, 604800, h.cookiePath, "", h.cookieSecure, true)

	response.Success(c, http.StatusOK, gin.H{
		"tokens": gin.H{
			"access_token": result.AccessToken,
		},
	})
}

// Logout завершает сессию пользователя.
// @Summary		Logout
// @Description	Invalidates the current session and clears the refresh token cookie.
// @Tags		Auth
// @Success		204	"No Content"
// @Router		/auth/logout [post]
func (h *Handler) Logout(c *gin.Context) {
	refreshRaw, err := c.Cookie("refresh_token")
	if err == nil && strings.TrimSpace(refreshRaw) != "" {
		if logoutErr := h.service.Logout(c.Request.Context(), refreshRaw); logoutErr != nil {
			response.CustomError(c, http.StatusInternalServerError, "LOGOUT_FAILED", "Failed to logout")
			return
		}
	}

	c.SetSameSite(parseSameSite(h.cookieSameSite))
	c.SetCookie("refresh_token", "", 0, h.cookiePath, "", h.cookieSecure, true)
	c.Status(http.StatusNoContent)
}

// RegisterClient регистрирует нового клиента на платформе.
// @Summary		Register client
// @Description	Creates a new client account and returns registration payload with user data and verification flag.
// @Tags		Auth
// @Accept		json
// @Produce		json
// @Param		body	body	RegisterClientRequest	true	"payload"
// @Success		201	{object}		RegisterClientResponseSwagger
// @Failure		400	{object}		ErrorResponseSwagger
// @Failure		409	{object}		ErrorResponseSwagger
// @Failure		500	{object}		ErrorResponseSwagger
// @Router		/auth/register/client [post]
func (h *Handler) RegisterClient(c *gin.Context) {
	var req RegisterClientRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.CustomError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body")
		return
	}

	user, verificationSent, err := h.service.RegisterClient(c.Request.Context(), req)
	if err != nil {
		if err == ErrEmailAlreadyExists {
			response.CustomError(c, http.StatusConflict, "EMAIL_EXISTS", "This email is already registered")
			return
		}
		response.CustomError(c, http.StatusInternalServerError, "REGISTRATION_FAILED", "Failed to register client")
		return
	}

	response.Success(c, http.StatusCreated, gin.H{
		"user": gin.H{
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

// Login авторизует пользователя на платформе и выдаёт JWT токен.
// @Summary		Войти в аккаунт
// @Description	Авторизует пользователя (клиента или владельца студии) по email и паролю. Возвращает JWT токен для последующих запросов к защищённым эндпоинтам.
// @Tags		Auth
// @Param		request	body	LoginRequest		true	"Учётные данные (email, password)"
// @Success		200	{object}		map[string]interface{} "Успешная авторизация, возвращается JWT токен"
// @Failure		400	{object}		map[string]interface{} "Ошибка валидации: неверный формат данных"
// @Failure		401	{object}		map[string]interface{} "Ошибка: неверный email или пароль"
// @Failure		500	{object}		map[string]interface{} "Ошибка сервера при авторизации"
// @Router		/auth/login [POST]
func (h *Handler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.CustomError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body")
		return
	}

	loginResult, err := h.service.Login(c.Request.Context(), req, c.Request.UserAgent(), c.ClientIP())
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidCredentials):
			response.CustomError(c, http.StatusUnauthorized, "INVALID_CREDENTIALS", "Email or password is incorrect")
		case errors.Is(err, ErrEmailNotVerified):
			response.CustomError(c, http.StatusForbidden, "EMAIL_NOT_VERIFIED", "Email must be verified before login")
		case errors.Is(err, ErrAccountBanned):
			response.CustomError(c, http.StatusForbidden, "ACCOUNT_BANNED", "Account is banned")
		case errors.Is(err, ErrAccountLocked):
			response.CustomError(c, http.StatusForbidden, "ACCOUNT_LOCKED", "Account is temporarily locked")
		default:
			response.CustomError(c, http.StatusInternalServerError, "LOGIN_FAILED", "Failed to login")
		}
		return
	}

	c.SetSameSite(parseSameSite(h.cookieSameSite))
	c.SetCookie("refresh_token", loginResult.RefreshToken, 604800, h.cookiePath, "", h.cookieSecure, true)

	response.Success(c, http.StatusOK, gin.H{
		"user": gin.H{
			"id":            loginResult.User.ID,
			"email":         loginResult.User.Email,
			"name":          loginResult.User.Name,
			"role":          loginResult.User.Role,
			"phone":         loginResult.User.Phone,
			"studio_status": loginResult.User.StudioStatus,
		},
		"tokens": gin.H{
			"access_token": loginResult.AccessToken,
		},
	})
}

// GetMe получает профиль текущего авторизованного пользователя.
// @Summary		Получить профиль пользователя
// @Description	Возвращает полный профиль текущего авторизованного пользователя (клиента или владельца). Может включать статистику бронирований и недавние брони. При include_stats=true добавляет количество бронирований.
// @Tags		Auth
// @Security	BearerAuth
// @Param		include_stats	query	boolean	false	"Включить статистику бронирований (true/false)"
// @Success		200	{object}		map[string]interface{} "Профиль пользователя с информацией и статистикой"
// @Failure		401	{object}		map[string]interface{} "Ошибка аутентификации: токен не предоставлен или истёк"
// @Failure		404	{object}		map[string]interface{} "Ошибка: пользователь не найден"
// @Failure		500	{object}		map[string]interface{} "Ошибка сервера при получении профиля"
// @Router		/users/me [GET]
func (h *Handler) GetMe(c *gin.Context) {
	userIDAny, exists := c.Get("user_id")
	if !exists {
		response.CustomError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
		return
	}
	userID := userIDAny.(int64)

	user, err := h.service.GetCurrentUser(c.Request.Context(), userID)
	if err != nil {
		response.CustomError(c, http.StatusNotFound, "NOT_FOUND", "User not found")
		return
	}

	// Fetch user stats
	var stats *UserStats
	if h.bookingReader != nil {
		s, err := h.bookingReader.GetStatsByUserID(user.ID)
		if err == nil && s != nil {
			stats = &UserStats{
				TotalBookings:     int(s.Total),
				UpcomingBookings:  int(s.Upcoming),
				CompletedBookings: int(s.Completed),
				CancelledBookings: int(s.Cancelled),
			}
		}
	}

	// Fetch profile data
	var profileData interface{}
	if h.profileService != nil {
		switch user.Role {
		case RoleClient:
			profileData, _ = h.profileService.EnsureClientProfile(c.Request.Context(), user.ID)
		case RoleStudioOwner:
			// Note: Lead conversion handles initial owner profile, but EnsureOwnerProfile
			// can be used here as a fallback if we decide on a default structure.
			// For now, only client and admin (if requested) are auto-created from scratch.
			profileData, _ = h.profileService.GetOwnerProfile(c.Request.Context(), user.ID)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"user":    user,
		"stats":   stats,
		"profile": profileData,
	})
}

// UpdateProfile обновляет информацию профиля текущего пользователя.
// @Summary		Обновить профиль пользователя
// @Description	Обновляет информацию о профиле: имя, телефон и другие поля. Email не может быть изменён через этот эндпоинт. Требуется аутентификация.
// @Tags		Auth
// @Security	BearerAuth
// @Param		request	body	UpdateProfileRequest	true	"Данные для обновления (name, phone, avatar_url, etc.)"
// @Success		200	{object}		map[string]interface{} "Профиль успешно обновлён"
// @Failure		400	{object}		map[string]interface{} "Ошибка валидации: неверный формат данных"
// @Failure		401	{object}		map[string]interface{} "Ошибка аутентификации: требуется токен"
// @Failure		500	{object}		map[string]interface{} "Ошибка сервера при обновлении профиля"
// @Router		/users/me [PUT]
func (h *Handler) UpdateProfile(c *gin.Context) {
	userID, _ := c.Get("user_id")

	var req UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.CustomError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body")
		return
	}

	user, err := h.service.UpdateProfile(c.Request.Context(), userID.(int64), req)
	if err != nil {
		response.CustomError(c, http.StatusInternalServerError, "UPDATE_FAILED", "Could not update profile")
		return
	}

	response.Success(c, http.StatusOK, gin.H{
		"user": gin.H{
			"id":    user.ID,
			"name":  user.Name,
			"phone": user.Phone,
			"email": user.Email,
			"role":  user.Role,
		},
	})
}

// UploadVerificationDocuments загружает документы для верификации владельца студии.
// @Summary		Загрузить документы верификации
// @Description	Загружает документы (паспорт, свидетельство о регистрации, и т.д.) для верификации владельца студии. Документы необходимы для одобрения заявки администратором. Максимальный размер файла 10MB.
// @Tags		Auth
// @Security	BearerAuth
// @Accept		multipart/form-data
// @Param		documents	formData	file		true	"Файлы документов для загрузки (несколько файлов допускаются)"
// @Success		200	{object}		map[string]interface{} "Документы загружены успешно, возвращены URL для доступа"
// @Failure		400	{object}		map[string]interface{} "Ошибка: отсутствуют файлы или неверный формат запроса"
// @Failure		401	{object}		map[string]interface{} "Ошибка аутентификации: требуется токен"
// @Failure		413	{object}		map[string]interface{} "Ошибка: файл слишком большой (макс 10MB)"
// @Failure		500	{object}		map[string]interface{} "Ошибка сервера при загрузке документов"
// @Router		/users/verification/documents [POST]
func (h *Handler) UploadVerificationDocuments(c *gin.Context) {
	userID, _ := c.Get("user_id")

	// Parse multipart form (max 10MB)
	if err := c.Request.ParseMultipartForm(10 << 20); err != nil {
		response.CustomError(c, http.StatusBadRequest, "INVALID_FORM", "Failed to parse form")
		return
	}

	files := c.Request.MultipartForm.File["documents"]
	if len(files) == 0 {
		response.CustomError(c, http.StatusBadRequest, "NO_FILES", "No files uploaded")
		return
	}

	uploadDir := "./uploads/verification"
	if err := os.MkdirAll(uploadDir, os.ModePerm); err != nil {
		response.CustomError(c, http.StatusInternalServerError, "STORAGE_ERROR", "Failed to create upload directory")
		return
	}

	var uploadedURLs []string
	for _, file := range files {
		// Generate unique filename
		filename := fmt.Sprintf("%d_%s", time.Now().UnixNano(), file.Filename)
		savePath := filepath.Join(uploadDir, filename)

		if err := c.SaveUploadedFile(file, savePath); err != nil {
			response.CustomError(c, http.StatusInternalServerError, "SAVE_FAILED", "Failed to save file")
			return
		}

		// URL that will be served via static route
		url := "/static/verification/" + filename
		uploadedURLs = append(uploadedURLs, url)
	}

	// Save URLs to database
	if err := h.service.AppendVerificationDocs(c.Request.Context(), userID.(int64), uploadedURLs); err != nil {
		response.CustomError(c, http.StatusInternalServerError, "DB_ERROR", "Failed to save document references")
		return
	}

	response.Success(c, http.StatusOK, gin.H{
		"message":       "Documents uploaded successfully",
		"uploaded_urls": uploadedURLs,
	})
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
