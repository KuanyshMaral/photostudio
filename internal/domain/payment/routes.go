package payment

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (h *Handler) RegisterPublicWebhookRoutes(r *gin.Engine) {
	w := r.Group("/webhooks/robokassa")
	{
		w.POST("/result", h.ResultCallback)
		w.GET("/success", h.SuccessCallback)
		w.POST("/success", h.SuccessCallback)
		w.GET("/fail", h.FailCallback)
		w.POST("/fail", h.FailCallback)
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

func (h *Handler) InitPayment(c *gin.Context) {
	var req InitPaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	resp, err := h.service.InitPayment(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}
