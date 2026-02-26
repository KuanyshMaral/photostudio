package payment

import (
	"context"
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/url"
	"os"
	"photostudio/internal/domain/booking"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

var (
	ErrInvalidSignature = errors.New("invalid signature")
	ErrAmountMismatch   = errors.New("amount mismatch")
	ErrReplayDetected   = errors.New("replay detected")
	ErrPaymentNotFound  = errors.New("payment not found")
	ErrMisconfigured    = errors.New("robokassa is misconfigured")
)

type Service struct {
	payments      paymentRepo
	bookings      bookingReader
	bookingWriter bookingPaymentWriter
	repo          *Repository
	loggerf       func(format string, args ...interface{})

	merchantLogin string
	password1     string
	password2     string
	baseURL       string
	resultURL     string
	successURL    string
	failURL       string
	frontSuccess  string
	frontFail     string
	isTest        string
	hashAlgo      string
}

func NewService(payments paymentRepo, bookings bookingReader, bookingWriter bookingPaymentWriter, loggerf func(format string, args ...interface{}), repo ...*Repository) *Service {
	if loggerf == nil {
		loggerf = func(string, ...interface{}) {}
	}
	r := (*Repository)(nil)
	if len(repo) > 0 {
		r = repo[0]
	}
	isTest := envOrDefault("ROBOKASSA_IS_TEST", "1")
	password1 := os.Getenv("ROBOKASSA_PROD_PASSWORD_1")
	password2 := os.Getenv("ROBOKASSA_PROD_PASSWORD_2")
	if isTest == "1" {
		password1 = os.Getenv("ROBOKASSA_TEST_PASSWORD_1")
		password2 = os.Getenv("ROBOKASSA_TEST_PASSWORD_2")
	}
	if password1 == "" {
		password1 = os.Getenv("ROBOKASSA_PASSWORD1")
	}
	if password2 == "" {
		password2 = os.Getenv("ROBOKASSA_PASSWORD2")
	}
	return &Service{payments: payments, bookings: bookings, bookingWriter: bookingWriter, repo: r, loggerf: loggerf,
		merchantLogin: os.Getenv("ROBOKASSA_MERCHANT_LOGIN"),
		password1:     password1, password2: password2,
		baseURL:   envOrDefault("ROBOKASSA_BASE_URL", "https://auth.robokassa.ru/Merchant/Index.aspx"),
		resultURL: os.Getenv("ROBOKASSA_RESULT_URL"), successURL: os.Getenv("ROBOKASSA_SUCCESS_URL"), failURL: os.Getenv("ROBOKASSA_FAIL_URL"),
		frontSuccess: os.Getenv("ROBOKASSA_FRONTEND_SUCCESS_URL"), frontFail: os.Getenv("ROBOKASSA_FRONTEND_FAIL_URL"),
		isTest:   isTest,
		hashAlgo: strings.ToLower(envOrDefault("ROBOKASSA_HASH_ALGO", "md5")),
	}
}

func envOrDefault(name, def string) string {
	if v := os.Getenv(name); v != "" {
		return v
	}
	return def
}

func (s *Service) CreatePayment(ctx context.Context, userID, bookingID int64, amount string, description string, shpParams map[string]string, recurring bool, previousInvoiceID *int64, subscriptionID *string) (*InitPaymentResponse, error) {
	if s.repo == nil {
		return nil, fmt.Errorf("payment repository is not configured")
	}
	if err := s.validateInitConfig(); err != nil {
		return nil, err
	}
	bk, err := s.bookings.GetByID(ctx, bookingID)
	if err != nil {
		return nil, err
	}
	if bk.UserID != userID {
		return nil, fmt.Errorf("booking does not belong to user")
	}
	if !amountEqual(amount, fmt.Sprintf("%.2f", bk.TotalPrice)) {
		return nil, ErrAmountMismatch
	}

	invID := time.Now().UnixNano()
	shp := map[string]string{"user_id": strconv.FormatInt(userID, 10), "booking_id": strconv.FormatInt(bookingID, 10)}
	for k, v := range shpParams {
		shp[k] = v
	}
	if subscriptionID != nil {
		shp["subscription_id"] = *subscriptionID
	}
	sig := s.generateSignatureForInit(amount, invID, shp)

	u := url.Values{}
	u.Set("MerchantLogin", s.merchantLogin)
	u.Set("OutSum", amount)
	u.Set("InvId", strconv.FormatInt(invID, 10))
	u.Set("Description", description)
	u.Set("SignatureValue", sig)
	u.Set("IsTest", s.isTest)
	u.Set("Encoding", "utf-8")
	if s.resultURL != "" {
		u.Set("ResultURL", s.resultURL)
	}
	if s.successURL != "" {
		u.Set("SuccessURL", s.successURL)
	}
	if s.failURL != "" {
		u.Set("FailURL", s.failURL)
	}
	if recurring {
		u.Set("Recurring", "true")
	}
	if previousInvoiceID != nil {
		u.Set("PreviousInvoiceID", strconv.FormatInt(*previousInvoiceID, 10))
	}
	for k, v := range shp {
		u.Set("Shp_"+k, v)
	}
	paymentURL := s.baseURL + "?" + u.Encode()

	p := &Payment{UserID: userID, BookingID: &bookingID, SubscriptionID: subscriptionID, RobokassaInvoiceID: invID, Amount: amount, Status: "created", IsRecurring: recurring}
	if err := s.repo.CreatePayment(ctx, p); err != nil {
		return nil, err
	}
	if _, err := s.bookingWriter.UpdatePaymentStatusSystem(ctx, bookingID, booking.PaymentUnpaid); err != nil {
		return nil, err
	}

	return &InitPaymentResponse{InvID: invID, PaymentURL: paymentURL, Signature: sig, Status: "created"}, nil
}

func (s *Service) CreateSubscription(ctx context.Context, userID int64, amount string) (*InitPaymentResponse, error) {
	if s.repo == nil {
		return nil, fmt.Errorf("payment repository is not configured")
	}
	sub := &RecurringSubscription{ID: uuid.NewString(), UserID: userID, Status: "pending"}
	if err := s.repo.CreateSubscription(ctx, sub); err != nil {
		return nil, err
	}
	return s.createSubscriptionPayment(ctx, sub, amount, nil)
}

func (s *Service) createSubscriptionPayment(ctx context.Context, sub *RecurringSubscription, amount string, previous *int64) (*InitPaymentResponse, error) {
	if err := s.validateInitConfig(); err != nil {
		return nil, err
	}
	invID := time.Now().UnixNano()
	shp := map[string]string{"user_id": strconv.FormatInt(sub.UserID, 10), "subscription_id": sub.ID}
	sig := s.generateSignatureForInit(amount, invID, shp)
	u := url.Values{}
	u.Set("MerchantLogin", s.merchantLogin)
	u.Set("OutSum", amount)
	u.Set("InvId", strconv.FormatInt(invID, 10))
	u.Set("Description", "Monthly subscription")
	u.Set("SignatureValue", sig)
	u.Set("IsTest", s.isTest)
	u.Set("Encoding", "utf-8")
	u.Set("Recurring", "true")
	if s.resultURL != "" {
		u.Set("ResultURL", s.resultURL)
	}
	if s.successURL != "" {
		u.Set("SuccessURL", s.successURL)
	}
	if s.failURL != "" {
		u.Set("FailURL", s.failURL)
	}
	if previous != nil {
		u.Set("PreviousInvoiceID", strconv.FormatInt(*previous, 10))
	}
	for k, v := range shp {
		u.Set("Shp_"+k, v)
	}
	paymentURL := s.baseURL + "?" + u.Encode()
	if err := s.repo.CreatePayment(ctx, &Payment{UserID: sub.UserID, SubscriptionID: &sub.ID, RobokassaInvoiceID: invID, Amount: amount, Status: "created", IsRecurring: true}); err != nil {
		return nil, err
	}
	if sub.FirstInvoiceID == nil {
		next := time.Now().UTC().AddDate(0, 1, 0)
		if err := s.repo.UpdateSubscription(ctx, sub.ID, map[string]interface{}{"first_invoice_id": invID, "next_billing_at": next}); err != nil {
			return nil, err
		}
	}
	return &InitPaymentResponse{InvID: invID, PaymentURL: paymentURL, Signature: sig, Status: "created"}, nil
}

// Legacy init endpoint compatibility.
func (s *Service) InitPayment(ctx context.Context, req InitPaymentRequest) (*InitPaymentResponse, error) {
	if err := s.validateInitConfig(); err != nil {
		return nil, err
	}
	if _, err := s.bookings.GetByID(ctx, req.BookingID); err != nil {
		return nil, fmt.Errorf("booking check failed: %w", err)
	}
	invID := time.Now().UnixNano()
	signature := s.generateSignatureForInit(req.OutSum, invID, req.ShpParams)
	u := url.Values{}
	u.Set("MerchantLogin", s.merchantLogin)
	u.Set("OutSum", req.OutSum)
	u.Set("InvId", strconv.FormatInt(invID, 10))
	u.Set("Description", req.Description)
	u.Set("SignatureValue", signature)
	u.Set("IsTest", s.isTest)
	u.Set("Encoding", "utf-8")
	if s.resultURL != "" {
		u.Set("ResultURL", s.resultURL)
	}
	if s.successURL != "" {
		u.Set("SuccessURL", s.successURL)
	}
	if s.failURL != "" {
		u.Set("FailURL", s.failURL)
	}
	for k, v := range req.ShpParams {
		u.Set("Shp_"+k, v)
	}
	paymentURL := s.baseURL + "?" + u.Encode()
	shpRaw, _ := json.Marshal(req.ShpParams)
	p := &RobokassaPayment{BookingID: req.BookingID, OutSum: req.OutSum, InvID: invID, Description: req.Description, Status: PaymentStatusCreated, Signature: signature, RobokassaURL: paymentURL, ShpParams: string(shpRaw)}
	if err := s.payments.Create(ctx, p); err != nil {
		return nil, err
	}
	if _, err := s.bookingWriter.UpdatePaymentStatusSystem(ctx, req.BookingID, booking.PaymentUnpaid); err != nil {
		return nil, err
	}
	return &InitPaymentResponse{InvID: invID, PaymentURL: paymentURL, Signature: signature, Status: string(PaymentStatusCreated)}, nil
}

func (s *Service) HandleResultCallback(ctx context.Context, outSum string, invID int64, signature string, shpParams map[string]string, rawBody string) (string, error) {
	if err := s.validateCallbackConfig(); err != nil {
		return "", err
	}
	if !strings.EqualFold(signature, s.generateSignatureForResult(outSum, invID, shpParams)) {
		return "", ErrInvalidSignature
	}
	if s.repo != nil {
		ack, err := s.handleResultV2(ctx, outSum, invID)
		if err == nil {
			return ack, nil
		}
		if !errors.Is(err, ErrPaymentNotFound) {
			return "", err
		}
	}
	if s.payments == nil {
		return "", ErrInvalidSignature
	}
	p, err := s.payments.GetByInvID(ctx, invID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", ErrInvalidSignature
		}
		return "", err
	}
	if !amountEqual(outSum, p.OutSum) {
		if err := s.payments.UpdateStatus(ctx, invID, PaymentStatusFailed, rawBody, "amount mismatch", nil); err != nil {
			return "", err
		}
		return "", ErrAmountMismatch
	}
	_, err = s.payments.MarkPaidIdempotent(ctx, invID, rawBody, time.Now().UTC())
	if err != nil {
		return "", err
	}
	if _, err := s.bookingWriter.UpdatePaymentStatusSystem(ctx, p.BookingID, booking.PaymentPaid); err != nil {
		return "", err
	}
	return "OK" + strconv.FormatInt(invID, 10), nil
}

