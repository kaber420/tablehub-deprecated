package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	appOnboarding "github.com/tablehub/cloud/internal/application/onboarding"
	"github.com/tablehub/cloud/internal/application/tunnel"
	appwebhook "github.com/tablehub/cloud/internal/application/webhook"
	"github.com/tablehub/cloud/internal/config"
	"github.com/tablehub/cloud/internal/domain/hub"
	"github.com/tablehub/cloud/internal/domain/user"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/tablehub/cloud/internal/infrastructure/db"
	"github.com/tablehub/cloud/internal/infrastructure/logto"
	"github.com/tablehub/cloud/internal/infrastructure/mock"
	infranats "github.com/tablehub/cloud/internal/infrastructure/nats"
	logger "github.com/tablehub/cloud/internal/infrastructure/observability"
	infrawebhook "github.com/tablehub/cloud/internal/infrastructure/webhook"
	"github.com/tablehub/cloud/internal/infrastructure/zitadel"
	"github.com/tablehub/cloud/internal/interfaces/rest"
	"github.com/tablehub/cloud/internal/interfaces/ws"
	"github.com/tablehub/cloud/pkg/auth"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Carga .env y levanta el servidor HTTP/WS de la nube",
	Run:   runServe,
}

func init() {
	serveCmd.Flags().IntP("port", "p", 8080, "Puerto del servidor HTTP")
	serveCmd.Flags().String("host", "0.0.0.0", "Host del servidor")
	serveCmd.Flags().StringP("env", "e", "dev", "Entorno: dev, staging, prod")
	serveCmd.Flags().String("log-level", "info", "Nivel de log: debug, info, warn, error")
	serveCmd.Flags().String("graceful-timeout", "30s", "Timeout para graceful shutdown")
	rootCmd.AddCommand(serveCmd)
}

