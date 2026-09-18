package tunnel

// Sender abstracts the ability to send a message to a connected hub.
// The WebSocket HubConn implements this interface, but the application
// layer never depends on the concrete ws.HubConn type.
type Sender interface {
	Send(msg TunnelMessage) error
	HubID() string
}

// ConnectionRegistry manages the in-memory registry of active hub connections.
// It maps hub IDs and organization IDs to their respective Sender implementations.
type ConnectionRegistry interface {
	// Store registers a Sender for a given hub and organization pair.
	// If a connection already exists for the hubID, the old one is evicted.
	Store(hubID string, organizationID string, sender Sender)

	// Get returns the Sender for a given hubID, or nil if not found.
	Get(hubID string) Sender

	// GetByOrganizationID returns the Sender for the hub associated with a organization.
	GetByOrganizationID(organizationID string) (Sender, error)

	// Delete removes the connection for a given hubID from the registry.
	Delete(hubID string)

	// Shutdown closes all connections and clears the registry.
	// It returns a list of hubIDs that were connected at the time of shutdown.
	Shutdown() []string
}
