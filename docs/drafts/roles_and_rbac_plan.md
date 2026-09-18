# Implementación de Roles y Permisos (RBAC y Multi-tenancy con Zitadel)

Este plan técnico detalla la arquitectura para integrar el control de acceso basado en roles (RBAC) y la validación estricta de multi-tenancy en el backend de Go (`cloud`), asegurando que todos los endpoints estén protegidos desde el inicio y que no haya fugas de datos entre franquicias.

## Contexto Actual
Actualmente, el backend de Go en `cloud/pkg/auth/jwks.go` valida el token de Zitadel y extrae únicamente el campo `sub` (Zitadel User ID). 
Para un entorno SaaS, es obligatorio extraer los **Roles**, el **OrgID**, y verificar los **Permisos de Acción** sobre un Tenant específico.

---

## 1. Estructuras Tipadas de Zitadel (Claims)

Zitadel inyecta los roles bajo el claim `urn:zitadel:iam:org:project:roles`. **IMPORTANTE:** Zitadel envía este claim como un mapa (Objeto JSON) donde la clave es el nombre del rol, y el valor contiene los detalles, no como un arreglo. 

Tampoco podemos confiar en que `org_id` venga como un claim directo a menos que se configure explícitamente, por lo que el `orgId` se lee desde los roles.

**Archivo a modificar:** `cloud/pkg/auth/claims.go`

```go
package auth

import "github.com/golang-jwt/jwt/v5"

type CustomClaims struct {
    jwt.RegisteredClaims
    // El mapa donde la clave es el RoleName (ej: "admin", "viewer")
    Roles ZitadelRoles `json:"urn:zitadel:iam:org:project:roles"`
}

type ZitadelRoles map[string]RoleClaim

type RoleClaim struct {
    OrgID          string `json:"orgId"`
    ProjectGrantID string `json:"projectGrantId,omitempty"`
    IsGranted      bool   `json:"isGranted,omitempty"`
}

// Extrae el OrgID del usuario basándose en los roles (asumiendo que pertenece a 1 org principal en este token)
func (c *CustomClaims) GetPrimaryOrgID() string {
    for _, role := range c.Roles {
        if role.OrgID != "" {
            return role.OrgID
        }
    }
    return ""
}
```

---

## 2. Inyección Segura en el Contexto

Centralizaremos las claves de contexto y la inyección en un solo lugar para evitar duplicidades entre middlewares.

**Archivo a modificar/crear:** `cloud/pkg/auth/context.go`

```go
package auth

import "context"

type contextKey string

const ClaimsKey contextKey = "claims"
const UserKey   contextKey = "user" // Para el usuario de BD

func ClaimsFromContext(ctx context.Context) *CustomClaims {
    claims, _ := ctx.Value(ClaimsKey).(*CustomClaims)
    return claims
}
```

---

## 3. Validación Multi-tenant Estricta

Para un SaaS, validar que el usuario pertenece al Tenant (Organización) correcto es la pieza central. No se deja como un "TODO", se implementa como middleware explícito.

**Archivo a crear:** `cloud/internal/interfaces/rest/tenant.go`

```go
package rest

import (
    "net/http"
    "github.com/go-chi/chi/v5"
    "github.com/rs/zerolog"
    "github.com/tablehub/cloud/pkg/auth"
)

func RequireOrgAccess() func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            claims := auth.ClaimsFromContext(r.Context())
            if claims == nil {
                http.Error(w, "unauthorized", http.StatusUnauthorized)
                return
            }

            // El OrgID al que se intenta acceder (por URL o Header)
            requestedOrgID := chi.URLParam(r, "orgID")
            if requestedOrgID == "" {
                requestedOrgID = r.Header.Get("X-Org-ID")
            }

            // Validar que el token del usuario contenga roles para ESA organización
            userOrgID := claims.GetPrimaryOrgID()
            if requestedOrgID == "" || userOrgID != requestedOrgID {
                zerolog.Ctx(r.Context()).Warn().
                    Str("user_id", claims.Subject).
                    Str("requested_org", requestedOrgID).
                    Str("token_org", userOrgID).
                    Msg("cross-org access denied")

                http.Error(w, "forbidden: cross-org access denied", http.StatusForbidden)
                return
            }
            next.ServeHTTP(w, r)
        })
    }
}
```

---

## 4. Middleware de Permisos (Acciones vs Roles)

Un middleware que solo verifica strings de roles es frágil. Lo correcto es mapear roles a permisos granulares, facilitando la escalabilidad del sistema.

**Archivo a modificar:** `cloud/pkg/auth/permissions.go`

```go
package auth

var rolePermissions = map[string][]string{
    "admin":           {"hub:create", "hub:delete", "hub:update", "hub:list"},
    "hub_operator":    {"hub:list", "hub:get"},
}

func HasPermission(roles ZitadelRoles, action string) bool {
    for roleName := range roles {
        perms, ok := rolePermissions[roleName]
        if ok {
            for _, p := range perms {
                if p == action {
                    return true
                }
            }
        }
    }
    return false
}
```

**Middleware de Autorización (`cloud/internal/interfaces/rest/rbac.go`):**

