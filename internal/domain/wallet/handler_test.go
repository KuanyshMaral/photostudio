package wallet

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type mockWalletService struct {
	wallet         *FakeWallet
	txns           []FakeTransaction
	addErr         error
	spendErr       error
	walletErr      error
	listErr        error
	addCallCount   int
	spendCallCount int
}

func (m *mockWalletService) GetOrCreateWallet(_ context.Context, _ int64) (*FakeWallet, error) {
	if m.walletErr != nil {
		return nil, m.walletErr
	}
	if m.wallet == nil {
		m.wallet = &FakeWallet{ID: uuid.New(), UserID: 42, Balance: 0}
	}
	return m.wallet, nil
}

func (m *mockWalletService) Add(_ context.Context, _ int64, amount int64) (*FakeWallet, *FakeTransaction, error) {
	m.addCallCount++
	if m.addErr != nil {
		return nil, nil, m.addErr
	}
	if m.wallet == nil {
		m.wallet = &FakeWallet{ID: uuid.New(), UserID: 42, Balance: 0}
	}
	m.wallet.Balance += amount
	txn := FakeTransaction{ID: uuid.New(), WalletID: m.wallet.ID, Amount: amount, Type: TransactionTypeAdd}
	m.txns = append([]FakeTransaction{txn}, m.txns...)
	return m.wallet, &txn, nil
}

func (m *mockWalletService) Spend(_ context.Context, _ int64, amount int64) (*FakeWallet, *FakeTransaction, error) {
	m.spendCallCount++
	if m.spendErr != nil {
		return nil, nil, m.spendErr
	}
	if m.wallet == nil {
		m.wallet = &FakeWallet{ID: uuid.New(), UserID: 42, Balance: 0}
	}
	if m.wallet.Balance < amount {
		return nil, nil, ErrInsufficientFunds
	}
	m.wallet.Balance -= amount
	txn := FakeTransaction{ID: uuid.New(), WalletID: m.wallet.ID, Amount: amount, Type: TransactionTypeSpend}
	m.txns = append([]FakeTransaction{txn}, m.txns...)
	return m.wallet, &txn, nil
}

func (m *mockWalletService) ListTransactions(_ context.Context, _ int64) ([]FakeTransaction, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	return m.txns, nil
}

func setupTestRouter() (*gin.Engine, *mockWalletService) {
	gin.SetMode(gin.TestMode)
	mockSvc := &mockWalletService{}
	h := NewHandler(mockSvc)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		if c.GetHeader("X-Test-User-ID") != "" {
			c.Set("user_id", int64(42))
		}
		c.Next()
	})
	v1 := r.Group("/api/v1")
	h.RegisterRoutes(v1)
	return r, mockSvc
}

func doJSONRequest(t *testing.T, r http.Handler, method, path string, body any, authorized bool) *httptest.ResponseRecorder {
	t.Helper()
	var reader *bytes.Reader
	if body == nil {
		reader = bytes.NewReader(nil)
	} else {
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal request body: %v", err)
		}
		reader = bytes.NewReader(b)
	}
	req := httptest.NewRequest(method, path, reader)
	req.Header.Set("Content-Type", "application/json")
	if authorized {
		req.Header.Set("X-Test-User-ID", "42")
	}
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	return rr
}

func decodeJSON(t *testing.T, data []byte, out any) {
	t.Helper()
	if err := json.Unmarshal(data, out); err != nil {
		t.Fatalf("decode json: %v, body=%s", err, string(data))
	}
}

func TestWalletEndpoints_Unauthorized(t *testing.T) {
	r, _ := setupTestRouter()
	cases := []struct {
		method string
		path   string
		body   any
	}{
		{method: http.MethodGet, path: "/api/v1/wallets/me"},
		{method: http.MethodPost, path: "/api/v1/wallets/me/add", body: map[string]any{"amount": 10}},
		{method: http.MethodPost, path: "/api/v1/wallets/me/spend", body: map[string]any{"amount": 10}},
		{method: http.MethodGet, path: "/api/v1/wallets/me/transactions"},
	}
	for _, tc := range cases {
		rr := doJSONRequest(t, r, tc.method, tc.path, tc.body, false)
		if rr.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401 for %s %s, got %d", tc.method, tc.path, rr.Code)
		}
	}
}

