package catalog

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"

	"photostudio/internal/domain/attachment"
	"photostudio/internal/domain/auth"
	"photostudio/internal/pkg/chicontext"
	"photostudio/internal/pkg/response"
)

type Handler struct {
	service       *Service
	userRepo      *auth.UserRepository
	attachmentSvc *attachment.Service
}

func NewHandler(service *Service, userRepo *auth.UserRepository, attachmentSvc *attachment.Service) *Handler {
	return &Handler{service: service, userRepo: userRepo, attachmentSvc: attachmentSvc}
}

/* ---------- STUDIO HANDLERS ---------- */

// GetStudios
//
//	@Summary		List studios
//	@Description	Get a list of studios with filtering and pagination
//	@Tags			Catalog
//	@Accept			json
//	@Produce		json
//	@Param			city		query		string	false	"City name"
//	@Param			room_type	query		string	false	"Room type"
//	@Param			search		query		string	false	"Search by name"
//	@Param			sort_by		query		string	false	"Sort field (rating, price)"	default(rating)
//	@Param			sort_order	query		string	false	"Sort order (asc, desc)"		default(desc)
//	@Param			min_price	query		number	false	"Minimum price"
//	@Param			max_price	query		number	false	"Maximum price"
//	@Param			limit		query		int		false	"Limit"	default(20)
//	@Param			page		query		int		false	"Page"	default(1)
//	@Success		200			{object}	map[string]interface{}
//	@Failure		400			{object}	map[string]interface{}
//	@Failure		500			{object}	map[string]interface{}
//	@Router			/studios [get]
func (h *Handler) GetStudios(w http.ResponseWriter, r *http.Request) {
	var f StudioFilters

	q := r.URL.Query()
	f.City = q.Get("city")
	f.RoomType = q.Get("room_type")
	f.Search = q.Get("search")
	f.SortBy = q.Get("sort_by")
	if f.SortBy == "" {
		f.SortBy = "rating"
	}
	f.SortOrder = q.Get("sort_order")
	if f.SortOrder == "" {
		f.SortOrder = "desc"
	}

	if minPrice := q.Get("min_price"); minPrice != "" {
		if val, err := strconv.ParseFloat(minPrice, 64); err == nil {
			f.MinPrice = val
		}
	}
	if maxPrice := q.Get("max_price"); maxPrice != "" {
		if val, err := strconv.ParseFloat(maxPrice, 64); err == nil {
			f.MaxPrice = val
		}
	}

	f.Limit = 20
	if limit := q.Get("limit"); limit != "" {
		if val, err := strconv.Atoi(limit); err == nil && val > 0 && val <= 100 {
			f.Limit = val
		}
	}

	f.Offset = 0
	if page := q.Get("page"); page != "" {
		if val, err := strconv.Atoi(page); err == nil && val > 0 {
			f.Offset = (val - 1) * f.Limit
		}
	}

	studios, total, err := h.service.studioRepo.GetAll(r.Context(), f)
	if err != nil {
		handleErrorHTTP(w, r, err)
		return
	}

	for i := range studios {
		h.enrichStudioWithAttachments(r.Context(), &studios[i])
	}

	totalPages := (int(total) + f.Limit - 1) / f.Limit
	currentPage := (f.Offset / f.Limit) + 1

	response.JSON(w, http.StatusOK, response.H{
		"success": true,
		"data": response.H{
			"studios": studios,
			"pagination": response.H{
				"page":        currentPage,
				"limit":       f.Limit,
				"total":       total,
				"total_pages": totalPages,
			},
		},
	})
}

// GetCities
//
//	@Summary		List cities
//	@Description	Get a list of unique cities where active studios are located
//	@Tags			References
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	map[string]interface{}
//	@Failure		500	{object}	map[string]interface{}
//	@Router			/references/cities [get]
func (h *Handler) GetCities(w http.ResponseWriter, r *http.Request) {
	var cities []string
	err := h.service.studioRepo.DB().Table("studios").
		Where("deleted_at IS NULL").
		Where("is_active = true").
		Distinct("city").
		Where("city IS NOT NULL AND city != ''").
		Pluck("city", &cities).Error

	if err != nil {
		response.CustomError(w, r, http.StatusInternalServerError, "INTERNAL", "Failed to fetch cities")
		return
	}

	response.Success(w, http.StatusOK, response.H{"cities": cities})
}

