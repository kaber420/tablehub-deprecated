package webhook

import (
	"sync"

	appwebhook "github.com/tablehub/cloud/internal/application/webhook"
	domainwebhook "github.com/tablehub/cloud/internal/domain/webhook"
)

// InMemoryProviderRegistry is an in-memory implementation of appwebhook.ProviderRegistry.
type InMemoryProviderRegistry struct {
	mu        sync.RWMutex
	providers map[string]appwebhook.Provider
}

// NewInMemoryProviderRegistry creates a new registry.
func NewInMemoryProviderRegistry() *InMemoryProviderRegistry {
	return &InMemoryProviderRegistry{
		providers: make(map[string]appwebhook.Provider),
	}
}

// Register adds a provider to the registry.
func (r *InMemoryProviderRegistry) Register(provider appwebhook.Provider) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.providers[provider.Name()] = provider
}

// Get retrieves a provider by name.
func (r *InMemoryProviderRegistry) Get(name string) (appwebhook.Provider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	provider, ok := r.providers[name]
	if !ok {
		return nil, domainwebhook.ErrUnsupportedProvider
	}
	return provider, nil
}
