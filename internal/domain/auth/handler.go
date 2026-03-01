package auth

import (
	"errors"
	"net/http"
	"strings"

	"photostudio/internal/pkg/response"
	"photostudio/internal/pkg/utils"
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
//	@Param		body	body		VerifyRequestDTO		true	"payload"
//	@Success	200		{object}	swaggerVerifyResponse	"Подтверждение отправки"
//	@Failure	400		{object}	response.ErrorResponse	"Ошибка валидации"
//	@Failure	429		{object}	response.ErrorResponse	"Слишком много запросов"
//	@Failure	500		{object}	response.ErrorResponse	"Ошибка сервера"
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
//	@Param		body	body		VerifyConfirmDTO		true	"payload"
//	@Success	200		{object}	swaggerVerifyResponse	"Успешное подтверждение"
//	@Failure	400		{object}	response.ErrorResponse	"Ошибка валидации или неверный код"
//	@Failure	429		{object}	response.ErrorResponse	"Слишком много запросов"
//	@Failure	500		{object}	response.ErrorResponse	"Ошибка сервера"
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
//	@Summary	Обновление токенов
//	@Description	Обновляет access/refresh токены по валидному refresh token.
//	@Tags		Auth
//
//	@Accept		json
//	@Produce	json
//	@Param		request	body		RefreshTokenRequest		true	"Refresh token"
//	@Success	200		{object}	swaggerTokensResponse	"Токены обновлены"
//	@Failure	401		{object}	response.ErrorResponse	"Неавторизован или токен не валиден"
//	@Failure	403		{object}	response.ErrorResponse	"Доступ запрещен"
//	@Failure	500		{object}	response.ErrorResponse	"Ошибка сервера"
//	@Router		/auth/refresh [post]
func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req RefreshTokenRequest
	if err := response.BindJSON(r, &req); err != nil || strings.TrimSpace(req.RefreshToken) == "" {
		response.CustomError(w, r, http.StatusUnauthorized, "INVALID_REFRESH_TOKEN", "Refresh token is missing or invalid")
		return
	}

	clientIP := utils.GetClientIP(r)

	result, err := h.service.RefreshSession(r.Context(), req.RefreshToken, r.UserAgent(), clientIP)
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

	response.Success(w, http.StatusOK, response.H{
		"tokens": response.H{
			"access_token":  result.AccessToken,
			"refresh_token": result.RefreshToken,
		},
	})
}

// Logout
//
//	@Summary	Logout
//	@Tags		Auth
//
//	@Accept		json
//	@Param		request	body	RefreshTokenRequest	false	"Refresh token to revoke"
//	@Success	204		"No Content"
//	@Failure	500		{object}	response.ErrorResponse	"Ошибка сервера"
//	@Router		/auth/logout [post]
func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	var req RefreshTokenRequest
	// Best-effort: parse body but don't fail if missing
	_ = response.BindJSON(r, &req)
	if strings.TrimSpace(req.RefreshToken) != "" {
		if logoutErr := h.service.Logout(r.Context(), req.RefreshToken); logoutErr != nil {
			response.CustomError(w, r, http.StatusInternalServerError, "LOGOUT_FAILED", "Failed to logout")
			return
		}
	}
	w.WriteHeader(http.StatusNoContent)
}

// RegisterClient
//
//	@Summary	Register client
//	@Tags		Auth
//
//	@Accept		json
//	@Produce	json
//	@Param		body	body		RegisterClientRequest			true	"payload (email, password)"
//	@Success	201		{object}	swaggerRegisterClientResponse	"Успешная регистрация"
//	@Failure	400		{object}	response.ErrorResponse			"Ошибка валидации"
//	@Failure	409		{object}	response.ErrorResponse			"Email уже существует"
//	@Failure	500		{object}	response.ErrorResponse			"Ошибка сервера"
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
			"role":          user.Role,
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
//	@Param		request	body		LoginRequest			true	"Учётные данные"
//	@Success	200		{object}	swaggerLoginResponse	"Успешный вход"
//	@Failure	400		{object}	response.ErrorResponse	"Ошибка валидации"
//	@Failure	401		{object}	response.ErrorResponse	"Неверные учетные данные"
//	@Failure	403		{object}	response.ErrorResponse	"Доступ запрещен"
//	@Failure	500		{object}	response.ErrorResponse	"Ошибка сервера"
//
//	@Router		/auth/login [post]
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := response.BindJSON(r, &req); err != nil {
		response.CustomError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body")
		return
	}

	clientIP := utils.GetClientIP(r)

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

	h.setRefreshCookie(w, loginResult.RefreshToken) // keep cookie for backward compat during transition
	response.Success(w, http.StatusOK, response.H{
		"user": response.H{
			"id":            loginResult.User.ID,
			"email":         loginResult.User.Email,
			"role":          loginResult.User.Role,
			"studio_status": loginResult.User.StudioStatus,
		},
		"tokens": response.H{
			"access_token":  loginResult.AccessToken,
			"refresh_token": loginResult.RefreshToken,
		},
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
