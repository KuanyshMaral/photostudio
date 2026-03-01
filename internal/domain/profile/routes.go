package profile

import "github.com/go-chi/chi/v5"

// RegisterRoutes registers all profile routes
func RegisterRoutes(r chi.Router, clientHandler *ClientHandler, ownerHandler *OwnerHandler, adminHandler *AdminHandler) {

	// Client profile (role: client)
	r.Get("/client", clientHandler.GetProfile)
	r.Put("/client", clientHandler.UpdateProfile)

	// Owner profile (role: studio_owner)
	r.Get("/owner", ownerHandler.GetProfile)
	r.Put("/owner", ownerHandler.UpdateProfile)

	// Admin profile (role: admin)
	r.Get("/admin", adminHandler.GetProfile)
	r.Put("/admin", adminHandler.UpdateProfile)

}
