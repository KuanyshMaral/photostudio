package auth

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"photostudio/internal/pkg/response"

	"github.com/gin-gonic/gin"
)

// Handler manages all HTTP interactions for authentication
type Handler struct {
	service       *Service
	bookingReader BookingStatsReader
}

// NewHandler creates a new auth handler with injected service
func NewHandler(service *Service, bookingReader BookingStatsReader) *Handler {
	return &Handler{
		service:       service,
		bookingReader: bookingReader,
	}
}

func (h *Handler) RegisterPublicRoutes(v1 *gin.RouterGroup) {
	authGroup := v1.Group("/auth")
	{
		authGroup.POST("/register/client", h.RegisterClient)
		authGroup.POST("/register/studio", h.RegisterStudioOwner)
		authGroup.POST("/login", h.Login)
		authGroup.POST("/refresh", h.Refresh)
		authGroup.POST("/logout", h.Logout)
	}
}

func (h *Handler) RegisterProtectedRoutes(protected *gin.RouterGroup) {
	userGroup := protected.Group("/users")
	{
		userGroup.GET("/me", h.GetMe)
		userGroup.PUT("/me", h.UpdateProfile)
		userGroup.POST("/verification/documents", h.UploadVerificationDocuments)
	}
}

// RegisterClient — POST /auth/register/client
func (h *Handler) RegisterClient(c *gin.Context) {
	var req RegisterClientRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body")
		return
	}

	user, tokens, err := h.service.RegisterClient(c.Request.Context(), req)
	if err != nil {
		if err == ErrEmailAlreadyExists {
			response.Error(c, http.StatusConflict, "EMAIL_EXISTS", "This email is already registered")
			return
		}
		response.Error(c, http.StatusInternalServerError, "REGISTRATION_FAILED", "Failed to register client")
		return
	}

	h.setRefreshCookie(c, tokens.RefreshToken)
	response.Success(c, http.StatusCreated, gin.H{
		"user": gin.H{
			"id":            user.ID,
			"email":         user.Email,
			"name":          user.Name,
			"role":          user.Role,
			"phone":         user.Phone,
			"studio_status": user.StudioStatus,
		},
		"tokens": tokens,
	})
}

// RegisterStudioOwner — POST /auth/register/studio
func (h *Handler) RegisterStudioOwner(c *gin.Context) {
	var req RegisterStudioRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body")
		return
	}

	user, tokens, err := h.service.RegisterStudioOwner(c.Request.Context(), req)
	if err != nil {
		if err == ErrEmailAlreadyExists {
			response.Error(c, http.StatusConflict, "EMAIL_EXISTS", "This email is already registered")
			return
		}
		response.Error(c, http.StatusInternalServerError, "REGISTRATION_FAILED", "Failed to register studio owner")
		return
	}

	h.setRefreshCookie(c, tokens.RefreshToken)
	response.Success(c, http.StatusCreated, gin.H{
		"user": gin.H{
			"id":            user.ID,
			"email":         user.Email,
			"name":          user.Name,
			"role":          user.Role,
			"phone":         user.Phone,
			"studio_status": user.StudioStatus,
		},
		"tokens": tokens,
	})
}

// Login — POST /auth/login
func (h *Handler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body")
		return
	}

	user, tokens, err := h.service.Login(c.Request.Context(), req)
	if err != nil {
		if errors.Is(err, ErrInvalidCredentials) {
			response.Error(c, http.StatusUnauthorized, "INVALID_CREDENTIALS", "Email or password is incorrect")
			return
		}
		response.Error(c, http.StatusInternalServerError, "LOGIN_FAILED", "Failed to login")
		return
	}

	h.setRefreshCookie(c, tokens.RefreshToken)
	response.Success(c, http.StatusOK, gin.H{
		"user": gin.H{
			"id":            user.ID,
			"email":         user.Email,
			"name":          user.Name,
			"role":          user.Role,
			"phone":         user.Phone,
			"studio_status": user.StudioStatus,
		},
		"tokens": tokens,
	})
}

// Refresh — POST /auth/refresh
func (h *Handler) Refresh(c *gin.Context) {
	refresh, err := c.Cookie("refresh_token")
	if err != nil || refresh == "" {
		response.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "Refresh token is missing")
		return
	}

	user, tokens, err := h.service.Refresh(c.Request.Context(), refresh)
	if err != nil {
		response.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "Invalid refresh token")
		return
	}

	h.setRefreshCookie(c, tokens.RefreshToken)
	response.Success(c, http.StatusOK, gin.H{
		"user": gin.H{
			"id":            user.ID,
			"email":         user.Email,
			"name":          user.Name,
			"role":          user.Role,
			"phone":         user.Phone,
			"studio_status": user.StudioStatus,
		},
		"tokens": tokens,
	})
}

