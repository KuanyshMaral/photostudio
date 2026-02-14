package payment

import "github.com/gin-gonic/gin"

func (h *Handler) RegisterProtectedRoutes(rg *gin.RouterGroup) {
	rg.POST("/payments/robokassa/init", h.InitPayment)
}

func (h *Handler) RegisterPublicRoutes(rg *gin.RouterGroup) {
	rg.POST("/payments/robokassa/result", h.ResultCallback)
	rg.GET("/payments/robokassa/success", h.SuccessCallback)
}

