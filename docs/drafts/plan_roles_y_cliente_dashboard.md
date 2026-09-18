# Plan v3: Eliminar `IsSuperadmin`, Rediseñar Roles, Crear Vista de Cliente

> Incorpora feedback de Mimo (12 puntos) y del usuario (no IDs hardcodeados, org nullable para staff).

---

## 1. Sistema de Roles

### Roles de Staff (sin organización — `organization_id = NULL`)

| Rol | Descripción | Permisos |
|---|---|---|
| `platform_admin` | Control total del SaaS | `global:manage`, `hubs:pre_register`, `branch:write`, `hub:claim`, `hub:view`, `clients:view`, `billing:view` |
| `support` | Soporte técnico, ve datos de clientes | `hub:view`, `clients:view` |
| `billing` | Solo facturación | `billing:view` |

### Roles de Cliente (con organización — `organization_id = UUID`)

| Rol | Descripción | Permisos |
|---|---|---|
| `owner` | Dueño del restaurante | `billing:manage`, `users:authorize`, `branch:write`, `hub:claim`, `hub:view` |
| `admin` | Gerente que configura | `users:authorize`, `branch:write`, `hub:claim`, `hub:view` |
| `viewer` | Solo lectura | `hub:view` |

### Cómo se distingue staff de cliente

**Solo por el rol.** No hay IDs hardcodeados, no hay campos extra, no hay `.env` especial.

- Rol es `platform_admin`, `support`, o `billing` → **Staff**. `organization_id` es `NULL`. Acceso global.
- Rol es `owner`, `admin`, o `viewer` → **Cliente**. `organization_id` tiene un UUID. Acceso solo a su org.

### Cómo el staff accede a datos de orgs de clientes

Los roles de staff se registran en el token OIDC con scope `"global"`. El middleware `RequireOrgAccess` ya soporta el concepto de `"global"` en `GetRolesForOrg()`. Al eliminar el bypass `IsSuperadmin()`, el flujo pasa por `GetGlobalRoles()` que retorna los roles con scope global → acceso a cualquier org.

### Flujo de datos: JWT → Middleware → Frontend

```
Token OIDC (Logto)
  └─ roles claim: {"platform_admin": {"global": ""}}  ← staff
  └─ roles claim: {"owner": {"org-uuid": ""}}          ← cliente

      ↓ RequireAuth (valida JWT, extrae TokenInfo)
      ↓ RequireOrgAccess (checa GetGlobalRoles() o GetRolesForOrg())  
      ↓ RequirePermission (checa rolePermissions[role])

      ↓ /v1/me → devuelve {role: "owner", organization_id: "uuid"}

Frontend
  └─ STAFF_ROLES.includes(profile.role) → Dashboard SaaS
  └─ else → ClientDashboard
```

---

## 2. Cambios por Archivo

### DB - Nueva migración

#### [NEW] [010_roles_redesign.sql](file:///home/kaber420/Documentos/proyectos/tablehub2/cloud/internal/infrastructure/db/migrations/010_roles_redesign.sql)

> [!NOTE]
> La migración `005` ya hizo `organization_id` nullable y agregó `superadmin` al CHECK. Solo necesitamos actualizar el CHECK constraint para los nuevos roles.

```sql
-- 010_roles_redesign.sql
-- Replaces 'superadmin' role with staff roles: platform_admin, support, billing

BEGIN;

-- Update existing superadmin users to platform_admin
UPDATE users SET role = 'platform_admin' WHERE role = 'superadmin';

-- Replace role constraint
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_role_check;
ALTER TABLE users ADD CONSTRAINT users_role_check 
    CHECK (role IN ('platform_admin', 'support', 'billing', 'owner', 'admin', 'viewer'));

COMMIT;
```

---

### Backend - Modelo de Dominio

#### [MODIFY] [user.go](file:///home/kaber420/Documentos/proyectos/tablehub2/cloud/internal/domain/user/user.go)

