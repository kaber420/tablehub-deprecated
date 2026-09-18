package tunnel

// TunnelMessage is the canonical message structure for tunnel communication.
// It belongs to the application layer to avoid coupling with the WebSocket
// infrastructure (ws.TunnelMessage). The interfaces layer is responsible for
// converting between wire formats and this type.
type TunnelMessage struct {
	Event     string      `json:"event"`
	Timestamp int64       `json:"timestamp"`
	Payload   interface{} `json:"payload,omitempty"`
}
