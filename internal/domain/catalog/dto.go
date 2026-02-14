package catalog

import (
	"fmt"
	"strings"
	"time"
)

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
	Name         string                 `json:"name" validate:"required"`
	Description  string                 `json:"description"`
	Address      string                 `json:"address" validate:"required"`
	District     string                 `json:"district"`
	City         string                 `json:"city" validate:"required"`
	Phone        string                 `json:"phone"`
	Email        string                 `json:"email"`
	Website      string                 `json:"website"`
	WorkingHours WorkingHoursMap `json:"working_hours,omitempty"`
}

// ---------- STUDIO UPDATE ----------

type UpdateStudioRequest struct {
	Name         string                 `json:"name" validate:"required"`
	Description  string                 `json:"description"`
	Address      string                 `json:"address" validate:"required"`
	City         string                 `json:"city" validate:"required"`
	Phone        string                 `json:"phone"`
	Email        string                 `json:"email"`
	Website      string                 `json:"website"`
	WorkingHours WorkingHoursMap `gorm:"type:jsonb" json:"working_hours,omitempty"`
	District     string                 `json:"district,omitempty"`
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

// ---------- WORKING HOURS ----------

// WorkingHoursResponse — ответ с часами работы
type WorkingHoursResponse struct {
	StudioID     int64                 `json:"studio_id"`
	Hours        []WorkingHours `json:"hours"`
	CompactText  string                `json:"compact_text"` // "Пн-Пт: 10:00-20:00"
	IsOpenNow    bool                  `json:"is_open_now"`
	StatusText   string                `json:"status_text"`              // "Открыто (закроется в 20:00)"
	NextOpenTime string                `json:"next_open_time,omitempty"` // Когда откроется
}

// FormatCompactHours форматирует часы в компактный вид
func FormatCompactHours(hours []WorkingHours) string {
	if len(hours) == 0 {
		return "Часы не указаны"
	}

	// Группируем дни с одинаковым расписанием
	scheduleMap := make(map[string][]string)

	for _, h := range hours {
		if h.IsClosed {
			continue
		}
		key := fmt.Sprintf("%s-%s", h.OpenTime, h.CloseTime)
		dayNames := []string{"Вс", "Пн", "Вт", "Ср", "Чт", "Пт", "Сб"}
		if days, exists := scheduleMap[key]; exists {
			scheduleMap[key] = append(days, dayNames[h.DayOfWeek])
		} else {
			scheduleMap[key] = []string{dayNames[h.DayOfWeek]}
		}
	}

	if len(scheduleMap) == 0 {
		return "Выходной"
	}

	var parts []string
	for schedule, days := range scheduleMap {
		if len(days) == 1 {
			parts = append(parts, fmt.Sprintf("%s: %s", days[0], schedule))
		} else {
			parts = append(parts, fmt.Sprintf("%s-%s: %s", days[0], days[len(days)-1], schedule))
		}
	}

	return strings.Join(parts, ", ")
}

// CalculateLiveStatus вычисляет текущий статус (открыто/закрыто)
func CalculateLiveStatus(hours []WorkingHours) (bool, string, string) {
	now := time.Now()
	dayOfWeek := int(now.Weekday()) // 0=Вс, 1=Пн, ...
	currentTime := now.Format("15:04")

	var todayHours *WorkingHours
	for _, h := range hours {
		if h.DayOfWeek == dayOfWeek {
			todayHours = &h
			break
		}
	}

	if todayHours == nil || todayHours.IsClosed {
		// Найти когда откроется
		nextOpen := findNextOpenTime(hours, dayOfWeek)
		return false, "Закрыто", nextOpen
	}

	// Проверяем текущее время
	if currentTime >= todayHours.OpenTime && currentTime < todayHours.CloseTime {
		return true, fmt.Sprintf("Открыто (закроется в %s)", todayHours.CloseTime), ""
	}

	if currentTime < todayHours.OpenTime {
		return false, fmt.Sprintf("Закрыто (откроется в %s)", todayHours.OpenTime), ""
	}

	// После закрытия
	nextOpen := findNextOpenTime(hours, dayOfWeek)
	return false, "Закрыто", nextOpen
}

func findNextOpenTime(hours []WorkingHours, currentDay int) string {
	for i := 1; i <= 7; i++ {
		nextDay := (currentDay + i) % 7
		for _, h := range hours {
			if h.DayOfWeek == nextDay && !h.IsClosed {
				dayNames := []string{"Вс", "Пн", "Вт", "Ср", "Чт", "Пт", "Сб"}
				return dayNames[nextDay] + " в " + h.OpenTime
			}
		}
	}
	return ""
}

// UpdateWorkingHoursRequest — запрос на обновление рабочих часов
type UpdateWorkingHoursRequest struct {
	Hours []WorkingHours `json:"hours" binding:"required"`
}
