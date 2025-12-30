package review

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) RegisterRoutes(public, protected *gin.RouterGroup) {
	// Public routes (no auth required)
	if public != nil {
		public.GET("/studios/:id/reviews", h.GetByStudio)
	}

	// Protected routes (auth required)
	if protected != nil {
		protected.POST("/reviews", h.Create)
		protected.POST("/reviews/:id/response", h.AddOwnerResponse)
	}
}

func (h *Handler) Create(c *gin.Context) {
	var req CreateReviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"code": "INVALID_REQUEST", "message": "Invalid request body"}})
		return
	}

	userID := c.GetInt64("user_id")
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": gin.H{"code": "UNAUTHORIZED", "message": "Authentication required"}})
		return
	}

	rv, err := h.svc.Create(c.Request.Context(), userID, req)
	if err != nil {
		switch err {
		case ErrInvalidRequest:
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"code": "INVALID_REQUEST", "message": "Invalid input"}})
		case ErrReviewNotAllowed:
			c.JSON(http.StatusForbidden, gin.H{"success": false, "error": gin.H{"code": "FORBIDDEN", "message": "You can review only after completed booking"}})
		case ErrConflict:
			c.JSON(http.StatusConflict, gin.H{"success": false, "error": gin.H{"code": "CONFLICT", "message": "Only one review per user per studio"}})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": gin.H{"code": "INTERNAL", "message": "Internal error"}})
		}
		return
	}

	c.JSON(http.StatusCreated, gin.H{"success": true, "data": rv})
}

func (h *Handler) GetByStudio(c *gin.Context) {
	studioID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || studioID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"code": "INVALID_ID", "message": "Invalid studio ID"}})
		return
	}

	limit, _ := strconv.Atoi(c.Query("limit"))
	offset, _ := strconv.Atoi(c.Query("offset"))

	items, err := h.svc.GetByStudio(c.Request.Context(), studioID, limit, offset)
	if err != nil {
		if err == ErrInvalidRequest {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"code": "INVALID_REQUEST", "message": "Invalid input"}})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": gin.H{"code": "INTERNAL", "message": "Internal error"}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": items})
}

func (h *Handler) AddOwnerResponse(c *gin.Context) {
	reviewID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || reviewID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"code": "INVALID_ID", "message": "Invalid review ID"}})
		return
	}

	var req OwnerResponseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"code": "INVALID_REQUEST", "message": "Invalid request body"}})
		return
	}

	userID := c.GetInt64("user_id")
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": gin.H{"code": "UNAUTHORIZED", "message": "Authentication required"}})
		return
	}

	rv, err := h.svc.AddOwnerResponse(c.Request.Context(), reviewID, userID, req.Response)
	if err != nil {
		switch err {
		case ErrInvalidRequest:
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"code": "INVALID_REQUEST", "message": "Invalid input"}})
		case ErrForbidden:
			c.JSON(http.StatusForbidden, gin.H{"success": false, "error": gin.H{"code": "FORBIDDEN", "message": "You don't own this studio"}})
		case ErrNotFound:
			c.JSON(http.StatusNotFound, gin.H{"success": false, "error": gin.H{"code": "NOT_FOUND", "message": "Review not found"}})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": gin.H{"code": "INTERNAL", "message": "Internal error"}})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": rv})
}
