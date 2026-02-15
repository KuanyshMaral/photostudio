package auth

import "github.com/gin-gonic/gin"

func (h *Handler) RegisterPublicRoutes(v1 *gin.RouterGroup) {
	authGroup := v1.Group("/auth")
	{
		authGroup.POST("/register/client", h.RegisterClient)

		authGroup.POST("/login", h.Login)
		authGroup.POST("/verify/request", h.RequestEmailVerification)
		authGroup.POST("/verify/confirm", h.ConfirmEmailVerification)
		authGroup.POST("/refresh", h.Refresh)
		authGroup.POST("/logout", h.Logout)
	}
}

func (h *Handler) RegisterProtectedRoutes(protected *gin.RouterGroup) {
	userGroup := protected.Group("/users")
	{
		userGroup.GET("/me", h.GetMe)
		userGroup.PUT("/me", h.UpdateProfile)
		userGroup.POST("/verification/documents", h.UploadVerificationDocuments)
	}
}
