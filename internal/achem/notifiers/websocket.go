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
	done       chan struct{}
	wg         sync.WaitGroup
}

// NewWebSocketNotifier creates a new WebSocket notifier
func NewWebSocketNotifier(id string) *WebSocketNotifier {
	notifier := &WebSocketNotifier{
		id:         id,
		clients:    make(map[*websocket.Conn]bool),
		broadcast:  make(chan achem.NotificationEvent, 256),
		register:   make(chan *websocket.Conn),
		unregister: make(chan *websocket.Conn),
		done:       make(chan struct{}),
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		},
	}

	// Start the broadcaster goroutine
	notifier.wg.Add(1)
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
	select {
	case wsn.register <- conn:
	case <-wsn.done:
		// Notifier is closing, ignore
	}
}

// UnregisterClient unregisters a WebSocket client connection
func (wsn *WebSocketNotifier) UnregisterClient(conn *websocket.Conn) {
	select {
	case wsn.unregister <- conn:
	case <-wsn.done:
		// Notifier is closing, ignore
	}
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
	defer wsn.wg.Done()
	for {
		select {
		case <-wsn.done:
			// Shutdown signal received, exit gracefully
			return

		case conn := <-wsn.register:
			if conn == nil {
				continue
			}
			wsn.mu.Lock()
			wsn.clients[conn] = true
			wsn.mu.Unlock()

		case conn := <-wsn.unregister:
			if conn == nil {
				continue
			}
			wsn.mu.Lock()
			if _, ok := wsn.clients[conn]; ok {
				delete(wsn.clients, conn)
				conn.Close()
			}
			wsn.mu.Unlock()

		case event, ok := <-wsn.broadcast:
			if !ok {
				// Channel closed, exit
				return
			}
			jsonData, err := event.JSON()
			if err != nil {
				continue
			}

			// Collect connections to write to (to avoid holding lock during write)
			wsn.mu.RLock()
			conns := make([]*websocket.Conn, 0, len(wsn.clients))
			for conn := range wsn.clients {
				conns = append(conns, conn)
			}
			wsn.mu.RUnlock()

			// Write to each connection (outside the lock to avoid blocking)
			var toRemove []*websocket.Conn
			for _, conn := range conns {
				if conn == nil {
					continue
				}
				// Use defer/recover to handle potential panics from closed connections
				func() {
					defer func() {
						if r := recover(); r != nil {
							// Connection was closed, mark for removal
							toRemove = append(toRemove, conn)
						}
					}()
					conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
					if err := conn.WriteMessage(websocket.TextMessage, jsonData); err != nil {
						// Mark for removal
						toRemove = append(toRemove, conn)
						conn.Close()
					}
				}()
			}

			// Remove failed connections
			if len(toRemove) > 0 {
				wsn.mu.Lock()
				for _, conn := range toRemove {
					delete(wsn.clients, conn)
				}
				wsn.mu.Unlock()
			}
		}
	}
}

// Close closes all WebSocket connections and stops the goroutine
func (wsn *WebSocketNotifier) Close() error {
	// Signal the goroutine to stop
	close(wsn.done)

	// Close all client connections
	wsn.mu.Lock()
	for conn := range wsn.clients {
		conn.Close()
		delete(wsn.clients, conn)
	}
	wsn.mu.Unlock()

	// Close channels (this will cause the goroutine to exit if it's waiting on them)
	close(wsn.broadcast)
	close(wsn.register)
	close(wsn.unregister)

	// Wait for the goroutine to finish
	wsn.wg.Wait()

	return nil
}

// GetUpgrader returns the WebSocket upgrader for HTTP handlers
func (wsn *WebSocketNotifier) GetUpgrader() websocket.Upgrader {
	return wsn.upgrader
}

