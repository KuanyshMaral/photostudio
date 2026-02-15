package lead

// SubmitLeadRequest represents public lead submission
type SubmitLeadRequest struct {
	// Contact person
	ContactName     string `json:"contact_name" validate:"required"`
	ContactEmail    string `json:"contact_email" validate:"required,email"`
	ContactPhone    string `json:"contact_phone" validate:"required"`
	ContactPosition string `json:"contact_position"`

	// Company info
	CompanyName  string `json:"company_name" validate:"required"`
	Bin          string `json:"bin"`
	LegalAddress string `json:"legal_address"`
	Website      string `json:"website"`

	// Application details
	UseCase    string `json:"use_case"`
	HowFoundUs string `json:"how_found_us"`

	// UTM parameters (optional, from query params)
	UTMSource   string `json:"utm_source"`
	UTMMedium   string `json:"utm_medium"`
	UTMCampaign string `json:"utm_campaign"`
}

// UpdateLeadStatusRequest represents status update
type UpdateLeadStatusRequest struct {
	Status Status `json:"status" validate:"required,oneof=new contacted qualified rejected lost"`
	Notes  string `json:"notes"`
	Reason string `json:"reason"` // For rejection
}

// AssignLeadRequest represents lead assignment
type AssignLeadRequest struct {
	AdminID  string `json:"admin_id" validate:"required,uuid"`
	Priority int    `json:"priority"`
}

// ConvertLeadRequest represents lead conversion to owner account
type ConvertLeadRequest struct {
	Password     string `json:"password" validate:"required,min=8"`
	OrgType      string `json:"org_type" validate:"required,oneof=ip llp"`
	LegalName    string `json:"legal_name" validate:"required"`
	Bin          string `json:"bin" validate:"required"`
	LegalAddress string `json:"legal_address" validate:"required"`
}

// LeadListResponse represents paginated list
type LeadListResponse struct {
	Leads []OwnerLead `json:"leads"`
	Total int         `json:"total"`
}
