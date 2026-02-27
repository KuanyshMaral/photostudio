package payment

import (
	"context"
	"crypto/md5"
	"crypto/rand"
	"crypto/sha256"
	"crypto/sha512"
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
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

var (
	ErrInvalidSignature      = errors.New("invalid signature")
	ErrAmountMismatch        = errors.New("amount mismatch")
	ErrReplayDetected        = errors.New("replay detected")
	ErrPaymentNotFound       = errors.New("payment not found")
	ErrMisconfigured         = errors.New("robokassa is misconfigured")
	ErrInvalidAmount         = errors.New("invalid amount")
	ErrInvalidSubscriptionID = errors.New("invalid subscription_id: must be UUID")
)

var invSeq uint32

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
	isTest := normalizeRobokassaIsTest(envOrDefault("ROBOKASSA_IS_TEST", "0"))
	hashAlgo := normalizeRobokassaHashAlgo(envOrDefault("ROBOKASSA_HASH_ALGO", "sha256"))
	password1, password2 := selectRobokassaPasswords(isTest)
	merchantLogin := strings.TrimSpace(os.Getenv("ROBOKASSA_MERCHANT_LOGIN"))
	baseURL := robokassaBaseURL(merchantLogin)
	if isTest == "1" {
		if v := strings.TrimSpace(os.Getenv("ROBOKASSA_TEST_BASE_URL")); v != "" {
			baseURL = v
		}
	} else if v := strings.TrimSpace(os.Getenv("ROBOKASSA_PROD_BASE_URL")); v != "" {
		baseURL = v
	}
	return &Service{payments: payments, bookings: bookings, bookingWriter: bookingWriter, repo: r, loggerf: loggerf,
		merchantLogin: merchantLogin,
		password1:     password1, password2: password2,
		baseURL:   baseURL,
		resultURL: os.Getenv("ROBOKASSA_RESULT_URL"), successURL: os.Getenv("ROBOKASSA_SUCCESS_URL"), failURL: os.Getenv("ROBOKASSA_FAIL_URL"),
		frontSuccess: os.Getenv("ROBOKASSA_FRONTEND_SUCCESS_URL"), frontFail: os.Getenv("ROBOKASSA_FRONTEND_FAIL_URL"),
		isTest:   isTest,
		hashAlgo: hashAlgo,
	}
}

func normalizeRobokassaHashAlgo(v string) string {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "sha256":
		return "sha256"
	case "sha512":
		return "sha512"
	default:
		return "md5"
	}
}

func selectRobokassaPasswords(isTest string) (string, string) {
	if isTest == "1" {
		p1 := firstEnv(
			"ROBOKASSA_TEST_PASSWORD_1",
			"ROBOKASSA_TEST_PASSWORD1",
		)
		p2 := firstEnv(
			"ROBOKASSA_TEST_PASSWORD_2",
			"ROBOKASSA_TEST_PASSWORD2",
		)
		return p1, p2
	}

	p1 := firstEnv(
		"ROBOKASSA_PROD_PASSWORD_1",
		"ROBOKASSA_PROD_PASSWORD1",
		"ROBOKASSA_PASSWORD_1",
		"ROBOKASSA_PASSWORD1",
	)
	p2 := firstEnv(
		"ROBOKASSA_PROD_PASSWORD_2",
		"ROBOKASSA_PROD_PASSWORD2",
		"ROBOKASSA_PASSWORD_2",
		"ROBOKASSA_PASSWORD2",
	)
	return p1, p2
}

func firstEnv(names ...string) string {
	for _, name := range names {
		if v := strings.TrimSpace(os.Getenv(name)); v != "" {
			return v
		}
	}
	return ""
}

func robokassaBaseURL(merchantLogin string) string {
	if configured := strings.TrimSpace(os.Getenv("ROBOKASSA_BASE_URL")); configured != "" {
		return configured
	}
	if strings.HasSuffix(strings.ToLower(strings.TrimSpace(merchantLogin)), "_kz") {
		return "https://auth.robokassa.kz/Merchant/Index.aspx"
	}
	return "https://auth.robokassa.ru/Merchant/Index.aspx"
}