```diff
 const (
-	RoleSuperadmin Role = "superadmin"
-	RoleOwner      Role = "owner"
-	RoleAdmin      Role = "admin"
-	RoleViewer     Role = "viewer"
+	// Staff roles (platform-level, no organization)
+	RolePlatformAdmin Role = "platform_admin"
+	RoleSupport       Role = "support"
+	RoleBilling       Role = "billing"
+
+	// Client roles (organization-scoped)
+	RoleOwner  Role = "owner"
+	RoleAdmin  Role = "admin"
+	RoleViewer Role = "viewer"
 )
+
+// IsStaffRole returns true if the role belongs to platform staff.
+func (r Role) IsStaffRole() bool {
+	switch r {
+	case RolePlatformAdmin, RoleSupport, RoleBilling:
+		return true
+	}
+	return false
+}
```

---

### Backend - Auth

#### [MODIFY] [jwks.go](file:///home/kaber420/Documentos/proyectos/tablehub2/cloud/pkg/auth/jwks.go)

**1. Eliminar `IsSuperadmin()` (líneas 68-75). Reemplazar con:**

```go
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
```

**2. Corregir mock token (líneas 80-91):**

```diff
 if strings.HasPrefix(tokenStr, "mock-token") || a.jwks == nil {
 	return &TokenInfo{
-		Sub:   "mock-user-123",
-		Email: "mock-admin@tablehub.com",
-		Name:  "Mock Admin",
+		Sub:   "mock-owner-123",
+		Email: "mock-owner@tablehub.com",
+		Name:  "Mock Owner",
 		Roles: ZitadelRoles{
-			"superadmin": map[string]string{"global": ""},
-			"admin":      map[string]string{"global": ""},
+			"owner": map[string]string{"mock-org-001": "Mock Org"},
 		},
 	}, nil
 }
```

---

#### [MODIFY] [jwks_test.go](file:///home/kaber420/Documentos/proyectos/tablehub2/cloud/pkg/auth/jwks_test.go)

Eliminar `TestTokenInfo_IsSuperadmin` (líneas 80-102). Agregar:

```go
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
```

---

### Backend - Middleware

#### [MODIFY] [middleware.go](file:///home/kaber420/Documentos/proyectos/tablehub2/cloud/internal/interfaces/rest/middleware.go)

**1. `RequireOrgAccess` (línea 89-93):**

```diff
-		// Bypasear verificación para superadministradores de la plataforma
-		if tokenInfo.IsSuperadmin() {
-			next.ServeHTTP(w, r)
-			return
-		}
+		// Staff con scope global tiene acceso a cualquier org
+		if len(tokenInfo.GetGlobalRoles()) > 0 {
+			next.ServeHTTP(w, r)
+			return
+		}
```

**2. `RequirePermission` (líneas 156-161):**

```diff
-		if tokenInfo.IsSuperadmin() {
-			if perms, ok := rolePermissions[user.RoleSuperadmin]; ok && perms[requiredPermission] {
-				next.ServeHTTP(w, r)
-				return
-			}
-		}
+		// Checar roles con scope global primero (no necesitan org)
+		for _, rName := range tokenInfo.GetGlobalRoles() {
+			if perms, ok := rolePermissions[user.Role(rName)]; ok && perms[requiredPermission] {
+				next.ServeHTTP(w, r)
+				return
+			}
+		}
```

**3. Mapa de permisos (líneas 117-144):**

```diff
 var rolePermissions = map[user.Role]map[string]bool{
-	user.RoleSuperadmin: {
+	user.RolePlatformAdmin: {
 		"global:manage":     true,
 		"hubs:pre_register": true,
 		"branch:write":      true,
 		"hub:claim":         true,
-		"clients:move":      true,
+		"clients:view":      true,
 		"hub:view":          true,
+		"billing:view":      true,
+	},
+	user.RoleSupport: {
+		"hub:view":     true,
+		"clients:view": true,
+	},
+	user.RoleBilling: {
+		"billing:view": true,
 	},
 	user.RoleOwner: {
 		"billing:manage":  true,
 		"users:authorize": true,
 		"branch:write":    true,
 		"hub:claim":       true,
-		"clients:move":    true,
 		"hub:view":        true,
 	},
 	user.RoleAdmin: {
 		"users:authorize": true,
 		"branch:write":    true,
 		"hub:claim":       true,
-		"clients:move":    true,
 		"hub:view":        true,
 	},
 	user.RoleViewer: {
 		"hub:view": true,
 	},
 }
```

> [!NOTE]
> `global:manage` solo lo tiene `platform_admin`. Las rutas `/v1/admin/*` requieren `global:manage`, por lo que solo `platform_admin` las accede. `support` y `billing` no.

