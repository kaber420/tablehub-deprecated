# Plan de Migración de Zitadel a Logto (Arquitectura Limpia)

Este documento define la estrategia para migrar el sistema de gestión de identidades y autenticación de **Zitadel** a **Logto**. Aprovechando la arquitectura limpia y hexagonal del proyecto, el objetivo es aislar la integración de Zitadel como código legacy e inyectar el nuevo adaptador de Logto sin romper las reglas de negocio.

---

## 1. Estrategia de Preservación de Código (Legacy)
No eliminaremos el código existente de Zitadel. En su lugar, lo moveremos o mantendremos como un adaptador opcional dentro de la capa de infraestructura.

* **Ubicación Legacy:** Mantener `cloud/internal/infrastructure/zitadel/` o moverlo a `cloud/internal/infrastructure/legacy/zitadel/`.
* **Ubicación Nueva:** Crear `cloud/internal/infrastructure/logto/` para el nuevo adaptador.
* **Inyección de Dependencias:** El punto de entrada (`cloud/cmd/server/main.go`) leerá `AUTH_PROVIDER` (valores: `logto` o `zitadel`) desde la configuración e instanciará el cliente correspondiente usando una interfaz común.

---

## 2. Rediseño de la Estructura de Tokens (Claims)

El middleware actual en `cloud/pkg/auth/jwks.go` espera una estructura de claims anidada específica de Zitadel (`ZitadelRoles`). Para Logto, actualizaremos la estructura `TokenInfo` para soportar claims planos basados en arrays de organizaciones y roles.

### Estructura de TokenInfo en `jwks.go`
```go
type TokenInfo struct {
	Sub               string   `json:"sub"`
	Email             string   `json:"email"`
	Name              string   `json:"name"`
	Organizations     []string `json:"organizations"`       // Lista de org IDs a las que pertenece
	OrganizationRoles []string `json:"organization_roles"` // Array con formato "org_id:role_name"
	IsSuperadmin      bool     `json:"superadmin"`          // Mapeado a claim personalizada
}
```

### Funciones de Validación de Roles (Go)
Actualizaremos las funciones del middleware para adaptarse a este formato plano:
```go
// HasRoleInOrg verifica si el token tiene el rol dado para una organización específica
func (t *TokenInfo) HasRoleInOrg(orgID, targetRole string) bool {
	if t.IsSuperadmin {
		return true
	}
	target := orgID + ":" + targetRole
	for _, role := range t.OrganizationRoles {
		if role == target {
			return true
		}
	}
	return false
}
```

---

## 3. Cambios Requeridos por Componente

### Backend (Go)

| Componente / Archivo | Cambio Sugerido |
| :--- | :--- |
| **`cloud/pkg/auth/jwks.go`** | Rediseñar `TokenInfo` para parsear los arrays `organizations` y `organization_roles` del JWT de Logto. |
| **`cloud/internal/config/config.go`** | Agregar variables de entorno para Logto (`LOGTO_ENDPOINT`, `LOGTO_APP_ID`, `LOGTO_APP_SECRET`, `AUTH_PROVIDER`). |
| **`cloud/internal/domain/user/user.go`** | Renombrar struct/campos: `ZitadelUserID` ➔ `LogtoUserID` (o un campo genérico `ProviderUserID`). |
| **`cloud/internal/infrastructure/db/`** | Actualizar consultas SQL para buscar por el nuevo identificador genérico en la base de datos. |
| **`cloud/internal/interfaces/rest/middleware.go`** | Adaptar el middleware de autorización para utilizar el nuevo formato plano del token de Logto. |

### Base de Datos (PostgreSQL)
* **Migración de Tabla:** Crear una migración que renombre la columna `zitadel_user_id` a `logto_user_id` en la tabla `users` (o renombrarla a `provider_user_id` para mayor abstracción).
* **Organizaciones:** El identificador `organization_id` devuelto por la API de Logto se guardará en la columna correspondiente de la base de datos de Tablehub, vinculando el ciclo de vida de negocio al de identidad.

### Frontend (Svelte)
* **Configuración del SDK:** Instalar `@logto/browser` o utilizar el cliente OIDC genérico configurando el endpoint OIDC de Logto.
* **Scopes Obligatorios:** Configurar el frontend para solicitar scopes de organización:
  ```javascript
  // cloud/web/src/lib/auth.js
  const config = {
    // ...
    scopes: ["openid", "profile", "email", "urn:logto:scope:organizations"]
  };
  ```

---

## 4. Infraestructura de Desarrollo (Docker Compose)
Reemplazaremos la instancia de Zitadel en desarrollo por un servicio de Logto ligero:

```yaml
version: '3.8'

services:
  postgres-logto:
    image: postgres:14-alpine
    container_name: logto-postgres
    ports:
      - "5435:5432" # Puerto alternativo para evitar colisiones
    environment:
      POSTGRES_USER: logto
      POSTGRES_PASSWORD: logto_password
      POSTGRES_DB: logto
    volumes:
      - logto_postgres_data:/var/lib/postgresql/data

  logto:
    image: ghcr.io/logto-io/logto:latest
    container_name: logto-server
    ports:
      - "3002:3002"
      - "3005:3005"
    environment:
      - PORT=3002
      - ADMIN_PORT=3005
      - DB_URL=postgresql://logto:logto_password@postgres-logto:5432/logto
      - ENDPOINT=http://localhost:3002
      - ADMIN_ENDPOINT=http://localhost:3005
    depends_on:
      - postgres-logto

volumes:
  logto_postgres_data:
```

