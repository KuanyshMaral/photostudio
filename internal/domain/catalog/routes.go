package catalog

import "github.com/gin-gonic/gin"

func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	// Public routes
	studios := r.Group("/studios")
	{
		studios.GET("", h.GetStudios)                                   // GET /api/v1/studios?city=...&room_type=...
		studios.GET("/:id", h.GetStudioByID)                            // GET /api/v1/studios/:id
		studios.GET("/:id/working-hours", h.GetStudioWorkingHours)      // GET /api/v1/studios/:id/working-hours (legacy)
		studios.GET("/:id/working-hours/v2", h.GetStudioWorkingHoursV2) // GET /api/v1/studios/:id/working-hours/v2 (new)
	}

	r.GET("/room-types", h.GetRoomTypes)
}

func (h *Handler) RegisterProtectedRoutes(r *gin.RouterGroup) {
	// Working hours update (только владелец)
	r.PUT("/studios/:id/working-hours", h.UpdateStudioWorkingHours)
}

