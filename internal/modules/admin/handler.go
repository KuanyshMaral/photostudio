package admin

import (
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"photostudio/internal/pkg/response"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(admin *gin.RouterGroup) {
	admin.GET("/studios/pending", h.GetPendingStudios)
	admin.POST("/studios/:id/verify", h.VerifyStudio)
	admin.POST("/studios/:id/reject", h.RejectStudio)
	admin.GET("/statistics", h.GetStatistics)
}

func (h *Handler) GetPendingStudios(c *gin.Context) {
	if !isAdmin(c) {
		response.Error(c, http.StatusForbidden, "FORBIDDEN", "Admin access required")
		return
	}

	page := parseIntDefault(c.Query("page"), 1)
	limit := parseIntDefault(c.Query("limit"), 20)

	log.Printf("admin action: GetPendingStudios page=%d limit=%d", page, limit)

	res, err := h.service.GetPendingStudios(page, limit)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	response.Success(c, http.StatusOK, res)
}

func (h *Handler) VerifyStudio(c *gin.Context) {
	if !isAdmin(c) {
		response.Error(c, http.StatusForbidden, "FORBIDDEN", "Admin access required")
		return
	}

	studioID, err := parseIDParam(c, "id")
	if err != nil {
		response.Error(c, http.StatusBadRequest, "INVALID_ID", "Invalid studio ID")
		return
	}

	var req VerifyStudioRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}

	log.Printf("admin action: VerifyStudio studio_id=%d notes=%q", studioID, req.AdminNotes)

	if err := h.service.VerifyStudio(studioID, req.AdminNotes); err != nil {
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	response.Success(c, http.StatusOK, gin.H{"status": "verified"})
}

func (h *Handler) RejectStudio(c *gin.Context) {
	if !isAdmin(c) {
		response.Error(c, http.StatusForbidden, "FORBIDDEN", "Admin access required")
		return
	}

	studioID, err := parseIDParam(c, "id")
	if err != nil {
		response.Error(c, http.StatusBadRequest, "INVALID_ID", "Invalid studio ID")
		return
	}

	var req RejectStudioRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}

	log.Printf("admin action: RejectStudio studio_id=%d reason=%q", studioID, req.Reason)

	if err := h.service.RejectStudio(studioID, req.Reason); err != nil {
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	response.Success(c, http.StatusOK, gin.H{"status": "rejected"})
}

func (h *Handler) GetStatistics(c *gin.Context) {
	if !isAdmin(c) {
		response.Error(c, http.StatusForbidden, "FORBIDDEN", "Admin access required")
		return
	}

	log.Printf("admin action: GetStatistics")

	stats, err := h.service.GetStatistics()
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	response.Success(c, http.StatusOK, stats)
}

func isAdmin(c *gin.Context) bool {
	role, ok := c.Get("role")
	if !ok {
		return false
	}
	rs, ok := role.(string)
	return ok && rs == "admin"
}

func parseIDParam(c *gin.Context, name string) (int64, error) {
	return strconv.ParseInt(c.Param(name), 10, 64)
}

func parseIntDefault(v string, def int) int {
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil || n <= 0 {
		return def
	}
	return n
}
