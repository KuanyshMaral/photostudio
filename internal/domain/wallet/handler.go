package wallet

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

type amountRequest struct {
	Amount int64 `json:"amount" binding:"required,gt=0"`
}

func (h *Handler) GetMyWallet(c *gin.Context) {
	userID := c.GetInt64("user_id")
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	wallet, err := h.service.GetOrCreateWallet(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get wallet"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"balance": wallet.Balance})
}

func (h *Handler) AddToMyWallet(c *gin.Context) {
	userID := c.GetInt64("user_id")
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req amountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	wallet, txn, err := h.service.Add(c.Request.Context(), userID, req.Amount)
	if err != nil {
		if errors.Is(err, ErrInvalidAmount) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to add funds"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"wallet": wallet, "transaction": txn})
}

func (h *Handler) SpendFromMyWallet(c *gin.Context) {
	userID := c.GetInt64("user_id")
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req amountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	wallet, txn, err := h.service.Spend(c.Request.Context(), userID, req.Amount)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidAmount), errors.Is(err, ErrInsufficientFunds):
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to spend funds"})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"wallet": wallet, "transaction": txn})
}

func (h *Handler) ListMyTransactions(c *gin.Context) {
	userID := c.GetInt64("user_id")
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	txns, err := h.service.ListTransactions(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list transactions"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"transactions": txns})
}
