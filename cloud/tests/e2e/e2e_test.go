//go:build e2e

package e2e

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/tablehub/cloud/internal/application/tunnel"
	appwebhook "github.com/tablehub/cloud/internal/application/webhook"
	"github.com/tablehub/cloud/internal/domain/hub"
	"github.com/tablehub/cloud/internal/domain/organization"
	domainwebhook "github.com/tablehub/cloud/internal/domain/webhook"
	"github.com/tablehub/cloud/internal/infrastructure/db"
	infranats "github.com/tablehub/cloud/internal/infrastructure/nats"
	infrawebhook "github.com/tablehub/cloud/internal/infrastructure/webhook"
	"github.com/tablehub/cloud/internal/interfaces/rest"
	"github.com/tablehub/cloud/internal/interfaces/ws"
	"github.com/tablehub/cloud/pkg/types"
)

func setupTestDB(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()

	// 1. Limpiar la DB
	_, err := pool.Exec(context.Background(), "DROP SCHEMA public CASCADE; CREATE SCHEMA public;")
	require.NoError(t, err, "failed to drop and recreate schema")

	// 2. Aplicar el esquema
	schemaPath := "../../internal/infrastructure/db/schema.sql"
	content, err := os.ReadFile(schemaPath)
	require.NoError(t, err, "failed to read schema file %s", schemaPath)
	
	_, err = pool.Exec(context.Background(), string(content))
	require.NoError(t, err, "failed to execute schema")
}

func startNATS(ctx context.Context, t *testing.T) (testcontainers.Container, string) {
	req := testcontainers.ContainerRequest{
		Image:        "nats:2.10-alpine",
		ExposedPorts: []string{"4222/tcp"},
		Cmd:          []string{"-js"},
		WaitingFor:   wait.ForLog("Server is ready"),
	}
	natsC, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	require.NoError(t, err)

	ip, err := natsC.Host(ctx)
	require.NoError(t, err)
	port, err := natsC.MappedPort(ctx, "4222")
	require.NoError(t, err)

	return natsC, "nats://" + ip + ":" + port.Port()
}

