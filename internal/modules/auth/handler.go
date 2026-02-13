package auth

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
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

// RegisterClient регистрирует нового клиента на платформе.
// @Summary		Зарегистрировать клиента
// @Description	Создаёт новый аккаунт клиента на платформе. Клиент получает возможность искать студии и делать бронирования. Автоматически генерируется JWT токен для сессии.
// @Tags		Автентификация
// @Param		request	body	RegisterClientRequest	true	"Данные для регистрации (email, password, name, phone)"
// @Success		201	{object}		map[string]interface{} "Клиент успешно зарегистрирован, возвращается JWT токен"
// @Failure		400	{object}		map[string]interface{} "Ошибка валидации: неверный формат данных"
// @Failure		409	{object}		map[string]interface{} "Ошибка: email уже зарегистрирован на платформе"
// @Failure		500	{object}		map[string]interface{} "Ошибка сервера при создании аккаунта"
// @Router		/auth/register/client [POST]
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

// RegisterStudioOwner регистрирует нового владельца студии на платформе.
// @Summary		Зарегистрировать владельца студии
// @Description	Создаёт новый аккаунт владельца студии. После регистрации требуется модерация администратором перед открытием студии. Владелец получает токен для доступа к личному кабинету.
// @Tags		Автентификация
// @Param		request	body	RegisterStudioRequest	true	"Данные для регистрации владельца (email, password, name, phone, studio_name, description, etc.)"
// @Success		201	{object}		map[string]interface{} "Владелец зарегистрирован, статус studio_status=pending, возвращается JWT токен"
// @Failure		400	{object}		map[string]interface{} "Ошибка валидации: неверный формат данных"
// @Failure		409	{object}		map[string]interface{} "Ошибка: email уже зарегистрирован на платформе"
// @Failure		500	{object}		map[string]interface{} "Ошибка сервера при создании аккаунта владельца"
// @Router		/auth/register/studio [POST]
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

// Login авторизует пользователя на платформе и выдаёт JWT токен.
// @Summary		Войти в аккаунт
// @Description	Авторизует пользователя (клиента или владельца студии) по email и паролю. Возвращает JWT токен для последующих запросов к защищённым эндпоинтам.
// @Tags		Автентификация
// @Param		request	body	LoginRequest		true	"Учётные данные (email, password)"
// @Success		200	{object}		map[string]interface{} "Успешная авторизация, возвращается JWT токен"
// @Failure		400	{object}		map[string]interface{} "Ошибка валидации: неверный формат данных"
// @Failure		401	{object}		map[string]interface{} "Ошибка: неверный email или пароль"
// @Failure		500	{object}		map[string]interface{} "Ошибка сервера при авторизации"
// @Router		/auth/login [POST]
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

// GetMe получает профиль текущего авторизованного пользователя.
// @Summary		Получить профиль пользователя
// @Description	Возвращает полный профиль текущего авторизованного пользователя (клиента или владельца). Может включать статистику бронирований и недавние брони. При include_stats=true добавляет количество бронирований.
// @Tags		Профиль и аутентификация
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

// UpdateProfile обновляет информацию профиля текущего пользователя.
// @Summary		Обновить профиль пользователя
// @Description	Обновляет информацию о профиле: имя, телефон и другие поля. Email не может быть изменён через этот эндпоинт. Требуется аутентификация.
// @Tags		Профиль и аутентификация
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

// UploadVerificationDocuments загружает документы для верификации владельца студии.
// @Summary		Загрузить документы верификации
// @Description	Загружает документы (паспорт, свидетельство о регистрации, и т.д.) для верификации владельца студии. Документы необходимы для одобрения заявки администратором. Максимальный размер файла 10MB.
// @Tags		Профиль и аутентификация
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


