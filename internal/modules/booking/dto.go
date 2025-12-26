package booking

import "time"

type CreateBookingRequest struct {
	RoomID    int64     `json:"room_id" binding:"required"`
	StudioID  int64     `json:"studio_id" binding:"required"`
	UserID    int64     `json:"user_id" binding:"required"`
	StartTime time.Time `json:"start_time" binding:"required"`
	EndTime   time.Time `json:"end_time" binding:"required"`
	Notes     string    `json:"notes"`
}
