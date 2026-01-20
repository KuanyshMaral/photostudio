package favorite

import (
	"net/http"
	"strconv"

	"photostudio/internal/repository"

	"github.com/gin-gonic/gin"
)

// Handler обрабатывает HTTP запросы для избранного
type Handler struct {
	repo repository.FavoriteRepository
}

// NewHandler создаёт новый handler
func NewHandler(repo repository.FavoriteRepository) *Handler {
	return &Handler{repo: repo}
}

// RegisterRoutes регистрирует routes для избранного
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	favorites := rg.Group("/favorites")
	{
		favorites.GET("", h.GetFavorites)
		favorites.POST("/:studioId", h.AddFavorite)
		favorites.DELETE("/:studioId", h.RemoveFavorite)
		favorites.GET("/:studioId/check", h.CheckFavorite)
	}
}

// GetFavorites возвращает список избранных студий текущего пользователя
// @Summary Получить избранное
// @Tags Favorites
// @Security BearerAuth
// @Param page query int false "Номер страницы" default(1)
// @Param per_page query int false "Элементов на страницу" default(20)
// @Success 200 {object} FavoriteListResponse
// @Router /favorites [get]
func (h *Handler) GetFavorites(c *gin.Context) {
	// Получаем user_id из JWT (установлен middleware)
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	// Парсим pagination параметры
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "20"))

	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}

	offset := (page - 1) * perPage

	// Получаем избранное из репозитория
	favorites, total, err := h.repo.GetByUserID(userID.(int64), perPage, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get favorites"})
		return
	}

	// Конвертируем в DTO и отправляем
	response := ToFavoriteListResponse(favorites, total, page, perPage)
	c.JSON(http.StatusOK, response)
}

// AddFavorite добавляет студию в избранное
// @Summary Добавить в избранное
// @Tags Favorites
// @Security BearerAuth
// @Param studioId path int true "ID студии"
// @Success 201 {object} FavoriteResponse
// @Failure 400 {object} ErrorResponse "Studio already in favorites"
// @Failure 404 {object} ErrorResponse "Studio not found"
// @Router /favorites/{studioId} [post]
func (h *Handler) AddFavorite(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	// Парсим studioId из URL
	studioIDStr := c.Param("studioId")
	studioID, err := strconv.ParseInt(studioIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid studio id"})
		return
	}

	// Добавляем в избранное
	favorite, err := h.repo.Add(userID.(int64), studioID)
	if err != nil {
		if err.Error() == "studio already in favorites" {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to add favorite"})
		return
	}

	response := ToFavoriteResponse(favorite)
	c.JSON(http.StatusCreated, response)
}

// RemoveFavorite удаляет студию из избранного
// @Summary Удалить из избранного
// @Tags Favorites
// @Security BearerAuth
// @Param studioId path int true "ID студии"
// @Success 204 "No Content"
// @Failure 404 {object} ErrorResponse "Favorite not found"
// @Router /favorites/{studioId} [delete]
func (h *Handler) RemoveFavorite(c *gin.Context) {
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

	err = h.repo.Remove(userID.(int64), studioID)
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

// CheckFavorite проверяет, есть ли студия в избранном
// @Summary Проверить избранное
// @Tags Favorites
// @Security BearerAuth
// @Param studioId path int true "ID студии"
// @Success 200 {object} CheckFavoriteResponse
// @Router /favorites/{studioId}/check [get]
func (h *Handler) CheckFavorite(c *gin.Context) {
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

	isFavorite, err := h.repo.Exists(userID.(int64), studioID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to check favorite"})
		return
	}

	c.JSON(http.StatusOK, CheckFavoriteResponse{IsFavorite: isFavorite})
}

// ErrorResponse для документации swagger
type ErrorResponse struct {
	Error string `json:"error"`
}
