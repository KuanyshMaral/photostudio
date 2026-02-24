package chat

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHubPublishToUser_MultiConnectionDelivery(t *testing.T) {
	h := NewHub()

	c1 := &connection{userID: 7, send: make(chan []byte, 1), rooms: map[string]bool{}}
	c2 := &connection{userID: 7, send: make(chan []byte, 1), rooms: map[string]bool{}}
	h.register(c1)
	h.register(c2)

	e := &WSEvent{Type: "notification.created", Payload: map[string]any{"id": 55}}
	h.PublishToUser(7, e)

	select {
	case msg := <-c1.send:
		var got WSEvent
		require.NoError(t, json.Unmarshal(msg, &got))
		require.Equal(t, "notification.created", got.Type)
	default:
		t.Fatal("expected message for first connection")
	}

	select {
	case msg := <-c2.send:
		var got WSEvent
		require.NoError(t, json.Unmarshal(msg, &got))
		require.Equal(t, "notification.created", got.Type)
	default:
		t.Fatal("expected message for second connection")
	}
}

func TestHubPublishToUser_RemovesStaleConnection(t *testing.T) {
	h := NewHub()

	stale := &connection{userID: 9, send: make(chan []byte), rooms: map[string]bool{}}
	good := &connection{userID: 9, send: make(chan []byte, 1), rooms: map[string]bool{}}
	h.register(stale)
	h.register(good)

	e := &WSEvent{Type: "notification.created", Payload: map[string]any{"id": 66}}
	h.PublishToUser(9, e)

	// good connection gets the event
	select {
	case <-good.send:
	default:
		t.Fatal("expected message for healthy connection")
	}

	h.mu.RLock()
	defer h.mu.RUnlock()
	_, staleExists := h.connections[9][stale]
	_, goodExists := h.connections[9][good]
	require.False(t, staleExists)
	require.True(t, goodExists)
}
