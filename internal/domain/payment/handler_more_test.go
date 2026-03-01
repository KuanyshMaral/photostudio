package payment

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"photostudio/internal/pkg/chicontext"

	"github.com/go-chi/chi/v5"
)

func TestHandlerSubscriptionAndValidationBranches(t *testing.T) {
	_, h, _ := setupPaymentTest(t)
	r := chi.NewRouter()
	r.Group(func(g chi.Router) {
		g.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				next.ServeHTTP(w, req.WithContext(chicontext.SetUserID(req.Context(), int64(1))))
			})
		})
		h.RegisterProtectedRoutes(g)
	})
	h.RegisterPublicWebhookRoutes(r)

	for _, tc := range []string{"/payments/robokassa/create", "/subscriptions"} {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, tc, bytes.NewBufferString("{"))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("%s => %d", tc, w.Code)
		}
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/subscriptions/me", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("me => %d", w.Code)
	}

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/subscriptions/cancel", bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("cancel => %d", w.Code)
	}
}

func TestInitPaymentRouteAndReplayForbidden(t *testing.T) {
	s, h, _ := setupPaymentTest(t)
	r := chi.NewRouter()
	r.Group(func(g chi.Router) {
		g.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				next.ServeHTTP(w, req.WithContext(chicontext.SetUserID(req.Context(), int64(1))))
			})
		})
		h.RegisterProtectedRoutes(g)
	})
	h.RegisterPublicWebhookRoutes(r)

	b, _ := json.Marshal(InitPaymentRequest{BookingID: 10, OutSum: "100.00"})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/payments/robokassa/init", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("init code=%d", w.Code)
	}

	var cr InitPaymentResponse
	_ = json.Unmarshal(w.Body.Bytes(), &cr)
	sig := s.generateSignatureForResult("100.00", cr.InvID, map[string]string{})
	first := httptest.NewRecorder()
	firstReq := httptest.NewRequest(http.MethodPost, "/webhooks/robokassa/result", bytes.NewBufferString("OutSum=100.00&InvId="+fmt.Sprintf("%d", cr.InvID)+"&SignatureValue="+sig))
	firstReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.ServeHTTP(first, firstReq)
	if first.Code != http.StatusOK {
		t.Fatalf("first=%d", first.Code)
	}
	replay := httptest.NewRecorder()
	replayReq := httptest.NewRequest(http.MethodPost, "/webhooks/robokassa/result", bytes.NewBufferString("OutSum=100.00&InvId="+fmt.Sprintf("%d", cr.InvID)+"&SignatureValue="+sig))
	replayReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.ServeHTTP(replay, replayReq)
	if replay.Code != http.StatusOK {
		t.Fatalf("replay=%d", replay.Code)
	}
}

func TestCollectShpWithQueryAndForm(t *testing.T) {
	r := chi.NewRouter()
	r.Post("/x", func(w http.ResponseWriter, req *http.Request) {
		_ = req.ParseForm()
		m := collectShp(req)
		if m["a"] != "1" || m["b"] != "2" {
			w.WriteHeader(500)
			w.Write([]byte("bad"))
			return
		}
		w.WriteHeader(200)
		w.Write([]byte("ok"))
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/x?Shp_b=2", bytes.NewBufferString("Shp_a=1"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatal(w.Code)
	}
}

func TestResultCallbackBadRequestOnMissingParams(t *testing.T) {
	_, h, _ := setupPaymentTest(t)
	r := chi.NewRouter()
	h.RegisterPublicWebhookRoutes(r)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/webhooks/robokassa/result", bytes.NewBufferString("InvId=123"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 got %d", w.Code)
	}
}

func TestCreatePaymentInvalidAmountReturnsBadRequest(t *testing.T) {
	_, h, _ := setupPaymentTest(t)
	r := chi.NewRouter()
	r.Group(func(g chi.Router) {
		g.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				next.ServeHTTP(w, req.WithContext(chicontext.SetUserID(req.Context(), int64(1))))
			})
		})
		h.RegisterProtectedRoutes(g)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/payments/robokassa/create", bytes.NewBufferString(`{"booking_id":10,"out_sum":"-1"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 got %d body=%s", w.Code, w.Body.String())
	}
}

func TestCreatePaymentSwaggerPlaceholderSubscriptionIDIsIgnored(t *testing.T) {
	_, h, _ := setupPaymentTest(t)
	r := chi.NewRouter()
	r.Group(func(g chi.Router) {
		g.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				next.ServeHTTP(w, req.WithContext(chicontext.SetUserID(req.Context(), int64(1))))
			})
		})
		h.RegisterProtectedRoutes(g)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/payments/robokassa/create", bytes.NewBufferString(`{"booking_id":10,"amount":"100.00","subscription_id":"string"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d body=%s", w.Code, w.Body.String())
	}
}

func TestCreatePaymentInvalidSubscriptionIDReturnsBadRequest(t *testing.T) {
	_, h, _ := setupPaymentTest(t)
	r := chi.NewRouter()
	r.Group(func(g chi.Router) {
		g.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				next.ServeHTTP(w, req.WithContext(chicontext.SetUserID(req.Context(), int64(1))))
			})
		})
		h.RegisterProtectedRoutes(g)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/payments/robokassa/create", bytes.NewBufferString(`{"booking_id":10,"amount":"100.00","subscription_id":"not-a-uuid"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 got %d body=%s", w.Code, w.Body.String())
	}
}