func TestWalletEndpoints_InvalidPayloadDoesNotCallService(t *testing.T) {
	r, svc := setupTestRouter()

	// missing amount
	rr := doJSONRequest(t, r, http.MethodPost, "/api/v1/wallets/me/add", map[string]any{}, true)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing amount, got %d", rr.Code)
	}
	if svc.addCallCount != 0 {
		t.Fatalf("expected add service not called, got %d", svc.addCallCount)
	}

	// zero amount fails gt=0
	rr = doJSONRequest(t, r, http.MethodPost, "/api/v1/wallets/me/spend", map[string]any{"amount": 0}, true)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for zero amount, got %d", rr.Code)
	}
	if svc.spendCallCount != 0 {
		t.Fatalf("expected spend service not called, got %d", svc.spendCallCount)
	}
}

func TestWalletEndpoints_FullFlow(t *testing.T) {
	r, _ := setupTestRouter()

	rr := doJSONRequest(t, r, http.MethodGet, "/api/v1/wallets/me", nil, true)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 get wallet, got %d", rr.Code)
	}
	var walletResp map[string]any
	decodeJSON(t, rr.Body.Bytes(), &walletResp)
	if walletResp["balance"].(float64) != 0 {
		t.Fatalf("expected 0, got %v", walletResp["balance"])
	}

	rr = doJSONRequest(t, r, http.MethodPost, "/api/v1/wallets/me/add", map[string]any{"amount": -5}, true)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 invalid add, got %d", rr.Code)
	}

	rr = doJSONRequest(t, r, http.MethodPost, "/api/v1/wallets/me/add", map[string]any{"amount": 150}, true)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 add, got %d body=%s", rr.Code, rr.Body.String())
	}

	rr = doJSONRequest(t, r, http.MethodPost, "/api/v1/wallets/me/spend", map[string]any{"amount": 500}, true)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 overspend, got %d", rr.Code)
	}

	rr = doJSONRequest(t, r, http.MethodPost, "/api/v1/wallets/me/spend", map[string]any{"amount": 40}, true)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 spend, got %d", rr.Code)
	}

	rr = doJSONRequest(t, r, http.MethodGet, "/api/v1/wallets/me/transactions", nil, true)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 txns, got %d", rr.Code)
	}
	var txResp struct {
		Transactions []map[string]any `json:"transactions"`
	}
	decodeJSON(t, rr.Body.Bytes(), &txResp)
	if len(txResp.Transactions) != 2 {
		t.Fatalf("expected 2 txns, got %d", len(txResp.Transactions))
	}
	if txResp.Transactions[0]["type"] != TransactionTypeSpend || txResp.Transactions[1]["type"] != TransactionTypeAdd {
		t.Fatalf("unexpected txn order/types: %+v", txResp.Transactions)
	}

	rr = doJSONRequest(t, r, http.MethodGet, "/api/v1/wallets/me", nil, true)
	decodeJSON(t, rr.Body.Bytes(), &walletResp)
	if walletResp["balance"].(float64) != 110 {
		t.Fatalf("expected 110, got %v", walletResp["balance"])
	}
}

func TestWalletEndpoints_ServiceErrors(t *testing.T) {
	r, svc := setupTestRouter()
	svc.walletErr = errors.New("db down")
	if rr := doJSONRequest(t, r, http.MethodGet, "/api/v1/wallets/me", nil, true); rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rr.Code)
	}

	svc.walletErr = nil
	svc.addErr = errors.New("db down")
	if rr := doJSONRequest(t, r, http.MethodPost, "/api/v1/wallets/me/add", map[string]any{"amount": 10}, true); rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rr.Code)
	}

	svc.addErr = nil
	svc.spendErr = errors.New("db down")
	if rr := doJSONRequest(t, r, http.MethodPost, "/api/v1/wallets/me/spend", map[string]any{"amount": 10}, true); rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rr.Code)
	}

	svc.spendErr = nil
	svc.listErr = errors.New("db down")
	if rr := doJSONRequest(t, r, http.MethodGet, "/api/v1/wallets/me/transactions", nil, true); rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rr.Code)
	}
}
