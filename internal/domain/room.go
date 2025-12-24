package domain

import "time"

type Room struct {
	ID        int64     `json:"id"`
	StudioID  int64     `json:"studio_id"`
	Name      string    `json:"name"`
	RoomType  string    `json:"room_type"`
	PriceMin  float64   `json:"price_per_hour_min"`
	PriceMax  float64   `json:"price_per_hour_max"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	Equipment []Equipment `json:"equipment,omitempty"`
}
