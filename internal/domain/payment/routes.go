package payment

import "github.com/gin-gonic/gin"

func (h *Handler) RegisterPublicWebhookRoutes(r *gin.Engine) {
	w := r.Group("/webhooks/robokassa")
	{
		w.POST("/result", h.ResultCallback)
	}
}

func (h *Handler) RegisterPublicCallbackRoutes(r *gin.RouterGroup) {
	rb := r.Group("/payments/robokassa")
	{
		rb.POST("/success", h.SuccessCallback)
		rb.POST("/fail", h.FailCallback)
	}
}

func (h *Handler) RegisterProtectedRoutes(r *gin.RouterGroup) {
	rb := r.Group("/payments/robokassa")
	{
		rb.POST("/create", h.CreatePayment)
		rb.POST("/init", h.InitPayment)
	}
	s := r.Group("/subscriptions")
	{
		s.POST("", h.CreateSubscription)
		s.GET("/me", h.MySubscription)
		s.POST("/cancel", h.CancelSubscription)
	}
}

func (h *Handler) InitPayment(c *gin.Context) { h.CreatePayment(c) }
