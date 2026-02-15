package profile

import (
	"github.com/google/uuid"
)

// UpdateClientProfileRequest represents client profile update
type UpdateClientProfileRequest struct {
	Name      *string `json:"name"`
	Nickname  *string `json:"nickname"`
	Phone     *string `json:"phone"`
	AvatarURL *string `json:"avatar_url"`
}

// UpdateOwnerProfileRequest represents owner profile update
type UpdateOwnerProfileRequest struct {
	CompanyName     *string `json:"company_name"`
	Bin             *string `json:"bin"`
	LegalAddress    *string `json:"legal_address"`
	ContactPerson   *string `json:"contact_person"`
	ContactPosition *string `json:"contact_position"`
	Phone           *string `json:"phone"`
	Email           *string `json:"email"`
	Website         *string `json:"website"`
}

// UpdateAdminProfileRequest represents admin profile update
type UpdateAdminProfileRequest struct {
	FullName *string `json:"full_name"`
	Position *string `json:"position"`
	Phone    *string `json:"phone"`
}

// CreateOwnerProfileRequest represents owner profile creation (from lead)
type CreateOwnerProfileRequest struct {
	CompanyName     string `json:"company_name" validate:"required"`
	Bin             string `json:"bin"`
	LegalAddress    string `json:"legal_address"`
	ContactPerson   string `json:"contact_person"`
	ContactPosition string `json:"contact_position"`
	Phone           string `json:"phone"`
	Email           string `json:"email"`
	Website         string `json:"website"`
}

// CreateAdminProfileRequest represents admin profile creation
type CreateAdminProfileRequest struct {
	FullName  string    `json:"full_name" validate:"required"`
	Position  string    `json:"position"`
	Phone     string    `json:"phone"`
	CreatedBy uuid.UUID `json:"-"` // Set from context
}
