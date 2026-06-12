package ws

import (
	"encoding/json"
	"go.uber.org/zap"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type WebSocketManager struct {
	mu          sync.RWMutex
	connections map[string]map[*websocket.Conn]bool
}

var Manager = &WebSocketManager{
	connections: make(map[string]map[*websocket.Conn]bool),
}

// Connect adds a connection to a specific channel
func (m *WebSocketManager) Connect(channel string, conn *websocket.Conn) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.connections[channel] == nil {
		m.connections[channel] = make(map[*websocket.Conn]bool)
	}
	m.connections[channel][conn] = true
	zap.L().Sugar().Infof("WS Connected to channel %s. Total: %d", channel, len(m.connections[channel]))
}

// Disconnect removes a connection from a specific channel
func (m *WebSocketManager) Disconnect(channel string, conn *websocket.Conn) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.connections[channel] != nil {
		delete(m.connections[channel], conn)
		if len(m.connections[channel]) == 0 {
			delete(m.connections, channel)
		}
	}
	conn.Close()
	zap.L().Sugar().Infof("WS Disconnected from channel %s", channel)
}

// Broadcast sends a message to all clients subscribed to a specific channel
func (m *WebSocketManager) Broadcast(channel string, data interface{}) {
	m.mu.RLock()
	conns := make([]*websocket.Conn, 0, len(m.connections[channel]))
	for conn := range m.connections[channel] {
		conns = append(conns, conn)
	}
	m.mu.RUnlock()

	if len(conns) == 0 {
		return
	}

	payload, err := json.Marshal(data)
	if err != nil {
		zap.L().Sugar().Infof("WS failed to marshal broadcast message: %v", err)
		return
	}

	for _, conn := range conns {
		err := conn.WriteMessage(websocket.TextMessage, payload)
		if err != nil {
			zap.L().Sugar().Infof("WS write failed to channel %s, removing connection: %v", channel, err)
			m.Disconnect(channel, conn)
		}
	}
}

// GetConnectionCount returns number of active clients on a channel
func (m *WebSocketManager) GetConnectionCount(channel string) int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.connections[channel])
}

// StartHeartbeat sends periodic ping messages to all clients
func (m *WebSocketManager) StartHeartbeat() {
	ticker := time.NewTicker(30 * time.Second)
	go func() {
		for range ticker.C {
			pingMsg := map[string]interface{}{"type": "ping"}
			m.mu.RLock()
			var channels []string
			for ch := range m.connections {
				channels = append(channels, ch)
			}
			m.mu.RUnlock()

			for _, ch := range channels {
				m.Broadcast(ch, pingMsg)
			}
		}
	}()
}
