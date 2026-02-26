package payment

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
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
	h.RegisterPublicWebhookRoutes(r)

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
	sw := httptest.NewRecorder()
	sreq := httptest.NewRequest(http.MethodGet, "/webhooks/robokassa/success?OutSum=100.00&InvId="+strconv.FormatInt(cr.Data.InvID, 10)+"&SignatureValue="+sig, nil)
	r.ServeHTTP(sw, sreq)
	if sw.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d", sw.Code)
	}

	fw := httptest.NewRecorder()
	freq := httptest.NewRequest(http.MethodGet, "/webhooks/robokassa/fail?InvId="+strconv.FormatInt(cr.Data.InvID, 10), nil)
	r.ServeHTTP(fw, freq)
	if fw.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d", fw.Code)
	}
}

func TestSuccessEndpointAcceptsLegacyInitPayment(t *testing.T) {
	s, h, _ := setupPaymentTest(t)
	gin.SetMode(gin.TestMode)
	r := gin.New()
	g := r.Group("/")
	g.Use(func(c *gin.Context) { c.Set("user_id", int64(1)); c.Next() })
	h.RegisterProtectedRoutes(g)
	h.RegisterPublicWebhookRoutes(r)

	initBody, _ := json.Marshal(InitPaymentRequest{BookingID: 10, OutSum: "100.00", Description: "legacy"})
	iw := httptest.NewRecorder()
	ireq := httptest.NewRequest(http.MethodPost, "/payments/robokassa/init", bytes.NewReader(initBody))
	ireq.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(iw, ireq)
	if iw.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d", iw.Code)
	}
	var ir InitPaymentResponse
	_ = json.Unmarshal(iw.Body.Bytes(), &ir)

	sig := s.generateSignatureForSuccess("100.00", ir.InvID, map[string]string{})
	sw := httptest.NewRecorder()
	sreq := httptest.NewRequest(http.MethodGet, "/webhooks/robokassa/success?OutSum=100.00&InvId="+strconv.FormatInt(ir.InvID, 10)+"&SignatureValue="+sig, nil)
	r.ServeHTTP(sw, sreq)
	if sw.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d", sw.Code)
	}
}
