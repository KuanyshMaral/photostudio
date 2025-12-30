package auth

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
	"photostudio/internal/pkg/response"
)

// Handler manages all HTTP interactions for authentication
type Handler struct {
	service *Service
}

// NewHandler creates a new auth handler with injected service
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterPublicRoutes(v1 *gin.RouterGroup) {
	authGroup := v1.Group("/auth")
	{
		authGroup.POST("/register/client", h.RegisterClient)
		authGroup.POST("/register/studio", h.RegisterStudioOwner)
		authGroup.POST("/login", h.Login)
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

	user, token, err := h.service.RegisterClient(c.Request.Context(), req)
	if err != nil {
		if err == ErrEmailAlreadyExists {
			response.Error(c, http.StatusConflict, "EMAIL_EXISTS", "This email is already registered")
			return
		}
		response.Error(c, http.StatusInternalServerError, "REGISTRATION_FAILED", "Failed to register client")
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
		"token": token,
	})
}

// RegisterStudioOwner — POST /auth/register/studio
func (h *Handler) RegisterStudioOwner(c *gin.Context) {
	var req RegisterStudioRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body")
		return
	}

	user, token, err := h.service.RegisterStudioOwner(c.Request.Context(), req)
	if err != nil {
		if err == ErrEmailAlreadyExists {
			response.Error(c, http.StatusConflict, "EMAIL_EXISTS", "This email is already registered")
			return
		}
		response.Error(c, http.StatusInternalServerError, "REGISTRATION_FAILED", "Failed to register studio owner")
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
		"token": token,
	})
}

// Login — POST /auth/login
func (h *Handler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body")
		return
	}

	user, token, err := h.service.Login(c.Request.Context(), req)
	if err != nil {
		if errors.Is(err, ErrInvalidCredentials) {
			response.Error(c, http.StatusUnauthorized, "INVALID_CREDENTIALS", "Email or password is incorrect")
			return
		}
		response.Error(c, http.StatusInternalServerError, "LOGIN_FAILED", "Failed to login")
		return
	}

	response.Success(c, http.StatusOK, gin.H{
		"user": gin.H{
			"id":            user.ID,
			"email":         user.Email,
			"name":          user.Name,
			"role":          user.Role,
			"phone":         user.Phone,
			"studio_status": user.StudioStatus,
		},
		"token": token,
	})
}

// GetMe — GET /users/me (protected)
func (h *Handler) GetMe(c *gin.Context) {
	userID, _ := c.Get("user_id")

	fmt.Println("User id: ", userID)

	user, err := h.service.GetCurrentUser(c.Request.Context(), userID.(int64))
	if err != nil {
		response.Error(c, http.StatusNotFound, "USER_NOT_FOUND", "User not found")
		return
	}

	response.Success(c, http.StatusOK, gin.H{
		"user": gin.H{
			"id":             user.ID,
			"email":          user.Email,
			"name":           user.Name,
			"phone":          user.Phone,
			"role":           user.Role,
			"email_verified": user.EmailVerified,
			"studio_status":  user.StudioStatus,
			"created_at":     user.CreatedAt,
		},
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
