package websocket

import (
	"encoding/json"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"
)

const (
	// defaultBroadcastBufferSize is the default buffer size for the broadcast channel
	defaultBroadcastBufferSize = 256
)

// Hub manages WebSocket connections and broadcasts
type Hub struct {
	// Registered clients per job ID
	clients map[string]map[*Client]bool

	// Register requests from clients
	register chan *Client

	// Unregister requests from clients
	unregister chan *Client

	// Broadcast messages to clients
	broadcast chan *Message

	// Mutex for thread-safe operations
	mu sync.RWMutex
}

// Client represents a WebSocket client connection
type Client struct {
	hub    *Hub
	conn   *websocket.Conn
	send   chan []byte
	jobID  string
	userID string
}

// Message represents a message to be broadcast
type Message struct {
	JobID   string      `json:"job_id"`
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

// NewHub creates a new WebSocket hub
func NewHub() *Hub {
	return &Hub{
		clients:    make(map[string]map[*Client]bool),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan *Message, defaultBroadcastBufferSize),
	}
}

// Run starts the hub's main loop
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			if _, ok := h.clients[client.jobID]; !ok {
				h.clients[client.jobID] = make(map[*Client]bool)
			}
			h.clients[client.jobID][client] = true
			h.mu.Unlock()
			log.Info().
				Str("job_id", client.jobID).
				Str("user_id", client.userID).
				Msg("Client connected to WebSocket")

		case client := <-h.unregister:
			h.mu.Lock()
			if clients, ok := h.clients[client.jobID]; ok {
				if _, ok := clients[client]; ok {
					delete(clients, client)
					close(client.send)
					if len(clients) == 0 {
						delete(h.clients, client.jobID)
					}
				}
			}
			h.mu.Unlock()
			log.Info().
				Str("job_id", client.jobID).
				Str("user_id", client.userID).
				Msg("Client disconnected from WebSocket")

		case message := <-h.broadcast:
			h.mu.RLock()
			clients := h.clients[message.JobID]
			h.mu.RUnlock()

			// Serialize message
			data, err := json.Marshal(message)
			if err != nil {
				log.Error().
					Err(err).
					Str("job_id", message.JobID).
					Msg("Failed to marshal WebSocket message")
				continue
			}

			// Broadcast to all clients for this job
			for client := range clients {
				select {
				case client.send <- data:
				default:
					// Client buffer is full, close connection
					h.mu.Lock()
					close(client.send)
					delete(clients, client)
					h.mu.Unlock()
				}
			}
		}
	}
}

// BroadcastProgress sends a progress update to all clients watching a job
func (h *Hub) BroadcastProgress(jobID string, payload interface{}) {
	h.broadcast <- &Message{
		JobID:   jobID,
		Type:    "progress",
		Payload: payload,
	}
}

// BroadcastCompletion sends a completion message to all clients watching a job
func (h *Hub) BroadcastCompletion(jobID string, payload interface{}) {
	h.broadcast <- &Message{
		JobID:   jobID,
		Type:    "completion",
		Payload: payload,
	}
}

// BroadcastError sends an error message to all clients watching a job
func (h *Hub) BroadcastError(jobID string, payload interface{}) {
	h.broadcast <- &Message{
		JobID:   jobID,
		Type:    "error",
		Payload: payload,
	}
}

// GetClientCount returns the number of connected clients for a job
func (h *Hub) GetClientCount(jobID string) int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients[jobID])
}