---

#### [MODIFY] [middleware_test.go](file:///home/kaber420/Documentos/proyectos/tablehub2/cloud/internal/interfaces/rest/middleware_test.go)

Actualizar test cases:

```diff
 {
-	name: "Access granted - Superadmin bypass",
+	name: "Access granted - Staff global role",
 	tokenInfo: &auth.TokenInfo{
-		Sub: "superadmin-1",
+		Sub: "staff-1",
 		Roles: auth.ZitadelRoles{
-			"superadmin": {"system": "System"},
+			"platform_admin": {"global": "Platform"},
 		},
 	},
 	requestedOrgID: "org-789",
 	expectedStatus: http.StatusOK,
 },
```

Y en `TestRequirePermission_Middleware`:

```diff
 {
-	name: "Permission granted - Superadmin bypass",
+	name: "Permission granted - Platform admin global role",
 	tokenInfo: &auth.TokenInfo{
-		Sub: "superadmin-1",
+		Sub: "staff-1",
 		Roles: auth.ZitadelRoles{
-			"superadmin": {"system": "System"},
+			"platform_admin": {"global": "Platform"},
 		},
 	},
 	requestedOrgID:     "org-456",
 	requiredPermission: "global:manage",
 	expectedStatus:     http.StatusOK,
 },
```

---

### Backend - Onboarding / GetMe

#### [MODIFY] [onboarding.go](file:///home/kaber420/Documentos/proyectos/tablehub2/cloud/internal/interfaces/rest/onboarding.go)

En `GetMe` (línea 200-213) — quitar `is_superadmin`:

```diff
 json.NewEncoder(w).Encode(map[string]interface{}{
 	"registered":      true,
 	"pending":         false,
 	"id":              u.ID.String(),
 	"organization_id": u.OrganizationID.String(),
 	"role":            string(u.Role),
-	"is_superadmin":   tokenInfo.IsSuperadmin(),
 	"name":            u.Name,
 	"email":           u.Email,
 })
```

---

### Backend - Users Handler

#### [MODIFY] [users.go](file:///home/kaber420/Documentos/proyectos/tablehub2/cloud/internal/interfaces/rest/users.go)

Actualizar validación de roles en `InviteUser` (línea 63):

```diff
 	var targetRole user.Role
 	switch req.Role {
 	case string(user.RoleAdmin):
 		targetRole = user.RoleAdmin
 	case string(user.RoleViewer):
 		targetRole = user.RoleViewer
 	default:
 		http.Error(w, `{"error":"invalid role, must be 'admin' or 'viewer'"}`, http.StatusBadRequest)
 		return
 	}
```

Sin cambios aquí — la invitación ya solo permite `admin` y `viewer`, que son roles de cliente. Correcto.

---

### Frontend - App

#### [MODIFY] [App.svelte](file:///home/kaber420/Documentos/proyectos/tablehub2/cloud/web/src/App.svelte)

```diff
+import ClientDashboard from "./pages/ClientDashboard.svelte";

 // User profile state
 let profile = null;

+// Staff role detection
+const STAFF_ROLES = ['platform_admin', 'support', 'billing'];
+$: isStaff = profile && STAFF_ROLES.includes(profile.role);
```

Routing condicional (líneas 106-113):

```diff
 {#if activeView === 'dashboard'}
-    <Dashboard />
+    {#if isStaff}
+        <Dashboard />
+    {:else}
+        <ClientDashboard {profile} />
+    {/if}
 {:else if activeView === 'organizations'}
-    {#if profile.is_superadmin}
+    {#if isStaff}
         <Organizations />
     {:else}
-        <Dashboard />
+        <ClientDashboard {profile} />
     {/if}
```

---

### Frontend - Dock

#### [MODIFY] [Dock.svelte](file:///home/kaber420/Documentos/proyectos/tablehub2/cloud/web/src/lib/Dock.svelte)

