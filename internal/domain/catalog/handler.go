package catalog

import (
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"net/http"
	"os"
	"path/filepath"
	"photostudio/internal/domain/auth"
	"photostudio/internal/pkg/response"
	"strconv"
	"strings"
	"time"
)

type Handler struct {
	service  *Service
	userRepo *auth.UserRepository
}

func NewHandler(service *Service, userRepo *auth.UserRepository) *Handler {
	return &Handler{
		service:  service,
		userRepo: userRepo,
	}
}

/* ---------- STUDIO HANDLERS ---------- */

// GetStudios получение списка студий с фильтрацией и поиском
// @Summary Получить список студий
// @Description Получает список всех студий с возможностью фильтрации по городу, типу комнаты, цене и поиску по названию. Поддерживает сортировку и пагинацию.
// @Tags Catalog - Студии
// @Accept json
// @Produce json
// @Param city query string false "Фильтр по городу" example("Moscow")
// @Param room_type query string false "Фильтр по типу комнаты (Fashion, Portrait, Creative, Commercial)" example("Fashion")
// @Param search query string false "Поиск по названию студии" example("My Studio")
// @Param min_price query number false "Минимальная цена в час" example(100)
// @Param max_price query number false "Максимальная цена в час" example(1000)
// @Param sort_by query string false "Поле для сортировки (rating, price, created_at)" example("rating")
// @Param sort_order query string false "Порядок сортировки (asc, desc)" example("desc")
// @Param page query integer false "Номер страницы" example(1)
// @Param limit query integer false "Количество студий на странице (максимум 100)" example(20)
// @Success 200 {object} map[string]interface{} "Успешный ответ со списком студий и информацией о пагинации"
// @Failure 400 {object} map[string]interface{} "Некорректные параметры запроса"
// @Failure 500 {object} map[string]interface{} "Внутренняя ошибка сервера"
// @Router /api/v1/studios [get]
func (h *Handler) GetStudios(c *gin.Context) {
	var f StudioFilters

	// Parse query parameters
	f.City = c.Query("city")
	f.RoomType = c.Query("room_type")
	// Search + sorting
	f.Search = c.Query("search")
	f.SortBy = c.DefaultQuery("sort_by", "rating")
	f.SortOrder = c.DefaultQuery("sort_order", "desc")

	if minPrice := c.Query("min_price"); minPrice != "" {
		if val, err := strconv.ParseFloat(minPrice, 64); err == nil {
			f.MinPrice = val
		}
	}

	if maxPrice := c.Query("max_price"); maxPrice != "" {
		if val, err := strconv.ParseFloat(maxPrice, 64); err == nil {
			f.MaxPrice = val
		}
	}

	// Pagination
	f.Limit = 20 // default
	if limit := c.Query("limit"); limit != "" {
		if val, err := strconv.Atoi(limit); err == nil && val > 0 && val <= 100 {
			f.Limit = val
		}
	}

	f.Offset = 0
	if page := c.Query("page"); page != "" {
		if val, err := strconv.Atoi(page); err == nil && val > 0 {
			f.Offset = (val - 1) * f.Limit
		}
	}

	studios, total, err := h.service.studioRepo.GetAll(c.Request.Context(), f)
	if err != nil {
		handleError(c, err)
		return
	}

	totalPages := (int(total) + f.Limit - 1) / f.Limit
	currentPage := (f.Offset / f.Limit) + 1

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"studios": studios,
			"pagination": gin.H{
				"page":        currentPage,
				"limit":       f.Limit,
				"total":       total,
				"total_pages": totalPages,
			},
		},
	})
}

// GetStudioByID получение информации о студии по ID
// @Summary Получить студию по ID
// @Description Получает полную информацию о студии, включая все комнаты, оборудование и фотографии по уникальному идентификатору.
// @Tags Catalog - Студии
// @Accept json
// @Produce json
// @Param id path integer true "Уникальный идентификатор студии" example(1)
// @Success 200 {object} map[string]interface{} "Успешный ответ с информацией о студии"
// @Failure 400 {object} map[string]interface{} "Некорректный формат ID"
// @Failure 404 {object} map[string]interface{} "Студия не найдена"
// @Failure 500 {object} map[string]interface{} "Внутренняя ошибка сервера"
// @Router /api/v1/studios/{id} [get]
func (h *Handler) GetStudioByID(c *gin.Context) {
	studioID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "INVALID_ID",
				"message": "Invalid studio ID",
			},
		})
		return
	}

	studio, err := h.service.studioRepo.GetByID(c.Request.Context(), studioID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "NOT_FOUND",
					"message": "Studio not found",
				},
			})
			return
		}
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"studio": studio,
		},
	})
}

