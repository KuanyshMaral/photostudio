package chat

import "github.com/gin-gonic/gin"

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	chatGroup := rg.Group("/chat")
	{
		chatGroup.POST("/conversations", h.CreateConversation)
		chatGroup.GET("/conversations", h.ListConversations)

		chatGroup.GET("/conversations/:id/messages", h.GetMessages)
		chatGroup.POST("/conversations/:id/messages", h.SendMessage)
		chatGroup.POST("/conversations/:id/messages/upload", h.UploadImage) // <-- NEW
		chatGroup.POST("/conversations/:id/read", h.MarkAsRead)

		chatGroup.POST("/users/:id/block", h.BlockUser)
		chatGroup.DELETE("/users/:id/block", h.UnblockUser)
	}
}