func TestE2EFullFlow(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	ctx := context.Background()

	// 1. Start PostgreSQL
	pgContainer, err := postgres.Run(ctx,
		"docker.io/postgres:16-alpine",
		postgres.WithDatabase("tablehub_cloud"),
		postgres.WithUsername("tablehub"),
		postgres.WithPassword("tablehub"),
		testcontainers.WithWaitStrategyAndDeadline(
			60*time.Second,
			wait.ForListeningPort("5432/tcp"),
		),
	)
	require.NoError(t, err)
	defer func() { _ = pgContainer.Terminate(ctx) }()

	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	pool, err := db.NewPool(ctx, db.DefaultConfig(connStr))
	require.NoError(t, err)
	defer pool.Close()

	setupTestDB(t, pool)

	// 2. Start NATS
	natsContainer, natsURL := startNATS(ctx, t)
	defer func() { _ = natsContainer.Terminate(ctx) }()

	nc, err := nats.Connect(natsURL)
	require.NoError(t, err)
	defer nc.Close()
	js, err := jetstream.New(nc)
	require.NoError(t, err)

	_, err = js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name:     "TENANT_ORDERS",
		Subjects: []string{"tenant.*.pos.order"},
	})
	require.NoError(t, err)

	natsPublisher := infranats.NewEventPublisher(js)

	// 3. Init Apps
	hubRepo := db.NewHubRepo(pool)
	keyProvider := db.NewDBKeyProvider(hubRepo)
	organizationRepo := db.NewOrganizationRepo(pool)
	webhookEventRepo := db.NewWebhookEventRepo(pool)
	userRepo := db.NewUserRepository(pool)

	hubRegistry := ws.NewHubRegistry()
	providerRegistry := infrawebhook.NewInMemoryProviderRegistry()
	providerRegistry.Register(infrawebhook.NewTastyIgniterProvider())

	connectUC := tunnel.NewConnectHubUseCase(hubRegistry, hubRepo)
	disconnectUC := tunnel.NewDisconnectHubUseCase(hubRegistry, hubRepo, webhookEventRepo)
	forwardUC := tunnel.NewForwardMessageUseCase(hubRegistry, webhookEventRepo)
	receiveWebhookUC := appwebhook.NewReceiveWebhookUseCase(organizationRepo, webhookEventRepo, providerRegistry, natsPublisher)

	wsCfg := ws.WSConfig{
		WriteWait:      500 * time.Millisecond,
		PongWait:       2 * time.Second,
		PingPeriod:     200 * time.Millisecond,
		MaxMessageSize: 65536,
		SendBufferSize: 256,
	}

	wsHandler := ws.NewWSHandler(keyProvider, wsCfg, connectUC, disconnectUC, forwardUC)
	hubHandler := rest.NewHubHandler(hubRepo)
	webhookHandler := rest.NewWebhookHandler(receiveWebhookUC)
	adminHandler := rest.NewAdminHandler(hubRegistry)

	router := rest.NewRouter(wsHandler, hubHandler, webhookHandler, adminHandler, nil, nil, nil, nil, nil, nil, userRepo, []string{"*"})
	srv := httptest.NewServer(router)
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/ws"
	webhookURL := srv.URL + "/v1/webhooks/tastyigniter/"

	pubKey, privKey, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	hubID := "e2e-test-hub"
	restID := types.NewOrganizationID()

	rest := &organization.Organization{
		ID:   restID,
		Slug: "e2e-test-organization",
		Name: "E2E Test Organization",
		Plan: organization.PlanFree,
		Settings: organization.Settings{
			WebhookToken: "e2e-test-token",
		},
	}
	err = organizationRepo.Create(ctx, rest)
	require.NoError(t, err)

	hubEntity := &hub.Hub{
		ID:               hubID,
		OrganizationID:     restID,
		PublicKey:        hex.EncodeToString(pubKey),
		ConnectionStatus: hub.StatusOffline,
		Metadata: hub.Metadata{
			FirmwareVersion: "1.0.0-test",
		},
	}
	err = hubRepo.Register(ctx, hubEntity)
	require.NoError(t, err)

	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer conn.Close()

	msgCh := make(chan tunnel.TunnelMessage, 64)
	go func() {
		for {
			var msg tunnel.TunnelMessage
			if err := conn.ReadJSON(&msg); err != nil {
				return
			}
			select {
			case msgCh <- msg:
			default:
			}
		}
	}()

	readMsg := func() tunnel.TunnelMessage {
		select {
		case msg := <-msgCh:
			return msg
		case <-time.After(10 * time.Second):
			t.Fatal("timeout waiting for WebSocket message")
			return tunnel.TunnelMessage{}
		}
	}

	writeMsg := func(msg tunnel.TunnelMessage) {
		require.NoError(t, conn.WriteJSON(msg))
	}

	challengeMsg := readMsg()
	require.Equal(t, "auth_challenge", challengeMsg.Event)

	challengePayload := challengeMsg.Payload.(map[string]interface{})
	challengeHex := challengePayload["challenge_hex"].(string)

	challengeBytes, err := hex.DecodeString(challengeHex)
	require.NoError(t, err)
	signature := ed25519.Sign(privKey, challengeBytes)

	writeMsg(tunnel.TunnelMessage{
		Event:     "auth_response",
		Timestamp: time.Now().UnixMilli(),
		Payload: map[string]interface{}{
			"hub_id":        hubID,
			"organization_id": restID.String(),
			"signature_hex": hex.EncodeToString(signature),
		},
	})

	successMsg := readMsg()
	require.Equal(t, "auth_success", successMsg.Event)

	// Subscribe to NATS to verify webhook delivery
	sub, err := nc.SubscribeSync("tenant.*.pos.order")
	require.NoError(t, err)

	webhookPayload := `{"event":"order.created","order_id":"e2e-order-123"}`
	req, err := http.NewRequest("POST", webhookURL+restID.String()+"?token=e2e-test-token", strings.NewReader(webhookPayload))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	msg, err := sub.NextMsg(5 * time.Second)
	require.NoError(t, err)
	require.Contains(t, string(msg.Data), "e2e-order-123")

	events, err := webhookEventRepo.GetEventsByOrganization(ctx, restID, 10, 0)
	require.NoError(t, err)
	require.Len(t, events, 1)
	event := events[0]
	require.Equal(t, domainwebhook.StatusDelivered, event.Status)

	hubFromDB, err := hubRepo.GetByID(ctx, hubID)
	require.NoError(t, err)
	assert.Equal(t, hub.StatusOnline, hubFromDB.ConnectionStatus)
	assert.NotNil(t, hubFromDB.LastSeen)
}

