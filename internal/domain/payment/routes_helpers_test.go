package payment

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRouteRegistrationsAndHelpers(t *testing.T) {
	_, h, _ := setupPaymentTest(t)
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h.RegisterPublicWebhookRoutes(r)
	p := r.Group("/")
	p.Use(func(c *gin.Context) { c.Set("user_id", int64(1)); c.Next() })
	h.RegisterProtectedRoutes(p)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/webhooks/robokassa/result", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected bad request got %d", w.Code)
	}

	if trimShpKey("Shp_a") != "a" {
		t.Fatal("trim failed")
	}
	if trimShpKey("ab") != "ab" {
		t.Fatal("short trim failed")
	}
}