// GetStudioWorkingHours получение часов работы студии (устаревший формат)
// @Summary Получить часы работы студии (v1)
// @Description Получает информацию о часах работы студии и её текущем статусе (открыта/закрыта) в устаревшем формате. Рекомендуется использовать v2 endpoint.
// @Tags Catalog - Часы работы
// @Accept json
// @Produce json
// @Param id path integer true "Уникальный идентификатор студии" example(1)
// @Success 200 {object} map[string]interface{} "Успешный ответ с часами работы и статусом"
// @Failure 400 {object} map[string]interface{} "Некорректный формат ID"
// @Failure 404 {object} map[string]interface{} "Студия не найдена"
// @Failure 500 {object} map[string]interface{} "Внутренняя ошибка сервера"
// @Deprecated
// @Router /api/v1/studios/{id}/working-hours [get]
func (h *Handler) GetStudioWorkingHours(c *gin.Context) {
	studioID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.CustomError(c, http.StatusBadRequest, "INVALID_ID", "Invalid studio ID")
		return
	}

	status, err := h.service.GetStudioWorkingStatus(c.Request.Context(), studioID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.CustomError(c, http.StatusNotFound, "NOT_FOUND", "Studio not found")
			return
		}
		response.CustomError(c, http.StatusInternalServerError, "FETCH_FAILED", err)
		return
	}

	response.Success(c, http.StatusOK, gin.H{
		"is_open":       status.IsOpen,
		"message":       status.Message,
		"open_time":     status.OpenTime,
		"close_time":    status.CloseTime,
		"working_hours": status.WorkingHours,
	})
}

// GetStudioWorkingHoursV2 получение часов работы студии с текущим статусом
// @Summary Получить часы работы студии (v2)
// @Description Получает информацию о часах работы студии в новом формате с подробной информацией о текущем статусе (открыта/закрыта), времени открытия/закрытия и полном расписании по дням недели.
// @Tags Catalog - Часы работы
// @Accept json
// @Produce json
// @Param id path integer true "Уникальный идентификатор студии" example(1)
// @Success 200 {object} map[string]interface{} "Успешный ответ с информацией о часах работы и текущем статусе"
// @Failure 400 {object} map[string]interface{} "Некорректный формат ID"
// @Failure 404 {object} map[string]interface{} "Студия не найдена"
// @Failure 500 {object} map[string]interface{} "Внутренняя ошибка сервера"
// @Router /api/v1/studios/{id}/working-hours/v2 [get]
func (h *Handler) GetStudioWorkingHoursV2(c *gin.Context) {
	studioIDStr := c.Param("id")
	studioID, err := strconv.ParseInt(studioIDStr, 10, 64)
	if err != nil {
		response.CustomError(c, http.StatusBadRequest, "INVALID_ID", "Invalid studio ID")
		return
	}

	hoursResponse, err := h.service.GetStudioWorkingHours(c.Request.Context(), studioID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.CustomError(c, http.StatusNotFound, "NOT_FOUND", "Studio not found")
			return
		}
		response.CustomError(c, http.StatusInternalServerError, "FETCH_FAILED", err)
		return
	}

	response.Success(c, http.StatusOK, hoursResponse)
}