// GetStudioByID
//
//	@Summary		Get studio by ID
//	@Description	Get detailed information about a specific studio
//	@Tags			Catalog
//	@Accept			json
//	@Produce		json
//	@Param			id	path		int	true	"Studio ID"
//	@Success		200	{object}	map[string]interface{}
//	@Failure		400	{object}	map[string]interface{}
//	@Failure		404	{object}	map[string]interface{}
//	@Failure		500	{object}	map[string]interface{}
//	@Router			/studios/{id} [get]
func (h *Handler) GetStudioByID(w http.ResponseWriter, r *http.Request) {
	studioID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		response.JSON(w, http.StatusBadRequest, response.H{
			"success": false,
			"error":   response.H{"code": "INVALID_ID", "message": "Invalid studio ID"},
		})
		return
	}

	studio, err := h.service.studioRepo.GetByID(r.Context(), studioID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.JSON(w, http.StatusNotFound, response.H{
				"success": false,
				"error":   response.H{"code": "NOT_FOUND", "message": "Studio not found"},
			})
			return
		}
		handleErrorHTTP(w, r, err)
		return
	}

	h.enrichStudioWithAttachments(r.Context(), studio)

	response.JSON(w, http.StatusOK, response.H{
		"success": true,
		"data":    response.H{"studio": studio},
	})
}

// GetStudioWorkingHours
//
//	@Summary		Get studio working hours (Legacy)
//	@Description	Get today's working hours for a studio
//	@Tags			Catalog
//	@Accept			json
//	@Produce		json
//	@Param			id	path		int	true	"Studio ID"
//	@Success		200	{object}	map[string]interface{}
//	@Failure		400	{object}	map[string]interface{}
//	@Failure		404	{object}	map[string]interface{}
//	@Failure		500	{object}	map[string]interface{}
//	@Router			/studios/{id}/working-hours [get]
func (h *Handler) GetStudioWorkingHours(w http.ResponseWriter, r *http.Request) {
	studioID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		response.CustomError(w, r, http.StatusBadRequest, "INVALID_ID", "Invalid studio ID")
		return
	}

	status, err := h.service.GetStudioWorkingStatus(r.Context(), studioID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.CustomError(w, r, http.StatusNotFound, "NOT_FOUND", "Studio not found")
			return
		}
		response.CustomError(w, r, http.StatusInternalServerError, "FETCH_FAILED", err)
		return
	}

	response.Success(w, http.StatusOK, response.H{
		"is_open":       status.IsOpen,
		"message":       status.Message,
		"open_time":     status.OpenTime,
		"close_time":    status.CloseTime,
		"working_hours": status.WorkingHours,
	})
}

// GetStudioWorkingHoursV2
//
//	@Summary		Get studio full working hours
//	@Description	Get comprehensive 7-day schedule for a studio
//	@Tags			Catalog
//	@Accept			json
//	@Produce		json
//	@Param			id	path		int	true	"Studio ID"
//	@Success		200	{object}	map[string]interface{}
//	@Failure		400	{object}	map[string]interface{}
//	@Failure		404	{object}	map[string]interface{}
//	@Failure		500	{object}	map[string]interface{}
//	@Router			/studios/{id}/working-hours/v2 [get]
func (h *Handler) GetStudioWorkingHoursV2(w http.ResponseWriter, r *http.Request) {
	studioID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		response.CustomError(w, r, http.StatusBadRequest, "INVALID_ID", "Invalid studio ID")
		return
	}

	hoursResponse, err := h.service.GetStudioWorkingHours(r.Context(), studioID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.CustomError(w, r, http.StatusNotFound, "NOT_FOUND", "Studio not found")
			return
		}
		response.CustomError(w, r, http.StatusInternalServerError, "FETCH_FAILED", err)
		return
	}

	response.Success(w, http.StatusOK, hoursResponse)
}

