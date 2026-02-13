package auth

type RegisterClientRequest struct {
	Name     string `json:"name" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
	Phone    string `json:"phone"`
	Password string `json:"password" binding:"required,min=6"`
}
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}
type UserPublic struct {
	ID    int64  `json:"id"`
	Role  string `json:"role"`
	Name  string `json:"name"`
	Email string `json:"email"`
}
type RegisterStudioRequest struct {
	Name            string `json:"name" validate:"required,min=2"`
	Email           string `json:"email" validate:"required,email"`
	Phone           string `json:"phone" validate:"required"`
	Password        string `json:"password" validate:"required,min=8"`
	CompanyName     string `json:"company_name" validate:"required"`
	BIN             string `json:"bin" validate:"required,len=12"`
	LegalAddress    string `json:"legal_address,omitempty"`
	ContactPerson   string `json:"contact_person,omitempty"`
	ContactPosition string `json:"contact_position,omitempty"`
}
type UpdateProfileRequest struct {
	Name  string `json:"name,omitempty" validate:"omitempty,min=2"`
	Phone string `json:"phone,omitempty" validate:"omitempty,e164"` // optional: use phone validation
}

// =========================
// PROFILE RESPONSE DTOs
// =========================

// UserStats содержит статистику бронирований пользователя
type UserStats struct {
	TotalBookings     int `json:"total_bookings"`
	UpcomingBookings  int `json:"upcoming_bookings"`
	CompletedBookings int `json:"completed_bookings"`
	CancelledBookings int `json:"cancelled_bookings"`
}

// RecentBooking — краткая информация о бронировании для профиля
type RecentBooking struct {
	ID         int64  `json:"id"`
	StudioName string `json:"studio_name"`
	RoomName   string `json:"room_name"`
	Date       string `json:"date"`
	Status     string `json:"status"`
}

// UserProfileResponse расширенный ответ для профиля
type UserProfileResponse struct {
	ID             int64           `json:"id"`
	Email          string          `json:"email"`
	Name           string          `json:"name"`
	Phone          string          `json:"phone,omitempty"`
	Role           string          `json:"role"`
	AvatarURL      string          `json:"avatar_url,omitempty"`
	CreatedAt      string          `json:"created_at"`
	Stats          *UserStats      `json:"stats,omitempty"`
	RecentBookings []RecentBooking `json:"recent_bookings,omitempty"`
}

type VerifyRequestDTO struct {
	Email string `json:"email" binding:"required,email"`
}

type VerifyConfirmDTO struct {
	Email string `json:"email" binding:"required,email"`
	Code  string `json:"code" binding:"required"`
}
