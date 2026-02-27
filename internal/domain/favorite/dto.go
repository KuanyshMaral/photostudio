package favorite

import "time"

type AddFavoriteRequest struct {
	EntityType string `json:"entity_type" validate:"required"`
	EntityID   int64  `json:"entity_id" validate:"required,gt=0"`
}

type FavoriteResponse struct {
	ID         int64     `json:"id"`
	EntityType string    `json:"entity_type"`
	EntityID   int64     `json:"entity_id"`
	CreatedAt  time.Time `json:"created_at"`
}

type FavoriteListResponse struct {
	Favorites  []FavoriteResponse `json:"favorites"`
	Total      int64              `json:"total"`
	Page       int                `json:"page"`
	PerPage    int                `json:"per_page"`
	TotalPages int                `json:"total_pages"`
}

type CheckFavoriteResponse struct {
	IsFavorite bool `json:"is_favorite"`
}

func ToFavoriteResponse(f *Favorite) FavoriteResponse {
	return FavoriteResponse{
		ID:         f.ID,
		EntityType: f.EntityType,
		EntityID:   f.EntityID,
		CreatedAt:  f.CreatedAt,
	}
}

func ToFavoriteListResponse(favorites []Favorite, total int64, page, perPage int) FavoriteListResponse {
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
