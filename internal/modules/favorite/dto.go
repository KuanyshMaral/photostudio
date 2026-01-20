package favorite

import (
	"photostudio/internal/domain"
	"time"
)

// AddFavoriteRequest — запрос на добавление в избранное
// StudioID берётся из URL path, поэтому body пустой
type AddFavoriteRequest struct {
	// StudioID передаётся через path parameter
}

// FavoriteResponse — ответ с информацией об избранном
type FavoriteResponse struct {
	ID        int64        `json:"id"`
	StudioID  int64        `json:"studio_id"`
	Studio    *StudioBrief `json:"studio,omitempty"`
	CreatedAt time.Time    `json:"created_at"`
}

// StudioBrief — краткая информация о студии для списка избранного
type StudioBrief struct {
	ID       int64    `json:"id"`
	Name     string   `json:"name"`
	Address  string   `json:"address"`
	Rating   float64  `json:"rating"`
	Photos   []string `json:"photos"`
	District string   `json:"district,omitempty"`
}

// FavoriteListResponse — ответ со списком избранного
type FavoriteListResponse struct {
	Favorites  []FavoriteResponse `json:"favorites"`
	Total      int64              `json:"total"`
	Page       int                `json:"page"`
	PerPage    int                `json:"per_page"`
	TotalPages int                `json:"total_pages"`
}

// CheckFavoriteResponse — ответ на проверку "в избранном ли"
type CheckFavoriteResponse struct {
	IsFavorite bool `json:"is_favorite"`
}

// ToFavoriteResponse конвертирует domain.Favorite в API response
func ToFavoriteResponse(f *domain.Favorite) FavoriteResponse {
	resp := FavoriteResponse{
		ID:        f.ID,
		StudioID:  f.StudioID,
		CreatedAt: f.CreatedAt,
	}

	if f.Studio != nil {
		resp.Studio = &StudioBrief{
			ID:       f.Studio.ID,
			Name:     f.Studio.Name,
			Address:  f.Studio.Address,
			Rating:   f.Studio.Rating,
			Photos:   f.Studio.Photos,
			District: f.Studio.District,
		}
	}

	return resp
}

// ToFavoriteListResponse конвертирует slice favorites в paginated response
func ToFavoriteListResponse(favorites []domain.Favorite, total int64, page, perPage int) FavoriteListResponse {
	items := make([]FavoriteResponse, len(favorites))
	for i, f := range favorites {
		items[i] = ToFavoriteResponse(&f)
	}

	totalPages := int(total) / perPage
	if int(total)%perPage > 0 {
		totalPages++
	}

	return FavoriteListResponse{
		Favorites:  items,
		Total:      total,
		Page:       page,
		PerPage:    perPage,
		TotalPages: totalPages,
	}
}
