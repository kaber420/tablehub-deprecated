package ws

import (
	"context"
	"encoding/json"
	"math/rand"
	"time"

	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"

	"github.com/tablehub/cloud/internal/application/tunnel"
)

// HubConn represents a WebSocket connection to a single hub device.
// It implements the tunnel.Sender interface from the application layer.
type HubConn struct {
	conn           *websocket.Conn
	hubID          string
	OrganizationID string
	send           chan []byte
	cancelCtx      context.CancelFunc
	config         WSConfig
	msgHandler     tunnel.MessageHandler
}

// NewHubConn creates a new HubConn. The msgHandler parameter injects the
// application-layer message handler to process incoming messages without
// coupling the WS layer to business logic.
func NewHubConn(conn *websocket.Conn, hubID string, organizationID string, config WSConfig, msgHandler tunnel.MessageHandler) *HubConn {
	return &HubConn{
		conn:           conn,
		hubID:          hubID,
		OrganizationID: organizationID,
		send:           make(chan []byte, config.SendBufferSize),
		config:         config,
		msgHandler:     msgHandler,
	}
}

// Start launches the read, write, and heartbeat pumps.
// It returns a channel that is closed when the readPump exits,
// signaling the supervisor goroutine in the handler to orchestrate cleanup.
func (c *HubConn) Start(ctx context.Context) <-chan struct{} {
	childCtx, cancel := context.WithCancel(ctx)
	c.cancelCtx = cancel

	done := make(chan struct{})

	go func() {
		c.readPump()
		close(done)
	}()
	go c.writePump(childCtx)
	go c.heartbeat(childCtx)

	return done
}

// Close cancels the context and closes the underlying WebSocket connection.
func (c *HubConn) Close() {
	if c.cancelCtx != nil {
		c.cancelCtx()
	}
	c.conn.Close()
}

// HubID returns the ID of the connected hub.
func (c *HubConn) HubID() string {
	return c.hubID
}

// readPump reads messages from the WebSocket. On exit, it only closes the
// socket via defer c.Close(). It does NOT touch the registry — the supervisor
// goroutine in the handler is the single owner of disconnect orchestration,
// preventing race conditions (DeepSeek fix).
func (c *HubConn) readPump() {
	defer func() {
		c.Close()
	}()

	c.conn.SetReadLimit(c.config.MaxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(c.config.PongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(c.config.PongWait))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Error().Err(err).Str("hub_id", c.hubID).Msg("unexpected close error")
			}
			break
		}

		var tm tunnel.TunnelMessage
		if err := json.Unmarshal(message, &tm); err != nil {
			log.Warn().Err(err).Str("hub_id", c.hubID).Msg("failed to unmarshal tunnel message")
			continue
		}

		log.Debug().Str("hub_id", c.hubID).Str("event", tm.Event).Msg("received message")

		// Dispatch to the application layer via the injected MessageHandler.
		if c.msgHandler != nil {
			c.msgHandler.HandleMessage(c.hubID, tm)
		}
	}
}

func (c *HubConn) writePump(ctx context.Context) {
	pingTicker := time.NewTicker(c.config.PingPeriod)
	defer pingTicker.Stop()
	defer func() {
		c.Close()
	}()

	for {
		select {
		case <-ctx.Done():
			c.conn.WriteMessage(websocket.CloseMessage, []byte{})
			return
		case <-pingTicker.C:
			c.conn.SetWriteDeadline(time.Now().Add(c.config.WriteWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Error().Err(err).Str("hub_id", c.hubID).Msg("failed to send ping")
				return
			}
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(c.config.WriteWait))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}
		}
	}
}

func (c *HubConn) heartbeat(ctx context.Context) {
	for {
		jitter := time.Duration((rand.Float64()*0.3 - 0.15) * float64(c.config.PingPeriod))
		timer := time.NewTimer(c.config.PingPeriod + jitter)

		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
			pingMsg := tunnel.TunnelMessage{
				Event:     "ping",
				Timestamp: time.Now().UnixMilli(),
			}
			if err := c.Send(pingMsg); err != nil {
				log.Error().Err(err).Str("hub_id", c.hubID).Msg("failed to queue ping")
				c.cancelCtx()
				return
			}
		}
	}
}

// Send implements the tunnel.Sender interface.
// It serializes a TunnelMessage and queues it for the writePump.
func (c *HubConn) Send(msg tunnel.TunnelMessage) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	select {
	case c.send <- data:
		return nil
	default:
		return context.DeadlineExceeded
	}
}
