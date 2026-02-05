package mwork

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"photostudio/internal/database"
	"photostudio/internal/domain"
	"photostudio/internal/middleware"
	"photostudio/internal/repository"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type syncResponse struct {
	Data SyncUserResponse `json:"data"`
}

type errorResponse struct {
	Error struct {
		Code    string         `json:"code"`
		Message string         `json:"message"`
		Details map[string]any `json:"details"`
	} `json:"error"`
}

func setupRouter(t *testing.T) (*gin.Engine, *gorm.DB) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	db, err := database.Connect(":memory:")
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&domain.User{}))

	userRepo := repository.NewUserRepository(db)
	service := NewService(userRepo)
	handler := NewHandler(service)

	router := gin.New()
	internal := router.Group("/internal")
	internal.Use(middleware.InternalTokenAuth())
	handler.RegisterRoutes(internal)

	return router, db
}

func performRequest(router *gin.Engine, method, path string, body interface{}, token string) *httptest.ResponseRecorder {
	return performRequestWithHeaders(router, method, path, body, token, nil)
}

func performRequestWithHeaders(router *gin.Engine, method, path string, body interface{}, token string, headers map[string]string) *httptest.ResponseRecorder {
	var buf bytes.Buffer
	if body != nil {
		_ = json.NewEncoder(&buf).Encode(body)
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	return resp
}

func TestSyncUserCreate(t *testing.T) {
	t.Setenv("MWORK_SYNC_TOKEN", "test-token")
	router, db := setupRouter(t)

	req := SyncUserRequest{
		MworkUserID: uuid.NewString(),
		Email:       "new.user@example.com",
		Role:        "model",
	}

	resp := performRequest(router, http.MethodPost, "/internal/mwork/users/sync", req, "test-token")
	require.Equal(t, http.StatusCreated, resp.Code)
	requireJSONContentType(t, resp)

	var payload syncResponse
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&payload))
	require.Equal(t, req.MworkUserID, payload.Data.MworkUserID)

	var user domain.User
	err := db.Where("mwork_user_id = ?", req.MworkUserID).First(&user).Error
	require.NoError(t, err)
	require.Equal(t, "new.user@example.com", user.Email)
}

func TestSyncUserUpdateByMworkUserID(t *testing.T) {
	t.Setenv("MWORK_SYNC_TOKEN", "test-token")
	router, db := setupRouter(t)

	user := domain.User{
		Email:        "old@example.com",
		PasswordHash: "hash",
		Role:         domain.RoleClient,
		Name:         "Old Name",
		MworkUserID:  uuid.NewString(),
		MworkRole:    "model",
	}
	require.NoError(t, db.Create(&user).Error)

	req := SyncUserRequest{
		MworkUserID: user.MworkUserID,
		Email:       "new@example.com",
		Role:        "agency",
	}

	resp := performRequest(router, http.MethodPost, "/internal/mwork/users/sync", req, "test-token")
	require.Equal(t, http.StatusOK, resp.Code)
	requireJSONContentType(t, resp)

	var updated domain.User
	require.NoError(t, db.First(&updated, user.ID).Error)
	require.Equal(t, "new@example.com", updated.Email)
	require.Equal(t, "agency", updated.MworkRole)
}

func TestSyncUserLinkByEmail(t *testing.T) {
	t.Setenv("MWORK_SYNC_TOKEN", "test-token")
	router, db := setupRouter(t)

	user := domain.User{
		Email:        "Case@Example.com",
		PasswordHash: "hash",
		Role:         domain.RoleClient,
		Name:         "Case User",
	}
	require.NoError(t, db.Create(&user).Error)

	req := SyncUserRequest{
		MworkUserID: uuid.NewString(),
		Email:       "case@example.com",
		Role:        "employer",
	}

	resp := performRequest(router, http.MethodPost, "/internal/mwork/users/sync", req, "test-token")
	require.Equal(t, http.StatusOK, resp.Code)
	requireJSONContentType(t, resp)

	var updated domain.User
	require.NoError(t, db.First(&updated, user.ID).Error)
	require.Equal(t, req.MworkUserID, updated.MworkUserID)
}

func TestSyncUserLinkByEmailNormalized(t *testing.T) {
	t.Setenv("MWORK_SYNC_TOKEN", "test-token")
	router, db := setupRouter(t)

	user := domain.User{
		Email:        "Case@Example.com",
		PasswordHash: "hash",
		Role:         domain.RoleClient,
		Name:         "Case User",
	}
	require.NoError(t, db.Create(&user).Error)

	req := SyncUserRequest{
		MworkUserID: uuid.NewString(),
		Email:       "  CASE@example.com ",
		Role:        "model",
	}

	resp := performRequest(router, http.MethodPost, "/internal/mwork/users/sync", req, "test-token")
	require.Equal(t, http.StatusOK, resp.Code)
	requireJSONContentType(t, resp)

	var updated domain.User
	require.NoError(t, db.First(&updated, user.ID).Error)
	require.Equal(t, req.MworkUserID, updated.MworkUserID)
	require.Equal(t, "case@example.com", updated.Email)
}

