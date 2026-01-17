package catalog

import "photostudio/internal/domain"

// ---------- ROOMS ----------

type CreateRoomRequest struct {
	Name            string   `json:"name" validate:"required"`
	Description     string   `json:"description"`
	AreaSqm         int      `json:"area_sqm" validate:"required,gt=0"`
	Capacity        int      `json:"capacity" validate:"required,gt=0"`
	RoomType        string   `json:"room_type" validate:"required"`
	PricePerHourMin float64  `json:"price_per_hour_min" validate:"required,gte=0"`
	PricePerHourMax *float64 `json:"price_per_hour_max,omitempty"`
	Amenities       []string `json:"amenities,omitempty"`
	Photos          []string `json:"photos"`
}

// ---------- EQUIPMENT ----------

type CreateEquipmentRequest struct {
	Name        string  `json:"name" validate:"required"`
	Category    string  `json:"category,omitempty"`
	Brand       string  `json:"brand"`
	Model       string  `json:"model"`
	Quantity    int     `json:"quantity" validate:"required,gt=0"`
	RentalPrice float64 `json:"rental_price"`
}

// ---------- STUDIO ----------

type CreateStudioRequest struct {
	Name         string              `json:"name" validate:"required"`
	Description  string              `json:"description"`
	Address      string              `json:"address" validate:"required"`
	District     string              `json:"district"`
	City         string              `json:"city" validate:"required"`
	Phone        string              `json:"phone"`
	Email        string              `json:"email"`
	Website      string              `json:"website"`
	WorkingHours domain.WorkingHours `json:"working_hours,omitempty"`
}

// ---------- STUDIO UPDATE ----------

type UpdateStudioRequest struct {
	Name         string              `json:"name" validate:"required"`
	Description  string              `json:"description"`
	Address      string              `json:"address" validate:"required"`
	City         string              `json:"city" validate:"required"`
	Phone        string              `json:"phone"`
	Email        string              `json:"email"`
	Website      string              `json:"website"`
	WorkingHours domain.WorkingHours `gorm:"type:jsonb" json:"working_hours,omitempty"`
	District     string              `json:"district,omitempty"`
}

type UpdateRoomRequest struct {
	Name            *string   `json:"name,omitempty"`
	Description     *string   `json:"description,omitempty"`
	AreaSqm         *int      `json:"area_sqm,omitempty"`
	Capacity        *int      `json:"capacity,omitempty"`
	RoomType        *string   `json:"room_type,omitempty"`
	PricePerHourMin *float64  `json:"price_per_hour_min,omitempty"`
	PricePerHourMax *float64  `json:"price_per_hour_max,omitempty"`
	Amenities       *[]string `json:"amenities,omitempty"`
	Photos          *[]string `json:"photos,omitempty"`
}
