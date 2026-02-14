package auth

// RegisterUserSwagger describes user fields returned in registration payload.
type RegisterUserSwagger struct {
	ID           int64  `json:"id"`
	Email        string `json:"email"`
	Name         string `json:"name"`
	Role         string `json:"role"`
	Phone        string `json:"phone"`
	StudioStatus string `json:"studio_status"`
}

// RegisterClientDataSwagger describes registration data envelope for client registration.
type RegisterClientDataSwagger struct {
	User             RegisterUserSwagger `json:"user"`
	VerificationSent bool                `json:"verification_sent"`
}

// RegisterClientResponseSwagger describes successful client registration response.
type RegisterClientResponseSwagger struct {
	Success bool                      `json:"success"`
	Data    RegisterClientDataSwagger `json:"data"`
}

// RegisterStudioDataSwagger describes registration data envelope for studio owner registration.
type RegisterStudioDataSwagger struct {
	User             RegisterUserSwagger `json:"user"`
	VerificationSent bool                `json:"verification_sent"`
}

// RegisterStudioResponseSwagger describes successful studio owner registration response.
type RegisterStudioResponseSwagger struct {
	Success bool                      `json:"success"`
	Data    RegisterStudioDataSwagger `json:"data"`
}

// ErrorDetailsSwagger describes error payload.
type ErrorDetailsSwagger struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// ErrorResponseSwagger describes common error response.
type ErrorResponseSwagger struct {
	Success bool                `json:"success"`
	Error   ErrorDetailsSwagger `json:"error"`
}
