package favorite

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	repo FavoriteRepository
}

func NewHandler(repo FavoriteRepository) *Handler {
	return &Handler{repo: repo}
}

// GetFavorites возвращает список избранных сущностей
//
// @Summary Получить список избранного
// @Description Получает список объектов (студий, профилей и т.д.), добавленных в избранное
// @Tags Favorite
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param entity_type query string false "Фильтр по типу сущности (например, studio)"
// @Param page query int false "Номер страницы" default(1)
// @Param per_page query int false "Элементов на страницу" default(20)
// @Success 200 {object} FavoriteListResponse "Список избранного"
// @Failure 401 {object} ErrorResponse "Пользователь не авторизован"
// @Failure 500 {object} ErrorResponse "Ошибка при получении списка"
// @Router /favorites [get]
func (h *Handler) GetFavorites(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "20"))
	entityType := c.Query("entity_type")

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

	favorites, total, err := h.repo.GetByUserID(userID.(int64), t, perPage, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get favorites"})
		return
	}

	response := ToFavoriteListResponse(favorites, total, page, perPage)
	c.JSON(http.StatusOK, response)
}

// AddFavorite добавляет сущность в избранное
//
// @Summary Добавить в избранное
// @Description Добавляет объект в список избранного (студию, профиль и т.д.)
// @Tags Favorite
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body AddFavoriteRequest true "Данные (entity_type, entity_id)"
// @Success 201 {object} FavoriteResponse "Сущность успешно добавлена в избранное"
// @Failure 400 {object} ErrorResponse "Сущность уже находится в избранном или некорректные данные"
// @Failure 401 {object} ErrorResponse "Пользователь не авторизован"
// @Failure 500 {object} ErrorResponse "Ошибка сервера"
// @Router /favorites [post]
func (h *Handler) AddFavorite(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req AddFavoriteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	favorite, err := h.repo.Add(userID.(int64), req.EntityType, req.EntityID)
	if err != nil {
		if err.Error() == "entity already in favorites" {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to add favorite"})
		return
	}

	response := ToFavoriteResponse(favorite)
	c.JSON(http.StatusCreated, response)
}

// AddFavoriteLegacy добавляет студию в избранное (Legacy)
// @Router /favorites/{studioId} [post]
func (h *Handler) AddFavoriteLegacy(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	studioIDStr := c.Param("studioId")
	studioID, err := strconv.ParseInt(studioIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid studio id"})
		return
	}

	favorite, err := h.repo.Add(userID.(int64), "studio", studioID)
	if err != nil {
		if err.Error() == "entity already in favorites" {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to add favorite"})
		return
	}

	response := ToFavoriteResponse(favorite)
	c.JSON(http.StatusCreated, response)
}

// RemoveFavorite удаляет сущность из избранного
//
// @Summary Удалить из избранного
// @Description Удаляет объект из списка избранного
// @Tags Favorite
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param type path string true "Тип сущности"
// @Param id path int64 true "ID сущности"
// @Success 204 "Успешно удалено из избранного"
// @Failure 400 {object} ErrorResponse "Некорректный ID или тип"
// @Failure 401 {object} ErrorResponse "Пользователь не авторизован"
// @Failure 404 {object} ErrorResponse "Отсутствует в избранном"
// @Failure 500 {object} ErrorResponse "Ошибка сервера"
// @Router /favorites/{type}/{id} [delete]
func (h *Handler) RemoveFavorite(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	entityType := c.Param("type")
	entityID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || entityType == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid type or id"})
		return
	}

	err = h.repo.Remove(userID.(int64), entityType, entityID)
	if err != nil {
		if err.Error() == "favorite not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to remove favorite"})
		return
	}

	c.Status(http.StatusNoContent)
}

// RemoveFavoriteLegacy (Legacy)
// @Router /favorites/{studioId} [delete]
func (h *Handler) RemoveFavoriteLegacy(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	studioIDStr := c.Param("studioId")
	studioID, err := strconv.ParseInt(studioIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid studio id"})
		return
	}

	err = h.repo.Remove(userID.(int64), "studio", studioID)
	if err != nil {
		if err.Error() == "favorite not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to remove favorite"})
		return
	}

	c.Status(http.StatusNoContent)
}

// CheckFavorite проверяет, находится ли сущность в избранном пользователя
//
// @Summary Проверить наличие в избранном
// @Description Проверяет наличие объекта в избранном текущего пользователя
// @Tags Favorite
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param type path string true "Тип сущности"
// @Param id path int64 true "ID сущности"
// @Success 200 {object} CheckFavoriteResponse "Результат проверки"
// @Failure 400 {object} ErrorResponse "Некорректный ID или тип"
// @Failure 401 {object} ErrorResponse "Пользователь не авторизован"
// @Failure 500 {object} ErrorResponse "Ошибка сервера"
// @Router /favorites/{type}/{id}/check [get]
func (h *Handler) CheckFavorite(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	entityType := c.Param("type")
	entityID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || entityType == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid type or id"})
		return
	}

	isFavorite, err := h.repo.Exists(userID.(int64), entityType, entityID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to check favorite"})
		return
	}

	c.JSON(http.StatusOK, CheckFavoriteResponse{IsFavorite: isFavorite})
}

// CheckFavoriteLegacy (Legacy)
// @Router /favorites/{studioId}/check [get]
func (h *Handler) CheckFavoriteLegacy(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	studioIDStr := c.Param("studioId")
	studioID, err := strconv.ParseInt(studioIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid studio id"})
		return
	}

	isFavorite, err := h.repo.Exists(userID.(int64), "studio", studioID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to check favorite"})
		return
	}

	c.JSON(http.StatusOK, CheckFavoriteResponse{IsFavorite: isFavorite})
}

// ErrorResponse для Swagger документации
type ErrorResponse struct {
	Error string `json:"error"`
}
