package payment

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"photostudio/internal/domain"
)

var (
	ErrInvalidSignature = errors.New("invalid signature")
	ErrAmountMismatch   = errors.New("amount mismatch")
)

type Service struct {
	payments      paymentRepo
	bookings      bookingReader
	bookingWriter bookingPaymentWriter
	loggerf       func(format string, args ...interface{})

	merchantLogin string
	password1     string
	password2     string
	baseURL       string
	resultURL     string
	successURL    string
	isTest        string
}

func NewService(payments paymentRepo, bookings bookingReader, bookingWriter bookingPaymentWriter, loggerf func(format string, args ...interface{})) *Service {
	if loggerf == nil {
		loggerf = func(string, ...interface{}) {}
	}
	return &Service{
		payments:      payments,
		bookings:      bookings,
		bookingWriter: bookingWriter,
		loggerf:       loggerf,
		merchantLogin: os.Getenv("ROBOKASSA_MERCHANT_LOGIN"),
		password1:     os.Getenv("ROBOKASSA_PASSWORD1"),
		password2:     os.Getenv("ROBOKASSA_PASSWORD2"),
		baseURL:       envOrDefault("ROBOKASSA_BASE_URL", "https://auth.robokassa.ru/Merchant/Index.aspx"),
		resultURL:     os.Getenv("ROBOKASSA_RESULT_URL"),
		successURL:    os.Getenv("ROBOKASSA_SUCCESS_URL"),
		isTest:        envOrDefault("ROBOKASSA_IS_TEST", "1"),
	}
}

func envOrDefault(name, def string) string {
	if v := os.Getenv(name); v != "" {
		return v
	}
	return def
}

func (s *Service) InitPayment(ctx context.Context, req InitPaymentRequest) (*InitPaymentResponse, error) {
	if s.merchantLogin == "" || s.password1 == "" || s.password2 == "" {
		return nil, fmt.Errorf("robokassa credentials are not configured")
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
	if s.resultURL != "" {
		u.Set("ResultURL", s.resultURL)
	}
	if s.successURL != "" {
		u.Set("SuccessURL", s.successURL)
	}
	for k, v := range req.ShpParams {
		u.Set("Shp_"+k, v)
	}
	paymentURL := s.baseURL + "?" + u.Encode()

	shpRaw, _ := json.Marshal(req.ShpParams)
	p := &domain.RobokassaPayment{
		BookingID:    req.BookingID,
		OutSum:       req.OutSum,
		InvID:        invID,
		Description:  req.Description,
		Status:       domain.PaymentStatusCreated,
		Signature:    signature,
		RobokassaURL: paymentURL,
		ShpParams:    string(shpRaw),
	}
	if err := s.payments.Create(ctx, p); err != nil {
		return nil, fmt.Errorf("save payment failed: %w", err)
	}
	if _, err := s.bookingWriter.UpdatePaymentStatus(ctx, req.BookingID, domain.PaymentUnpaid); err != nil {
		s.loggerf("level=error msg=failed to sync booking payment status on init booking_id=%d err=%v", req.BookingID, err)
	}

	return &InitPaymentResponse{InvID: invID, PaymentURL: paymentURL, Signature: signature, Status: string(domain.PaymentStatusCreated)}, nil
}

func (s *Service) HandleResultCallback(ctx context.Context, outSum string, invID int64, signature string, shpParams map[string]string, rawBody string) (string, error) {
	valid := strings.EqualFold(signature, s.generateSignatureForResult(outSum, invID, shpParams))
	s.loggerf("level=info msg=robokassa result signature validation inv_id=%d signature_valid=%t", invID, valid)
	if !valid {
		_ = s.payments.UpdateStatus(ctx, invID, domain.PaymentStatusFailed, rawBody, "invalid signature", nil)
		return "", ErrInvalidSignature
	}

	p, err := s.payments.GetByInvID(ctx, invID)
	if err != nil {
		return "", err
	}
	if !amountEqual(outSum, p.OutSum) {
		reason := fmt.Sprintf("amount mismatch callback=%s expected=%s", outSum, p.OutSum)
		_ = s.payments.UpdateStatus(ctx, invID, domain.PaymentStatusFailed, rawBody, reason, nil)
		return "", ErrAmountMismatch
	}

	changed, err := s.payments.MarkPaidIdempotent(ctx, invID, rawBody, time.Now().UTC())
	if err != nil {
		return "", err
	}
	_, berr := s.bookingWriter.UpdatePaymentStatus(ctx, p.BookingID, domain.PaymentPaid)
	if berr != nil {
		s.loggerf("level=error msg=failed to update booking payment status to paid booking_id=%d err=%v", p.BookingID, berr)
	}

	if !changed {
		s.loggerf("level=info msg=idempotent callback already paid inv_id=%d", invID)
	}
	return "OK" + strconv.FormatInt(invID, 10), nil
}

func (s *Service) HandleSuccessCallback(ctx context.Context, outSum string, invID int64, signature string, shpParams map[string]string, rawBody string) (bool, error) {
	if err := s.payments.SaveSuccessRawBody(ctx, invID, rawBody); err != nil {
		s.loggerf("level=error msg=failed to save success callback body inv_id=%d err=%v", invID, err)
	}
	valid := strings.EqualFold(signature, s.generateSignatureForSuccess(outSum, invID, shpParams))
	s.loggerf("level=info msg=robokassa success signature validation inv_id=%d signature_valid=%t", invID, valid)
	if !valid {
		return false, ErrInvalidSignature
	}

	p, err := s.payments.GetByInvID(ctx, invID)
	if err != nil {
		return false, err
	}
	if !amountEqual(outSum, p.OutSum) {
		s.loggerf("level=error msg=amount mismatch on success callback inv_id=%d callback_out_sum=%s expected_out_sum=%s", invID, outSum, p.OutSum)
		return false, ErrAmountMismatch
	}

	if err := s.payments.UpdateStatusPendingIfNotPaid(ctx, invID, rawBody); err != nil {
		s.loggerf("level=error msg=failed to set pending status from success callback inv_id=%d err=%v", invID, err)
		return false, err
	}
	return true, nil
}

func (s *Service) generateSignatureForInit(outSum string, invID int64, shpParams map[string]string) string {
	parts := []string{s.merchantLogin, outSum, strconv.FormatInt(invID, 10), s.password1}
	parts = append(parts, flattenShpParams(shpParams)...)
	return md5Hex(strings.Join(parts, ":"))
}

func (s *Service) generateSignatureForResult(outSum string, invID int64, shpParams map[string]string) string {
	parts := []string{outSum, strconv.FormatInt(invID, 10), s.password2}
	parts = append(parts, flattenShpParams(shpParams)...)
	return md5Hex(strings.Join(parts, ":"))
}

func (s *Service) generateSignatureForSuccess(outSum string, invID int64, shpParams map[string]string) string {
	parts := []string{outSum, strconv.FormatInt(invID, 10), s.password1}
	parts = append(parts, flattenShpParams(shpParams)...)
	return md5Hex(strings.Join(parts, ":"))
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

func md5Hex(s string) string {
	h := md5.Sum([]byte(s))
	return strings.ToUpper(hex.EncodeToString(h[:]))
}
