package notifiers

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/daniacca/achemdb/internal/achem"
	"github.com/gorilla/websocket"
)

// WebSocketNotifier sends notifications via WebSocket connections
type WebSocketNotifier struct {
	id         string
	mu         sync.RWMutex
	clients    map[*websocket.Conn]bool
	upgrader   websocket.Upgrader
	broadcast  chan achem.NotificationEvent
	register   chan *websocket.Conn
	unregister chan *websocket.Conn
}

// NewWebSocketNotifier creates a new WebSocket notifier
func NewWebSocketNotifier(id string) *WebSocketNotifier {
	notifier := &WebSocketNotifier{
		id:         id,
		clients:    make(map[*websocket.Conn]bool),
		broadcast:  make(chan achem.NotificationEvent, 256),
		register:   make(chan *websocket.Conn),
		unregister: make(chan *websocket.Conn),
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		},
	}

	// Start the broadcaster goroutine
	go notifier.run()

	return notifier
}

// ID returns the notifier ID
func (wsn *WebSocketNotifier) ID() string {
	return wsn.id
}

// Type returns the notifier type
func (wsn *WebSocketNotifier) Type() string {
	return "websocket"
}

// RegisterClient registers a new WebSocket client connection
func (wsn *WebSocketNotifier) RegisterClient(conn *websocket.Conn) {
	wsn.register <- conn
}

// UnregisterClient unregisters a WebSocket client connection
func (wsn *WebSocketNotifier) UnregisterClient(conn *websocket.Conn) {
	wsn.unregister <- conn
}

// Notify sends the notification event to all connected WebSocket clients
func (wsn *WebSocketNotifier) Notify(ctx context.Context, event achem.NotificationEvent) error {
	select {
	case wsn.broadcast <- event:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(1 * time.Second):
		return fmt.Errorf("notification queue full")
	}
}

// run handles client registration/unregistration and message broadcasting
func (wsn *WebSocketNotifier) run() {
	for {
		select {
		case conn := <-wsn.register:
			wsn.mu.Lock()
			wsn.clients[conn] = true
			wsn.mu.Unlock()

		case conn := <-wsn.unregister:
			wsn.mu.Lock()
			if _, ok := wsn.clients[conn]; ok {
				delete(wsn.clients, conn)
				conn.Close()
			}
			wsn.mu.Unlock()

		case event := <-wsn.broadcast:
			jsonData, err := event.JSON()
			if err != nil {
				continue
			}

			wsn.mu.RLock()
			for conn := range wsn.clients {
				conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
				if err := conn.WriteMessage(websocket.TextMessage, jsonData); err != nil {
					// Remove failed connection
					delete(wsn.clients, conn)
					conn.Close()
				}
			}
			wsn.mu.RUnlock()
		}
	}
}

// Close closes all WebSocket connections
func (wsn *WebSocketNotifier) Close() error {
	wsn.mu.Lock()
	defer wsn.mu.Unlock()

	for conn := range wsn.clients {
		conn.Close()
		delete(wsn.clients, conn)
	}

	close(wsn.broadcast)
	close(wsn.register)
	close(wsn.unregister)

	return nil
}

// GetUpgrader returns the WebSocket upgrader for HTTP handlers
func (wsn *WebSocketNotifier) GetUpgrader() websocket.Upgrader {
	return wsn.upgrader
}

