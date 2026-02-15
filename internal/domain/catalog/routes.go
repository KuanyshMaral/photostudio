package catalog

import (
	"github.com/gin-gonic/gin"
)

// OwnershipMiddleware для проверки прав владения студией/комнатой
type OwnershipMiddleware interface {
	CheckStudioOwnership() gin.HandlerFunc
	CheckRoomOwnership() gin.HandlerFunc
}

// RegisterRoutes регистрирует публичные маршруты каталога
func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	// Public studio routes
	studios := r.Group("/studios")
	{
		studios.GET("", h.GetStudios)                                   // List studios with filtering
		studios.GET("/:id", h.GetStudioByID)                            // Get studio details
		studios.GET("/:id/working-hours", h.GetStudioWorkingHours)      // Get working hours
		studios.GET("/:id/working-hours/v2", h.GetStudioWorkingHoursV2) // Get working hours v2
	}

	// Получение типов комнат
	r.GET("/room-types", h.GetRoomTypes)

	// Public routes для комнат
	r.GET("/rooms", h.GetRooms)
	r.GET("/rooms/:id", h.GetRoomByID)
}

// RegisterProtectedRoutes регистрирует защищенные маршруты (требуется авторизация)
func (h *Handler) RegisterProtectedRoutes(r *gin.RouterGroup, ownershipChecker OwnershipMiddleware) {
	// Studio management (Owner only)
	studios := r.Group("/studios")
	{
		studios.POST("", h.CreateStudio)
		studios.PUT("/:id", ownershipChecker.CheckStudioOwnership(), h.UpdateStudio)
		studios.PUT("/:id/working-hours", ownershipChecker.CheckStudioOwnership(), h.UpdateStudioWorkingHours)

		// Room management within studio
		studios.POST("/:id/rooms", ownershipChecker.CheckStudioOwnership(), h.CreateRoom)
		studios.POST("/:id/photos", ownershipChecker.CheckStudioOwnership(), h.UploadStudioPhotos)
	}

	// Direct room management
	rooms := r.Group("/rooms")
	{
		rooms.PUT("/:id", ownershipChecker.CheckRoomOwnership(), h.UpdateRoom)
		rooms.DELETE("/:id", ownershipChecker.CheckRoomOwnership(), h.DeleteRoom)
		rooms.POST("/:id/equipment", ownershipChecker.CheckRoomOwnership(), h.AddEquipment)
	}

	// User's studios
	r.GET("/studios/my", h.GetMyStudios)
}
