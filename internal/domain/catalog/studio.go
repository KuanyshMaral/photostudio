package catalog

import (
	"time"
)

// WorkingHours представляет часы работы на один день
type WorkingHours struct {
	DayOfWeek int    `json:"day_of_week"` // 0=Вс, 1=Пн, ..., 6=Сб
	OpenTime  string `json:"open_time"`   // "09:00"
	CloseTime string `json:"close_time"`  // "21:00"
	IsClosed  bool   `json:"is_closed"`   // true = выходной
}

// StudioWorkingHours хранит часы работы студии в отдельной таблице
type StudioWorkingHours struct {
	ID       int64          `json:"id" gorm:"primaryKey"`
	StudioID int64          `json:"studio_id" gorm:"uniqueIndex"`
	Hours    []WorkingHours `json:"hours" gorm:"serializer:json"` // ✅ изменили на []WorkingHours

	Studio *Studio `json:"-" gorm:"foreignKey:StudioID"`
}

func (StudioWorkingHours) TableName() string {
	return "studio_working_hours"
}

// DefaultWorkingHours возвращает стандартные часы (Пн-Пт 10:00-20:00)
func DefaultWorkingHours() []WorkingHours {
	hours := make([]WorkingHours, 7)
	for i := 0; i < 7; i++ {
		hours[i] = WorkingHours{
			DayOfWeek: i,
			OpenTime:  "10:00",
			CloseTime: "20:00",
			IsClosed:  i == 0 || i == 6, // Выходные по умолчанию
		}
	}
	return hours
}

// DaySchedule сохраняем для обратной совместимости
type DaySchedule struct {
	Open  string `json:"open"`  // "09:00"
	Close string `json:"close"` // "22:00"
}

type WorkingHoursMap map[string]DaySchedule

type Studio struct {
	ID           int64           `json:"id"`
	OwnerID      int64           `json:"owner_id"`
	Name         string          `json:"name" validate:"required"`
	Description  string          `json:"description,omitempty"`
	Photos       []string        `json:"photos" gorm:"serializer:json;type:jsonb;default:'[]'"`
	Address      string          `json:"address" validate:"required"`
	District     string          `json:"district,omitempty"`
	City         string          `json:"city" validate:"required"`
	Latitude     *float64        `json:"latitude,omitempty"`
	Longitude    *float64        `json:"longitude,omitempty"`
	Rating       float64         `json:"rating"`
	TotalReviews int             `json:"total_reviews"`
	Phone        string          `json:"phone,omitempty"`
	Email        string          `json:"email,omitempty"`
	Website      string          `json:"website,omitempty"`
	WorkingHours WorkingHoursMap `gorm:"-" json:"working_hours,omitempty"`
	DeletedAt    *time.Time      `json:"-"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`

	// Relations (loaded separately)
	Rooms []Room `json:"rooms,omitempty"`
}