// UpdateStudioWorkingHours обновление часов работы студии
// @Summary Обновить часы работы студии
// @Description Обновляет расписание работы студии. Требует аутентификации. Только владелец студии может обновлять её часы работы. Принимает массив объектов с информацией о рабочих днях и часах.
// @Tags Catalog - Часы работы
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path integer true "Уникальный идентификатор студии" example(1)
// @Param body body array true "Массив объектов WorkingHours с расписанием работы по дням недели"
// @Success 200 {object} map[string]interface{} "Успешное обновление часов работы"
// @Failure 400 {object} map[string]interface{} "Некорректный формат запроса"
// @Failure 401 {object} map[string]interface{} "Требуется аутентификация"
// @Failure 403 {object} map[string]interface{} "Недостаточно прав для обновления этой студии"
// @Failure 500 {object} map[string]interface{} "Внутренняя ошибка сервера"
// @Router /api/v1/studios/{id}/working-hours [put]
func (h *Handler) UpdateStudioWorkingHours(c *gin.Context) {
	studioID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.CustomError(c, http.StatusBadRequest, "INVALID_ID", "Invalid studio ID")
		return
	}

	var hours []WorkingHours
	if err := c.ShouldBindJSON(&hours); err != nil {
		response.CustomError(c, http.StatusBadRequest, "VALIDATION_ERROR", err)
		return
	}

	userID := c.GetInt64("user_id")
	if userID == 0 {
		response.CustomError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
		return
	}

	err = h.service.UpdateStudioWorkingHours(c.Request.Context(), userID, studioID, hours)
	if err != nil {
		if errors.Is(err, ErrForbidden) {
			response.CustomError(c, http.StatusForbidden, "FORBIDDEN", "You don't own this studio")
			return
		}
		response.CustomError(c, http.StatusBadRequest, "UPDATE_ERROR", err)
		return
	}

	response.Success(c, http.StatusOK, gin.H{"message": "Working hours updated"})
}

// GetMyStudios получение всех студий текущего владельца
// @Summary Получить мои студии
// @Description Получает список всех студий, принадлежащих текущему авторизованному владельцу студии. Требует валидного JWT токена в заголовке Authorization.
// @Tags Catalog - Студии
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{} "Успешный ответ со списком студий владельца"
// @Failure 401 {object} map[string]interface{} "Требуется аутентификация"
// @Failure 500 {object} map[string]interface{} "Внутренняя ошибка сервера"
// @Router /api/v1/studios/my [get]
func (h *Handler) GetMyStudios(c *gin.Context) {
	userID := c.GetInt64("user_id")
	if userID == 0 {
		response.CustomError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
		return
	}

	studios, err := h.service.GetStudiosByOwner(c.Request.Context(), userID)
	if err != nil {
		response.CustomError(c, http.StatusInternalServerError, "FETCH_FAILED", "Failed to get studios")
		return
	}

	response.Success(c, http.StatusOK, gin.H{"studios": studios})
}

// CreateStudio создание новой студии
// @Summary Создать новую студию
// @Description Создает новую студию в каталоге. Требует аутентификации и верификации пользователя как владельца студии. Пользователь должен иметь роль студии-владельца и пройти верификацию.
// @Tags Catalog - Студии
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body CreateStudioRequest true "Данные для создания студии (название, описание, адрес, город, цена и т.д.)"
// @Success 201 {object} map[string]interface{} "Студия успешно создана, возвращает объект созданной студии"
// @Failure 400 {object} map[string]interface{} "Некорректный формат запроса"
// @Failure 401 {object} map[string]interface{} "Требуется аутентификация"
// @Failure 403 {object} map[string]interface{} "Пользователь не является верифицированным владельцем студии"
// @Failure 500 {object} map[string]interface{} "Внутренняя ошибка сервера"
// @Router /api/v1/studios [post]
func (h *Handler) CreateStudio(c *gin.Context) {
	var req CreateStudioRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "INVALID_REQUEST",
				"message": "Invalid request body",
				"details": err.Error(),
			},
		})
		return
	}

	// Get user_id and role from context (set by auth middleware)
	userID := c.GetInt64("user_id")
	//role = c.GetString("role")

	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "UNAUTHORIZED",
				"message": "User not authenticated",
			},
		})
		return
	}

	// Create minimal user object for service
	userObj, err := h.userRepo.GetByID(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "INTERNAL_ERROR",
				"message": "Failed to load user",
			},
		})
		return
	}

	studio, err := h.service.CreateStudio(c.Request.Context(), userObj, req)
	if err != nil {
		if errors.Is(err, ErrForbidden) {
			c.JSON(http.StatusForbidden, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "FORBIDDEN",
					"message": "Only verified studio owners can create studios",
				},
			})
			return
		}
		handleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data": gin.H{
			"studio": studio,
		},
		"message": "Studio created successfully",
	})
}

