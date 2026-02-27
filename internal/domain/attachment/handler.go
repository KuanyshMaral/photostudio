package attachment

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// Handler handles HTTP requests for the attachment domain.
type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// Attach godoc
// @Summary Attach uploads to an entity
// @Description Links one or more existing upload IDs to a business entity (studio gallery, room gallery, chat message, etc.)
// @Tags Attachments
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body attachRequest true "Attach request"
// @Success 201 {object} map[string]interface{}
// @Failure 400,401,422,500 {object} map[string]interface{}
// @Router /attachments [post]
func (h *Handler) Attach(c *gin.Context) {
	callerID := mustUserID(c)
	if callerID == 0 {
		return
	}

	var req attachRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}
	if len(req.UploadIDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "upload_ids required"})
		return
	}

	results, err := h.service.Attach(
		c.Request.Context(),
		req.UploadIDs,
		callerID,
		TargetType(req.TargetType),
		req.TargetID,
		Metadata{Caption: req.Caption},
	)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidTarget):
			c.JSON(http.StatusUnprocessableEntity, gin.H{"success": false, "error": err.Error()})
		case errors.Is(err, ErrNotOwner):
			c.JSON(http.StatusForbidden, gin.H{"success": false, "error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "attach failed"})
		}
		return
	}

	c.JSON(http.StatusCreated, gin.H{"success": true, "data": results})
}

// ListByTarget godoc
// @Summary List attachments for an entity
// @Tags Attachments
// @Produce json
// @Security BearerAuth
// @Param target_type query string true "Target type (studio_gallery, room_gallery, review_photos, chat_message)"
// @Param target_id query int true "Target entity ID"
// @Success 200 {object} map[string]interface{}
// @Failure 400,422,500 {object} map[string]interface{}
// @Router /attachments [get]
func (h *Handler) ListByTarget(c *gin.Context) {
	targetType := TargetType(c.Query("target_type"))
	targetIDStr := c.Query("target_id")
	targetID, err := strconv.ParseInt(targetIDStr, 10, 64)
	if err != nil || targetID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "valid target_id required"})
		return
	}

	results, err := h.service.ListByTarget(c.Request.Context(), targetType, targetID)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidTarget):
			c.JSON(http.StatusUnprocessableEntity, gin.H{"success": false, "error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "list failed"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": results})
}

// Delete godoc
// @Summary Delete an attachment (unlink upload from entity)
// @Description Removes the link but does NOT delete the underlying file. Use DELETE /uploads/{id} for that.
// @Tags Attachments
// @Produce json
// @Security BearerAuth
// @Param id path int true "Attachment ID"
// @Success 200 {object} map[string]interface{}
// @Failure 403,404,500 {object} map[string]interface{}
// @Router /attachments/{id} [delete]
func (h *Handler) Delete(c *gin.Context) {
	callerID := mustUserID(c)
	if callerID == 0 {
		return
	}

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid attachment id"})
		return
	}

	if err := h.service.Delete(c.Request.Context(), id, callerID); err != nil {
		switch {
		case errors.Is(err, ErrNotFound):
			c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "attachment not found"})
		case errors.Is(err, ErrNotOwner):
			c.JSON(http.StatusForbidden, gin.H{"success": false, "error": "you do not own this attachment"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "delete failed"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "attachment removed"})
}

// Reorder godoc
// @Summary Reorder attachments for an entity
// @Tags Attachments
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body reorderRequest true "Ordered list of attachment IDs"
// @Success 200 {object} map[string]interface{}
// @Failure 400,500 {object} map[string]interface{}
// @Router /attachments/reorder [patch]
func (h *Handler) Reorder(c *gin.Context) {
	callerID := mustUserID(c)
	if callerID == 0 {
		return
	}

	var req reorderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}

	if err := h.service.Reorder(c.Request.Context(), req.IDs); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "reorder failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "reordered"})
}

// ─── Request DTOs ────────────────────────────────────────────────────────────

type attachRequest struct {
	UploadIDs  []string `json:"upload_ids" binding:"required"`
	TargetType string   `json:"target_type" binding:"required"`
	TargetID   int64    `json:"target_id" binding:"required,gt=0"`
	Caption    string   `json:"caption"`
}

type reorderRequest struct {
	IDs []int64 `json:"ids" binding:"required"`
}

// ─── Auth helper ─────────────────────────────────────────────────────────────

func mustUserID(c *gin.Context) int64 {
	id, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "unauthorized"})
		return 0
	}
	switch v := id.(type) {
	case int64:
		return v
	case float64:
		return int64(v)
	}
	c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "invalid user id"})
	return 0
}
