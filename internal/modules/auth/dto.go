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
