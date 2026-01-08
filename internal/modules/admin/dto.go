package admin

import "photostudio/internal/domain"

type VerifyStudioRequest struct {
	AdminNotes string `json:"admin_notes"`
}

type RejectStudioRequest struct {
	Reason string `json:"reason" binding:"required"`
}

type BlockUserRequest struct {
	Reason string `json:"reason" binding:"required"`
}

type StudioListResponse struct {
	Studios []domain.Studio `json:"studios"`
	Total   int             `json:"total"`
	Page    int             `json:"page"`
	Limit   int             `json:"limit"`
}

type StatisticsResponse struct {
	TotalUsers     int `json:"total_users"`
	TotalStudios   int `json:"total_studios"`
	TotalBookings  int `json:"total_bookings"`
	PendingStudios int `json:"pending_studios"`
	TodayBookings  int `json:"today_bookings"`
}

type UserListFilter struct {
	Role    string `form:"role"`
	Blocked *bool  `form:"blocked"`
	Query   string `form:"q"` // name/email contains
}

type UserListResponse struct {
	Users []domain.User `json:"users"`
	Total int           `json:"total"`
	Page  int           `json:"page"`
	Limit int           `json:"limit"`
}

type ReviewListFilter struct {
	StudioID *int64 `form:"studio_id"`
	UserID   *int64 `form:"user_id"`
	Hidden   *bool  `form:"hidden"`
}

type ReviewListResponse struct {
	Reviews []domain.Review `json:"reviews"`
	Total   int             `json:"total"`
	Page    int             `json:"page"`
	Limit   int             `json:"limit"`
}
