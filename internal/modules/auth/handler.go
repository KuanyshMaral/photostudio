package auth

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type jwtService interface {
	GenerateToken(userID int64, role string) (string, error)
}

type Handler struct {
	jwt jwtService
}

func NewHandler(jwt jwtService) *Handler {
	return &Handler{jwt: jwt}
}

func (h *Handler) Login(c *gin.Context) {
	token, _ := h.jwt.GenerateToken(1, "studio_owner")

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"token": token,
		},
	})
}
