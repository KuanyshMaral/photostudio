package domain

import (
	"errors"
	"time"
)

type RoomType string

const (
	RoomFashion    RoomType = "Fashion"
	RoomPortrait   RoomType = "Portrait"
	RoomCreative   RoomType = "Creative"
	RoomCommercial RoomType = "Commercial"
)

// ValidRoomTypes returns all valid room types
func ValidRoomTypes() []RoomType {
	return []RoomType{
		RoomFashion,
		RoomPortrait,
		RoomCreative,
		RoomCommercial,
	}
}

// IsValidRoomType checks if the given string is a valid RoomType
func IsValidRoomType(s string) bool {
	switch RoomType(s) {
	case RoomFashion, RoomPortrait, RoomCreative, RoomCommercial:
		return true
	}
	return false
}

// ParseRoomType converts string to RoomType with validation
func ParseRoomType(s string) (RoomType, error) {
	rt := RoomType(s)
	if !IsValidRoomType(s) {
		return "", errors.New("invalid room type: must be Fashion, Portrait, Creative, or Commercial")
	}
	return rt, nil
}

type Room struct {
	ID              int64     `json:"id"`
	StudioID        int64     `json:"studio_id"`
	Name            string    `json:"name" validate:"required"`
	Description     string    `json:"description,omitempty"`
	AreaSqm         int       `json:"area_sqm" validate:"required,gt=0"`
	Capacity        int       `json:"capacity" validate:"required,gt=0"`
	RoomType        RoomType  `json:"room_type" validate:"required"`
	PricePerHourMin float64   `json:"price_per_hour_min" validate:"required,gte=0"`
	PricePerHourMax *float64  `json:"price_per_hour_max,omitempty"`
	Amenities       []string  `gorm:"type:jsonb" json:"amenities,omitempty"`
	Photos          []string  `gorm:"type:jsonb" json:"photos,omitempty"`
	IsActive        bool      `json:"is_active"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`

	// Relations
	Equipment []Equipment `json:"equipment,omitempty"`
}

type Equipment struct {
	ID          int64     `json:"id"`
	RoomID      int64     `json:"room_id"`
	Name        string    `json:"name" validate:"required"`
	Category    string    `json:"category,omitempty"`
	Brand       string    `json:"brand,omitempty"`
	Model       string    `json:"model,omitempty"`
	Quantity    int       `json:"quantity" validate:"required,gt=0"`
	RentalPrice float64   `json:"rental_price,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}