func normalizeRobokassaIsTest(v string) string {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "1", "true", "yes", "on":
		return "1"
	default:
		return "0"
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
	normalizedAmount, err := normalizeAmount(amount)
	if err != nil {
		return nil, err
	}
	bk, err := s.bookings.GetByID(ctx, bookingID)
	if err != nil {
		return nil, err
	}
	if bk.UserID != userID {
		return nil, fmt.Errorf("booking does not belong to user")
	}
	if !amountEqual(normalizedAmount, fmt.Sprintf("%.2f", bk.TotalPrice)) {
		return nil, ErrAmountMismatch
	}
	if subscriptionID != nil {
		sid := strings.TrimSpace(*subscriptionID)
		if sid == "" {
			subscriptionID = nil
		} else {
			if _, err := uuid.Parse(sid); err != nil {
				return nil, ErrInvalidSubscriptionID
			}
			subscriptionID = &sid
		}
	}

	// Recurring payments are temporarily disabled until they are enabled on Robokassa side
	// (merchant error 34/71: recurring service is not allowed for the store).
	// Keep create-payment flow strictly one-time even if client sends recurring params.
	recurring = false
	subscriptionID = nil
	previousInvoiceID = nil

	invID := generateInvoiceID()
	shp := map[string]string{"user_id": strconv.FormatInt(userID, 10), "booking_id": strconv.FormatInt(bookingID, 10)}
	for k, v := range sanitizeShpParams(shpParams) {
		if _, protected := shp[k]; protected {
			continue
		}
		// Do not let client-provided Shp params re-introduce recurring-only values.
		if strings.EqualFold(k, "subscription_id") {
			continue
		}
		shp[k] = v
	}

	// Robokassa error 34 is usually caused by signature mismatches: parameter order,
	// amount formatting, or missing/extra Shp_* values. Signature must be built from
	// MerchantLogin:OutSum:InvId:Password1 plus alphabetically sorted Shp_* params.
	sig := s.generateSignatureForInit(normalizedAmount, invID, shp)

	u := url.Values{}
	u.Set("MerchantLogin", s.merchantLogin)
	u.Set("OutSum", normalizedAmount)
	u.Set("InvId", strconv.FormatInt(invID, 10))
	u.Set("Description", description)
	u.Set("SignatureValue", sig)
	if s.isTest == "1" {
		u.Set("IsTest", "1")
	}
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
	addRecurringInitParams(u, recurring, previousInvoiceID)
	for k, v := range shp {
		u.Set("Shp_"+k, v)
	}
	paymentURL := s.baseURL + "?" + u.Encode()

	p := &Payment{UserID: userID, BookingID: &bookingID, SubscriptionID: subscriptionID, RobokassaInvoiceID: invID, Amount: normalizedAmount, Status: "created", IsRecurring: recurring}
	if err := s.repo.CreatePayment(ctx, p); err != nil {
		return nil, err
	}
	if _, err := s.bookingWriter.UpdatePaymentStatusSystem(ctx, bookingID, booking.PaymentUnpaid); err != nil {
		return nil, err
	}

	return &InitPaymentResponse{InvID: invID, PaymentURL: paymentURL, Signature: sig, Status: "created"}, nil
}

