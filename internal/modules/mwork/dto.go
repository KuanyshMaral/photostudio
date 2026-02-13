package mwork

type SyncUserRequest struct {
	MworkUserID string `json:"mwork_user_id" binding:"required"`
	Email       string `json:"email" binding:"required,email"`
	Role        string `json:"role" binding:"required"`
}

type SyncUserResponse struct {
	ID          int64  `json:"id"`
	MworkUserID string `json:"mwork_user_id"`
	Email       string `json:"email"`
	Role        string `json:"role"`
}


