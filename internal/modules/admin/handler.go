package admin

import (
	"log"
	"net/http"
	"strconv"

	"photostudio/internal/pkg/response"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(admin *gin.RouterGroup) {
	// studios moderation
	admin.GET("/studios/pending", h.GetPendingStudios)
	admin.POST("/studios/:id/verify", h.VerifyStudio)
	admin.POST("/studios/:id/reject", h.RejectStudio)

	// statistics
	admin.GET("/statistics", h.GetStatistics)

	// users moderation
	admin.GET("/users", h.GetUsers)
	admin.POST("/users/:id/block", h.BlockUser)
	admin.POST("/users/:id/unblock", h.UnblockUser)

	// reviews moderation
	admin.GET("/reviews", h.GetReviews)
	admin.POST("/reviews/:id/hide", h.HideReview)
	admin.POST("/reviews/:id/show", h.ShowReview)
}

func (h *Handler) GetPendingStudios(c *gin.Context) {
	if !isAdmin(c) {
		response.Error(c, http.StatusForbidden, "FORBIDDEN", "Admin access required")
		return
	}

	page := parseIntDefault(c.Query("page"), 1)
	limit := parseIntDefault(c.Query("limit"), 20)

	log.Printf("admin action: GetPendingStudios page=%d limit=%d", page, limit)

	studios, total, err := h.service.GetPendingStudios(c.Request.Context(), page, limit)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	response.Success(c, http.StatusOK, StudioListResponse{
		Studios: studios,
		Total:   total,
		Page:    page,
		Limit:   limit,
	})
}

func (h *Handler) VerifyStudio(c *gin.Context) {
	if !isAdmin(c) {
		response.Error(c, http.StatusForbidden, "FORBIDDEN", "Admin access required")
		return
	}

	adminID := c.GetInt64("user_id")
	if adminID == 0 {
		response.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
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

	log.Printf("admin action: VerifyStudio admin_id=%d studio_id=%d notes=%q", adminID, studioID, req.AdminNotes)

	studio, err := h.service.VerifyStudio(c.Request.Context(), studioID, adminID, req.AdminNotes)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	response.Success(c, http.StatusOK, studio)
}

func (h *Handler) RejectStudio(c *gin.Context) {
	if !isAdmin(c) {
		response.Error(c, http.StatusForbidden, "FORBIDDEN", "Admin access required")
		return
	}

	adminID := c.GetInt64("user_id")
	if adminID == 0 {
		response.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
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

	log.Printf("admin action: RejectStudio admin_id=%d studio_id=%d reason=%q", adminID, studioID, req.Reason)

	studio, err := h.service.RejectStudio(c.Request.Context(), studioID, adminID, req.Reason)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	response.Success(c, http.StatusOK, studio)
}

func (h *Handler) GetStatistics(c *gin.Context) {
	if !isAdmin(c) {
		response.Error(c, http.StatusForbidden, "FORBIDDEN", "Admin access required")
		return
	}

	log.Printf("admin action: GetStatistics")

	stats, err := h.service.GetStatistics(c.Request.Context())
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	response.Success(c, http.StatusOK, stats)
}

// -------------------- Users --------------------

func (h *Handler) BlockUser(c *gin.Context) {
	if !isAdmin(c) {
		response.Error(c, http.StatusForbidden, "FORBIDDEN", "Admin access required")
		return
	}

	userID, err := parseIDParam(c, "id")
	if err != nil {
		response.Error(c, http.StatusBadRequest, "INVALID_ID", "Invalid user ID")
		return
	}

	var req BlockUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}

	log.Printf("admin action: BlockUser user_id=%d reason=%q", userID, req.Reason)

	u, err := h.service.BlockUser(c.Request.Context(), userID, req.Reason)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	response.Success(c, http.StatusOK, u)
}

func (h *Handler) UnblockUser(c *gin.Context) {
	if !isAdmin(c) {
		response.Error(c, http.StatusForbidden, "FORBIDDEN", "Admin access required")
		return
	}

	userID, err := parseIDParam(c, "id")
	if err != nil {
		response.Error(c, http.StatusBadRequest, "INVALID_ID", "Invalid user ID")
		return
	}

	log.Printf("admin action: UnblockUser user_id=%d", userID)

	u, err := h.service.UnblockUser(c.Request.Context(), userID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	response.Success(c, http.StatusOK, u)
}

func (h *Handler) GetUsers(c *gin.Context) {
	if !isAdmin(c) {
		response.Error(c, http.StatusForbidden, "FORBIDDEN", "Admin access required")
		return
	}

	page := parseIntDefault(c.Query("page"), 1)
	limit := parseIntDefault(c.Query("limit"), 20)

	var filter UserListFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}

	log.Printf("admin action: GetUsers page=%d limit=%d", page, limit)

	users, total, err := h.service.ListUsers(c.Request.Context(), filter, page, limit)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	response.Success(c, http.StatusOK, UserListResponse{
		Users: users,
		Total: total,
		Page:  page,
		Limit: limit,
	})
}

// -------------------- Reviews --------------------

func (h *Handler) GetReviews(c *gin.Context) {
	if !isAdmin(c) {
		response.Error(c, http.StatusForbidden, "FORBIDDEN", "Admin access required")
		return
	}

	page := parseIntDefault(c.Query("page"), 1)
	limit := parseIntDefault(c.Query("limit"), 20)

	var filter ReviewListFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}

	log.Printf("admin action: GetReviews page=%d limit=%d", page, limit)

	reviews, total, err := h.service.ListReviews(c.Request.Context(), filter, page, limit)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	response.Success(c, http.StatusOK, ReviewListResponse{
		Reviews: reviews,
		Total:   total,
		Page:    page,
		Limit:   limit,
	})
}

func (h *Handler) HideReview(c *gin.Context) {
	if !isAdmin(c) {
		response.Error(c, http.StatusForbidden, "FORBIDDEN", "Admin access required")
		return
	}

	reviewID, err := parseIDParam(c, "id")
	if err != nil {
		response.Error(c, http.StatusBadRequest, "INVALID_ID", "Invalid review ID")
		return
	}

	log.Printf("admin action: HideReview review_id=%d", reviewID)

	rv, err := h.service.HideReview(c.Request.Context(), reviewID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	response.Success(c, http.StatusOK, rv)
}

func (h *Handler) ShowReview(c *gin.Context) {
	if !isAdmin(c) {
		response.Error(c, http.StatusForbidden, "FORBIDDEN", "Admin access required")
		return
	}

	reviewID, err := parseIDParam(c, "id")
	if err != nil {
		response.Error(c, http.StatusBadRequest, "INVALID_ID", "Invalid review ID")
		return
	}

	log.Printf("admin action: ShowReview review_id=%d", reviewID)

	rv, err := h.service.ShowReview(c.Request.Context(), reviewID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	response.Success(c, http.StatusOK, rv)
}

// -------------------- helpers --------------------

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
