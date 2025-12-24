package domain

import "time"

type Equipment struct {
	ID          int64     `json:"id"`
	RoomID      int64     `json:"room_id"`
	Name        string    `json:"name"`
	Category    string    `json:"category"`
	Quantity    int       `json:"quantity"`
	RentalPrice float64   `json:"rental_price"`
	CreatedAt   time.Time `json:"created_at"`
}
