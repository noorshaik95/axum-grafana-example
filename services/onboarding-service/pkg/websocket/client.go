package websocket

import (
	"time"

	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"
)

const (
	// Time allowed to write a message to the peer
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer
	pongWait = 60 * time.Second

	// pingPeriodNumerator is the multiplier for calculating ping period
	pingPeriodNumerator = 9
	// pingPeriodDenominator is the divisor for calculating ping period
	pingPeriodDenominator = 10

	// Send pings to peer with this period. Must be less than pongWait
	pingPeriod = (pongWait * pingPeriodNumerator) / pingPeriodDenominator

	// Maximum message size allowed from peer
	maxMessageSize = 512

	// defaultSendBufferSize is the default buffer size for client send channel
	defaultSendBufferSize = 256
)

// ReadPump pumps messages from the WebSocket connection to the hub
func (c *Client) ReadPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	if err := c.conn.SetReadDeadline(time.Now().Add(pongWait)); err != nil {
		log.Error().Err(err).Msg("Failed to set read deadline")
		return
	}
	c.conn.SetPongHandler(func(string) error {
		if err := c.conn.SetReadDeadline(time.Now().Add(pongWait)); err != nil {
			log.Error().Err(err).Msg("Failed to set read deadline in pong handler")
		}
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Error().
					Err(err).
					Str("job_id", c.jobID).
					Msg("WebSocket unexpected close error")
			}
			break
		}

		log.Debug().
			Str("job_id", c.jobID).
			Str("message", string(message)).
			Msg("Received message from WebSocket client")
	}
}

// WritePump pumps messages from the hub to the WebSocket connection
func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			if err := c.conn.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
				log.Error().Err(err).Msg("Failed to set write deadline")
				return
			}
			if !ok {
				// Hub closed the channel
				if err := c.conn.WriteMessage(websocket.CloseMessage, []byte{}); err != nil {
					log.Error().Err(err).Msg("Failed to write close message")
				}
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			if _, err := w.Write(message); err != nil {
				log.Error().Err(err).Msg("Failed to write message")
				return
			}

			// Add queued messages to the current WebSocket message
			n := len(c.send)
			for i := 0; i < n; i++ {
				if _, err := w.Write([]byte{'\n'}); err != nil {
					log.Error().Err(err).Msg("Failed to write newline")
					return
				}
				if _, err := w.Write(<-c.send); err != nil {
					log.Error().Err(err).Msg("Failed to write queued message")
					return
				}
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			if err := c.conn.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
				log.Error().Err(err).Msg("Failed to set write deadline for ping")
				return
			}
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// NewClient creates a new WebSocket client
func NewClient(hub *Hub, conn *websocket.Conn, jobID, userID string) *Client {
	return &Client{
		hub:    hub,
		conn:   conn,
		send:   make(chan []byte, defaultSendBufferSize),
		jobID:  jobID,
		userID: userID,
	}
}

// Register registers the client with the hub
func (c *Client) Register() {
	c.hub.register <- c
}
