package catalog

import (
	"net/http"
	"strconv"

	"photostudio/internal/domain"
	"photostudio/internal/repository"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

/* ---------- STUDIO ---------- */

func (h *Handler) GetStudios(c *gin.Context) {
	var f repository.StudioFilters
	if err := c.ShouldBindQuery(&f); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid filters"})
		return
	}

	studios, total, err := h.service.studioRepo.GetAll(c.Request.Context(), f)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  studios,
		"total": total,
	})
}

func (h *Handler) CreateStudio(c *gin.Context) {
	var req CreateStudioRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "invalid request"})
		return
	}

	user := c.MustGet("user").(*domain.User)

	if err := h.service.CreateStudio(c.Request.Context(), user, req); err != nil {
		handleError(c, err)
		return
	}

	c.JSON(201, gin.H{"success": true})
}

func (h *Handler) UpdateStudio(c *gin.Context) {
	var req UpdateStudioRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "invalid request"})
		return
	}

	userID := c.GetInt64("user_id")
	studioID, _ := strconv.ParseInt(c.Param("id"), 10, 64)

	if err := h.service.UpdateStudio(c.Request.Context(), userID, studioID, req); err != nil {
		handleError(c, err)
		return
	}

	c.JSON(200, gin.H{"success": true})
}

/* ---------- ROOM ---------- */

func (h *Handler) CreateRoom(c *gin.Context) {
	var req CreateRoomRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "invalid request"})
		return
	}

	userID := c.GetInt64("user_id")
	studioID, _ := strconv.ParseInt(c.Param("id"), 10, 64)

	if err := h.service.CreateRoom(c.Request.Context(), userID, studioID, req); err != nil {
		handleError(c, err)
		return
	}

	c.JSON(201, gin.H{"success": true})
}

/* ---------- EQUIPMENT ---------- */

func (h *Handler) AddEquipment(c *gin.Context) {
	var req CreateEquipmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "invalid request"})
		return
	}

	userID := c.GetInt64("user_id")
	roomID, _ := strconv.ParseInt(c.Param("id"), 10, 64)

	if err := h.service.AddEquipment(c.Request.Context(), userID, roomID, req); err != nil {
		handleError(c, err)
		return
	}

	c.JSON(201, gin.H{"success": true})
}

func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	r.GET("/studios", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
	})
}

func handleError(c *gin.Context, err error) {
	switch err {
	case nil:
		return
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}
}