func (s *Service) handleResultV2(ctx context.Context, outSum string, invID int64) (string, error) {
	p, err := s.repo.GetPaymentByInvoiceID(ctx, invID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", ErrPaymentNotFound
		}
		return "", err
	}
	if !amountEqual(outSum, p.Amount) {
		return "", ErrAmountMismatch
	}
	now := time.Now().UTC()
	changed, err := s.repo.MarkPaymentStatus(ctx, invID, "paid", &now)
	if err != nil {
		return "", err
	}
	if p.BookingID != nil {
		if _, err := s.bookingWriter.UpdatePaymentStatusSystem(ctx, *p.BookingID, booking.PaymentPaid); err != nil {
			return "", err
		}
	}
	if p.SubscriptionID != nil && changed {
		next := now.AddDate(0, 1, 0)
		if err := s.repo.UpdateSubscription(ctx, *p.SubscriptionID, map[string]interface{}{"status": "active", "next_billing_at": next}); err != nil {
			return "", err
		}
	}
	if !changed {
		s.loggerf("level=warn msg=duplicate robokassa result callback inv_id=%d", invID)
	}
	return "OK" + strconv.FormatInt(invID, 10), nil
}

func (s *Service) HandleSuccessCallback(ctx context.Context, outSum string, invID int64, signature string, shpParams map[string]string, rawBody string) (bool, error) {
	if err := s.validateInitConfig(); err != nil {
		return false, err
	}
	valid := strings.EqualFold(signature, s.generateSignatureForSuccess(outSum, invID, shpParams))
	if !valid {
		return false, ErrInvalidSignature
	}
	if s.repo != nil {
		p, err := s.repo.GetPaymentByInvoiceID(ctx, invID)
		if err == nil {
			if !amountEqual(outSum, p.Amount) {
				return false, ErrAmountMismatch
			}
			return true, nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return false, err
		}
	}
	if s.payments == nil {
		return false, ErrInvalidSignature
	}
	p, err := s.payments.GetByInvID(ctx, invID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, ErrInvalidSignature
		}
		return false, err
	}
	if !amountEqual(outSum, p.OutSum) {
		return false, ErrAmountMismatch
	}
	if err := s.payments.UpdateStatusPendingIfNotPaid(ctx, invID, rawBody); err != nil {
		return false, err
	}
	return true, nil
}

