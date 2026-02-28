package chat

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestWebSocket_RequiresUpgradeHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)

	h := NewHandler(nil, NewHub())
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("user_id", int64(2))
		c.Next()
	})
	r.GET("/chats/ws", h.WebSocket)

	req := httptest.NewRequest(http.MethodGet, "/chats/ws", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUpgradeRequired, w.Code)
	assert.Contains(t, w.Body.String(), "websocket upgrade required")
}

func TestWebSocket_RejectsHandshakeWithoutSecWebSocketKey(t *testing.T) {
	gin.SetMode(gin.TestMode)

	h := NewHandler(nil, NewHub())
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("user_id", int64(2))
		c.Next()
	})
	r.GET("/chats/ws", h.WebSocket)

	req := httptest.NewRequest(http.MethodGet, "/chats/ws", nil)
	req.Header.Set("Connection", "Upgrade")
	req.Header.Set("Upgrade", "websocket")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "missing Sec-WebSocket-Key")
}