func normalizePreviousInvoiceID(recurring bool, previousInvoiceID *int64) *int64 {
	if !recurring || previousInvoiceID == nil {
		return nil
	}
	if *previousInvoiceID <= 0 {
		return nil
	}
	return previousInvoiceID
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
	normalizedAmount, err := normalizeAmount(amount)
	if err != nil {
		return nil, err
	}
	invID := generateInvoiceID()
	shp := map[string]string{"user_id": strconv.FormatInt(sub.UserID, 10), "subscription_id": sub.ID}
	sig := s.generateSignatureForInit(normalizedAmount, invID, shp)
	u := url.Values{}
	u.Set("MerchantLogin", s.merchantLogin)
	u.Set("OutSum", normalizedAmount)
	u.Set("InvId", strconv.FormatInt(invID, 10))
	u.Set("Description", "Monthly subscription")
	u.Set("SignatureValue", sig)
	if s.isTest == "1" {
		u.Set("IsTest", "1")
	}
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
	if err := s.repo.CreatePayment(ctx, &Payment{UserID: sub.UserID, SubscriptionID: &sub.ID, RobokassaInvoiceID: invID, Amount: normalizedAmount, Status: "created", IsRecurring: true}); err != nil {
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
	normalizedAmount, err := normalizeAmount(req.OutSum)
	if err != nil {
		return nil, err
	}
	invID := generateInvoiceID()
	shp := sanitizeShpParams(req.ShpParams)
	signature := s.generateSignatureForInit(normalizedAmount, invID, shp)
	u := url.Values{}
	u.Set("MerchantLogin", s.merchantLogin)
	u.Set("OutSum", normalizedAmount)
	u.Set("InvId", strconv.FormatInt(invID, 10))
	u.Set("Description", req.Description)
	u.Set("SignatureValue", signature)
	if s.isTest == "1" {
		u.Set("IsTest", "1")
	}
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
	for k, v := range shp {
		u.Set("Shp_"+k, v)
	}
	paymentURL := s.baseURL + "?" + u.Encode()
	shpRaw, _ := json.Marshal(shp)
	p := &RobokassaPayment{BookingID: req.BookingID, OutSum: normalizedAmount, InvID: invID, Description: req.Description, Status: PaymentStatusCreated, Signature: signature, RobokassaURL: paymentURL, ShpParams: string(shpRaw)}
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
	changed, err := s.payments.MarkPaidIdempotent(ctx, invID, rawBody, time.Now().UTC())
	if err != nil {
		return "", err
	}
	if changed {
		if _, err := s.bookingWriter.UpdatePaymentStatusSystem(ctx, p.BookingID, booking.PaymentPaid); err != nil {
			return "", err
		}
	} else {
		s.loggerf("level=warn msg=duplicate robokassa result callback inv_id=%d", invID)
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
		if changed {
			if _, err := s.bookingWriter.UpdatePaymentStatusSystem(ctx, *p.BookingID, booking.PaymentPaid); err != nil {
				return "", err
			}
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

func addRecurringInitParams(values url.Values, recurring bool, previousInvoiceID *int64) {
	if !recurring {
		return
	}
	values.Set("Recurring", "true")
	if previousInvoiceID != nil {
		values.Set("PreviousInvoiceID", strconv.FormatInt(*previousInvoiceID, 10))
	}
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
	switch s.hashAlgo {
	case "sha256":
		h := sha256.Sum256([]byte(input))
		return hex.EncodeToString(h[:])
	case "sha512":
		h := sha512.Sum512([]byte(input))
		return hex.EncodeToString(h[:])
	default:
		h := md5.Sum([]byte(input))
		return hex.EncodeToString(h[:])
	}
}

func sanitizeShpParams(shp map[string]string) map[string]string {
	keys := make([]string, 0, len(shp))
	for rawK := range shp {
		keys = append(keys, rawK)
	}
	sort.Slice(keys, func(i, j int) bool {
		li := strings.ToLower(strings.TrimSpace(keys[i]))
		lj := strings.ToLower(strings.TrimSpace(keys[j]))
		if li == lj {
			return keys[i] < keys[j]
		}
		return li < lj
	})

	out := make(map[string]string, len(keys))
	for _, rawK := range keys {
		rawV := shp[rawK]
		k := strings.TrimSpace(rawK)
		if k == "" {
			continue
		}
		if len(k) >= 4 && strings.EqualFold(k[:4], "shp_") {
			k = k[4:]
		}
		k = strings.ToLower(strings.TrimSpace(k))
		if k == "" || strings.ContainsAny(k, "=:\x00") {
			continue
		}
		v := strings.TrimSpace(rawV)
		if strings.ContainsAny(v, "\x00") {
			continue
		}
		out[k] = v
	}
	return out
}

func normalizeAmount(v string) (string, error) {
	v = strings.TrimSpace(v)
	if v == "" {
		return "", ErrInvalidAmount
	}
	if strings.HasPrefix(v, "+") {
		v = strings.TrimPrefix(v, "+")
	}
	if strings.HasPrefix(v, "-") {
		return "", ErrInvalidAmount
	}
	if strings.Count(v, ".") > 1 {
		return "", ErrInvalidAmount
	}
	parts := strings.SplitN(v, ".", 2)
	if len(parts[0]) == 0 {
		return "", ErrInvalidAmount
	}
	for _, ch := range parts[0] {
		if ch < '0' || ch > '9' {
			return "", ErrInvalidAmount
		}
	}
	frac := ""
	if len(parts) == 2 {
		frac = parts[1]
		if len(frac) == 0 || len(frac) > 2 {
			return "", ErrInvalidAmount
		}
		for _, ch := range frac {
			if ch < '0' || ch > '9' {
				return "", ErrInvalidAmount
			}
		}
	}
	whole := strings.TrimLeft(parts[0], "0")
	if whole == "" {
		whole = "0"
	}
	if len(frac) == 0 {
		frac = "00"
	} else if len(frac) == 1 {
		frac += "0"
	}
	return whole + "." + frac, nil
}

func generateInvoiceID() int64 {
	const maxRobokassaInvoiceID = int64(2_147_483_647)

	n, err := rand.Int(rand.Reader, big.NewInt(maxRobokassaInvoiceID))
	if err == nil {
		return n.Int64() + 1
	}

	// fallback preserves process-level uniqueness while staying in Robokassa range
	next := atomic.AddUint32(&invSeq, 1)
	return int64(next%uint32(maxRobokassaInvoiceID)) + 1
}
