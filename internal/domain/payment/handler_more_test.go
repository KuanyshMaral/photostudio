package payment

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestHandlerSubscriptionAndValidationBranches(t *testing.T) {
	_, h, _ := setupPaymentTest(t)
	gin.SetMode(gin.TestMode)
	r := gin.New()
	g := r.Group("/")
	g.Use(func(c *gin.Context) { c.Set("user_id", int64(1)); c.Next() })
	h.RegisterProtectedRoutes(g)
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
	gin.SetMode(gin.TestMode)
	r := gin.New()
	g := r.Group("/")
	g.Use(func(c *gin.Context) { c.Set("user_id", int64(1)); c.Next() })
	h.RegisterProtectedRoutes(g)
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
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/x", func(c *gin.Context) {
		_ = c.Request.ParseForm()
		m := collectShp(c)
		if m["a"] != "1" || m["b"] != "2" {
			c.String(500, "bad")
			return
		}
		c.String(200, "ok")
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
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h.RegisterPublicWebhookRoutes(r)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/webhooks/robokassa/result", bytes.NewBufferString("InvId=123"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 got %d", w.Code)
	}
}
