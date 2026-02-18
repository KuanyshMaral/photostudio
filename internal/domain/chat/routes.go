package chat

import "github.com/gin-gonic/gin"

// RegisterRoutes registers all chat routes under the protected group
func RegisterRoutes(r *gin.RouterGroup, h *Handler) {
	rooms := r.Group("/rooms")
	{
		// Room creation
		rooms.POST("/direct", h.CreateDirectRoom)
		rooms.POST("/group", h.CreateGroupRoom)

		// Room listing & unread
		rooms.GET("", h.ListRooms)
		rooms.GET("/unread", h.GetUnreadCount)

		// WebSocket
		rooms.GET("/ws", h.WebSocket)

		// Per-room operations
		rooms.GET("/:id/messages", h.GetMessages)
		rooms.POST("/:id/messages", h.SendMessage)
		rooms.POST("/:id/read", h.MarkAsRead)
		rooms.POST("/:id/leave", h.LeaveRoom)

		// Member management (group admin only)
		rooms.GET("/:id/members", h.GetMembers)
		rooms.POST("/:id/members", h.AddMember)
		rooms.DELETE("/:id/members/:user_id", h.RemoveMember)
	}
}
