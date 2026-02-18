package chat

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	writeWait  = 10 * time.Second
	pongWait   = 60 * time.Second
	pingPeriod = (pongWait * 9) / 10
	maxMsgSize = 512 * 1024 // 512 KB
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true }, // Allow all origins (configure in prod)
}

// WSEvent is a real-time event pushed to clients
type WSEvent struct {
	Type    string      `json:"type"`
	RoomID  string      `json:"room_id"`
	Payload interface{} `json:"payload,omitempty"`
}

const (
	EventNewMessage = "new_message"
	EventTyping     = "typing"
	EventRead       = "read"
)

// connection represents a single WebSocket client
type connection struct {
	userID int64
	conn   *websocket.Conn
	send   chan []byte
	rooms  map[string]bool // subscribed room IDs
}

// Hub manages all active WebSocket connections
type Hub struct {
	mu          sync.RWMutex
	connections map[int64]*connection // userID -> connection
}

func NewHub() *Hub {
	return &Hub{
		connections: make(map[int64]*connection),
	}
}

func (h *Hub) register(c *connection) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.connections[c.userID] = c
}

func (h *Hub) unregister(c *connection) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if existing, ok := h.connections[c.userID]; ok && existing == c {
		delete(h.connections, c.userID)
		close(c.send)
	}
}

// BroadcastToRoom sends an event to all members of a room who are connected
func (h *Hub) BroadcastToRoom(roomID string, event *WSEvent) {
	data, err := json.Marshal(event)
	if err != nil {
		return
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, c := range h.connections {
		if c.rooms[roomID] {
			select {
			case c.send <- data:
			default:
				// Client too slow — skip
			}
		}
	}
}

// ServeWS registers a new connection and starts read/write loops
func (h *Hub) ServeWS(conn *websocket.Conn, userID int64, initialRooms []string) {
	c := &connection{
		userID: userID,
		conn:   conn,
		send:   make(chan []byte, 256),
		rooms:  make(map[string]bool),
	}

	// Auto-subscribe to existing rooms
	for _, rid := range initialRooms {
		c.rooms[rid] = true
	}

	h.register(c)

	go h.writePump(c)
	h.readPump(c) // blocks until disconnect
}

func (h *Hub) readPump(c *connection) {
	defer func() {
		h.unregister(c)
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMsgSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, msg, err := c.conn.ReadMessage()
		if err != nil {
			break
		}

		var event struct {
			Type   string `json:"type"`
			RoomID string `json:"room_id"`
		}
		if err := json.Unmarshal(msg, &event); err != nil {
			continue
		}

		switch event.Type {
		case "subscribe":
			// Client subscribes to a room to receive events
			h.mu.Lock()
			c.rooms[event.RoomID] = true
			h.mu.Unlock()
		case "unsubscribe":
			h.mu.Lock()
			delete(c.rooms, event.RoomID)
			h.mu.Unlock()
		case "typing":
			h.BroadcastToRoom(event.RoomID, &WSEvent{
				Type:    EventTyping,
				RoomID:  event.RoomID,
				Payload: map[string]int64{"user_id": c.userID},
			})
		}
	}
}

func (h *Hub) writePump(c *connection) {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case msg, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
