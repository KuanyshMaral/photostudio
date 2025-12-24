package catalog

import (
	"net/http"
	"strconv"

	"photostudio/internal/repository"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

func NewHandler(repo *repository.StudioRepository) *Handler {
	return &Handler{
		service: NewService(repo),
	}
}

func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	r.GET("/studios", h.GetStudios)
	r.GET("/studios/:id", h.GetStudioByID)
}

func (h *Handler) GetStudios(c *gin.Context) {
	f := repository.StudioFilters{
		City:     c.Query("city"),
		RoomType: c.Query("room_type"),
		Limit:    20,
		Offset:   0,
	}

	if v := c.Query("min_price"); v != "" {
		f.MinPrice, _ = strconv.ParseFloat(v, 64)
	}

	studios, total, err := h.service.GetStudios(c.Request.Context(), f)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"studios": studios,
			"pagination": gin.H{
				"total":  total,
				"limit":  f.Limit,
				"offset": f.Offset,
			},
		},
	})
}

func (h *Handler) GetStudioByID(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid studio id",
		})
		return
	}

	studio, err := h.service.GetStudioByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "studio not found",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    studio,
	})
}
