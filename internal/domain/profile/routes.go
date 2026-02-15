package profile

import "github.com/gin-gonic/gin"

// RegisterRoutes registers all profile routes
func RegisterRoutes(r *gin.RouterGroup, clientHandler *ClientHandler, ownerHandler *OwnerHandler, adminHandler *AdminHandler) {
	profile := r.Group("/profile")
	{
		// Client profile (role: client)
		profile.GET("/client", clientHandler.GetProfile)
		profile.PUT("/client", clientHandler.UpdateProfile)

		// Owner profile (role: studio_owner)
		profile.GET("/owner", ownerHandler.GetProfile)
		profile.PUT("/owner", ownerHandler.UpdateProfile)

		// Admin profile (role: admin)
		profile.GET("/admin", adminHandler.GetProfile)
		profile.PUT("/admin", adminHandler.UpdateProfile)
	}
}
