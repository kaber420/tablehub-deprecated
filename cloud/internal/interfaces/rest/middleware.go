package rest

import (
	"context"
	"log"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/tablehub/cloud/internal/domain/user"
	"github.com/tablehub/cloud/pkg/auth"
)

type contextKey string

const (
	authContextKey contextKey = "auth_user"
	oidcContextKey contextKey = "oidc_token"
)

// RequireAuth validates the OIDC JWT token and injects the auth.TokenInfo into the context.
func RequireAuth(authenticator *auth.Authenticator) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, "authorization header missing", http.StatusUnauthorized)
				return
			}

			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
				http.Error(w, "invalid authorization header format", http.StatusUnauthorized)
				return
			}

			tokenStr := parts[1]
			tokenInfo, err := authenticator.ValidateToken(tokenStr)
			if err != nil {
				log.Printf("Token validation failed: %v", err)
				http.Error(w, "invalid token", http.StatusUnauthorized)
				return
			}
			log.Printf("Token validation SUCCESS: Sub=%s, Email=%s, Name=%s", tokenInfo.Sub, tokenInfo.Email, tokenInfo.Name)

			ctx := context.WithValue(r.Context(), oidcContextKey, tokenInfo)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireRegistered checks if the OIDC user is registered in our database.
func RequireRegistered(userRepo user.Repository) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tokenInfo, ok := GetOIDCFromContext(r.Context())
			if !ok {
				http.Error(w, "unauthorized: token info missing", http.StatusUnauthorized)
				return
			}

			u, err := userRepo.GetByProviderUserID(r.Context(), tokenInfo.Sub)
			if err != nil {
				if err == user.ErrNotFound {
					http.Error(w, "user not registered in system", http.StatusForbidden)
					return
				}
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}

			ctx := context.WithValue(r.Context(), authContextKey, u)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireOrgAccess validates that the user belongs to the requested organization.
func RequireOrgAccess() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tokenInfo, ok := GetOIDCFromContext(r.Context())
			if !ok {
				http.Error(w, "unauthorized: token info missing", http.StatusUnauthorized)
				return
			}

			// Staff con scope global tiene acceso a cualquier org
			if len(tokenInfo.GetGlobalRoles()) > 0 {
				next.ServeHTTP(w, r)
				return
			}

			requestedOrgID := chi.URLParam(r, "orgID")
			if requestedOrgID == "" {
				requestedOrgID = r.Header.Get("X-Org-ID")
			}

			if requestedOrgID == "" {
				http.Error(w, "forbidden: missing organization scope", http.StatusForbidden)
				return
			}

			// Check DB User as fallback for local/logto mismatches
			if u, ok := GetUserFromContext(r.Context()); ok {
				if u.OrganizationID.String() == requestedOrgID {
					next.ServeHTTP(w, r)
					return
				}
			}

			userRoles := tokenInfo.GetRolesForOrg(requestedOrgID)
			if len(userRoles) == 0 {
				http.Error(w, "forbidden: cross-org access denied", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// Permisos mapeados a roles locales
var rolePermissions = map[user.Role]map[string]bool{
	user.RolePlatformAdmin: {
		"global:manage":     true,
		"hubs:pre_register": true,
		"branch:write":      true,
		"hub:claim":         true,
		"clients:view":      true,
		"hub:view":          true,
		"billing:view":      true,
	},
	user.RoleSupport: {
		"hub:view":     true,
		"clients:view": true,
	},
	user.RoleBilling: {
		"billing:view": true,
	},
	user.RoleOwner: {
		"billing:manage":  true,
		"users:authorize": true,
		"branch:write":    true,
		"hub:claim":       true,
		"hub:view":        true,
	},
	user.RoleAdmin: {
		"users:authorize": true,
		"branch:write":    true,
		"hub:claim":       true,
		"hub:view":        true,
	},
	user.RoleViewer: {
		"hub:view": true,
	},
}

// RequirePermission validates that the user's role has the required permission.
func RequirePermission(requiredPermission string) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tokenInfo, ok := GetOIDCFromContext(r.Context())
			if !ok {
				http.Error(w, "unauthorized: token info missing", http.StatusUnauthorized)
				return
			}

			// Checar roles con scope global primero (no necesitan org)
			for _, rName := range tokenInfo.GetGlobalRoles() {
				if perms, ok := rolePermissions[user.Role(rName)]; ok && perms[requiredPermission] {
					next.ServeHTTP(w, r)
					return
				}
			}

			requestedOrgID := chi.URLParam(r, "orgID")
			if requestedOrgID == "" {
				requestedOrgID = r.Header.Get("X-Org-ID")
			}

			if requestedOrgID == "" {
				http.Error(w, "forbidden: missing organization scope for permission check", http.StatusForbidden)
				return
			}

			// Check DB User as fallback
			if u, ok := GetUserFromContext(r.Context()); ok {
				if u.OrganizationID.String() == requestedOrgID {
					if perms, ok := rolePermissions[u.Role]; ok && perms[requiredPermission] {
						next.ServeHTTP(w, r)
						return
					}
				}
			}

			userRoles := tokenInfo.GetRolesForOrg(requestedOrgID)
			hasPerm := false
			for _, rName := range userRoles {
				if perms, ok := rolePermissions[user.Role(rName)]; ok {
					if perms[requiredPermission] {
						hasPerm = true
						break
					}
				}
			}

			if !hasPerm {
				http.Error(w, "forbidden: missing permission", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// GetUserFromContext retrieves the authenticated database user from the context.
func GetUserFromContext(ctx context.Context) (*user.User, bool) {
	u, ok := ctx.Value(authContextKey).(*user.User)
	return u, ok
}

// GetOIDCFromContext retrieves the OIDC TokenInfo from the context.
func GetOIDCFromContext(ctx context.Context) (*auth.TokenInfo, bool) {
	tokenInfo, ok := ctx.Value(oidcContextKey).(*auth.TokenInfo)
	return tokenInfo, ok
}

// CORSMiddleware handles Cross-Origin Resource Sharing headers.
func CORSMiddleware(allowedOrigins []string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if origin != "" {
				allowed := false
				for _, o := range allowedOrigins {
					if o == "*" || o == origin {
						allowed = true
						break
					}
				}

				if allowed {
					w.Header().Set("Access-Control-Allow-Origin", origin)
					w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE, PATCH")
					w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, X-Org-ID")
					w.Header().Set("Access-Control-Allow-Credentials", "true")
					w.Header().Set("Access-Control-Expose-Headers", "Content-Disposition")
				}
			}

			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
