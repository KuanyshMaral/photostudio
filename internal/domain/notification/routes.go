package notification

import "github.com/gin-gonic/gin"

// RegisterRoutes registers all notification-related routes
func RegisterRoutes(protected *gin.RouterGroup, handler *Handler, prefsHandler *PreferencesHandler, devicesHandler *DeviceTokensHandler) {
	// Notifications
	notifGroup := protected.Group("/notifications")
	{
		notifGroup.GET("", handler.GetNotifications)
		notifGroup.GET("/unread-count", handler.GetUnreadCount)
		notifGroup.PATCH("/:id/read", handler.MarkAsRead)
		notifGroup.POST("/read-all", handler.MarkAllAsRead)
		notifGroup.DELETE("/:id", handler.DeleteNotification)

		// Preferences
		prefsGroup := notifGroup.Group("/preferences")
		{
			prefsGroup.GET("", prefsHandler.GetPreferences)
			prefsGroup.PATCH("", prefsHandler.UpdatePreferences)
			prefsGroup.POST("/reset", prefsHandler.ResetPreferences)
		}

		// Device Tokens
		devicesGroup := notifGroup.Group("/device-tokens")
		{
			devicesGroup.POST("", devicesHandler.RegisterDeviceToken)
			devicesGroup.GET("", devicesHandler.ListDeviceTokens)
			devicesGroup.DELETE("/:id", devicesHandler.DeactivateDeviceToken)
		}
	}
}

