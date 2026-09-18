package auth

import (
	"testing"
)

func TestTokenInfo_HasRoleInOrg(t *testing.T) {
	roles := ZitadelRoles{
		"owner": {
			"org-1": "Org One",
		},
		"admin": {
			"org-2": "Org Two",
		},
	}

	token := &TokenInfo{
		Sub:   "user-1",
		Roles: roles,
	}

	if !token.HasRoleInOrg("org-1", "owner") {
		t.Error("expected owner role in org-1")
	}

	if token.HasRoleInOrg("org-1", "admin") {
		t.Error("did not expect admin role in org-1")
	}

	if !token.HasRoleInOrg("org-2", "admin") {
		t.Error("expected admin role in org-2")
	}

	if token.HasRoleInOrg("org-3", "admin") {
		t.Error("did not expect any roles in org-3")
	}
}

func TestTokenInfo_GetRolesForOrg(t *testing.T) {
	roles := ZitadelRoles{
		"owner": {
			"org-1": "Org One",
		},
		"admin": {
			"org-1": "Org One",
			"org-2": "Org Two",
		},
	}

	token := &TokenInfo{
		Sub:   "user-1",
		Roles: roles,
	}

	org1Roles := token.GetRolesForOrg("org-1")
	if len(org1Roles) != 2 {
		t.Errorf("expected 2 roles for org-1, got %d", len(org1Roles))
	}

	hasOwner := false
	hasAdmin := false
	for _, r := range org1Roles {
		if r == "owner" {
			hasOwner = true
		}
		if r == "admin" {
			hasAdmin = true
		}
	}
	if !hasOwner || !hasAdmin {
		t.Error("expected roles to contain 'owner' and 'admin'")
	}

	org2Roles := token.GetRolesForOrg("org-2")
	if len(org2Roles) != 1 || org2Roles[0] != "admin" {
		t.Errorf("expected only 'admin' role for org-2, got %v", org2Roles)
	}
}

func TestTokenInfo_HasGlobalRole(t *testing.T) {
	token := &TokenInfo{
		Roles: ZitadelRoles{
			"platform_admin": {"global": ""},
			"owner":          {"org-1": "Org One"},
		},
	}
	if !token.HasGlobalRole("platform_admin") {
		t.Error("expected platform_admin to have global scope")
	}
	if token.HasGlobalRole("owner") {
		t.Error("owner should not have global scope")
	}
	if token.HasGlobalRole("nonexistent") {
		t.Error("nonexistent role should not have global scope")
	}
}

func TestTokenInfo_GetGlobalRoles(t *testing.T) {
	token := &TokenInfo{
		Roles: ZitadelRoles{
			"platform_admin": {"global": ""},
			"support":        {"global": ""},
			"owner":          {"org-1": "Org One"},
		},
	}
	globalRoles := token.GetGlobalRoles()
	if len(globalRoles) != 2 {
		t.Errorf("expected 2 global roles, got %d", len(globalRoles))
	}

	tokenNoGlobal := &TokenInfo{
		Roles: ZitadelRoles{
			"owner": {"org-1": "Org One"},
		},
	}
	if len(tokenNoGlobal.GetGlobalRoles()) != 0 {
		t.Error("expected 0 global roles")
	}
}