```go
func RequirePermission(action string) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            claims := auth.ClaimsFromContext(r.Context())

            if claims == nil || !auth.HasPermission(claims.Roles, action) {
                zerolog.Ctx(r.Context()).Warn().
                    Str("user_id", claims.Subject).
                    Str("attempted_action", action).
                    Str("path", r.URL.Path).
                    Msg("authorization denied: missing permission")

                http.Error(w, "forbidden", http.StatusForbidden)
                return
            }
            next.ServeHTTP(w, r)
        })
    }
}
```

### Composición en el Router (`router.go`):
El orden de los middlewares en Chi es vital:
```go
r.Route("/v1/orgs/{orgID}", func(r chi.Router) {
    r.Use(rest.RequireAuth)          // Valida Token y Firma
    r.Use(rest.RequireOrgAccess())   // Valida Multi-tenancy (OrgID)

    r.Route("/hubs", func(r chi.Router) {
        r.Use(rest.RequirePermission("hub:list"))
        r.Get("/", hubHandler.List)

        r.With(rest.RequirePermission("hub:create")).Post("/", hubHandler.Create)
    })
})
```

---

## 5. Validación Estricta de Tokens (`jwks.go`)

Actualizamos `ValidateToken` para parsear en nuestra estructura `CustomClaims`:

```go
func (a *Authenticator) ValidateToken(tokenStr string) (*CustomClaims, error) {
    token, err := jwt.ParseWithClaims(tokenStr, &CustomClaims{}, a.jwks.Keyfunc)
    if err != nil {
        return nil, fmt.Errorf("invalid token: %w", err)
    }

    claims, ok := token.Claims.(*CustomClaims)
    if !ok || !token.Valid {
        return nil, errors.New("invalid token claims")
    }

    return claims, nil
}
```

---

## Estrategia de Testing (Casos Límite)

1. **Estructura Real (`claims_test.go`):** Comprobar que el unmarshal funciona con un JWT real de Zitadel:
   ```go
   raw := `{"urn:zitadel:iam:org:project:roles":{"admin":{"orgId":"123"}}}`
   // Assert len == 1, assert OrgID == "123".
   ```
2. **Cross-Org Access (`tenant_test.go`):** Validar que un token con roles de la Org A reciba un `HTTP 403` al intentar acceder a la ruta `/v1/orgs/B/hubs`.

---
## Resumen Ejecutivo de Acciones
- [ ] Cambiar a `jwt.ParseWithClaims` usando `CustomClaims`.
- [ ] Modelar el claim de roles de Zitadel como un Mapa (`ZitadelRoles`), no como Slice.
- [ ] Extraer el Tenant desde el mapa de roles (OrgID).
- [ ] Implementar middleware `RequireOrgAccess` explícito.
- [ ] Mapear roles a acciones granulares e implementar `RequirePermission`.
- [ ] Unificar el logger usando `zerolog` para seguridad auditada.

---

## 7. Omisiones y Mejoras Detectadas (Revisión de Plan)

Durante la revisión técnica del plan, se identificaron las siguientes áreas que deben ser abordadas:

### Faltas Críticas
1. **Migración de datos existentes:** Definir la estrategia para migrar usuarios actuales al modelo RBAC/Zitadel, considerando el impacto en el middleware que actualmente inyecta `user.User` en el contexto.
2. **Acceso a recursos específicos (granularidad):** El middleware `RequireOrgAccess` valida pertenencia al Tenant (`orgID`), pero se debe asegurar que se valide también el acceso a recursos específicos (ej: verificar que el Hub solicitado efectivamente pertenece a la Org del usuario).
3. **Logging de auditoría:** Integrar registros de auditoría estructurados con `zerolog` para operaciones sensibles (quién, cuándo, qué acción, sobre qué tenant/recurso).

### Mejoras Recomendadas
4. **Soporte Multi-Org:** La función `GetPrimaryOrgID()` retorna el primer org de los roles del token. Debe considerarse el caso donde un usuario pertenezca a múltiples organizaciones y el token traiga roles para más de una.
5. **Ciclo de vida del token & Refresh:** Detallar el flujo de renovación del token de sesión (`refresh token`) en el frontend (Svelte) cuando expire.
6. **Códigos de error estandarizados:** Retornar códigos de error específicos (ej: `TENANT_MISMATCH`, `PERMISSION_DENIED`) en las respuestas JSON del backend para que el frontend los maneje adecuadamente.
7. **Resolución de dependencias circulares:** Aclarar si el middleware `RequireAuth` seguirá dependiendo de `user.Repository` para cargar el usuario desde la base de datos o si se desacoplará completamente usando la información del token JWT.
8. **Estrategia de pruebas ampliada:** Añadir casos de prueba para:
   - Tokens con roles vacíos o nulos.
   - Formatos maliciosos o malformados de `orgID`.
   - Usuarios con múltiples roles asignados simultáneamente.
   - Verificación de jerarquías o herencia de permisos.
9. **Catálogo de roles:** Definir formalmente las capacidades y límites de cada rol (ej: `hub_operator`, `viewer`, `admin`).
10. **Rate Limiting por Tenant:** Adaptar el control de peticiones (`throttle`) actual para que sea por tenant (`orgID`) en lugar de global, previniendo la denegación de servicio entre tenants.