func runServe(cmd *cobra.Command, args []string) {
	// 1. Load configuration
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		os.Exit(1)
	}

	// 2. Initialize logger
	log := logger.New(cfg.LogLevel, cfg.IsDev())
	log.Info().
		Int("port", cfg.Port).
		Str("env", cfg.Env).
		Msg("Starting Tablehub Cloud Server")

	// 3. Initialize database connection pool
	ctx := context.Background()
	pool, err := db.NewPool(ctx, db.DefaultConfig(cfg.DatabaseURL))
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to database")
	}
	defer pool.Close()

	// 3.5 Initialize NATS JetStream
	nc, err := nats.Connect(cfg.NatsURL, nats.Name("tablehub-cloud"))
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to NATS")
	}
	defer nc.Close()

	js, err := jetstream.New(nc)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize JetStream")
	}

	_, err = js.CreateOrUpdateStream(context.Background(), jetstream.StreamConfig{
		Name:     "TENANT_ORDERS",
		Subjects: []string{"tenant.*.pos.order"},
	})
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create TENANT_ORDERS stream")
	}
	
	natsPublisher := infranats.NewEventPublisher(js)

	// 4. Initialize repositories
	hubRepo := db.NewHubRepo(pool)
	keyProvider := db.NewDBKeyProvider(hubRepo)
	organizationRepo := db.NewOrganizationRepo(pool)
	webhookEventRepo := db.NewWebhookEventRepo(pool)

	userRepo := db.NewUserRepository(pool)
	branchRepo := db.NewBranchRepository(pool)
	pendingOpRepo := db.NewPendingOpRepo(pool)

	// 5. Initialize in-memory connection registry and provider registry
	hubRegistry := ws.NewHubRegistry()
	providerRegistry := infrawebhook.NewInMemoryProviderRegistry()
	providerRegistry.Register(infrawebhook.NewTastyIgniterProvider())

	// 5.5 Initialize Authenticator
	authenticator, err := auth.NewAuthenticator(cfg.JWKSURL)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize Authenticator")
	}

	// 5.7 Initialize Identity Provider Client
	var identityProvider user.IdentityProvider
	switch cfg.AuthProvider {
	case "logto":
		log.Info().Str("endpoint", cfg.LogtoAPIURL).Msg("Using Logto Identity Provider")
		identityProvider = logto.NewClient(cfg.LogtoAPIURL, cfg.LogtoApplicationID, cfg.LogtoApplicationSecret, cfg.LogtoOrgID, log)
	case "mock":
		log.Info().Msg("Using Mock Identity Provider")
		identityProvider = mock.NewMockIdentityProvider()
	default:
		log.Info().Str("endpoint", cfg.ZitadelAPIURL).Msg("Using Zitadel Identity Provider")
		identityProvider = zitadel.NewClient(cfg.ZitadelAPIURL, cfg.ZitadelMachineKeyPath, cfg.ZitadelProjectID, cfg.ZitadelOrgID)
	}

	// 6. Initialize application use cases
	connectUC := tunnel.NewConnectHubUseCase(hubRegistry, hubRepo)
	disconnectUC := tunnel.NewDisconnectHubUseCase(hubRegistry, hubRepo, webhookEventRepo)
	forwardUC := tunnel.NewForwardMessageUseCase(hubRegistry, webhookEventRepo)
	provisionUC := tunnel.NewProvisionHubUseCase(hubRepo, cfg.CloudWSURL)
	receiveWebhookUC := appwebhook.NewReceiveWebhookUseCase(organizationRepo, webhookEventRepo, providerRegistry, natsPublisher)
	retryWorker := appwebhook.NewRetryWorker(webhookEventRepo, natsPublisher)

	// 7. Initialize interface handlers
	wsHandler := ws.NewWSHandler(keyProvider, ws.DefaultWSConfig(), connectUC, disconnectUC, forwardUC)
	hubHandler := rest.NewHubHandler(hubRepo)
	webhookHandler := rest.NewWebhookHandler(receiveWebhookUC)
	adminHandler := rest.NewAdminHandler(hubRegistry)
	provisionHandler := rest.NewProvisionHandler(provisionUC, hubRepo)
	onboardingHandler := rest.NewOnboardingHandler(organizationRepo, userRepo, identityProvider, pendingOpRepo, log)
	branchHandler := rest.NewBranchHandler(branchRepo)
	statsHandler := rest.NewStatsHandler(branchRepo, hubRepo)
	usersHandler := rest.NewUsersHandler(identityProvider, userRepo, organizationRepo)
	reconcWorker := appOnboarding.NewReconciliationWorker(pendingOpRepo, userRepo, organizationRepo, identityProvider, log)

	// 8. Setup HTTP router
	r := rest.NewRouter(
		wsHandler,
		hubHandler,
		webhookHandler,
		adminHandler,
		provisionHandler,
		onboardingHandler,
		branchHandler,
		statsHandler,
		usersHandler,
		authenticator,
		userRepo,
		cfg.AllowedOrigins,
	)

	// 9. Create HTTP server (timeouts disabled for WebSocket support)
	srv := &http.Server{
		Addr:         cfg.Addr(),
		Handler:      r,
		ReadTimeout:  0,
		WriteTimeout: 0,
		IdleTimeout:  120 * time.Second,
	}

	// 10. Start retry worker and reconciliation worker
	var wg sync.WaitGroup
	workerCtx, workerCancel := context.WithCancel(context.Background())
	wg.Add(1)
	go retryWorker.Start(workerCtx, &wg)
	wg.Add(1)
	go reconcWorker.Start(workerCtx, &wg)

	// 11. Start server in goroutine
	go func() {
		log.Info().Str("addr", cfg.Addr()).Msg("HTTP server listening")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("HTTP server failed")
		}
	}()

	// 12. Graceful shutdown on SIGTERM/SIGINT
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit

	log.Info().Str("signal", sig.String()).Msg("Shutdown signal received, draining connections...")

	// Shutdown order: HTTP server -> Worker -> HubRegistry -> DB offline updates -> pgxpool
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 11a. Stop accepting new HTTP connections.
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error().Err(err).Msg("HTTP server forced to shutdown")
	} else {
		log.Info().Msg("HTTP server stopped")
	}

	// 11b. Stop retry worker and wait for in-flight dispatches to finish.
	workerCancel()
	wg.Wait()
	log.Info().Msg("Retry worker stopped")

	// 11c. Close all WebSocket connections and get list of connected hub IDs.
	connectedHubIDs := hubRegistry.Shutdown()
	log.Info().Int("count", len(connectedHubIDs)).Msg("Hub registry shutdown complete")

	// 11d. Mark all previously-connected hubs as offline in the database.
	// This ensures no stale "online" records remain after a server restart.
	now := time.Now()
	for _, hubID := range connectedHubIDs {
		if err := hubRepo.UpdateStatus(shutdownCtx, hubID, hub.StatusOffline, &now); err != nil {
			log.Error().Err(err).Str("hub_id", hubID).Msg("Failed to set hub offline during shutdown")
		}
	}
	if len(connectedHubIDs) > 0 {
		log.Info().Int("count", len(connectedHubIDs)).Msg("All connected hubs marked offline in database")
	}

	// 11e. Close the database connection pool.
	pool.Close()
	log.Info().Msg("Database connection pool closed")

	log.Info().Msg("Server exited gracefully")
}
