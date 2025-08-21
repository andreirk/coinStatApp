package websocket

import (
	"coinStatApp/internal/domain/model"
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

// WebSocketBroadcaster implements Broadcaster interface for stats updates.
type WebSocketBroadcaster struct {
	clients  map[*websocket.Conn]struct{}
	mu       sync.Mutex
	upgrader websocket.Upgrader
}

func NewWebSocketBroadcaster() *WebSocketBroadcaster {
	return &WebSocketBroadcaster{
		clients:  make(map[*websocket.Conn]struct{}),
		upgrader: websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }},
	}
}

func (b *WebSocketBroadcaster) BroadcastStatistics(stats *model.Statistics) {
	b.mu.Lock()
	defer b.mu.Unlock()
	msg, err := json.Marshal(stats)
	if err != nil {
		log.Printf("failed to marshal stats: %v", err)
		return
	}
	for c := range b.clients {
		if err := c.WriteMessage(websocket.TextMessage, msg); err != nil {
			log.Printf("websocket write error: %v", err)
			c.Close()
			delete(b.clients, c)
		}
	}
}

// Handler returns an http.HandlerFunc to accept websocket connections.
func (b *WebSocketBroadcaster) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		conn, err := b.upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("websocket upgrade error: %v", err)
			return
		}
		b.mu.Lock()
		b.clients[conn] = struct{}{}
		b.mu.Unlock()
		// Optionally: read loop to keep connection alive
		go func() {
			defer func() {
				b.mu.Lock()
				delete(b.clients, conn)
				b.mu.Unlock()
				conn.Close()
			}()
			for {
				if _, _, err := conn.ReadMessage(); err != nil {
					break
				}
			}
		}()
	}
}
