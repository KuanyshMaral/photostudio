package notification

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"photostudio/internal/domain/auth"
	"photostudio/internal/domain/chat"
)

type realtimePublisherMock struct {
	calls  int
	userID int64
	event  *chat.WSEvent
}

func (m *realtimePublisherMock) PublishToUser(userID int64, event *chat.WSEvent) {
	m.calls++
	m.userID = userID
	m.event = event
}

func TestServiceCreate_PublishesRealtimeEvent(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&auth.User{}, &Notification{}, &UserPreferences{}, &DeviceToken{}))

	u := auth.User{ID: 321, Email: "ws@test.local", PasswordHash: "x", Role: auth.RoleClient, Name: "WS"}
	require.NoError(t, db.Create(&u).Error)

	repo := NewRepository(db)
	prefRepo := NewPreferencesRepository(db)
	deviceRepo := NewDeviceTokenRepository(db)
	svc := NewService(repo, prefRepo, deviceRepo)

	pub := &realtimePublisherMock{}
	svc.SetRealtimePublisher(pub)

	n, err := svc.Create(context.Background(), 321, TypeBookingCreated, "title", "body", nil)
	require.NoError(t, err)
	require.NotNil(t, n)

	require.Equal(t, 1, pub.calls)
	require.Equal(t, int64(321), pub.userID)
	require.NotNil(t, pub.event)
	require.Equal(t, EventNotificationCreated, pub.event.Type)

	payload, ok := pub.event.Payload.(*RealtimeNotificationPayload)
	require.True(t, ok)
	require.Equal(t, n.ID, payload.ID)
	require.Equal(t, "title", payload.Title)
	require.False(t, payload.IsRead)
}
