package rest

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"
	appwebhook "github.com/tablehub/cloud/internal/application/webhook"
	domainwebhook "github.com/tablehub/cloud/internal/domain/webhook"
	"github.com/tablehub/cloud/pkg/types"
)

// ReceiveWebhookUseCase defines the interface for receiving webhooks.
type ReceiveWebhookUseCase interface {
	Execute(ctx context.Context, providerName string, organizationID types.OrganizationID, r appwebhook.WebhookRequest) (*domainwebhook.WebhookEvent, error)
}



// WebhookHandler handles incoming webhooks from external providers.
type WebhookHandler struct {
	receiveUC ReceiveWebhookUseCase
}

// NewWebhookHandler creates a new WebhookHandler.
func NewWebhookHandler(receiveUC ReceiveWebhookUseCase) *WebhookHandler {
	return &WebhookHandler{
		receiveUC: receiveUC,
	}
}

// HandleWebhook receives the webhook and dispatches it.
func (h *WebhookHandler) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	provider := chi.URLParam(r, "provider")
	organizationIDStr := chi.URLParam(r, "organization_id")

	organizationID, err := types.ParseOrganizationID(organizationIDStr)
	if err != nil {
		http.Error(w, "invalid organization ID", http.StatusBadRequest)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "failed to read request body", http.StatusBadRequest)
		return
	}

	wr := appwebhook.WebhookRequest{
		Body:    body,
		Headers: r.Header,
		Query:   r.URL.Query(),
	}

	_, err = h.receiveUC.Execute(r.Context(), provider, organizationID, wr)
	if err != nil {
		if errors.Is(err, domainwebhook.ErrInvalidToken) {
			http.Error(w, "invalid token", http.StatusUnauthorized)
			return
		}
		if errors.Is(err, domainwebhook.ErrUnsupportedProvider) {
			http.Error(w, "unsupported provider", http.StatusBadRequest)
			return
		}
		if errors.Is(err, domainwebhook.ErrDuplicateEvent) {
			// Idempotency: it's a duplicate, return 200 OK without processing further
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status":"delivered"}`))
			return
		}
		// Any other error (e.g. failed to log event, organization not found)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	// In case of publishing error, we still want to return 200 to the POS 
	// because the event was successfully received and persisted in DB.
	// The retry mechanism should handle publishing.


	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "delivered"})
}