// UpdateStudioWorkingHours
//
//	@Summary		Update studio working hours
//	@Description	Set the 7-day schedule for a studio (Owner only)
//	@Tags			Catalog
//	@Accept			json
//	@Produce		json
//	@Param			id		path		int				true	"Studio ID"
//	@Param			hours	body		[]WorkingHours	true	"Working hours schedule"
//	@Success		200		{object}	map[string]interface{}
//	@Failure		400		{object}	map[string]interface{}
//	@Failure		401		{object}	map[string]interface{}
//	@Failure		403		{object}	map[string]interface{}
//	@Failure		500		{object}	map[string]interface{}
//	@Security		BearerAuth
//	@Router			/studios/{id}/working-hours [put]
func (h *Handler) UpdateStudioWorkingHours(w http.ResponseWriter, r *http.Request) {
	studioID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		response.CustomError(w, r, http.StatusBadRequest, "INVALID_ID", "Invalid studio ID")
		return
	}

	var hours []WorkingHours
	if err := response.BindJSON(r, &hours); err != nil {
		response.CustomError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", err)
		return
	}

	userID := chicontext.UserIDFromCtx(r.Context())
	if userID == 0 {
		response.CustomError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
		return
	}

	if err = h.service.UpdateStudioWorkingHours(r.Context(), userID, studioID, hours); err != nil {
		if errors.Is(err, ErrForbidden) {
			response.CustomError(w, r, http.StatusForbidden, "FORBIDDEN", "You don't own this studio")
			return
		}
		response.CustomError(w, r, http.StatusBadRequest, "UPDATE_ERROR", err)
		return
	}

	response.Success(w, http.StatusOK, response.H{"message": "Working hours updated"})
}

// GetMyStudios
//
//	@Summary		Get my studios
//	@Description	Get a list of studios owned by the authenticated user
//	@Tags			Catalog
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	map[string]interface{}
//	@Failure		401	{object}	map[string]interface{}
//	@Failure		500	{object}	map[string]interface{}
//	@Security		BearerAuth
//	@Router			/studios/my [get]
func (h *Handler) GetMyStudios(w http.ResponseWriter, r *http.Request) {
	userID := chicontext.UserIDFromCtx(r.Context())
	if userID == 0 {
		response.CustomError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
		return
	}

	studios, err := h.service.GetStudiosByOwner(r.Context(), userID)
	if err != nil {
		response.CustomError(w, r, http.StatusInternalServerError, "FETCH_FAILED", "Failed to get studios")
		return
	}

	for i := range studios {
		h.enrichStudioWithAttachments(r.Context(), &studios[i])
	}

	response.Success(w, http.StatusOK, response.H{"studios": studios})
}

// CreateStudio
//
//	@Summary		Create studio
//	@Description	Create a new studio (Owner only)
//	@Tags			Catalog
//	@Accept			json
//	@Produce		json
//	@Param			request	body		CreateStudioRequest	true	"Studio details"
//	@Success		201		{object}	map[string]interface{}
//	@Failure		400		{object}	map[string]interface{}
//	@Failure		401		{object}	map[string]interface{}
//	@Failure		403		{object}	map[string]interface{}
//	@Failure		500		{object}	map[string]interface{}
//	@Security		BearerAuth
//	@Router			/studios [post]
func (h *Handler) CreateStudio(w http.ResponseWriter, r *http.Request) {
	var req CreateStudioRequest
	if err := response.BindJSON(r, &req); err != nil {
		response.JSON(w, http.StatusBadRequest, response.H{
			"success": false,
			"error":   response.H{"code": "INVALID_REQUEST", "message": "Invalid request body", "details": err.Error()},
		})
		return
	}

	userID := chicontext.UserIDFromCtx(r.Context())
	if userID == 0 {
		response.JSON(w, http.StatusUnauthorized, response.H{
			"success": false,
			"error":   response.H{"code": "UNAUTHORIZED", "message": "User not authenticated"},
		})
		return
	}

	userObj, err := h.userRepo.GetByID(r.Context(), userID)
	if err != nil {
		response.JSON(w, http.StatusInternalServerError, response.H{
			"success": false,
			"error":   response.H{"code": "INTERNAL_ERROR", "message": "Failed to load user"},
		})
		return
	}

	studio, err := h.service.CreateStudio(r.Context(), userObj, req)
	if err != nil {
		if errors.Is(err, ErrForbidden) {
			response.CustomError(w, r, http.StatusForbidden, "FORBIDDEN", "Only verified studio owners can create studios")
			return
		}
		handleErrorHTTP(w, r, err)
		return
	}

	if len(req.UploadIDs) > 0 && h.attachmentSvc != nil {
		if attached, attachErr := h.attachmentSvc.Attach(
			r.Context(), req.UploadIDs, userID,
			attachment.TargetStudioGallery, studio.ID, attachment.Metadata{},
		); attachErr == nil {
			studio.Attachments = toAttachmentURLs(attached)
		}
	} else {
		h.enrichStudioWithAttachments(r.Context(), studio)
	}

	response.JSON(w, http.StatusCreated, response.H{
		"success": true,
		"data":    response.H{"studio": studio},
		"message": "Studio created successfully",
	})
}

