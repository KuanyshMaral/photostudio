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

		// Legacy API aliases for frontend backward compatibility using query params or similar structures if really needed.
		// However, in REST standard, if polymorphic route uses /:type/:id, 
		// we cannot keep an identical dynamic path /:studioId.
		// Therefore we will deprecate the old specific paths and rely on the polymorphic ones entirely or change the old path format (e.g., /studios/:id/favorite), which should be mapped at the top level router.
		// Since we want drop-in without crashing: 
		// Frontend must be updated to call /favorites/studio/123 instead of /favorites/123
	}
}
