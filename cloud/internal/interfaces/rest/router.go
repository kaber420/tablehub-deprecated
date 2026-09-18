package rest

import (
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/tablehub/cloud/internal/domain/user"
	"github.com/tablehub/cloud/internal/interfaces/ws"
	"github.com/tablehub/cloud/pkg/auth"
)

// NewRouter creates the main Chi router with all middleware and route configuration.
func NewRouter(
	wsHandler *ws.WSHandler,
	hubHandler *HubHandler,
	webhookHandler *WebhookHandler,
	adminHandler *AdminHandler,
	provisionHandler *ProvisionHandler,
	onboardingHandler *OnboardingHandler,
	branchHandler *BranchHandler,
	statsHandler *StatsHandler,
	usersHandler *UsersHandler,
	authenticator *auth.Authenticator,
	userRepo user.Repository,
	allowedOrigins []string,
) chi.Router {
	r := chi.NewRouter()

	// Global middleware stack
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(CORSMiddleware(allowedOrigins))
	r.Use(middleware.Timeout(30 * time.Second))
	r.Use(middleware.Throttle(100)) // Rate limiting for REST endpoints

	// Health check
	r.Get("/health", hubHandler.HealthCheck)

	// WebSocket upgrade route (no REST timeout/throttle — handled separately)
	r.Get("/ws", wsHandler.UpgradeHandler)

	// API v1 routes
	r.Route("/v1", func(r chi.Router) {
		// Webhooks (public for POS integrations)
		r.Post("/webhooks/{provider}/{organization_id}", webhookHandler.HandleWebhook)

		// Hub activation (public, authenticated via bootstrap_token)
		r.Post("/hubs/activate", provisionHandler.HandleActivate)

		// OIDC Authenticated only (Onboarding, Get Me & Access Request submission)
		r.Group(func(r chi.Router) {
			r.Use(RequireAuth(authenticator))
			r.Get("/me", onboardingHandler.GetMe)
			r.Post("/onboarding", onboardingHandler.HandleOnboarding)
		})

		// Organization scoped routes (requires OIDC auth)
		r.Route("/orgs/{orgID}", func(r chi.Router) {
			r.Use(RequireAuth(authenticator))
			r.Use(RequireRegistered(userRepo))
			r.Use(RequireOrgAccess())

			// Hubs management
			r.With(RequirePermission("hub:view")).Get("/hubs", hubHandler.ListHubs)
			r.With(RequirePermission("hub:view")).Get("/hubs/{id}", hubHandler.GetHub)
			r.With(RequireRegistered(userRepo), RequirePermission("hub:claim")).Post("/hubs", hubHandler.CreateHub)
			r.With(RequireRegistered(userRepo), RequirePermission("hub:claim")).Put("/hubs/{id}", hubHandler.UpdateHub)
			r.With(RequireRegistered(userRepo), RequirePermission("hub:claim")).Post("/hubs/{id}/provision", provisionHandler.HandleProvision)

			// Branches management
			r.With(RequirePermission("branch:write")).Post("/branches", branchHandler.CreateBranch)
			r.With(RequirePermission("hub:view")).Get("/branches", branchHandler.ListBranches)

			// Stats
			r.With(RequirePermission("hub:view")).Get("/stats", statsHandler.GetOrgStats)

			// Users management
			r.With(RequirePermission("users:authorize")).Post("/users/invite", usersHandler.InviteUser)
		})

		// Admin (Global platform operators only)
		r.Route("/admin", func(r chi.Router) {
			r.Use(RequireAuth(authenticator))
			r.Use(RequirePermission("global:manage"))

			r.Get("/stats", adminHandler.Stats)
			r.Get("/tunnels", adminHandler.Tunnels)
		})
	})

	return r
}