// UpdateStudio
//
//	@Summary		Update studio
//	@Description	Update an existing studio's details (Owner only)
//	@Tags			Catalog
//	@Accept			json
//	@Produce		json
//	@Param			id		path		int					true	"Studio ID"
//	@Param			request	body		UpdateStudioRequest	true	"Studio details"
//	@Success		200		{object}	map[string]interface{}
//	@Failure		400		{object}	map[string]interface{}
//	@Failure		401		{object}	map[string]interface{}
//	@Failure		403		{object}	map[string]interface{}
//	@Failure		404		{object}	map[string]interface{}
//	@Failure		500		{object}	map[string]interface{}
//	@Security		BearerAuth
//	@Router			/studios/{id} [put]
func (h *Handler) UpdateStudio(w http.ResponseWriter, r *http.Request) {
	studioID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		response.JSON(w, http.StatusBadRequest, response.H{
			"success": false,
			"error":   response.H{"code": "INVALID_ID", "message": "Invalid studio ID"},
		})
		return
	}

	var req UpdateStudioRequest
	if err := response.BindJSON(r, &req); err != nil {
		response.JSON(w, http.StatusBadRequest, response.H{
			"success": false,
			"error":   response.H{"code": "INVALID_REQUEST", "message": "Invalid request body", "details": err.Error()},
		})
		return
	}

	userID := chicontext.UserIDFromCtx(r.Context())
	if userID == 0 {
		response.JSON(w, http.StatusUnauthorized, response.H{
			"success": false,
			"error":   response.H{"code": "UNAUTHORIZED", "message": "User not authenticated"},
		})
		return
	}

	studio, err := h.service.UpdateStudio(r.Context(), userID, studioID, req)
	if err != nil {
		if errors.Is(err, ErrForbidden) {
			response.CustomError(w, r, http.StatusForbidden, "FORBIDDEN", "You don't have permission to update this studio")
			return
		}
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.CustomError(w, r, http.StatusNotFound, "NOT_FOUND", "Studio not found")
			return
		}
		handleErrorHTTP(w, r, err)
		return
	}

	if len(req.UploadIDs) > 0 && h.attachmentSvc != nil {
		if attached, attachErr := h.attachmentSvc.Attach(
			r.Context(), req.UploadIDs, userID,
			attachment.TargetStudioGallery, studio.ID, attachment.Metadata{},
		); attachErr == nil {
			studio.Attachments = toAttachmentURLs(attached)
		}
	} else {
		h.enrichStudioWithAttachments(r.Context(), studio)
	}

	response.JSON(w, http.StatusOK, response.H{
		"success": true,
		"data":    response.H{"studio": studio},
		"message": "Studio updated successfully",
	})
}

/* ---------- ROOM HANDLERS ---------- */

