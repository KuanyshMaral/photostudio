package domain

import "time"

type Studio struct {
	ID           int64      `json:"id"`
	OwnerID      int64      `json:"owner_id"`
	Name         string     `json:"name"`
	Description  string     `json:"description"`
	Address      string     `json:"address"`
	City         string     `json:"city"`
	Rating       float64    `json:"rating"`
	TotalReviews int        `json:"total_reviews"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	DeletedAt    *time.Time `json:"-"`

	Rooms []Room `json:"rooms,omitempty"`
}
