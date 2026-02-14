package mwork

import "github.com/gin-gonic/gin"

func (h *Handler) RegisterRoutes(internal *gin.RouterGroup) {
	mworkGroup := internal.Group("/mwork")
	{
		mworkGroup.POST("/users/sync", h.SyncUser)
	}
}