func TestSyncUserAuth(t *testing.T) {
	t.Setenv("MWORK_SYNC_TOKEN", "test-token")
	router, _ := setupRouter(t)

	req := SyncUserRequest{
		MworkUserID: uuid.NewString(),
		Email:       "user@example.com",
		Role:        "model",
	}

	resp := performRequest(router, http.MethodPost, "/internal/mwork/users/sync", req, "")
	require.Equal(t, http.StatusUnauthorized, resp.Code)
	assertErrorCode(t, resp, "AUTH_MISSING")
	requireJSONContentType(t, resp)

	resp = performRequest(router, http.MethodPost, "/internal/mwork/users/sync", req, "wrong-token")
	require.Equal(t, http.StatusForbidden, resp.Code)
	assertErrorCode(t, resp, "AUTH_INVALID")
	requireJSONContentType(t, resp)
}

func TestSyncUserDisabled(t *testing.T) {
	t.Setenv("MWORK_SYNC_TOKEN", "test-token")
	t.Setenv("MWORK_SYNC_ENABLED", "false")
	router, _ := setupRouter(t)

	req := SyncUserRequest{
		MworkUserID: uuid.NewString(),
		Email:       "user@example.com",
		Role:        "model",
	}

	resp := performRequest(router, http.MethodPost, "/internal/mwork/users/sync", req, "test-token")
	require.Equal(t, http.StatusForbidden, resp.Code)
	assertErrorCode(t, resp, "AUTH_INVALID")
	requireJSONContentType(t, resp)
}

func TestSyncUserAllowedIP(t *testing.T) {
	t.Setenv("MWORK_SYNC_TOKEN", "test-token")
	t.Setenv("MWORK_SYNC_ALLOWED_IPS", "203.0.113.10")
	router, _ := setupRouter(t)

	req := SyncUserRequest{
		MworkUserID: uuid.NewString(),
		Email:       "user@example.com",
		Role:        "model",
	}

	resp := performRequestWithHeaders(router, http.MethodPost, "/internal/mwork/users/sync", req, "test-token", map[string]string{
		"X-Forwarded-For": "203.0.113.10",
	})
	require.Equal(t, http.StatusCreated, resp.Code)
	requireJSONContentType(t, resp)

	resp = performRequestWithHeaders(router, http.MethodPost, "/internal/mwork/users/sync", req, "test-token", map[string]string{
		"X-Forwarded-For": "203.0.113.11",
	})
	require.Equal(t, http.StatusForbidden, resp.Code)
	assertErrorCode(t, resp, "AUTH_INVALID")
	requireJSONContentType(t, resp)
}

func TestSyncUserValidation(t *testing.T) {
	t.Setenv("MWORK_SYNC_TOKEN", "test-token")
	router, _ := setupRouter(t)

	tests := []SyncUserRequest{
		{MworkUserID: "not-a-uuid", Email: "user@example.com", Role: "model"},
		{MworkUserID: uuid.NewString(), Email: "invalid-email", Role: "model"},
		{MworkUserID: uuid.NewString(), Email: "user@example.com", Role: "invalid"},
	}

	for _, req := range tests {
		resp := performRequest(router, http.MethodPost, "/internal/mwork/users/sync", req, "test-token")
		require.Equal(t, http.StatusBadRequest, resp.Code)
		assertErrorCode(t, resp, "VALIDATION_ERROR")
		requireJSONContentType(t, resp)
	}
}

func TestSyncUserConflict(t *testing.T) {
	t.Setenv("MWORK_SYNC_TOKEN", "test-token")
	router, db := setupRouter(t)

	user := domain.User{
		Email:        "conflict@example.com",
		PasswordHash: "hash",
		Role:         domain.RoleClient,
		Name:         "Conflict User",
		MworkUserID:  uuid.NewString(),
		MworkRole:    "model",
	}
	require.NoError(t, db.Create(&user).Error)

	req := SyncUserRequest{
		MworkUserID: uuid.NewString(),
		Email:       "conflict@example.com",
		Role:        "agency",
	}

	resp := performRequest(router, http.MethodPost, "/internal/mwork/users/sync", req, "test-token")
	require.Equal(t, http.StatusConflict, resp.Code)
	assertErrorCode(t, resp, "CONFLICT")
	requireJSONContentType(t, resp)
}

func assertErrorCode(t *testing.T, resp *httptest.ResponseRecorder, code string) {
	t.Helper()
	var payload errorResponse
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&payload))
	require.Equal(t, code, payload.Error.Code)
	require.NotEmpty(t, payload.Error.Message)
}

func requireJSONContentType(t *testing.T, resp *httptest.ResponseRecorder) {
	t.Helper()
	contentType := resp.Header().Get("Content-Type")
	require.True(t, strings.Contains(contentType, "application/json"))
}
