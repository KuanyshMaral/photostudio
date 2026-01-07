package domain

import "time"

type Review struct {
	ID            int64      `json:"id"`
	StudioID      int64      `json:"studio_id"`
	UserID        int64      `json:"user_id"`
	BookingID     *int64     `json:"booking_id,omitempty"`
	Rating        int        `json:"rating"`
	Comment       string     `json:"comment,omitempty"`
	Photos        []string   `json:"photos,omitempty" gorm:"type:json"`
	OwnerResponse *string    `json:"owner_response,omitempty"`
	RespondedAt   *time.Time `json:"responded_at,omitempty"`
	IsVerified    bool       `json:"is_verified"`
	IsHidden      bool       `json:"is_hidden"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}
