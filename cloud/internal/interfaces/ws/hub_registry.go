package ws

import (
	"errors"
	"hash/fnv"
	"sync"

	"github.com/tablehub/cloud/internal/application/tunnel"
)

var ErrHubNotFound = errors.New("hub not found")

// connEntry wraps a Sender with its associated organization ID.
type connEntry struct {
	sender         tunnel.Sender
	organizationID string
}

// Shard is a single shard in the sharded registry.
type Shard struct {
	sync.RWMutex
	connections map[string]*connEntry
}

// HubRegistry is a sharded, thread-safe in-memory connection registry.
// It implements the tunnel.ConnectionRegistry interface from the application layer.
type HubRegistry struct {
	shards          [256]*Shard
	organizationIdx map[string]string // organizationID -> hubID
	idxMu           sync.RWMutex
}

// NewHubRegistry creates a new sharded HubRegistry.
func NewHubRegistry() *HubRegistry {
	r := &HubRegistry{
		organizationIdx: make(map[string]string),
	}
	for i := 0; i < 256; i++ {
		r.shards[i] = &Shard{
			connections: make(map[string]*connEntry),
		}
	}
	return r
}

func (r *HubRegistry) getShard(hubID string) *Shard {
	h := fnv.New32a()
	h.Write([]byte(hubID))
	return r.shards[h.Sum32()%256]
}

// Store registers a Sender for a given hub and organization pair.
// If a connection already exists for the hubID, the old one is evicted and closed.
func (r *HubRegistry) Store(hubID string, organizationID string, sender tunnel.Sender) {
	shard := r.getShard(hubID)

	r.idxMu.Lock()
	defer r.idxMu.Unlock()

	shard.Lock()
	defer shard.Unlock()

	if old, exists := shard.connections[hubID]; exists {
		if old.organizationID != "" {
			delete(r.organizationIdx, old.organizationID)
		}
		// Close the old connection if it implements io.Closer.
		if closer, ok := old.sender.(interface{ Close() }); ok {
			closer.Close()
		}
	}
	shard.connections[hubID] = &connEntry{
		sender:         sender,
		organizationID: organizationID,
	}

	if organizationID != "" {
		r.organizationIdx[organizationID] = hubID
	}
}

// Get returns the Sender for a given hubID, or nil if not found.
func (r *HubRegistry) Get(hubID string) tunnel.Sender {
	shard := r.getShard(hubID)
	shard.RLock()
	defer shard.RUnlock()
	entry := shard.connections[hubID]
	if entry == nil {
		return nil
	}
	return entry.sender
}

// GetByOrganizationID returns the Sender for the hub associated with a organization.
func (r *HubRegistry) GetByOrganizationID(organizationID string) (tunnel.Sender, error) {
	r.idxMu.RLock()
	hubID, ok := r.organizationIdx[organizationID]
	r.idxMu.RUnlock()

	if !ok {
		return nil, ErrHubNotFound
	}

	sender := r.Get(hubID)
	if sender == nil {
		return nil, ErrHubNotFound
	}
	return sender, nil
}

// Delete removes the connection for a given hubID from the registry.
func (r *HubRegistry) Delete(hubID string) {
	shard := r.getShard(hubID)

	r.idxMu.Lock()
	defer r.idxMu.Unlock()

	shard.Lock()
	defer shard.Unlock()

	if entry, exists := shard.connections[hubID]; exists {
		if entry.organizationID != "" {
			delete(r.organizationIdx, entry.organizationID)
		}
		delete(shard.connections, hubID)
	}
}

// Shutdown closes all connections and clears the registry.
// It returns a list of hubIDs that were connected at the time of shutdown,
// so the caller can update their database status to offline.
func (r *HubRegistry) Shutdown() []string {
	r.idxMu.Lock()
	r.organizationIdx = make(map[string]string)
	r.idxMu.Unlock()

	var hubIDs []string

	for i := 0; i < 256; i++ {
		shard := r.shards[i]
		shard.Lock()
		for hubID, entry := range shard.connections {
			hubIDs = append(hubIDs, hubID)
			if closer, ok := entry.sender.(interface{ Close() }); ok {
				closer.Close()
			}
		}
		shard.connections = make(map[string]*connEntry)
		shard.Unlock()
	}

	return hubIDs
}

// GetAllHubIDs returns a list of all currently connected hub IDs.
func (r *HubRegistry) GetAllHubIDs() []string {
	var hubIDs []string
	for i := range r.shards {
		shard := r.shards[i]
		shard.RLock()
		for hubID := range shard.connections {
			hubIDs = append(hubIDs, hubID)
		}
		shard.RUnlock()
	}
	return hubIDs
}
