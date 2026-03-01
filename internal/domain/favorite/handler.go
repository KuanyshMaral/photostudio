package favorite

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"photostudio/internal/pkg/chicontext"
	"photostudio/internal/pkg/response"
)

type Handler struct {
	repo FavoriteRepository
}

func NewHandler(repo FavoriteRepository) *Handler {
	return &Handler{repo: repo}
}

func (h *Handler) GetFavorites(w http.ResponseWriter, r *http.Request) {
	userID := chicontext.UserIDFromCtx(r.Context())
	if userID == 0 {
		response.CustomError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "unauthorized")
		return
	}

	q := r.URL.Query()
	page, _ := strconv.Atoi(q.Get("page"))
	perPage, _ := strconv.Atoi(q.Get("per_page"))
	entityType := q.Get("entity_type")

	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}

	offset := (page - 1) * perPage
	var t *string
	if entityType != "" {
		t = &entityType
	}

	favorites, total, err := h.repo.GetByUserID(userID, t, perPage, offset)
	if err != nil {
		response.CustomError(w, r, http.StatusInternalServerError, "FETCH_FAILED", "failed to get favorites")
		return
	}

	response.JSON(w, http.StatusOK, ToFavoriteListResponse(favorites, total, page, perPage))
}

func (h *Handler) AddFavorite(w http.ResponseWriter, r *http.Request) {
	userID := chicontext.UserIDFromCtx(r.Context())
	if userID == 0 {
		response.CustomError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "unauthorized")
		return
	}

	var req AddFavoriteRequest
	if err := response.BindJSON(r, &req); err != nil {
		response.CustomError(w, r, http.StatusBadRequest, "INVALID_BODY", "invalid request body")
		return
	}

	favorite, err := h.repo.Add(userID, req.EntityType, req.EntityID)
	if err != nil {
		if err.Error() == "entity already in favorites" {
			response.CustomError(w, r, http.StatusBadRequest, "ALREADY_FAVORITED", err.Error())
			return
		}
		response.CustomError(w, r, http.StatusInternalServerError, "ADD_FAILED", "failed to add favorite")
		return
	}

	response.JSON(w, http.StatusCreated, ToFavoriteResponse(favorite))
}

func (h *Handler) RemoveFavorite(w http.ResponseWriter, r *http.Request) {
	userID := chicontext.UserIDFromCtx(r.Context())
	if userID == 0 {
		response.CustomError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "unauthorized")
		return
	}

	entityType := chi.URLParam(r, "type")
	entityID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || entityType == "" {
		response.CustomError(w, r, http.StatusBadRequest, "INVALID_PARAMS", "invalid type or id")
		return
	}

	err = h.repo.Remove(userID, entityType, entityID)
	if err != nil {
		if err.Error() == "favorite not found" {
			response.CustomError(w, r, http.StatusNotFound, "NOT_FOUND", err.Error())
			return
		}
		response.CustomError(w, r, http.StatusInternalServerError, "REMOVE_FAILED", "failed to remove favorite")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) CheckFavorite(w http.ResponseWriter, r *http.Request) {
	userID := chicontext.UserIDFromCtx(r.Context())
	if userID == 0 {
		response.CustomError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "unauthorized")
		return
	}

	entityType := chi.URLParam(r, "type")
	entityID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || entityType == "" {
		response.CustomError(w, r, http.StatusBadRequest, "INVALID_PARAMS", "invalid type or id")
		return
	}

	isFavorite, err := h.repo.Exists(userID, entityType, entityID)
	if err != nil {
		response.CustomError(w, r, http.StatusInternalServerError, "CHECK_FAILED", "failed to check favorite")
		return
	}

	response.JSON(w, http.StatusOK, CheckFavoriteResponse{IsFavorite: isFavorite})
}

// ErrorResponse для Swagger документации
type ErrorResponse struct {
	Error string `json:"error"`
}