```diff
+  const STAFF_ROLES = ['platform_admin', 'support', 'billing'];
+  $: isStaff = profile && STAFF_ROLES.includes(profile.role);

   $: navItems = [
       { id: 'dashboard', label: 'Dashboard', icon: LayoutDashboard },
-      ...(profile && profile.is_superadmin ? [
+      ...(isStaff ? [
           { id: 'organizations', label: 'Clientes y Orgs', icon: Building }
       ] : []),
-      { id: 'devices', label: 'Devices / IoT', icon: Tablet },
-      { id: 'users', label: 'Staff / Users', icon: Users },
+      { id: 'devices', label: isStaff ? 'Devices / IoT' : 'Mis Hubs', icon: Tablet },
+      { id: 'users', label: isStaff ? 'Staff' : 'Mi Equipo', icon: Users },
       { id: 'settings', label: 'Settings', icon: Settings },
       { id: 'theme', label: 'Appearance', icon: Palette }
   ];
```

---

### Frontend - Nuevo Dashboard Cliente

#### [NEW] [ClientDashboard.svelte](file:///home/kaber420/Documentos/proyectos/tablehub2/cloud/web/src/pages/ClientDashboard.svelte)

Dashboard para clientes (`owner`/`admin`/`viewer`):

- **Header**: "Mi Organización" + nombre de la org (de `profile`)
- **KPIs**: Hubs activos, dispositivos registrados, sucursales (datos placeholder por ahora — se conectarán cuando exista un endpoint `/v1/orgs/{orgID}/stats`)
- **Panel de acciones rápidas**: Links a Mis Hubs, Mi Equipo, Configuración
- **Diseño**: Glass/neomorphic consistente con Dashboard.svelte

> [!NOTE]
> El `ClientDashboard` usará `profile.organization_id` para futuras llamadas a la API. Por ahora muestra datos de ejemplo. No se crea un endpoint nuevo de stats en este PR — eso es trabajo separado.

---

## 3. Resumen de Archivos (10 archivos)

| # | Archivo | Acción | Tipo |
|---|---|---|---|
| 1 | [010_roles_redesign.sql](file:///home/kaber420/Documentos/proyectos/tablehub2/cloud/internal/infrastructure/db/migrations/010_roles_redesign.sql) | NUEVO | Migración DB |
| 2 | [user.go](file:///home/kaber420/Documentos/proyectos/tablehub2/cloud/internal/domain/user/user.go) | Modificar | Modelo |
| 3 | [jwks.go](file:///home/kaber420/Documentos/proyectos/tablehub2/cloud/pkg/auth/jwks.go) | Modificar | Auth |
| 4 | [jwks_test.go](file:///home/kaber420/Documentos/proyectos/tablehub2/cloud/pkg/auth/jwks_test.go) | Modificar | Tests |
| 5 | [middleware.go](file:///home/kaber420/Documentos/proyectos/tablehub2/cloud/internal/interfaces/rest/middleware.go) | Modificar | Middleware |
| 6 | [middleware_test.go](file:///home/kaber420/Documentos/proyectos/tablehub2/cloud/internal/interfaces/rest/middleware_test.go) | Modificar | Tests |
| 7 | [onboarding.go](file:///home/kaber420/Documentos/proyectos/tablehub2/cloud/internal/interfaces/rest/onboarding.go) | Modificar | API |
| 8 | [App.svelte](file:///home/kaber420/Documentos/proyectos/tablehub2/cloud/web/src/App.svelte) | Modificar | Frontend |
| 9 | [Dock.svelte](file:///home/kaber420/Documentos/proyectos/tablehub2/cloud/web/src/lib/Dock.svelte) | Modificar | Frontend |
| 10 | [ClientDashboard.svelte](file:///home/kaber420/Documentos/proyectos/tablehub2/cloud/web/src/pages/ClientDashboard.svelte) | NUEVO | Frontend |

## 4. Verificación

1. `go test ./pkg/auth/... ./internal/interfaces/rest/...` — Tests unitarios backend
2. `go build ./...` — Compilación backend
3. `pnpm run build` en `cloud/web` — Compilación frontend
4. Login con mock token → entra como `owner` con `organization_id`, ve `ClientDashboard`
5. Verificar que el Dock muestra "Mis Hubs" y "Mi Equipo" (no "Devices/IoT" ni "Staff")

## 5. Lo que NO incluye este plan (trabajo futuro)

- Endpoint `/v1/orgs/{orgID}/stats` para datos reales del ClientDashboard
- CLI o UI para crear usuarios staff (hoy solo se crean directo en DB o via Logto)
- Permisos granulares adicionales para `support` y `billing`
