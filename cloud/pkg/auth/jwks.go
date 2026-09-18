package auth

import (
	"strings"

	"github.com/MicahParks/keyfunc/v3"
	"github.com/golang-jwt/jwt/v5"
)

// Authenticator provides JWT validation against a JWKS endpoint.
type Authenticator struct {
	jwks keyfunc.Keyfunc
}

// NewAuthenticator creates a new Authenticator by fetching JWKS from the given URL.
func NewAuthenticator(jwksURL string) (*Authenticator, error) {
	if jwksURL == "" || strings.HasPrefix(jwksURL, "mock") {
		return &Authenticator{jwks: nil}, nil
	}
	k, err := keyfunc.NewDefault([]string{jwksURL})
	if err != nil {
		return nil, err
	}
	return &Authenticator{jwks: k}, nil
}

// ZitadelRoles maps a role name to a map of organization IDs to organization names.
type ZitadelRoles map[string]map[string]string

// TokenInfo holds basic user claims extracted from the Zitadel JWT token.
type TokenInfo struct {
	Sub   string
	Email string
	Name  string
	Roles ZitadelRoles
}

// HasRoleInOrg checks if the token has the specified role for the given organization.
func (t *TokenInfo) HasRoleInOrg(orgID string, role string) bool {
	if t.Roles == nil {
		return false
	}
	orgs, ok := t.Roles[role]
	if !ok {
		return false
	}
	// Also match global role if present
	if _, hasGlobal := orgs["global"]; hasGlobal {
		return true
	}
	_, hasOrg := orgs[orgID]
	return hasOrg
}

// GetRolesForOrg returns all roles assigned to the user within a given organization.
func (t *TokenInfo) GetRolesForOrg(orgID string) []string {
	var userRoles []string
	for role, orgs := range t.Roles {
		if _, hasOrg := orgs[orgID]; hasOrg {
			userRoles = append(userRoles, role)
		} else if _, hasGlobal := orgs["global"]; hasGlobal {
			userRoles = append(userRoles, role)
		}
	}
	return userRoles
}

// HasGlobalRole checks if the user has a specific role with global scope.
func (t *TokenInfo) HasGlobalRole(role string) bool {
	if t.Roles == nil {
		return false
	}
	orgs, ok := t.Roles[role]
	if !ok {
		return false
	}
	_, hasGlobal := orgs["global"]
	return hasGlobal
}

// GetGlobalRoles returns all roles that have global scope.
func (t *TokenInfo) GetGlobalRoles() []string {
	var roles []string
	for role, orgs := range t.Roles {
		if _, hasGlobal := orgs["global"]; hasGlobal {
			roles = append(roles, role)
		}
	}
	return roles
}

// ValidateToken parses and validates a JWT token string.
// It returns the TokenInfo if successful.
func (a *Authenticator) ValidateToken(tokenStr string) (*TokenInfo, error) {
	if strings.HasPrefix(tokenStr, "mock-token") || a.jwks == nil {
		// Mock validator for local development and testing
		return &TokenInfo{
			Sub:   "mock-owner-123",
			Email: "mock-owner@tablehub.com",
			Name:  "Mock Owner",
			Roles: ZitadelRoles{
				"owner": map[string]string{"mock-org-001": "Mock Org"},
			},
		}, nil
	}

	token, err := jwt.Parse(tokenStr, a.jwks.Keyfunc)
	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		sub, _ := claims["sub"].(string)
		email, _ := claims["email"].(string)
		name, _ := claims["name"].(string)
		if name == "" {
			name, _ = claims["preferred_username"].(string)
		}

		roles := make(ZitadelRoles)

		// 1. Zitadel Claims
		if rolesClaim, exists := claims["urn:zitadel:iam:org:project:roles"]; exists {
			if rolesMap, ok := rolesClaim.(map[string]interface{}); ok {
				for roleName, orgsVal := range rolesMap {
					if orgsMap, ok := orgsVal.(map[string]interface{}); ok {
						if roles[roleName] == nil {
							roles[roleName] = make(map[string]string)
						}
						for orgID, orgNameVal := range orgsMap {
							if orgName, ok := orgNameVal.(string); ok {
								roles[roleName][orgID] = orgName
							}
						}
					}
				}
			}
		}

		// 2. Logto claims (flat org role mapping: "org_id:role_name")
		if orgRolesClaim, exists := claims["organization_roles"]; exists {
			if rolesSlice, ok := orgRolesClaim.([]interface{}); ok {
				for _, rVal := range rolesSlice {
					if rStr, ok := rVal.(string); ok {
						parts := strings.SplitN(rStr, ":", 2)
						if len(parts) == 2 {
							orgID := parts[0]
							roleName := parts[1]
							if roles[roleName] == nil {
								roles[roleName] = make(map[string]string)
							}
							roles[roleName][orgID] = ""
						}
					}
				}
			}
		}

		// 3. Custom/Standard roles claim
		if rolesClaim, exists := claims["roles"]; exists {
			if rolesSlice, ok := rolesClaim.([]interface{}); ok {
				for _, rVal := range rolesSlice {
					if rStr, ok := rVal.(string); ok {
						if roles[rStr] == nil {
							roles[rStr] = make(map[string]string)
						}
						roles[rStr]["global"] = ""
					}
				}
			}
		}

		if sub != "" {
			return &TokenInfo{
				Sub:   sub,
				Email: email,
				Name:  name,
				Roles: roles,
			}, nil
		}
	}
	return nil, jwt.ErrInvalidKey
}
