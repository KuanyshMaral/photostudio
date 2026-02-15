package payment

import "github.com/gin-gonic/gin"

func (h *Handler) RegisterWebhookRoutes(r *gin.RouterGroup) {
	robokassa := r.Group("/payments/robokassa")
	{
		robokassa.POST("/result", h.ResultCallback)
		robokassa.GET("/success", h.SuccessCallback)
	}
}

func (h *Handler) RegisterProtectedRoutes(r *gin.RouterGroup) {
	robokassa := r.Group("/payments/robokassa")
	{
		robokassa.POST("/init", h.InitPayment)
	}
}