func TestE2EWebhookInvalidToken(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	ctx := context.Background()

	pgContainer, err := postgres.Run(ctx,
		"docker.io/postgres:16-alpine",
		postgres.WithDatabase("tablehub_cloud"),
		postgres.WithUsername("tablehub"),
		postgres.WithPassword("tablehub"),
		testcontainers.WithWaitStrategyAndDeadline(
			60*time.Second,
			wait.ForListeningPort("5432/tcp"),
		),
	)
	require.NoError(t, err)
	defer func() { _ = pgContainer.Terminate(ctx) }()

	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	pool, err := db.NewPool(ctx, db.DefaultConfig(connStr))
	require.NoError(t, err)
	defer pool.Close()

	setupTestDB(t, pool)

	natsContainer, natsURL := startNATS(ctx, t)
	defer func() { _ = natsContainer.Terminate(ctx) }()

	nc, err := nats.Connect(natsURL)
	require.NoError(t, err)
	defer nc.Close()
	js, err := jetstream.New(nc)
	require.NoError(t, err)

	_, err = js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name:     "TENANT_ORDERS",
		Subjects: []string{"tenant.*.pos.order"},
	})
	require.NoError(t, err)

	natsPublisher := infranats.NewEventPublisher(js)

	organizationRepo := db.NewOrganizationRepo(pool)
	webhookEventRepo := db.NewWebhookEventRepo(pool)
	providerRegistry := infrawebhook.NewInMemoryProviderRegistry()
	providerRegistry.Register(infrawebhook.NewTastyIgniterProvider())

	receiveWebhookUC := appwebhook.NewReceiveWebhookUseCase(organizationRepo, webhookEventRepo, providerRegistry, natsPublisher)

	hubHandler := rest.NewHubHandler(db.NewHubRepo(pool))
	webhookHandler := rest.NewWebhookHandler(receiveWebhookUC)
	adminHandler := rest.NewAdminHandler(ws.NewHubRegistry())

	router := rest.NewRouter(nil, hubHandler, webhookHandler, adminHandler, nil, nil, nil, nil, nil, nil, db.NewUserRepository(pool), []string{"*"})
	srv := httptest.NewServer(router)
	defer srv.Close()

	restID := types.NewOrganizationID()
	rest := &organization.Organization{
		ID:   restID,
		Slug: "e2e-invalid-token",
		Name: "Invalid Token Test",
		Plan: organization.PlanFree,
		Settings: organization.Settings{
			WebhookToken: "real-token",
		},
	}
	err = organizationRepo.Create(ctx, rest)
	require.NoError(t, err)

	webhookPayload := `{"event":"order.created"}`
	req, err := http.NewRequest("POST", srv.URL+"/v1/webhooks/tastyigniter/"+restID.String()+"?token=wrong-token",
		strings.NewReader(webhookPayload))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	resp.Body.Close()
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}