// UpdateStudio обновление информации о студии
// @Summary Обновить студию
// @Description Обновляет информацию о студии (названия, описание, адрес, город, цена в час и другие параметры). Требует аутентификации. Только владелец студии может её обновлять.
// @Tags Catalog - Студии
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path integer true "Уникальный идентификатор студии" example(1)
// @Param body body UpdateStudioRequest true "Данные для обновления студии"
// @Success 200 {object} map[string]interface{} "Студия успешно обновлена, возвращает обновленный объект"
// @Failure 400 {object} map[string]interface{} "Некорректный формат запроса"
// @Failure 401 {object} map[string]interface{} "Требуется аутентификация"
// @Failure 403 {object} map[string]interface{} "Недостаточно прав для обновления этой студии"
// @Failure 404 {object} map[string]interface{} "Студия не найдена"
// @Failure 500 {object} map[string]interface{} "Внутренняя ошибка сервера"
// @Router /api/v1/studios/{id} [put]
func (h *Handler) UpdateStudio(c *gin.Context) {
	studioID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "INVALID_ID",
				"message": "Invalid studio ID",
			},
		})
		return
	}

	var req UpdateStudioRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "INVALID_REQUEST",
				"message": "Invalid request body",
				"details": err.Error(),
			},
		})
		return
	}

	userID := c.GetInt64("user_id")
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "UNAUTHORIZED",
				"message": "User not authenticated",
			},
		})
		return
	}

	studio, err := h.service.UpdateStudio(c.Request.Context(), userID, studioID, req)
	if err != nil {
		if errors.Is(err, ErrForbidden) {
			c.JSON(http.StatusForbidden, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "FORBIDDEN",
					"message": "You don't have permission to update this studio",
				},
			})
			return
		}
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "NOT_FOUND",
					"message": "Studio not found",
				},
			})
			return
		}
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"studio": studio,
		},
		"message": "Studio updated successfully",
	})
}

// UpdateRoom обновление информации о комнате
// @Summary Обновить комнату
// @Description Обновляет информацию о комнате в студии (названия, тип комнаты, описание и другие параметры). Требует аутентификации.
// @Tags Catalog - Комнаты
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path integer true "Уникальный идентификатор комнаты" example(1)
// @Param body body UpdateRoomRequest true "Данные для обновления комнаты"
// @Success 200 {object} map[string]interface{} "Комната успешно обновлена"
// @Failure 400 {object} map[string]interface{} "Некорректный формат запроса или тип комнаты"
// @Failure 401 {object} map[string]interface{} "Требуется аутентификация"
// @Failure 500 {object} map[string]interface{} "Внутренняя ошибка сервера"
// @Router /api/v1/rooms/{id} [put]
func (h *Handler) UpdateRoom(c *gin.Context) {
	roomID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.CustomError(c, http.StatusBadRequest, "INVALID_ID", "Invalid room ID")
		return
	}

	var req UpdateRoomRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.CustomError(c, http.StatusBadRequest, "INVALID_REQUEST", err)
		return
	}

	room, err := h.service.UpdateRoom(c.Request.Context(), roomID, req)
	if err != nil {
		if errors.Is(err, ErrInvalidRoomType) {
			response.CustomError(c, http.StatusBadRequest, "INVALID_ROOM_TYPE", err)
			return
		}
		handleError(c, err)
		return
	}

	response.Success(c, http.StatusOK, gin.H{"room": room})
}

// DeleteRoom удаление комнаты из студии
// @Summary Удалить комнату
// @Description Удаляет комнату из студии. Требует аутентификации. Только владелец студии может удалять комнаты.
// @Tags Catalog - Комнаты
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path integer true "Уникальный идентификатор комнаты" example(1)
// @Success 200 {object} map[string]interface{} "Комната успешно удалена"
// @Failure 400 {object} map[string]interface{} "Некорректный формат ID"
// @Failure 401 {object} map[string]interface{} "Требуется аутентификация"
// @Failure 500 {object} map[string]interface{} "Внутренняя ошибка сервера"
// @Router /api/v1/rooms/{id} [delete]
func (h *Handler) DeleteRoom(c *gin.Context) {
	roomID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.CustomError(c, http.StatusBadRequest, "INVALID_ID", "Invalid room ID")
		return
	}

	if err := h.service.DeleteRoom(c.Request.Context(), roomID); err != nil {
		handleError(c, err)
		return
	}

	response.Success(c, http.StatusOK, gin.H{"deleted": true})
}

