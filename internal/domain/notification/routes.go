package notification

import "github.com/go-chi/chi/v5"

// RegisterRoutes registers all notification-related routes
func RegisterRoutes(r chi.Router, handler *Handler, prefsHandler *PreferencesHandler, devicesHandler *DeviceTokensHandler) {

	r.Get("/", handler.GetNotifications)
	r.Get("/ws", handler.WebSocket)
	r.Get("/unread-count", handler.GetUnreadCount)
	r.Patch("/{id}/read", handler.MarkAsRead)
	r.Post("/read-all", handler.MarkAllAsRead)
	r.Delete("/{id}", handler.DeleteNotification)

	// Preferences
	r.Route("/preferences", func(r chi.Router) {
		r.Get("/", prefsHandler.GetPreferences)
		r.Patch("/", prefsHandler.UpdatePreferences)
		r.Post("/reset", prefsHandler.ResetPreferences)
	})

	// Device Tokens
	r.Route("/device-tokens", func(r chi.Router) {
		r.Post("/", devicesHandler.RegisterDeviceToken)
		r.Get("/", devicesHandler.ListDeviceTokens)
		r.Delete("/{id}", devicesHandler.DeactivateDeviceToken)
	})

}
