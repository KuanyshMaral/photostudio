package chat

import (
	"sync"

	"github.com/gorilla/websocket"
)

type Hub struct {
	connections map[int64]*websocket.Conn
	mutex       sync.RWMutex
}

func NewHub() *Hub {
	return &Hub{
		connections: make(map[int64]*websocket.Conn),
	}
}

func (h *Hub) Register(userID int64, conn *websocket.Conn) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	if oldConn, exists := h.connections[userID]; exists && oldConn != nil {
		_ = oldConn.Close()
	}

	h.connections[userID] = conn
}

func (h *Hub) Unregister(userID int64) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	if conn, exists := h.connections[userID]; exists && conn != nil {
		_ = conn.Close()
		delete(h.connections, userID)
	}
}

func (h *Hub) SendToUser(userID int64, message interface{}) bool {
	h.mutex.RLock()
	conn, exists := h.connections[userID]
	h.mutex.RUnlock()

	if !exists || conn == nil {
		return false
	}

	if err := conn.WriteJSON(message); err != nil {
		h.Unregister(userID)
		return false
	}

	return true
}

func (h *Hub) IsOnline(userID int64) bool {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	_, exists := h.connections[userID]
	return exists
}

func (h *Hub) GetOnlineCount() int {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	return len(h.connections)
}

func (h *Hub) BroadcastMessage(senderID, recipientID int64, message interface{}) bool {
	_ = h.SendToUser(senderID, message)
	return h.SendToUser(recipientID, message)
}

func (h *Hub) Close() {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	for userID, conn := range h.connections {
		if conn != nil {
			_ = conn.Close()
		}
		delete(h.connections, userID)
	}
}