/* ---------- PHOTO HANDLERS ---------- */

// UploadStudioPhotos загрузка фотографий студии
// @Summary Загрузить фотографии студии
// @Description Загружает фотографии студии. Поддерживает загрузку до 10 файлов одновременно. Допустимые форматы: JPEG, PNG, WebP. Максимальный размер файла: 5 МБ. Требует аутентификации. Только владелец студии может загружать фотографии.
// @Tags Catalog - Фотографии
// @Accept multipart/form-data
// @Produce json
// @Security BearerAuth
// @Param id path integer true "Уникальный идентификатор студии" example(1)
// @Param photos formData []file true "Файлы изображений для загрузки (до 10 файлов)" CollectionFormat(multi)
// @Success 200 {object} map[string]interface{} "Фотографии успешно загружены, возвращает список URL загруженных файлов"
// @Failure 400 {object} map[string]interface{} "Некорректный формат запроса, файл слишком большой или недопустимый формат"
// @Failure 401 {object} map[string]interface{} "Требуется аутентификация"
// @Failure 500 {object} map[string]interface{} "Внутренняя ошибка сервера"
// @Router /api/v1/studios/{id}/photos [post]
func (h *Handler) UploadStudioPhotos(c *gin.Context) {
	// 1. Extract studio ID from URL param
	studioIDStr := c.Param("id")
	studioID, err := strconv.ParseInt(studioIDStr, 10, 64)
	if err != nil {
		response.CustomError(c, http.StatusBadRequest, "INVALID_ID", "Invalid studio ID")
		return
	}

	// 2. Get userID from context (set by JWT middleware)
	v, ok := c.Get("user_id")
	if !ok {
		response.CustomError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Missing user_id in context")
		return
	}

	var userID int64
	switch t := v.(type) {
	case int64:
		userID = t
	case int:
		userID = int64(t)
	case float64:
		userID = int64(t)
	default:
		response.CustomError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Invalid user_id type in context")
		return
	}

	// 3. Parse multipart form
	form, err := c.MultipartForm()
	if err != nil {
		response.CustomError(c, http.StatusBadRequest, "INVALID_FORM", "Invalid multipart form")
		return
	}

	files := form.File["photos"]
	if len(files) == 0 {
		response.CustomError(c, http.StatusBadRequest, "NO_FILES", "No files provided")
		return
	}

	// Cut request to max 10 files (final limit is enforced in service too)
	if len(files) > 10 {
		files = files[:10]
	}

	// 4. Create upload dir
	uploadDir := fmt.Sprintf("./uploads/studios/%d", studioID)
	if err := os.MkdirAll(uploadDir, os.ModePerm); err != nil {
		response.CustomError(c, http.StatusInternalServerError, "UPLOAD_DIR_ERROR", err)
		return
	}

	var uploadedURLs []string
	for _, file := range files {
		// size limit 5MB
		if file.Size > 5*1024*1024 {
			continue
		}

		// extension whitelist: jpg, png, webp
		ext := strings.ToLower(filepath.Ext(file.Filename))
		if ext == ".jpeg" {
			ext = ".jpg"
		}
		if ext != ".jpg" && ext != ".png" && ext != ".webp" {
			continue
		}

		// Generate unique name
		newName := fmt.Sprintf("%d%s", time.Now().UnixNano(), ext)
		savePath := filepath.Join(uploadDir, newName)

		if err := c.SaveUploadedFile(file, savePath); err != nil {
			continue
		}

		url := fmt.Sprintf("/static/studios/%d/%s", studioID, newName)
		uploadedURLs = append(uploadedURLs, url)
	}

	if len(uploadedURLs) == 0 {
		response.CustomError(c, http.StatusBadRequest, "NO_VALID_FILES", "No valid files uploaded")
		return
	}

	// 5. Save URLs in DB (service enforces max 10 total and ownership)
	if err := h.service.AddStudioPhotos(c.Request.Context(), userID, studioID, uploadedURLs); err != nil {
		response.CustomError(c, http.StatusBadRequest, "PHOTO_UPLOAD_ERROR", err)
		return
	}

	response.Success(c, http.StatusOK, gin.H{
		"uploaded": len(uploadedURLs),
		"urls":     uploadedURLs,
	})
}

