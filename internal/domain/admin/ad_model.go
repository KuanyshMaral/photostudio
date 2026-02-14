package admin

import "time"

type Ad struct {
	ID          int64      `json:"id" gorm:"primaryKey"`
	Title       string     `json:"title"`
	ImageURL    string     `json:"image_url"`
	TargetURL   string     `json:"target_url,omitempty"`
	Placement   string     `json:"placement"`
	IsActive    bool       `json:"is_active"`
	StartDate   *time.Time `json:"start_date,omitempty"`
	EndDate     *time.Time `json:"end_date,omitempty"`
	Impressions int64      `json:"impressions"`
	Clicks      int64      `json:"clicks"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

func (Ad) TableName() string {
	return "ads"
}
