package booking

import "time"

type CreatePreBookingRequest struct {
	StudioID  int64     `json:"studio_id"`
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
}
