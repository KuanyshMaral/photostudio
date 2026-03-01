package payment

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"photostudio/internal/pkg/chicontext"

	"github.com/go-chi/chi/v5"
)

func TestRouteRegistrationsAndHelpers(t *testing.T) {
	_, h, _ := setupPaymentTest(t)
	r := chi.NewRouter()
	h.RegisterPublicWebhookRoutes(r)
	r.Group(func(p chi.Router) {
		p.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				next.ServeHTTP(w, req.WithContext(chicontext.SetUserID(req.Context(), int64(1))))
			})
		})
		h.RegisterProtectedRoutes(p)
	})

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