// UpdateRoom
//
//	@Summary		Update room
//	@Description	Update an existing room's details (Owner only)
//	@Tags			Catalog
//	@Accept			json
//	@Produce		json
//	@Param			id		path		int					true	"Room ID"
//	@Param			request	body		UpdateRoomRequest	true	"Room details"
//	@Success		200		{object}	map[string]interface{}
//	@Failure		400		{object}	map[string]interface{}
//	@Failure		401		{object}	map[string]interface{}
//	@Failure		403		{object}	map[string]interface{}
//	@Failure		404		{object}	map[string]interface{}
//	@Failure		500		{object}	map[string]interface{}
//	@Security		BearerAuth
//	@Router			/rooms/{id} [put]
func (h *Handler) UpdateRoom(w http.ResponseWriter, r *http.Request) {
	roomID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		response.CustomError(w, r, http.StatusBadRequest, "INVALID_ID", "Invalid room ID")
		return
	}

	var req UpdateRoomRequest
	if err := response.BindJSON(r, &req); err != nil {
		response.CustomError(w, r, http.StatusBadRequest, "INVALID_REQUEST", err)
		return
	}

	room, err := h.service.UpdateRoom(r.Context(), roomID, req)
	if err != nil {
		if errors.Is(err, ErrInvalidRoomType) {
			response.CustomError(w, r, http.StatusBadRequest, "INVALID_ROOM_TYPE", err)
			return
		}
		handleErrorHTTP(w, r, err)
		return
	}

	userID := chicontext.UserIDFromCtx(r.Context())
	if len(req.UploadIDs) > 0 && h.attachmentSvc != nil && userID != 0 {
		if attached, attachErr := h.attachmentSvc.Attach(
			r.Context(), req.UploadIDs, userID,
			attachment.TargetRoomGallery, room.ID, attachment.Metadata{},
		); attachErr == nil {
			room.Attachments = toAttachmentURLs(attached)
		}
	} else {
		h.enrichRoomWithAttachments(r.Context(), room)
	}

	response.Success(w, http.StatusOK, response.H{"room": room})
}

// DeleteRoom
//
//	@Summary		Delete room
//	@Description	Delete a room by ID (Owner only)
//	@Tags			Catalog
//	@Accept			json
//	@Produce		json
//	@Param			id	path		int	true	"Room ID"
//	@Success		200	{object}	map[string]interface{}
//	@Failure		400	{object}	map[string]interface{}
//	@Failure		401	{object}	map[string]interface{}
//	@Failure		403	{object}	map[string]interface{}
//	@Failure		404	{object}	map[string]interface{}
//	@Failure		500	{object}	map[string]interface{}
//	@Security		BearerAuth
//	@Router			/rooms/{id} [delete]
func (h *Handler) DeleteRoom(w http.ResponseWriter, r *http.Request) {
	roomID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		response.CustomError(w, r, http.StatusBadRequest, "INVALID_ID", "Invalid room ID")
		return
	}

	if err := h.service.DeleteRoom(r.Context(), roomID); err != nil {
		handleErrorHTTP(w, r, err)
		return
	}

	response.Success(w, http.StatusOK, response.H{"deleted": true})
}

/* ---------- PHOTO HANDLERS ---------- */

// UploadStudioPhotos
//
//	@Summary		Upload studio photos
//	@Description	Upload multiple photos for a studio (Legacy)
//	@Tags			Catalog
//	@Accept			multipart/form-data
//	@Produce		json
//	@Param			id		path		int		true	"Studio ID"
//	@Param			photos	formData	file	true	"Photos to upload (max 10 allowed, jpeg/png/webp, ≤5MB each)"
//	@Success		200		{object}	map[string]interface{}
//	@Failure		400		{object}	map[string]interface{}
//	@Failure		401		{object}	map[string]interface{}
//	@Failure		403		{object}	map[string]interface{}
//	@Failure		500		{object}	map[string]interface{}
//	@Security		BearerAuth
//	@Router			/studios/{id}/photos [post]
func (h *Handler) UploadStudioPhotos(w http.ResponseWriter, r *http.Request) {
	studioID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		response.CustomError(w, r, http.StatusBadRequest, "INVALID_ID", "Invalid studio ID")
		return
	}

	userID := chicontext.UserIDFromCtx(r.Context())
	if userID == 0 {
		response.CustomError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "Missing user_id in context")
		return
	}

	if err := r.ParseMultipartForm(50 << 20); err != nil {
		response.CustomError(w, r, http.StatusBadRequest, "INVALID_FORM", "Invalid multipart form")
		return
	}

	files := r.MultipartForm.File["photos"]
	if len(files) == 0 {
		response.CustomError(w, r, http.StatusBadRequest, "NO_FILES", "No files provided")
		return
	}
	if len(files) > 10 {
		files = files[:10]
	}

	uploadDir := fmt.Sprintf("./uploads/studios/%d", studioID)
	if err := os.MkdirAll(uploadDir, os.ModePerm); err != nil {
		response.CustomError(w, r, http.StatusInternalServerError, "UPLOAD_DIR_ERROR", err)
		return
	}

	var uploadedURLs []string
	for _, fh := range files {
		if fh.Size > 5*1024*1024 {
			continue
		}
		ext := strings.ToLower(filepath.Ext(fh.Filename))
		if ext == ".jpeg" {
			ext = ".jpg"
		}
		if ext != ".jpg" && ext != ".png" && ext != ".webp" {
			continue
		}

		newName := fmt.Sprintf("%d%s", time.Now().UnixNano(), ext)
		savePath := filepath.Join(uploadDir, newName)

		src, err := fh.Open()
		if err != nil {
			continue
		}
		dst, err := os.Create(savePath)
		if err != nil {
			_ = src.Close()
			continue
		}
		_, _ = io.Copy(dst, src)
		_ = src.Close()
		_ = dst.Close()

		url := fmt.Sprintf("/static/studios/%d/%s", studioID, newName)
		uploadedURLs = append(uploadedURLs, url)
	}

	if len(uploadedURLs) == 0 {
		response.CustomError(w, r, http.StatusBadRequest, "NO_VALID_FILES", "No valid files uploaded")
		return
	}

	if err := h.service.AddStudioPhotos(r.Context(), userID, studioID, uploadedURLs); err != nil {
		response.CustomError(w, r, http.StatusBadRequest, "PHOTO_UPLOAD_ERROR", err)
		return
	}

	response.Success(w, http.StatusOK, response.H{"uploaded": len(uploadedURLs), "urls": uploadedURLs})
}

