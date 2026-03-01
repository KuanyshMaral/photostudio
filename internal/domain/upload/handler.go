package upload

import (
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"

	"photostudio/internal/pkg/chicontext"
	"photostudio/internal/pkg/response"
)

// swaggerUploadData is a wrapper strictly for generating Swagger documentation.
type swaggerUploadData struct {
	ID        string `json:"id"`
	URL       string `json:"url"`
	Name      string `json:"name"`
	MimeType  string `json:"mime_type"`
	Size      int64  `json:"size"`
	CreatedAt string `json:"created_at"`
}

// swaggerUploadResponse is a wrapper strictly for generating Swagger documentation.
type swaggerUploadResponse struct {
	Success bool              `json:"success"`
	Data    swaggerUploadData `json:"data"`
}

// swaggerListUploadResponse is a wrapper strictly for generating Swagger documentation.
type swaggerListUploadResponse struct {
	Success bool                `json:"success"`
	Data    []swaggerUploadData `json:"data"`
}

// swaggerDeleteResponse is a wrapper strictly for generating Swagger documentation.
type swaggerDeleteResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// Handler handles HTTP requests for file uploads.
// Any authenticated user can upload. Ownership is tracked by user_id.
type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// Upload загрузка одного файла.
//
//	@Summary		Загрузка файла
//	@Description	Принимает multipart/form-data с полем "file". Возвращает метаданные загруженного файла.
//	@Tags			Uploads
//	@Accept			multipart/form-data
//	@Produce		json
//	@Security		BearerAuth
//	@Param			file			formData	file					true	"Файл для загрузки"
//	@Success		201				{object}	swaggerUploadResponse	"Успешная загрузка"
//	@Failure		400,401,413,422	{object}	response.ErrorResponse	"Ошибка запроса"
//	@Failure		500				{object}	response.ErrorResponse	"Внутренняя ошибка сервера"
//	@Router			/uploads [post]
func (h *Handler) Upload(w http.ResponseWriter, r *http.Request) {
	userID := chicontext.UserIDFromCtx(r.Context())
	if userID == 0 {
		response.CustomError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "unauthorized")
		return
	}

	if err := r.ParseMultipartForm(32 << 20); err != nil { // 32 MB max memory
		response.CustomError(w, r, http.StatusBadRequest, "INVALID_FORM", "failed to parse multipart form")
		return
	}

	f, fh, err := r.FormFile("file")
	if err != nil {
		response.CustomError(w, r, http.StatusBadRequest, "NO_FILE", "no file provided")
		return
	}
	defer f.Close()

	// Reconstruct a multipart.FileHeader-compatible value via the service.
	// The existing Service.Upload accepts *multipart.FileHeader directly.
	upload, err := h.service.Upload(r.Context(), userID, fh)
	if err != nil {
		switch err {
		case ErrEmptyFile:
			response.CustomError(w, r, http.StatusBadRequest, "EMPTY_FILE", err.Error())
		case ErrFileTooLarge:
			response.CustomError(w, r, http.StatusRequestEntityTooLarge, "FILE_TOO_LARGE", err.Error())
		case ErrInvalidMimeType:
			response.CustomError(w, r, http.StatusBadRequest, "INVALID_MIME", err.Error())
		default:
			response.CustomError(w, r, http.StatusInternalServerError, "UPLOAD_FAILED", "upload failed")
		}
		return
	}

	response.JSON(w, http.StatusCreated, response.H{
		"success": true,
		"data": response.H{
			"id":         upload.ID,
			"url":        upload.FileURL,
			"name":       upload.OriginalName,
			"mime_type":  upload.MimeType,
			"size":       upload.Size,
			"created_at": upload.CreatedAt,
		},
	})
}

// GetByID получение информации о загруженном файле по ID.
//
//	@Summary		Получить метаданные файла
//	@Description	Возвращает информацию о ранее загруженном файле по его идентификатору.
//	@Tags			Uploads
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		string					true	"ID файла"
//	@Success		200	{object}	swaggerUploadResponse	"Успех"
//	@Failure		401	{object}	response.ErrorResponse	"Не авторизован"
//	@Failure		404	{object}	response.ErrorResponse	"Файл не найден"
//	@Router			/uploads/{id} [get]
func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	upload, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		response.CustomError(w, r, http.StatusNotFound, "NOT_FOUND", "upload not found")
		return
	}
	response.JSON(w, http.StatusOK, response.H{
		"success": true,
		"data": response.H{
			"id":         upload.ID,
			"url":        upload.FileURL,
			"name":       upload.OriginalName,
			"mime_type":  upload.MimeType,
			"size":       upload.Size,
			"created_at": upload.CreatedAt,
		},
	})
}

// Delete удаление загруженного файла.
//
//	@Summary		Удалить файл
//	@Description	Удаляет файл и его метаданные. Доступно только владельцу файла.
//	@Tags			Uploads
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		string					true	"ID файла"
//	@Success		200		{object}	swaggerDeleteResponse	"Успешное удаление"
//	@Failure		401,403	{object}	response.ErrorResponse	"Ошибка прав доступа"
//	@Failure		404		{object}	response.ErrorResponse	"Файл не найден"
//	@Failure		500		{object}	response.ErrorResponse	"Внутренняя ошибка сервера"
//	@Router			/uploads/{id} [delete]
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	userID := chicontext.UserIDFromCtx(r.Context())
	if userID == 0 {
		response.CustomError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "unauthorized")
		return
	}

	id := chi.URLParam(r, "id")
	if err := h.service.Delete(r.Context(), id, userID); err != nil {
		switch err {
		case ErrUploadNotFound:
			response.CustomError(w, r, http.StatusNotFound, "NOT_FOUND", "upload not found")
		case ErrNotOwner:
			response.CustomError(w, r, http.StatusForbidden, "FORBIDDEN", "you do not own this upload")
		default:
			response.CustomError(w, r, http.StatusInternalServerError, "DELETE_FAILED", "delete failed")
		}
		return
	}

	response.JSON(w, http.StatusOK, response.H{"success": true, "message": "deleted"})
}

// ListMy список загруженных файлов текущего пользователя.
//
//	@Summary		Мои файлы
//	@Description	Возвращает список всех файлов, загруженных текущим авторизованным пользователем.
//	@Tags			Uploads
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	swaggerListUploadResponse	"Успех"
//	@Failure		401	{object}	response.ErrorResponse		"Не авторизован"
//	@Failure		500	{object}	response.ErrorResponse		"Внутренняя ошибка сервера"
//	@Router			/uploads [get]
func (h *Handler) ListMy(w http.ResponseWriter, r *http.Request) {
	userID := chicontext.UserIDFromCtx(r.Context())
	if userID == 0 {
		response.CustomError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "unauthorized")
		return
	}

	uploads, err := h.service.ListByUser(r.Context(), userID)
	if err != nil {
		response.CustomError(w, r, http.StatusInternalServerError, "FETCH_FAILED", "failed to list uploads")
		return
	}

	items := make([]response.H, 0, len(uploads))
	for _, u := range uploads {
		items = append(items, response.H{
			"id":         u.ID,
			"url":        u.FileURL,
			"name":       u.OriginalName,
			"mime_type":  u.MimeType,
			"size":       u.Size,
			"created_at": u.CreatedAt,
		})
	}
	response.JSON(w, http.StatusOK, response.H{"success": true, "data": items})
}

// ensure io is used (for future streaming use)
var _ = io.Discard
