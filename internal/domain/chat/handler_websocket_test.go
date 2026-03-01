package chat

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"photostudio/internal/pkg/chicontext"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
)

func TestWebSocket_RequiresUpgradeHeaders(t *testing.T) {
	h := NewHandler(nil, NewHub())
	r := chi.NewRouter()
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			ctx := chicontext.SetUserID(req.Context(), int64(2))
			next.ServeHTTP(w, req.WithContext(ctx))
		})
	})
	r.Get("/chats/ws", h.WebSocket)

	req := httptest.NewRequest(http.MethodGet, "/chats/ws", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUpgradeRequired, w.Code)
	assert.Contains(t, w.Body.String(), "websocket upgrade required")
}

func TestWebSocket_RejectsHandshakeWithoutSecWebSocketKey(t *testing.T) {
	h := NewHandler(nil, NewHub())
	r := chi.NewRouter()
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			ctx := chicontext.SetUserID(req.Context(), int64(2))
			next.ServeHTTP(w, req.WithContext(ctx))
		})
	})
	r.Get("/chats/ws", h.WebSocket)

	req := httptest.NewRequest(http.MethodGet, "/chats/ws", nil)
	req.Header.Set("Connection", "Upgrade")
	req.Header.Set("Upgrade", "websocket")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "missing Sec-WebSocket-Key")
}
