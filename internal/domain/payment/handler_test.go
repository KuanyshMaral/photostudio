package payment

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestSuccessAndFailEndpoints(t *testing.T) {
	s, h, _ := setupPaymentTest(t)
	gin.SetMode(gin.TestMode)
	r := gin.New()
	g := r.Group("/")
	g.Use(func(c *gin.Context) { c.Set("user_id", int64(1)); c.Next() })
	h.RegisterProtectedRoutes(g)

	createBody, _ := json.Marshal(CreatePaymentRequest{BookingID: 10, Amount: "100.00", Description: "x"})
	cw := httptest.NewRecorder()
	creq := httptest.NewRequest(http.MethodPost, "/payments/robokassa/create", bytes.NewReader(createBody))
	creq.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(cw, creq)
	var cr struct {
		Data InitPaymentResponse `json:"data"`
	}
	_ = json.Unmarshal(cw.Body.Bytes(), &cr)

	sig := s.generateSignatureForSuccess("100.00", cr.Data.InvID, map[string]string{})
	succBody, _ := json.Marshal(PaymentCallbackRequest{OutSum: "100.00", InvID: cr.Data.InvID, SignatureValue: sig})
	sw := httptest.NewRecorder()
	sreq := httptest.NewRequest(http.MethodPost, "/payments/robokassa/success", bytes.NewReader(succBody))
	sreq.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(sw, sreq)
	if sw.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d", sw.Code)
	}

	failBody, _ := json.Marshal(PaymentFailRequest{InvID: cr.Data.InvID})
	fw := httptest.NewRecorder()
	freq := httptest.NewRequest(http.MethodPost, "/payments/robokassa/fail", bytes.NewReader(failBody))
	freq.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(fw, freq)
	if fw.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d", fw.Code)
	}
}
