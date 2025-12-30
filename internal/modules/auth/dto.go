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
