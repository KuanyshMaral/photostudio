package favorite

import "github.com/gin-gonic/gin"

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	favorites := rg.Group("/favorites")
	{
		favorites.GET("", h.GetFavorites)
		favorites.POST("/:studioId", h.AddFavorite)
		favorites.DELETE("/:studioId", h.RemoveFavorite)
		favorites.GET("/:studioId/check", h.CheckFavorite)
	}
}

