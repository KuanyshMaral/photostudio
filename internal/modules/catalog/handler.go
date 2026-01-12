package catalog

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"photostudio/internal/pkg/response"
	"strconv"
	"time"

	"photostudio/internal/domain"
	"photostudio/internal/repository"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type Handler struct {
	service  *Service
	userRepo *repository.UserRepository
}

func NewHandler(service *Service, userRepo *repository.UserRepository) *Handler {
	return &Handler{
		service:  service,
		userRepo: userRepo,
	}
}

/* ---------- STUDIO HANDLERS ---------- */

// GetStudios handles GET /api/v1/studios with filters
func (h *Handler) GetStudios(c *gin.Context) {
	var f repository.StudioFilters

	// Parse query parameters
	f.City = c.Query("city")
	f.RoomType = c.Query("room_type")

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

// GetStudioByID handles GET /api/v1/studios/:id
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

// GetMyStudios — GET /studios/my
func (h *Handler) GetMyStudios(c *gin.Context) {
	userID := c.GetInt64("user_id")
	if userID == 0 {
		response.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
		return
	}

	studios, err := h.service.GetStudiosByOwner(c.Request.Context(), userID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "FETCH_FAILED", "Failed to get studios")
		return
	}

	response.Success(c, http.StatusOK, gin.H{"studios": studios})
}

// CreateStudio handles POST /api/v1/studios (protected)
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

// UpdateStudio handles PUT /api/v1/studios/:id (protected)
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

/* ---------- PHOTO HANDLERS ---------- */

// UploadStudioPhotos handler for uploading studio photos
func (h *Handler) UploadStudioPhotos(c *gin.Context) {
	// 1. Extract studio ID from URL param
	studioIDStr := c.Param("id")
	studioID, err := strconv.ParseInt(studioIDStr, 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "INVALID_ID", "Invalid studio ID")
		return
	}

	// 2. Extract authenticated user ID from JWT context (set by middleware)
	userIDVal, exists := c.Get("user_id")
	if !exists {
		response.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "User not authenticated")
		return
	}
	userID, ok := userIDVal.(int64)
	if !ok {
		response.Error(c, http.StatusInternalServerError, "CONTEXT_ERROR", "Invalid user ID in context")
		return
	}

	// 3. Optional: early role check (if you want to restrict to studio owners only)
	roleVal, _ := c.Get("role")
	if roleVal != string(domain.RoleStudioOwner) {
		response.Error(c, http.StatusForbidden, "FORBIDDEN", "Only studio owners can upload photos")
		return
	}

	// 4. Parse multipart form (max 10MB total)
	if err := c.Request.ParseMultipartForm(10 << 20); err != nil {
		response.Error(c, http.StatusBadRequest, "INVALID_FORM", "Failed to parse multipart form")
		return
	}

	files := c.Request.MultipartForm.File["photos"]
	if len(files) == 0 {
		response.Error(c, http.StatusBadRequest, "NO_FILES", "No photos uploaded")
		return
	}

	// 5. Prepare upload directory
	uploadDir := fmt.Sprintf("./uploads/studios/%d", studioID)
	if err := os.MkdirAll(uploadDir, os.ModePerm); err != nil {
		response.Error(c, http.StatusInternalServerError, "STORAGE_ERROR", "Failed to create directory")
		return
	}

	// 6. Save files and collect URLs
	var urls []string
	for _, file := range files {
		filename := fmt.Sprintf("%d_%s", time.Now().UnixNano(), file.Filename)
		savePath := filepath.Join(uploadDir, filename)

		if err := c.SaveUploadedFile(file, savePath); err != nil {
			response.Error(c, http.StatusInternalServerError, "SAVE_FAILED", "Failed to save photo")
			return
		}

		// Public URL (served by Gin static)
		url := fmt.Sprintf("/static/studios/%d/%s", studioID, filename)
		urls = append(urls, url)
	}

	// 7. Call service with BOTH userID and studioID (ownership check inside service)
	if err := h.service.AddStudioPhotos(c.Request.Context(), userID, studioID, urls); err != nil {
		if errors.Is(err, ErrForbidden) {
			response.Error(c, http.StatusForbidden, "FORBIDDEN", "You don't own this studio")
			return
		}
		response.Error(c, http.StatusInternalServerError, "DB_ERROR", "Failed to save photo URLs")
		return
	}

	// 8. Success response
	response.Success(c, http.StatusOK, gin.H{
		"message":       "Photos uploaded successfully",
		"uploaded_urls": urls,
	})
}

/* ---------- ROOM HANDLERS ---------- */

// GetRooms handles GET /api/v1/rooms
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

// GetRoomByID handles GET /api/v1/rooms/:id
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

// CreateRoom handles POST /api/v1/studios/:id/rooms (protected)
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

func (h *Handler) GetRoomTypes(c *gin.Context) {
	types := domain.ValidRoomTypes()

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

// AddEquipment handles POST /api/v1/rooms/:id/equipment (protected)
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
func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	// Public routes
	studios := r.Group("/studios")
	{
		studios.GET("", h.GetStudios)        // GET /api/v1/studios?city=...&room_type=...
		studios.GET("/:id", h.GetStudioByID) // GET /api/v1/studios/:id
	}

	r.GET("/room-types", h.GetRoomTypes)
	// Protected routes (require authentication)
	// Note: Auth middleware should be applied to these in main.go
	// studios.POST("", h.CreateStudio)              // POST /api/v1/studios
	// studios.PUT("/:id", h.UpdateStudio)           // PUT /api/v1/studios/:id
	// studios.POST("/:id/rooms", h.CreateRoom)      // POST /api/v1/studios/:id/rooms

	// Equipment routes
	// r.POST("/rooms/:id/equipment", h.AddEquipment)  // POST /api/v1/rooms/:id/equipment
}

// RegisterProtectedRoutes registers protected catalog routes that require authentication
//func (h *Handler) RegisterProtectedRoutes(r *gin.RouterGroup) {

//}

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