/* ---------- ROOM HANDLERS ---------- */

// GetRooms получение списка комнат студии
// @Summary Получить комнаты студии
// @Description Получает список всех комнат конкретной студии или всех комнат во всех студиях. Может быть отфильтровано по ID студии через параметр запроса.
// @Tags Catalog - Комнаты
// @Accept json
// @Produce json
// @Param studio_id query integer false "ID студии для фильтрации комнат. Если не указан, возвращает все комнаты" example(1)
// @Success 200 {object} map[string]interface{} "Успешный ответ со списком комнат"
// @Failure 500 {object} map[string]interface{} "Внутренняя ошибка сервера"
// @Router /api/v1/rooms [get]
func (h *Handler) GetRooms(c *gin.Context) {
	var studioIDPtr *int64
	if studioIDStr := c.Query("studio_id"); studioIDStr != "" {
		if studioID, err := strconv.ParseInt(studioIDStr, 10, 64); err == nil {
			studioIDPtr = &studioID
		}
	}

	rooms, err := h.service.roomRepo.GetAll(c.Request.Context(), studioIDPtr)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"rooms": rooms,
		},
	})
}

// GetRoomByID получение информации о комнате по ID
// @Summary Получить комнату по ID
// @Description Получает полную информацию о комнате, включая её характеристики, тип, оборудование и фотографии по уникальному идентификатору.
// @Tags Catalog - Комнаты
// @Accept json
// @Produce json
// @Param id path integer true "Уникальный идентификатор комнаты" example(1)
// @Success 200 {object} map[string]interface{} "Успешный ответ с информацией о комнате"
// @Failure 400 {object} map[string]interface{} "Некорректный формат ID"
// @Failure 404 {object} map[string]interface{} "Комната не найдена"
// @Failure 500 {object} map[string]interface{} "Внутренняя ошибка сервера"
// @Router /api/v1/rooms/{id} [get]
func (h *Handler) GetRoomByID(c *gin.Context) {
	roomID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "INVALID_ID",
				"message": "Invalid room ID",
			},
		})
		return
	}

	room, err := h.service.roomRepo.GetByID(c.Request.Context(), roomID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "NOT_FOUND",
					"message": "Room not found",
				},
			})
			return
		}
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"room": room,
		},
	})
}

// CreateRoom создание новой комнаты в студии
// @Summary Создать новую комнату
// @Description Создает новую комнату в студии. Требует аутентификации. Только владелец студии может создавать комнаты. Поддерживаемые типы комнат: Fashion, Portrait, Creative, Commercial.
// @Tags Catalog - Комнаты
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path integer true "Уникальный идентификатор студии" example(1)
// @Param body body CreateRoomRequest true "Данные для создания комнаты (названия, тип, описание и т.д.)"
// @Success 201 {object} map[string]interface{} "Комната успешно создана, возвращает объект созданной комнаты"
// @Failure 400 {object} map[string]interface{} "Некорректный формат запроса или тип комнаты"
// @Failure 401 {object} map[string]interface{} "Требуется аутентификация"
// @Failure 403 {object} map[string]interface{} "Недостаточно прав для добавления комнат в эту студию"
// @Failure 404 {object} map[string]interface{} "Студия не найдена"
// @Failure 500 {object} map[string]interface{} "Внутренняя ошибка сервера"
// @Router /api/v1/studios/{id}/rooms [post]
func (h *Handler) CreateRoom(c *gin.Context) {
	studioID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "INVALID_ID",
				"message": "Invalid studio ID",
			},
		})
		return
	}

	var req CreateRoomRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "INVALID_REQUEST",
				"message": "Invalid request body",
				"details": err.Error(),
			},
		})
		return
	}

	userID := c.GetInt64("user_id")
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "UNAUTHORIZED",
				"message": "User not authenticated",
			},
		})
		return
	}

	room, err := h.service.CreateRoom(c.Request.Context(), userID, studioID, req)
	if err != nil {

		if errors.Is(err, ErrInvalidRoomType) {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "INVALID_ROOM_TYPE",
					"message": "Invalid room type. Must be one of: Fashion, Portrait, Creative, Commercial",
				},
			})
			return
		}

		if errors.Is(err, ErrForbidden) {
			c.JSON(http.StatusForbidden, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "FORBIDDEN",
					"message": "You don't have permission to add rooms to this studio",
				},
			})
			return
		}
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "NOT_FOUND",
					"message": "Studio not found",
				},
			})
			return
		}
		handleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data": gin.H{
			"room": room,
		},
		"message": "Room created successfully",
	})
}

