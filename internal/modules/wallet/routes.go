package wallet

import "github.com/gin-gonic/gin"

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	wallets := rg.Group("/wallets")
	{
		wallets.GET("/me", h.GetMyWallet)
		wallets.POST("/me/add", h.AddToMyWallet)
		wallets.POST("/me/spend", h.SpendFromMyWallet)
		wallets.GET("/me/transactions", h.ListMyTransactions)
	}
}
