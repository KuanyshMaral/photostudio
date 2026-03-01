package chat

import "github.com/go-chi/chi/v5"

// RegisterRoutes registers all chat routes under the protected group
func RegisterRoutes(r chi.Router, h *Handler) {

	// Room creation
	r.Post("/direct", h.CreateDirectRoom)
	r.Post("/group", h.CreateGroupRoom)

	// Room listing & unread
	r.Get("/", h.ListRooms)
	r.Get("/unread", h.GetUnreadCount)

	// WebSocket
	r.Get("/ws", h.WebSocket)

	// Per-room operations
	r.Get("/{id}/messages", h.GetMessages)
	r.Post("/{id}/messages", h.SendMessage)
	r.Post("/{id}/read", h.MarkAsRead)
	r.Post("/{id}/leave", h.LeaveRoom)

	// Member management (group admin only)
	r.Get("/{id}/members", h.GetMembers)
	r.Post("/{id}/members", h.AddMember)
	r.Delete("/{id}/members/{user_id}", h.RemoveMember)

}
