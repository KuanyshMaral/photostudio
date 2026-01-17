package domain

import "time"

type WorkingHours map[string]DaySchedule

type DaySchedule struct {
	Open  string `json:"open"`  // "09:00"
	Close string `json:"close"` // "22:00"
}

type Studio struct {
	ID           int64        `json:"id"`
	OwnerID      int64        `json:"owner_id"`
	Name         string       `json:"name" validate:"required"`
	Description  string       `json:"description,omitempty"`
	Photos       []string     `json:"photos" gorm:"serializer:json;type:jsonb;default:'[]'"`
	Address      string       `json:"address" validate:"required"`
	District     string       `json:"district,omitempty"`
	City         string       `json:"city" validate:"required"`
	Latitude     *float64     `json:"latitude,omitempty"`
	Longitude    *float64     `json:"longitude,omitempty"`
	Rating       float64      `json:"rating"`
	TotalReviews int          `json:"total_reviews"`
	Phone        string       `json:"phone,omitempty"`
	Email        string       `json:"email,omitempty"`
	Website      string       `json:"website,omitempty"`
	WorkingHours WorkingHours `gorm:"-" json:"working_hours,omitempty"`
	DeletedAt    *time.Time   `json:"-"`
	CreatedAt    time.Time    `json:"created_at"`
	UpdatedAt    time.Time    `json:"updated_at"`

	// Relations (loaded separately)
	Rooms []Room `json:"rooms,omitempty"`
}
