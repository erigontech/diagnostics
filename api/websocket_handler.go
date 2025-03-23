package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	ActionSubscribe   = "subscribe"
	ActionUnsubscribe = "unsubscribe"
)

// SubscriptionResponse is the response sent back to the client after an action is processed.
type ClientResponse struct {
	Status  string  `json:"status"`
	Message string  `json:"message,omitempty"`
	Data    *string `json:"data,omitempty"`
}

type WebsocketHandler struct {
	mu         sync.Mutex
	writeQueue chan []byte
	conn       *websocket.Conn
	closeChan  chan struct{}
	closed     bool
}

// **NewWebsocketHandler initializes WebsocketHandler**
func NewWebsocketHandler(conn *websocket.Conn) *WebsocketHandler {
	handler := &WebsocketHandler{
		writeQueue: make(chan []byte, 100),
		conn:       conn,
		closeChan:  make(chan struct{}),
		closed:     false,
	}

	go handler.startWriter() // Start dedicated writer goroutine
	return handler
}

// **Sends response safely**
func (h *WebsocketHandler) sendResponse(response *ClientResponse) {
	resp, err := json.Marshal(response)
	if err != nil {
		fmt.Printf("Error marshaling response: %v\n", err)
		return
	}

	select {
	case h.writeQueue <- resp:
	default:
		fmt.Println("Warning: writeQueue is full, dropping message")
	}
}

// **Dedicated writer goroutine**
func (h *WebsocketHandler) startWriter() {
	for {
		select {
		case msg := <-h.writeQueue:
			h.mu.Lock()
			err := h.conn.WriteMessage(websocket.TextMessage, msg)
			h.mu.Unlock()

			if err != nil {
				fmt.Printf("Error writing response: %v\n", err)
				return
			}

		case <-h.closeChan:
			fmt.Println("Writer goroutine stopped")
			return
		}
	}
}

// **Close WebSocket connection and stop writer**
func (h *WebsocketHandler) closeConnection() {
	h.mu.Lock()
	if h.closed {
		h.mu.Unlock()
		return
	}
	h.closed = true
	h.mu.Unlock()

	close(h.closeChan)
	h.conn.Close()
}

// **WebSocket handler function**
func (h *APIHandler) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	type wsMessage struct {
		Service string `json:"service"`
		Action  string `json:"action"`
	}

	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Println("WebSocket upgrade failed:", err)
		return
	}
	defer conn.Close()

	handler := NewWebsocketHandler(conn)
	defer handler.closeConnection()

	channel := make(chan []byte)

	// **Goroutine to forward messages from the channel to the client**
	go func() {
		for {
			select {
			case <-r.Context().Done():
				handler.closeConnection()
				return
			case <-handler.closeChan: // Graceful shutdown
				return
			case message, ok := <-channel:
				if !ok {
					return
				}
				handler.sendResponse(&ClientResponse{
					Status:  "success",
					Message: string(message),
				})
			}
		}
	}()

	// **Enable Ping/Pong Handling**
	conn.SetPongHandler(func(appData string) error {
		return nil
	})

	pingTicker := time.NewTicker(10 * time.Second)
	defer pingTicker.Stop()

	go func() {
		for {
			select {
			case <-handler.closeChan:
				return
			case <-pingTicker.C:
				handler.mu.Lock()
				if handler.closed {
					handler.mu.Unlock()
					return
				}
				err := conn.WriteMessage(websocket.PingMessage, nil)
				handler.mu.Unlock()

				if err != nil {
					fmt.Println("Ping failed, closing connection:", err)
					handler.closeConnection()
					return
				}
			}
		}
	}()

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsCloseError(err, websocket.CloseNormalClosure) {
				fmt.Println("Client closed connection")
				break
			} else if websocket.IsUnexpectedCloseError(err, websocket.CloseNormalClosure) {
				fmt.Println("Client closed connection unexpectedly")
				break
			}

			fmt.Println("Error reading message:", err)
			break
		}
		fmt.Printf("Received: %s\n", msg)

		client, err := h.findNodeClient(r)
		if err != nil {
			handler.sendResponse(&ClientResponse{
				Status:  "error",
				Message: "Client not found: " + err.Error(),
			})
			return
		}

		var inMsg wsMessage
		if err := json.Unmarshal(msg, &inMsg); err != nil {
			handler.sendResponse(&ClientResponse{
				Status:  "error",
				Message: "Invalid JSON: " + err.Error(),
			})
			continue
		}

		switch inMsg.Action {
		case ActionSubscribe:
			go client.Subscribe(r.Context(), channel, inMsg.Service)
		case ActionUnsubscribe:
			client.Unsubscribe(r.Context(), channel, inMsg.Service)
		default:
			handler.sendResponse(&ClientResponse{
				Status:  "error",
				Message: "Unknown action " + inMsg.Action,
			})
		}
	}
}
