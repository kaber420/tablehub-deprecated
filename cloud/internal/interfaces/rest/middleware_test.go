package rest

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/tablehub/cloud/pkg/auth"
)

func TestRequireOrgAccess_Middleware(t *testing.T) {
	tests := []struct {
		name           string
		tokenInfo      *auth.TokenInfo
		requestedOrgID string
		expectedStatus int
	}{
		{
			name: "Access granted - User has role in org",
			tokenInfo: &auth.TokenInfo{
				Sub: "user-1",
				Roles: auth.ZitadelRoles{
					"viewer": {"org-123": "Org Name"},
				},
			},
			requestedOrgID: "org-123",
			expectedStatus: http.StatusOK,
		},
		{
			name: "Access denied - User org mismatch",
			tokenInfo: &auth.TokenInfo{
				Sub: "user-1",
				Roles: auth.ZitadelRoles{
					"viewer": {"org-123": "Org Name"},
				},
			},
			requestedOrgID: "org-456",
			expectedStatus: http.StatusForbidden,
		},
		{
			name: "Access granted - Staff global role",
			tokenInfo: &auth.TokenInfo{
				Sub: "staff-1",
				Roles: auth.ZitadelRoles{
					"platform_admin": {"global": "Platform"},
				},
			},
			requestedOrgID: "org-789",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := chi.NewRouter()
			r.Use(func(next http.Handler) http.Handler {
				return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					ctx := context.WithValue(r.Context(), oidcContextKey, tt.tokenInfo)
					next.ServeHTTP(w, r.WithContext(ctx))
				})
			})
			r.Route("/orgs/{orgID}", func(r chi.Router) {
				r.Use(RequireOrgAccess())
				r.Get("/test", func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
				})
			})

			req := httptest.NewRequest("GET", "/orgs/"+tt.requestedOrgID+"/test", nil)
			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rec.Code)
			}
		})
	}
}

func TestRequirePermission_Middleware(t *testing.T) {
	tests := []struct {
		name               string
		tokenInfo          *auth.TokenInfo
		requestedOrgID     string
		requiredPermission string
		expectedStatus     int
	}{
		{
			name: "Permission granted - Owner has write permission",
			tokenInfo: &auth.TokenInfo{
				Sub: "user-1",
				Roles: auth.ZitadelRoles{
					"owner": {"org-123": "Org Name"},
				},
			},
			requestedOrgID:     "org-123",
			requiredPermission: "branch:write",
			expectedStatus:     http.StatusOK,
		},
		{
			name: "Permission denied - Viewer lacks write permission",
			tokenInfo: &auth.TokenInfo{
				Sub: "user-2",
				Roles: auth.ZitadelRoles{
					"viewer": {"org-123": "Org Name"},
				},
			},
			requestedOrgID:     "org-123",
			requiredPermission: "branch:write",
			expectedStatus:     http.StatusForbidden,
		},
		{
			name: "Permission granted - Platform admin global role",
			tokenInfo: &auth.TokenInfo{
				Sub: "staff-1",
				Roles: auth.ZitadelRoles{
					"platform_admin": {"global": "Platform"},
				},
			},
			requestedOrgID:     "org-456",
			requiredPermission: "global:manage",
			expectedStatus:     http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := chi.NewRouter()
			r.Use(func(next http.Handler) http.Handler {
				return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					ctx := context.WithValue(r.Context(), oidcContextKey, tt.tokenInfo)
					next.ServeHTTP(w, r.WithContext(ctx))
				})
			})
			r.Route("/orgs/{orgID}", func(r chi.Router) {
				r.Use(RequirePermission(tt.requiredPermission))
				r.Get("/test", func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
				})
			})

			req := httptest.NewRequest("GET", "/orgs/"+tt.requestedOrgID+"/test", nil)
			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rec.Code)
			}
		})
	}
}