func (s *Service) validateInitConfig() error {
	if strings.TrimSpace(s.merchantLogin) == "" || strings.TrimSpace(s.password1) == "" {
		return ErrMisconfigured
	}
	return nil
}

func (s *Service) validateCallbackConfig() error {
	if strings.TrimSpace(s.password2) == "" {
		return ErrMisconfigured
	}
	return nil
}

func (s *Service) FailPayment(ctx context.Context, invID int64) error {
	if s.repo == nil {
		return nil
	}
	_, err := s.repo.MarkPaymentStatus(ctx, invID, "failed", nil)
	return err
}

func (s *Service) CancelSubscription(ctx context.Context, userID int64) error {
	sub, err := s.repo.GetSubscriptionByUserID(ctx, userID)
	if err != nil {
		return err
	}
	return s.repo.UpdateSubscription(ctx, sub.ID, map[string]interface{}{"status": "canceled"})
}

func (s *Service) GetMySubscription(ctx context.Context, userID int64) (*RecurringSubscription, error) {
	return s.repo.GetSubscriptionByUserID(ctx, userID)
}

func (s *Service) generateSignatureForInit(outSum string, invID int64, shpParams map[string]string) string {
	parts := []string{s.merchantLogin, outSum, strconv.FormatInt(invID, 10), s.password1}
	parts = append(parts, flattenShpParams(shpParams)...)
	return s.hashHex(strings.Join(parts, ":"))
}
func (s *Service) generateSignatureForResult(outSum string, invID int64, shpParams map[string]string) string {
	parts := []string{outSum, strconv.FormatInt(invID, 10), s.password2}
	parts = append(parts, flattenShpParams(shpParams)...)
	return s.hashHex(strings.Join(parts, ":"))
}
func (s *Service) generateSignatureForSuccess(outSum string, invID int64, shpParams map[string]string) string {
	parts := []string{outSum, strconv.FormatInt(invID, 10), s.password1}
	parts = append(parts, flattenShpParams(shpParams)...)
	return s.hashHex(strings.Join(parts, ":"))
}

func flattenShpParams(shp map[string]string) []string {
	keys := make([]string, 0, len(shp))
	for k := range shp {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]string, 0, len(keys))
	for _, k := range keys {
		out = append(out, "Shp_"+k+"="+shp[k])
	}
	return out
}
func amountEqual(a, b string) bool {
	ar, ok := new(big.Rat).SetString(strings.TrimSpace(a))
	if !ok {
		return false
	}
	br, ok := new(big.Rat).SetString(strings.TrimSpace(b))
	if !ok {
		return false
	}
	return ar.Cmp(br) == 0
}
func (s *Service) hashHex(input string) string {
	switch strings.ToLower(s.hashAlgo) {
	case "", "md5":
		h := md5.Sum([]byte(input))
		return strings.ToUpper(hex.EncodeToString(h[:]))
	case "sha256", "sha-256":
		h := sha256.Sum256([]byte(input))
		return strings.ToUpper(hex.EncodeToString(h[:]))
	default:
		h := md5.Sum([]byte(input))
		return strings.ToUpper(hex.EncodeToString(h[:]))
	}
}