// Logout — POST /auth/logout
func (h *Handler) Logout(c *gin.Context) {
	refresh, _ := c.Cookie("refresh_token")
	_ = h.service.Logout(c.Request.Context(), refresh)

	// clear cookie
	h.clearRefreshCookie(c)
	response.Success(c, http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) setRefreshCookie(c *gin.Context, token string) {
	if token == "" {
		return
	}

	// Cookie settings
	secure := os.Getenv("ENV") == "production" || os.Getenv("COOKIE_SECURE") == "true"
	domain := os.Getenv("COOKIE_DOMAIN")
	path := "/" // allow refresh from any endpoint

	// Lax is a sane default for SPA + same-site requests; if you do cross-site, use None + Secure.
	sameSite := http.SameSiteLaxMode
	if strings.EqualFold(os.Getenv("COOKIE_SAMESITE"), "none") {
		sameSite = http.SameSiteNoneMode
	}
	c.SetSameSite(sameSite)

	maxAge := h.service.RefreshTTLSeconds()
	c.SetCookie("refresh_token", token, maxAge, path, domain, secure, true)
}

func (h *Handler) clearRefreshCookie(c *gin.Context) {
	secure := os.Getenv("ENV") == "production" || os.Getenv("COOKIE_SECURE") == "true"
	domain := os.Getenv("COOKIE_DOMAIN")
	path := "/"
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie("refresh_token", "", -1, path, domain, secure, true)
}

// GetMe — GET /users/me (protected)
// GetMe — GET /users/me (protected)
func (h *Handler) GetMe(c *gin.Context) {
	userIDAny, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	userID := userIDAny.(int64)

	user, err := h.service.GetCurrentUser(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	includeStats := c.Query("include_stats") == "true"

	profile := UserProfileResponse{
		ID:        user.ID,
		Email:     user.Email,
		Name:      user.Name,
		Phone:     user.Phone,
		Role:      string(user.Role),
		AvatarURL: user.AvatarURL,
		CreatedAt: user.CreatedAt.Format("2006-01-02"),
	}

	if includeStats && h.bookingReader != nil {
		stats, err := h.bookingReader.GetStatsByUserID(userID)
		if err == nil && stats != nil {
			profile.Stats = &UserStats{
				TotalBookings:     int(stats.Total),
				UpcomingBookings:  int(stats.Upcoming),
				CompletedBookings: int(stats.Completed),
				CancelledBookings: int(stats.Cancelled),
			}
		}

		recent, err := h.bookingReader.GetRecentByUserID(userID, 3)
		if err == nil {
			profile.RecentBookings = make([]RecentBooking, 0, len(recent))
			for _, r := range recent {
				profile.RecentBookings = append(profile.RecentBookings, RecentBooking{
					ID:         r.ID,
					StudioName: r.StudioName,
					RoomName:   r.RoomName,
					Date:       r.StartTime.Format("02.01.2006"),
					Status:     r.Status,
				})
			}
		}
	}

	// Сохраняем стиль проекта (response.Success)
	response.Success(c, http.StatusOK, gin.H{
		"user": profile,
	})
}

// UpdateProfile — PUT /users/me (protected)
func (h *Handler) UpdateProfile(c *gin.Context) {
	userID, _ := c.Get("user_id")

	var req UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body")
		return
	}

	user, err := h.service.UpdateProfile(c.Request.Context(), userID.(int64), req)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "UPDATE_FAILED", "Could not update profile")
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

// UploadVerificationDocuments — POST /users/verification/documents (protected)
func (h *Handler) UploadVerificationDocuments(c *gin.Context) {
	userID, _ := c.Get("user_id")

	// Parse multipart form (max 10MB)
	if err := c.Request.ParseMultipartForm(10 << 20); err != nil {
		response.Error(c, http.StatusBadRequest, "INVALID_FORM", "Failed to parse form")
		return
	}

	files := c.Request.MultipartForm.File["documents"]
	if len(files) == 0 {
		response.Error(c, http.StatusBadRequest, "NO_FILES", "No files uploaded")
		return
	}

	uploadDir := "./uploads/verification"
	if err := os.MkdirAll(uploadDir, os.ModePerm); err != nil {
		response.Error(c, http.StatusInternalServerError, "STORAGE_ERROR", "Failed to create upload directory")
		return
	}

	var uploadedURLs []string
	for _, file := range files {
		// Generate unique filename
		filename := fmt.Sprintf("%d_%s", time.Now().UnixNano(), file.Filename)
		savePath := filepath.Join(uploadDir, filename)

		if err := c.SaveUploadedFile(file, savePath); err != nil {
			response.Error(c, http.StatusInternalServerError, "SAVE_FAILED", "Failed to save file")
			return
		}

		// URL that will be served via static route
		url := "/static/verification/" + filename
		uploadedURLs = append(uploadedURLs, url)
	}

	// Save URLs to database
	if err := h.service.AppendVerificationDocs(c.Request.Context(), userID.(int64), uploadedURLs); err != nil {
		response.Error(c, http.StatusInternalServerError, "DB_ERROR", "Failed to save document references")
		return
	}

	response.Success(c, http.StatusOK, gin.H{
		"message":       "Documents uploaded successfully",
		"uploaded_urls": uploadedURLs,
	})
}
