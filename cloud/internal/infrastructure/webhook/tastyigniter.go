package webhook

import (
	"crypto/subtle"
	"encoding/json"
	"fmt"

	appwebhook "github.com/tablehub/cloud/internal/application/webhook"
	domainwebhook "github.com/tablehub/cloud/internal/domain/webhook"
)

// TastyIgniterProvider implements the Provider interface for TastyIgniter.
type TastyIgniterProvider struct{}

// NewTastyIgniterProvider creates a new TastyIgniterProvider.
func NewTastyIgniterProvider() *TastyIgniterProvider {
	return &TastyIgniterProvider{}
}

// Name returns the provider name.
func (p *TastyIgniterProvider) Name() string {
	return "tastyigniter"
}

// ValidateRequest validates the TastyIgniter webhook token.
func (p *TastyIgniterProvider) ValidateRequest(r appwebhook.WebhookRequest, secret string) error {
	token := ""
	if vals, ok := r.Headers["X-TastyIgniter-Token"]; ok && len(vals) > 0 {
		token = vals[0]
	}
	if token == "" {
		if vals, ok := r.Query["token"]; ok && len(vals) > 0 {
			token = vals[0]
		}
	}

	// Timing-safe comparison to prevent timing attacks.
	if subtle.ConstantTimeCompare([]byte(token), []byte(secret)) != 1 {
		return domainwebhook.ErrInvalidToken
	}

	return nil
}

// ParsePayload extracts the event type and idempotency key from the payload.
func (p *TastyIgniterProvider) ParsePayload(r appwebhook.WebhookRequest) (eventType string, idempotencyKey *string, payload json.RawMessage, err error) {
	body := r.Body

	var genericPayload map[string]interface{}
	if err := json.Unmarshal(body, &genericPayload); err != nil {
		return "", nil, nil, err
	}

	eventType = "order.created"
	if ev, ok := genericPayload["event"].(string); ok && ev != "" {
		eventType = ev
	}

	// Try to extract an order ID to use as an idempotency key.
	if orderID, ok := genericPayload["order_id"].(string); ok && orderID != "" {
		idempotencyKey = &orderID
	} else if orderIDFloat, ok := genericPayload["order_id"].(float64); ok {
		orderIDStr := fmt.Sprintf("%v", orderIDFloat)
		idempotencyKey = &orderIDStr
	}

	return eventType, idempotencyKey, json.RawMessage(body), nil
}
