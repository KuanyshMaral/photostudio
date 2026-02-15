package wallet

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func setupTestRouter(t *testing.T) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)

	dsn := fmt.Sprintf("file:wallet_handler_test_%s?mode=memory&cache=shared", t.Name())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("failed to open sqlite db: %v", err)
	}
	if err := db.AutoMigrate(&FakeWallet{}, &FakeTransaction{}); err != nil {
		t.Fatalf("failed to migrate db: %v", err)
	}

	h := NewHandler(NewService(db))
	r := gin.New()
	r.Use(func(c *gin.Context) {
		if userID := c.GetHeader("X-Test-User-ID"); userID != "" {
			c.Set("user_id", int64(42))
		}
		c.Next()
	})

	v1 := r.Group("/api/v1")
	h.RegisterRoutes(v1)
	return r
}

func doJSONRequest(r http.Handler, method, path string, body any, authorized bool) *httptest.ResponseRecorder {
	var reader *bytes.Reader
	if body == nil {
		reader = bytes.NewReader(nil)
	} else {
		b, _ := json.Marshal(body)
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

func TestWalletEndpoints_Unauthorized(t *testing.T) {
	r := setupTestRouter(t)

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
		rr := doJSONRequest(r, tc.method, tc.path, tc.body, false)
		if rr.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401 for %s %s, got %d", tc.method, tc.path, rr.Code)
		}
	}
}

func TestWalletEndpoints_FullFlow(t *testing.T) {
	r := setupTestRouter(t)

	// GET /wallets/me initial balance
	rr := doJSONRequest(r, http.MethodGet, "/api/v1/wallets/me", nil, true)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 for get wallet, got %d body=%s", rr.Code, rr.Body.String())
	}
	var walletResp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &walletResp); err != nil {
		t.Fatalf("invalid get wallet response: %v", err)
	}
	if walletResp["balance"].(float64) != 0 {
		t.Fatalf("expected initial balance 0, got %v", walletResp["balance"])
	}

	// invalid add
	rr = doJSONRequest(r, http.MethodPost, "/api/v1/wallets/me/add", map[string]any{"amount": -5}, true)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid add, got %d body=%s", rr.Code, rr.Body.String())
	}

	// add funds
	rr = doJSONRequest(r, http.MethodPost, "/api/v1/wallets/me/add", map[string]any{"amount": 150}, true)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 for add, got %d body=%s", rr.Code, rr.Body.String())
	}

	// overspend
	rr = doJSONRequest(r, http.MethodPost, "/api/v1/wallets/me/spend", map[string]any{"amount": 500}, true)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for overspend, got %d body=%s", rr.Code, rr.Body.String())
	}

	// valid spend
	rr = doJSONRequest(r, http.MethodPost, "/api/v1/wallets/me/spend", map[string]any{"amount": 40}, true)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 for spend, got %d body=%s", rr.Code, rr.Body.String())
	}

	// list txns should have ADD + SPEND
	rr = doJSONRequest(r, http.MethodGet, "/api/v1/wallets/me/transactions", nil, true)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 for transactions, got %d body=%s", rr.Code, rr.Body.String())
	}
	var txResp struct {
		Transactions []map[string]any `json:"transactions"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &txResp); err != nil {
		t.Fatalf("invalid transactions response: %v", err)
	}
	if len(txResp.Transactions) != 2 {
		t.Fatalf("expected 2 transactions, got %d", len(txResp.Transactions))
	}

	// final balance should be 110
	rr = doJSONRequest(r, http.MethodGet, "/api/v1/wallets/me", nil, true)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 for final wallet get, got %d body=%s", rr.Code, rr.Body.String())
	}
	walletResp = map[string]any{}
	if err := json.Unmarshal(rr.Body.Bytes(), &walletResp); err != nil {
		t.Fatalf("invalid final wallet response: %v", err)
	}
	if walletResp["balance"].(float64) != 110 {
		t.Fatalf("expected final balance 110, got %v", walletResp["balance"])
	}
}