// GetRoomTypes получение списка доступных типов комнат
// @Summary Получить типы комнат
// @Description Возвращает список всех доступных типов комнат, которые могут быть использованы при создании или обновлении комнаты в студии. Например: Fashion, Portrait, Creative, Commercial.
// @Tags Catalog - Типы комнат
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{} "Успешный ответ со списком доступных типов комнат"
// @Router /api/v1/room-types [get]
func (h *Handler) GetRoomTypes(c *gin.Context) {
	types := ValidRoomTypes()

	typeStrings := make([]string, len(types))
	for i, t := range types {
		typeStrings[i] = string(t)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"room_types": typeStrings,
		},
	})
}

/* ---------- EQUIPMENT HANDLERS ---------- */

// AddEquipment добавление оборудования в комнату
// @Summary Добавить оборудование в комнату
// @Description Добавляет оборудование в комнату студии. Требует аутентификации. Только владелец студии может добавлять оборудование. Оборудование содержит названия и другую информацию о предметах в комнате.
// @Tags Catalog - Оборудование
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path integer true "Уникальный идентификатор комнаты" example(1)
// @Param body body CreateEquipmentRequest true "Данные оборудования для добавления (названия, тип и т.д.)"
// @Success 201 {object} map[string]interface{} "Оборудование успешно добавлено, возвращает объект добавленного оборудования"
// @Failure 400 {object} map[string]interface{} "Некорректный формат запроса"
// @Failure 401 {object} map[string]interface{} "Требуется аутентификация"
// @Failure 403 {object} map[string]interface{} "Недостаточно прав для добавления оборудования в эту комнату"
// @Failure 404 {object} map[string]interface{} "Комната не найдена"
// @Failure 500 {object} map[string]interface{} "Внутренняя ошибка сервера"
// @Router /api/v1/rooms/{id}/equipment [post]
func (h *Handler) AddEquipment(c *gin.Context) {
	roomID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "INVALID_ID",
				"message": "Invalid room ID",
			},
		})
		return
	}

	var req CreateEquipmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "INVALID_REQUEST",
				"message": "Invalid request body",
				"details": err.Error(),
			},
		})
		return
	}

	userID := c.GetInt64("user_id")
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "UNAUTHORIZED",
				"message": "User not authenticated",
			},
		})
		return
	}

	equipment, err := h.service.AddEquipment(c.Request.Context(), userID, roomID, req)
	if err != nil {
		if errors.Is(err, ErrForbidden) {
			c.JSON(http.StatusForbidden, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "FORBIDDEN",
					"message": "You don't have permission to add equipment to this room",
				},
			})
			return
		}
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "NOT_FOUND",
					"message": "Room not found",
				},
			})
			return
		}
		handleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data": gin.H{
			"equipment": equipment,
		},
		"message": "Equipment added successfully",
	})
}

/* ---------- ROUTE REGISTRATION ---------- */

// RegisterRoutes registers all catalog routes

// RegisterProtectedRoutes registers protected catalog routes that require authentication

/* ---------- ERROR HANDLING ---------- */

func handleError(c *gin.Context, err error) {
	if err == nil {
		return
	}

	// Check for specific error types
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "NOT_FOUND",
				"message": "Resource not found",
			},
		})
	case errors.Is(err, ErrForbidden):
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "FORBIDDEN",
				"message": "You don't have permission to perform this action",
			},
		})
	default:
		// Generic server error
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "INTERNAL_ERROR",
				"message": "An internal error occurred",
				"details": err.Error(),
			},
		})
	}
}