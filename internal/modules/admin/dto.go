package admin

import "photostudio/internal/domain"

type VerifyStudioRequest struct {
	AdminNotes string `json:"admin_notes"`
}

type RejectStudioRequest struct {
	Reason string `json:"reason" binding:"required"`
}

type PendingStudioDTO struct {
	ID          int64  `json:"id"`
	UserID      int64  `json:"user_id"`
	BIN         string `json:"bin"`
	CompanyName string `json:"company_name"`
	Status      string `json:"status"`
	CreatedAt   string `json:"created_at"`
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
	TotalUsers                 int `json:"total_users"`
	TotalStudios               int `json:"total_studios"`
	TotalBookings              int `json:"total_bookings"`
	PendingStudios             int `json:"pending_studios"`
	CompletedBookingsThisMonth int `json:"completed_bookings_this_month"`
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

type PlatformAnalytics struct {
	TotalUsers    int64   `json:"total_users"`
	TotalStudios  int64   `json:"total_studios"`
	TotalBookings int64   `json:"total_bookings"`
	TotalRevenue  float64 `json:"total_revenue"`

	NewUsersThisMonth   int64   `json:"new_users_this_month"`
	NewStudiosThisMonth int64   `json:"new_studios_this_month"`
	BookingsThisMonth   int64   `json:"bookings_this_month"`
	RevenueThisMonth    float64 `json:"revenue_this_month"`

	PlatformCommission  float64 `json:"platform_commission"`
	CommissionThisMonth float64 `json:"commission_this_month"`

	UsersByRole   map[string]int64 `json:"users_by_role"`
	BookingsByDay []DailyStats     `json:"bookings_by_day"`
	TopStudios    []StudioRanking  `json:"top_studios"`
	TopCities     []CityStats      `json:"top_cities"`
}

type StudioRanking struct {
	StudioID   int64   `json:"studio_id"`
	StudioName string  `json:"studio_name"`
	City       string  `json:"city"`
	Bookings   int64   `json:"bookings"`
	Revenue    float64 `json:"revenue"`
	Rating     float64 `json:"rating"`
	IsVIP      bool    `json:"is_vip"`
	IsGold     bool    `json:"is_gold"`
}

type CityStats struct {
	City     string  `json:"city"`
	Studios  int64   `json:"studios"`
	Bookings int64   `json:"bookings"`
	Revenue  float64 `json:"revenue"`
}

type DailyStats struct {
	Date     string  `json:"date"`
	Bookings int64   `json:"bookings"`
	Revenue  float64 `json:"revenue"`
}


