package notification

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"photostudio/internal/domain/auth"
	"photostudio/internal/pkg/chicontext"
)

func setupNotificationTest(t *testing.T) (*gorm.DB, *chi.Mux) {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&auth.User{}, &Notification{}, &UserPreferences{}, &DeviceToken{}))

	users := []auth.User{
		{ID: 101, Email: "a@test.local", PasswordHash: "x", Role: auth.RoleClient, Name: "A"},
		{ID: 202, Email: "b@test.local", PasswordHash: "x", Role: auth.RoleClient, Name: "B"},
	}
	for _, u := range users {
		require.NoError(t, db.Create(&u).Error)
	}

	repo := NewRepository(db)
	prefRepo := NewPreferencesRepository(db)
	deviceRepo := NewDeviceTokenRepository(db)
	service := NewService(repo, prefRepo, deviceRepo)

	nh := NewHandler(service)
	ph := NewPreferencesHandler(service)
	dh := NewDeviceTokensHandler(service)

	r := chi.NewRouter()
	r.Route("/api/v1", func(protected chi.Router) {
		protected.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				uid, _ := strconv.ParseInt(req.Header.Get("X-User-ID"), 10, 64)
				next.ServeHTTP(w, req.WithContext(chicontext.SetUserID(req.Context(), uid)))
			})
		})
		RegisterRoutes(protected, nh, ph, dh)
	})

	return db, r
}

func TestNotifications_Ownership404s(t *testing.T) {
	db, router := setupNotificationTest(t)

	n := Notification{
		UserID:    101,
		Type:      TypeBookingCreated,
		Title:     "test",
		Body:      sql.NullString{String: "body", Valid: true},
		CreatedAt: time.Now(),
	}
	require.NoError(t, db.Create(&n).Error)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/notifications/"+strconv.FormatInt(n.ID, 10)+"/read", nil)
	req.Header.Set("X-User-ID", "202")
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusNotFound, w.Code)

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodDelete, "/api/v1/notifications/"+strconv.FormatInt(n.ID, 10), nil)
	req.Header.Set("X-User-ID", "202")
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestDeviceTokens_Ownership404Deactivate(t *testing.T) {
	db, router := setupNotificationTest(t)

	token := DeviceToken{
		UserID:     101,
		Token:      "tok-a",
		Platform:   "web",
		DeviceName: "A",
		IsActive:   true,
		CreatedAt:  time.Now(),
		LastUsedAt: time.Now(),
	}
	require.NoError(t, db.Create(&token).Error)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/notifications/device-tokens/"+strconv.FormatInt(token.ID, 10), nil)
	req.Header.Set("X-User-ID", "202")
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestNotifications_MarkReadHappyPath(t *testing.T) {
	db, router := setupNotificationTest(t)

	n := Notification{
		UserID:    101,
		Type:      TypeBookingCreated,
		Title:     "test",
		Body:      sql.NullString{String: "body", Valid: true},
		CreatedAt: time.Now(),
	}
	require.NoError(t, db.Create(&n).Error)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/notifications/"+strconv.FormatInt(n.ID, 10)+"/read", nil)
	req.Header.Set("X-User-ID", "101")
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	var updated Notification
	require.NoError(t, db.First(&updated, n.ID).Error)
	require.True(t, updated.IsRead)
	require.True(t, updated.ReadAt.Valid)

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/notifications/unread-count", nil)
	req.Header.Set("X-User-ID", "101")
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		Success bool `json:"success"`
		Data    struct {
			UnreadCount int64 `json:"unread_count"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.True(t, resp.Success)
	require.Equal(t, int64(0), resp.Data.UnreadCount)
}

func TestPreferencesPatch_PartialUpdate(t *testing.T) {
	db, router := setupNotificationTest(t)

	prefs := UserPreferences{
		UserID:          101,
		EmailEnabled:    true,
		PushEnabled:     true,
		InAppEnabled:    true,
		DigestEnabled:   true,
		DigestFrequency: "weekly",
		PerTypeSettings: PerTypeSettingsMap{},
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}
	require.NoError(t, db.Create(&prefs).Error)

	body := []byte(`{"email_enabled": false}`)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/notifications/preferences", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", "101")
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	var updated UserPreferences
	require.NoError(t, db.Where("user_id = ?", 101).First(&updated).Error)
	require.False(t, updated.EmailEnabled)
	require.True(t, updated.PushEnabled)
	require.True(t, updated.InAppEnabled)
}

func TestPreferencesPatch_EmptyPayloadReturns400(t *testing.T) {
	_, router := setupNotificationTest(t)

	t.Run("empty_json_object", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPatch, "/api/v1/notifications/preferences", bytes.NewReader([]byte(`{}`)))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-User-ID", "101")
		router.ServeHTTP(w, req)
		require.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("empty_body", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPatch, "/api/v1/notifications/preferences", bytes.NewReader(nil))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-User-ID", "101")
		router.ServeHTTP(w, req)
		require.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestNotificationsList_OnlyUnreadFilter(t *testing.T) {
	db, router := setupNotificationTest(t)

	n1 := Notification{UserID: 101, Type: TypeBookingCreated, Title: "unread-1", CreatedAt: time.Now().Add(-3 * time.Minute)}
	n2 := Notification{UserID: 101, Type: TypeBookingCreated, Title: "unread-2", CreatedAt: time.Now().Add(-2 * time.Minute)}
	n3 := Notification{UserID: 101, Type: TypeBookingCreated, Title: "read-1", IsRead: true, ReadAt: sql.NullTime{Time: time.Now(), Valid: true}, CreatedAt: time.Now().Add(-1 * time.Minute)}
	require.NoError(t, db.Create(&n1).Error)
	require.NoError(t, db.Create(&n2).Error)
	require.NoError(t, db.Create(&n3).Error)

	t.Run("only_unread_true", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/notifications?only_unread=true", nil)
		req.Header.Set("X-User-ID", "101")
		router.ServeHTTP(w, req)
		require.Equal(t, http.StatusOK, w.Code)

		var resp struct {
			Success bool `json:"success"`
			Data    struct {
				Notifications []NotificationResponse `json:"notifications"`
				UnreadCount   int64                  `json:"unread_count"`
				Total         int64                  `json:"total"`
			} `json:"data"`
		}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		require.True(t, resp.Success)
		require.Len(t, resp.Data.Notifications, 2)
		require.Equal(t, int64(2), resp.Data.Total)
		require.Equal(t, int64(2), resp.Data.UnreadCount)
		for _, n := range resp.Data.Notifications {
			require.False(t, n.IsRead)
		}
	})

	t.Run("no_filter", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/notifications", nil)
		req.Header.Set("X-User-ID", "101")
		router.ServeHTTP(w, req)
		require.Equal(t, http.StatusOK, w.Code)

		var resp struct {
			Success bool `json:"success"`
			Data    struct {
				Notifications []NotificationResponse `json:"notifications"`
				UnreadCount   int64                  `json:"unread_count"`
				Total         int64                  `json:"total"`
			} `json:"data"`
		}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		require.True(t, resp.Success)
		require.Len(t, resp.Data.Notifications, 3)
		require.Equal(t, int64(3), resp.Data.Total)
		require.Equal(t, int64(2), resp.Data.UnreadCount)
	})

	t.Run("invalid_only_unread", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/notifications?only_unread=abc", nil)
		req.Header.Set("X-User-ID", "101")
		router.ServeHTTP(w, req)
		require.Equal(t, http.StatusBadRequest, w.Code)
	})
}
