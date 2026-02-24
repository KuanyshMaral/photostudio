package upload

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Handler handles HTTP requests for file uploads.
// Any authenticated user can upload. Ownership is tracked by user_id.
type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// Upload godoc
// @Summary Upload a file
// @Description Upload any file (image, video, PDF). Returns file ID and public URL.
// @Tags Uploads
// @Accept multipart/form-data
// @Produce json
// @Security BearerAuth
// @Param file formData file true "File to upload"
// @Success 201 {object} map[string]interface{}
// @Failure 400,401,413,500 {object} map[string]interface{}
// @Router /uploads [post]
func (h *Handler) Upload(c *gin.Context) {
	userID := mustUserID(c)
	if userID == 0 {
		return
	}

	fileHeader, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "no file provided"})
		return
	}

	upload, err := h.service.Upload(c.Request.Context(), userID, fileHeader)
	if err != nil {
		switch err {
		case ErrEmptyFile:
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		case ErrFileTooLarge:
			c.JSON(http.StatusRequestEntityTooLarge, gin.H{"success": false, "error": err.Error()})
		case ErrInvalidMimeType:
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "upload failed"})
		}
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data": gin.H{
			"id":         upload.ID,
			"url":        upload.FileURL,
			"name":       upload.OriginalName,
			"mime_type":  upload.MimeType,
			"size":       upload.Size,
			"created_at": upload.CreatedAt,
		},
	})
}

// GetByID godoc
// @Summary Get upload metadata by ID
// @Tags Uploads
// @Produce json
// @Security BearerAuth
// @Param id path string true "Upload ID"
// @Success 200 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Router /uploads/{id} [get]
func (h *Handler) GetByID(c *gin.Context) {
	id := c.Param("id")
	upload, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "upload not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{
		"id":         upload.ID,
		"url":        upload.FileURL,
		"name":       upload.OriginalName,
		"mime_type":  upload.MimeType,
		"size":       upload.Size,
		"created_at": upload.CreatedAt,
	}})
}

// Delete godoc
// @Summary Delete an upload (file + record)
// @Tags Uploads
// @Produce json
// @Security BearerAuth
// @Param id path string true "Upload ID"
// @Success 200 {object} map[string]interface{}
// @Failure 403,404,500 {object} map[string]interface{}
// @Router /uploads/{id} [delete]
func (h *Handler) Delete(c *gin.Context) {
	userID := mustUserID(c)
	if userID == 0 {
		return
	}

	id := c.Param("id")
	if err := h.service.Delete(c.Request.Context(), id, userID); err != nil {
		switch err {
		case ErrUploadNotFound:
			c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "upload not found"})
		case ErrNotOwner:
			c.JSON(http.StatusForbidden, gin.H{"success": false, "error": "you do not own this upload"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "delete failed"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "deleted"})
}

// ListMy godoc
// @Summary List my uploads
// @Tags Uploads
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{}
// @Router /uploads [get]
func (h *Handler) ListMy(c *gin.Context) {
	userID := mustUserID(c)
	if userID == 0 {
		return
	}

	uploads, err := h.service.ListByUser(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "failed to list uploads"})
		return
	}

	items := make([]gin.H, 0, len(uploads))
	for _, u := range uploads {
		items = append(items, gin.H{
			"id":         u.ID,
			"url":        u.FileURL,
			"name":       u.OriginalName,
			"mime_type":  u.MimeType,
			"size":       u.Size,
			"created_at": u.CreatedAt,
		})
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": items})
}

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
