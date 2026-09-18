package webhook

import "encoding/json"

// WebhookRequest abstracts the HTTP request for webhook processing.
type WebhookRequest struct {
	Body    []byte
	Headers map[string][]string
	Query   map[string][]string
}

// Provider abstracts a webhook source (TastyIgniter, UberEats, etc.)
type Provider interface {
	Name() string
	ValidateRequest(r WebhookRequest, secret string) error
	ParsePayload(r WebhookRequest) (eventType string, idempotencyKey *string, payload json.RawMessage, err error)
}

// ProviderRegistry manages registered webhook providers.
type ProviderRegistry interface {
	Register(provider Provider)
	Get(name string) (Provider, error)
}