/* ---------- ROOM LIST/CRUD ---------- */

// GetRooms
//
//	@Summary		List rooms
//	@Description	Get a list of rooms, optionally filtered by studio ID
//	@Tags			Catalog
//	@Accept			json
//	@Produce		json
//	@Param			studio_id	query		int	false	"Studio ID filter"
//	@Success		200			{object}	map[string]interface{}
//	@Failure		400			{object}	map[string]interface{}
//	@Failure		500			{object}	map[string]interface{}
//	@Router			/rooms [get]
func (h *Handler) GetRooms(w http.ResponseWriter, r *http.Request) {
	var studioIDPtr *int64
	if studioIDStr := r.URL.Query().Get("studio_id"); studioIDStr != "" {
		if studioID, err := strconv.ParseInt(studioIDStr, 10, 64); err == nil {
			studioIDPtr = &studioID
		}
	}

	rooms, err := h.service.roomRepo.GetAll(r.Context(), studioIDPtr)
	if err != nil {
		handleErrorHTTP(w, r, err)
		return
	}

	for i := range rooms {
		h.enrichRoomWithAttachments(r.Context(), &rooms[i])
	}

	response.JSON(w, http.StatusOK, response.H{
		"success": true,
		"data":    response.H{"rooms": rooms},
	})
}

// GetRoomByID
//
//	@Summary		Get room by ID
//	@Description	Get detailed information about a specific room
//	@Tags			Catalog
//	@Accept			json
//	@Produce		json
//	@Param			id	path		int	true	"Room ID"
//	@Success		200	{object}	map[string]interface{}
//	@Failure		400	{object}	map[string]interface{}
//	@Failure		404	{object}	map[string]interface{}
//	@Failure		500	{object}	map[string]interface{}
//	@Router			/rooms/{id} [get]
func (h *Handler) GetRoomByID(w http.ResponseWriter, r *http.Request) {
	roomID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		response.JSON(w, http.StatusBadRequest, response.H{
			"success": false,
			"error":   response.H{"code": "INVALID_ID", "message": "Invalid room ID"},
		})
		return
	}

	room, err := h.service.roomRepo.GetByID(r.Context(), roomID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.JSON(w, http.StatusNotFound, response.H{
				"success": false,
				"error":   response.H{"code": "NOT_FOUND", "message": "Room not found"},
			})
			return
		}
		handleErrorHTTP(w, r, err)
		return
	}

	h.enrichRoomWithAttachments(r.Context(), room)

	response.JSON(w, http.StatusOK, response.H{
		"success": true,
		"data":    response.H{"room": room},
	})
}

