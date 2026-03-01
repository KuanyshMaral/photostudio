package payment

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"photostudio/internal/domain/auth"
	"photostudio/internal/domain/booking"
	"strconv"
	"testing"
	"time"

	"photostudio/internal/pkg/chicontext"

	"github.com/go-chi/chi/v5"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupPaymentTest(t *testing.T) (*Service, *Handler, *gorm.DB) {
	t.Helper()
	os.Setenv("ROBOKASSA_MERCHANT_LOGIN", "photostudio_kz")
	os.Setenv("ROBOKASSA_IS_TEST", "1")
	os.Setenv("ROBOKASSA_TEST_PASSWORD_1", "PhotoTest1x7K9pM2w")
	os.Setenv("ROBOKASSA_TEST_PASSWORD_2", "PhotoTest2b4N6vR8c")
	db, err := gorm.Open(sqlite.Open("file:payment_test?mode=memory&cache=private"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&auth.User{}, &booking.Booking{}, &Payment{}, &RecurringSubscription{}, &RobokassaPayment{}); err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&auth.User{ID: 1, Email: "a@b.c"}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&booking.Booking{ID: 10, UserID: 1, TotalPrice: 100, RoomID: 1, StudioID: 1, StartTime: time.Now(), EndTime: time.Now().Add(time.Hour), Status: booking.BookingPending, PaymentStatus: booking.PaymentUnpaid}).Error; err != nil {
		t.Fatal(err)
	}
	bRepo := booking.NewBookingRepository(db)
	s := NewService(NewRobokassaPaymentRepository(db), bRepo, bRepo, nil, NewRepository(db))
	return s, NewHandler(s, nil), db
}

func TestBookingPaymentCreationAndWebhook(t *testing.T) {
	s, h, db := setupPaymentTest(t)
	r := chi.NewRouter()
	r.Group(func(authGroup chi.Router) {
		authGroup.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				next.ServeHTTP(w, req.WithContext(chicontext.SetUserID(req.Context(), int64(1))))
			})
		})
		h.RegisterProtectedRoutes(authGroup)
	})
	h.RegisterPublicWebhookRoutes(r)

	body, _ := json.Marshal(CreatePaymentRequest{BookingID: 10, Amount: "100.00", Description: "booking"})
	req := httptest.NewRequest(http.MethodPost, "/payments/robokassa/create", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("create status %d body %s", w.Code, w.Body.String())
	}

	var resp struct {
		Success bool                `json:"success"`
		Data    InitPaymentResponse `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)

	sig := s.generateSignatureForResult("100.00", resp.Data.InvID, map[string]string{})
	webReq := httptest.NewRequest(http.MethodPost, "/webhooks/robokassa/result", bytes.NewBufferString("OutSum=100.00&InvId="+strconv.FormatInt(resp.Data.InvID, 10)+"&SignatureValue="+sig))
	webReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	webW := httptest.NewRecorder()
	r.ServeHTTP(webW, webReq)
	if webW.Code != http.StatusOK {
		t.Fatalf("webhook status %d body %s", webW.Code, webW.Body.String())
	}

	p, _ := NewRepository(db).GetPaymentByInvoiceID(req.Context(), resp.Data.InvID)
	if p.Status != "paid" {
		t.Fatalf("expected paid got %s", p.Status)
	}
}

func TestWebhookInvalidSignature(t *testing.T) {
	_, h, _ := setupPaymentTest(t)
	r := chi.NewRouter()
	h.RegisterPublicWebhookRoutes(r)
	webReq := httptest.NewRequest(http.MethodPost, "/webhooks/robokassa/result", bytes.NewBufferString("OutSum=100.00&InvId=123&SignatureValue=bad"))
	webReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	webW := httptest.NewRecorder()
	r.ServeHTTP(webW, webReq)
	if webW.Code != http.StatusForbidden {
		t.Fatalf("unexpected status %d", webW.Code)
	}
}

func TestSubscriptionActivationAfterPayment(t *testing.T) {
	s, _, db := setupPaymentTest(t)
	resp, err := s.CreateSubscription(context.Background(), 1, "200.00")
	if err != nil {
		t.Fatal(err)
	}
	sig := s.generateSignatureForResult("200.00", resp.InvID, map[string]string{})
	_, err = s.HandleResultCallback(context.Background(), "200.00", resp.InvID, sig, map[string]string{}, "")
	if err != nil {
		t.Fatal(err)
	}
	sub, err := NewRepository(db).GetSubscriptionByUserID(context.Background(), 1)
	if err != nil {
		t.Fatal(err)
	}
	if sub.Status != "active" {
		t.Fatalf("expected active got %s", sub.Status)
	}
}
