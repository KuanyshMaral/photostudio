package favorite

import "github.com/gin-gonic/gin"

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	favorites := rg.Group("/favorites")
	{
		// Polymorphic routes
		favorites.GET("", h.GetFavorites)
		favorites.POST("", h.AddFavorite)
		favorites.DELETE("/:type/:id", h.RemoveFavorite)
		favorites.GET("/:type/:id/check", h.CheckFavorite)

		// Legacy aliases for studio
		favorites.POST("/:studioId", h.AddFavoriteLegacy)
		favorites.DELETE("/:studioId", h.RemoveFavoriteLegacy)
		favorites.GET("/:studioId/check", h.CheckFavoriteLegacy)
	}
}