// CreateRoom
//
//	@Summary		Create room
//	@Description	Add a new room to a studio (Owner only)
//	@Tags			Catalog
//	@Accept			json
//	@Produce		json
//	@Param			id		path		int					true	"Studio ID"
//	@Param			request	body		CreateRoomRequest	true	"Room details"
//	@Success		201		{object}	map[string]interface{}
//	@Failure		400		{object}	map[string]interface{}
//	@Failure		401		{object}	map[string]interface{}
//	@Failure		403		{object}	map[string]interface{}
//	@Failure		404		{object}	map[string]interface{}
//	@Failure		500		{object}	map[string]interface{}
//	@Security		BearerAuth
//	@Router			/studios/{id}/rooms [post]
func (h *Handler) CreateRoom(w http.ResponseWriter, r *http.Request) {
	studioID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		response.JSON(w, http.StatusBadRequest, response.H{
			"success": false,
			"error":   response.H{"code": "INVALID_ID", "message": "Invalid studio ID"},
		})
		return
	}

	var req CreateRoomRequest
	if err := response.BindJSON(r, &req); err != nil {
		response.JSON(w, http.StatusBadRequest, response.H{
			"success": false,
			"error":   response.H{"code": "INVALID_REQUEST", "message": "Invalid request body", "details": err.Error()},
		})
		return
	}

	userID := chicontext.UserIDFromCtx(r.Context())
	if userID == 0 {
		response.JSON(w, http.StatusUnauthorized, response.H{
			"success": false,
			"error":   response.H{"code": "UNAUTHORIZED", "message": "User not authenticated"},
		})
		return
	}

	room, err := h.service.CreateRoom(r.Context(), userID, studioID, req)
	if err != nil {
		if errors.Is(err, ErrInvalidRoomType) {
			response.JSON(w, http.StatusBadRequest, response.H{
				"success": false,
				"error":   response.H{"code": "INVALID_ROOM_TYPE", "message": "Invalid room type. Must be one of: Fashion, Portrait, Creative, Commercial"},
			})
			return
		}
		if errors.Is(err, ErrForbidden) {
			response.JSON(w, http.StatusForbidden, response.H{
				"success": false,
				"error":   response.H{"code": "FORBIDDEN", "message": "You don't have permission to add rooms to this studio"},
			})
			return
		}
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.JSON(w, http.StatusNotFound, response.H{
				"success": false,
				"error":   response.H{"code": "NOT_FOUND", "message": "Studio not found"},
			})
			return
		}
		handleErrorHTTP(w, r, err)
		return
	}

	if len(req.UploadIDs) > 0 && h.attachmentSvc != nil {
		if attached, attachErr := h.attachmentSvc.Attach(
			r.Context(), req.UploadIDs, userID,
			attachment.TargetRoomGallery, room.ID, attachment.Metadata{},
		); attachErr == nil {
			room.Attachments = toAttachmentURLs(attached)
		}
	} else {
		h.enrichRoomWithAttachments(r.Context(), room)
	}

	response.JSON(w, http.StatusCreated, response.H{
		"success": true,
		"data":    response.H{"room": room},
		"message": "Room created successfully",
	})
}

/* ---------- ROOM TYPES ---------- */

// GetRoomTypes
//
//	@Summary		Get room types
//	@Description	Get a list of available predefined room types
//	@Tags			Catalog
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	map[string]interface{}
//	@Router			/rooms/types [get]
func (h *Handler) GetRoomTypes(w http.ResponseWriter, r *http.Request) {
	types := ValidRoomTypes()
	typeStrings := make([]string, len(types))
	for i, t := range types {
		typeStrings[i] = string(t)
	}

	response.JSON(w, http.StatusOK, response.H{
		"success": true,
		"data":    response.H{"room_types": typeStrings},
	})
}

/* ---------- EQUIPMENT HANDLERS ---------- */

