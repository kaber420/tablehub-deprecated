package ws

import (
	"context"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"

	"github.com/tablehub/cloud/internal/application/tunnel"
)

var allowedOrigins = map[string]bool{
	"http://localhost:3000":  true,
	"http://localhost:9000":  true,
	"https://tablehub.local": true,
	// Aquí se pueden agregar dominios de producción en el futuro
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		// Clientes que no sean navegadores (como el Hub local) no envían encabezado Origin.
		// Permitimos acceso directo si no hay origin, o si está en la lista blanca.
		if origin == "" || allowedOrigins[origin] {
			return true
		}
		log.Warn().
			Str("origin", origin).
			Msg("websocket connection rejected due to CORS policy violation")
		return false
	},
}

// WSHandler orchestrates the WebSocket upgrade, authentication, and lifecycle
// of hub connections. It injects use cases from the application layer and
// delegates all business logic to them.
type WSHandler struct {
	provider   KeyProvider
	config     WSConfig
	connectUC  *tunnel.ConnectHubUseCase
	disconnUC  *tunnel.DisconnectHubUseCase
	msgHandler tunnel.MessageHandler
}

// NewWSHandler creates a new WSHandler wired with its use case dependencies.
func NewWSHandler(
	provider KeyProvider,
	config WSConfig,
	connectUC *tunnel.ConnectHubUseCase,
	disconnUC *tunnel.DisconnectHubUseCase,
	msgHandler tunnel.MessageHandler,
) *WSHandler {
	return &WSHandler{
		provider:   provider,
		config:     config,
		connectUC:  connectUC,
		disconnUC:  disconnUC,
		msgHandler: msgHandler,
	}
}

// UpgradeHandler handles the HTTP upgrade to WebSocket, authenticates the hub,
// and launches the supervisor goroutine that owns the connection lifecycle.
func (h *WSHandler) UpgradeHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Error().Err(err).Msg("failed to upgrade websocket connection")
		return
	}

	authResult, err := AuthenticateHub(conn, h.provider)
	if err != nil {
		log.Warn().Err(err).Msg("hub authentication failed")
		return
	}

	// Create the HubConn with the injected MessageHandler.
	hubConn := NewHubConn(conn, authResult.HubID, authResult.OrganizationID, h.config, h.msgHandler)

	// Execute the connect use case: register in memory + update DB to online.
	ctx := r.Context()
	if err := h.connectUC.Execute(ctx, authResult.HubID, authResult.OrganizationID, hubConn); err != nil {
		log.Error().Err(err).
			Str("hub_id", authResult.HubID).
			Msg("failed to execute connect use case")
		hubConn.Close()
		return
	}

	// Start the pumps and get the done channel.
	done := hubConn.Start(context.Background())

	log.Info().
		Str("hub_id", authResult.HubID).
		Str("organization_id", authResult.OrganizationID).
		Msg("hub connection established and registered")

	// Supervisor goroutine: waits for readPump to finish, then orchestrates
	// the full disconnect sequence. This is the ONLY goroutine that calls
	// DisconnectUseCase, eliminating race conditions.
	go func() {
		<-done

		log.Info().
			Str("hub_id", authResult.HubID).
			Msg("hub connection closed, executing disconnect use case")

		if err := h.disconnUC.Execute(context.Background(), authResult.HubID); err != nil {
			log.Error().Err(err).
				Str("hub_id", authResult.HubID).
				Msg("failed to execute disconnect use case")
		}
	}()
}