// AddEquipment
//
//	@Summary		Add equipment
//	@Description	Add new equipment to a room (Owner only)
//	@Tags			Catalog
//	@Accept			json
//	@Produce		json
//	@Param			id		path		int						true	"Room ID"
//	@Param			request	body		CreateEquipmentRequest	true	"Equipment details"
//	@Success		201		{object}	map[string]interface{}
//	@Failure		400		{object}	map[string]interface{}
//	@Failure		401		{object}	map[string]interface{}
//	@Failure		403		{object}	map[string]interface{}
//	@Failure		404		{object}	map[string]interface{}
//	@Failure		500		{object}	map[string]interface{}
//	@Security		BearerAuth
//	@Router			/rooms/{id}/equipment [post]
func (h *Handler) AddEquipment(w http.ResponseWriter, r *http.Request) {
	roomID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		response.JSON(w, http.StatusBadRequest, response.H{
			"success": false,
			"error":   response.H{"code": "INVALID_ID", "message": "Invalid room ID"},
		})
		return
	}

	var req CreateEquipmentRequest
	if err := response.BindJSON(r, &req); err != nil {
		response.JSON(w, http.StatusBadRequest, response.H{
			"success": false,
			"error":   response.H{"code": "INVALID_REQUEST", "message": "Invalid request body", "details": err.Error()},
		})
		return
	}

	userID := chicontext.UserIDFromCtx(r.Context())
	if userID == 0 {
		response.JSON(w, http.StatusUnauthorized, response.H{
			"success": false,
			"error":   response.H{"code": "UNAUTHORIZED", "message": "User not authenticated"},
		})
		return
	}

	equipment, err := h.service.AddEquipment(r.Context(), userID, roomID, req)
	if err != nil {
		if errors.Is(err, ErrForbidden) {
			response.JSON(w, http.StatusForbidden, response.H{
				"success": false,
				"error":   response.H{"code": "FORBIDDEN", "message": "You don't have permission to add equipment to this room"},
			})
			return
		}
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.JSON(w, http.StatusNotFound, response.H{
				"success": false,
				"error":   response.H{"code": "NOT_FOUND", "message": "Room not found"},
			})
			return
		}
		handleErrorHTTP(w, r, err)
		return
	}

	response.JSON(w, http.StatusCreated, response.H{
		"success": true,
		"data":    response.H{"equipment": equipment},
		"message": "Equipment added successfully",
	})
}

/* ---------- ERROR HANDLING ---------- */

func handleErrorHTTP(w http.ResponseWriter, r *http.Request, err error) {
	if err == nil {
		return
	}
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		response.CustomError(w, r, http.StatusNotFound, "NOT_FOUND", "Resource not found")
	case errors.Is(err, ErrForbidden):
		response.CustomError(w, r, http.StatusForbidden, "FORBIDDEN", "You don't have permission to perform this action")
	default:
		response.CustomError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", err)
	}
}

/* ---------- ATTACHMENT ENRICHMENT ---------- */

func toAttachmentURLs(attachments []*attachment.AttachmentWithURL) []AttachmentURL {
	out := make([]AttachmentURL, len(attachments))
	for i, a := range attachments {
		out[i] = AttachmentURL{
			ID: a.ID, URL: a.URL,
			OriginalName: a.OriginalName, MimeType: a.MimeType,
			SortOrder: a.SortOrder,
		}
	}
	return out
}

func (h *Handler) enrichStudioWithAttachments(ctx context.Context, studio *Studio) {
	if h.attachmentSvc == nil {
		return
	}
	attachments, err := h.attachmentSvc.ListByTarget(ctx, attachment.TargetStudioGallery, studio.ID)
	if err == nil {
		studio.Attachments = make([]AttachmentURL, len(attachments))
		for i, a := range attachments {
			studio.Attachments[i] = AttachmentURL{
				ID: a.ID, URL: a.URL,
				OriginalName: a.OriginalName, MimeType: a.MimeType,
				SortOrder: a.SortOrder, Caption: a.Metadata.Caption,
			}
		}
	}
}

func (h *Handler) enrichRoomWithAttachments(ctx context.Context, room *Room) {
	if h.attachmentSvc == nil {
		return
	}
	attachments, err := h.attachmentSvc.ListByTarget(ctx, attachment.TargetRoomGallery, room.ID)
	if err == nil {
		room.Attachments = make([]AttachmentURL, len(attachments))
		for i, a := range attachments {
			room.Attachments[i] = AttachmentURL{
				ID: a.ID, URL: a.URL,
				OriginalName: a.OriginalName, MimeType: a.MimeType,
				SortOrder: a.SortOrder, Caption: a.Metadata.Caption,
			}
		}
	}
}
