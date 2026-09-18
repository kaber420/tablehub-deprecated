# Rediseño de Arquitectura: Flujo de Onboarding B2B (Event-Driven Saga)

Tras una evaluación crítica profunda de nuestro borrador inicial, se determinó que dependía demasiado del frontend (esperando que el usuario no cerrara el navegador) y no protegía contra caídas graves o doble-clicks. 

Para construir un SaaS robusto nivel empresarial, implementaremos una **Arquitectura Híbrida de Saga Guiada por Eventos** (Event-Driven Saga).

## 1. El Core de la Solución (Desacoplamiento Absoluto)
El registro de identidad (Logto) y la pertenencia a una Organización (Workspace) son dominios separados. Un usuario puede existir en estado `profile_setup` sin pertenecer a nada, y la aplicación no debe bloquearse, sino mostrarle las opciones para crear o unirse a un entorno.

## 2. Nueva Máquina de Estados (Backend)
En `user.go`, expandimos los estados para soportar fallos asíncronos:

```go
type OnboardingState string

const (
    StateNew           OnboardingState = "new"           // Sin perfil local
    StateProfileSetup  OnboardingState = "profile_setup" // Perfil listo, sin org
    StateOrgPending    OnboardingState = "org_pending"   // Intención guardada, backend trabajando
    StateOrgCreating   OnboardingState = "org_creating"  // Llamada a Logto en curso
    StateOrgFailed     OnboardingState = "org_failed"    // Logto falló críticamente, requiere retry
    StateActive        OnboardingState = "active"        // 100% operativo
)
```

## 3. Resiliencia Nivel "Saga" (Sin Bloqueos)

Para que el usuario jamás quede atrapado ("parches pendejos"), introducimos una tabla de **Operaciones Pendientes** en PostgreSQL/SQLite. Si Logto se cae, el sistema no lanza error 500, sino que encola la tarea.

### Tabla `pending_operations`:
```sql
CREATE TABLE pending_operations (
    id UUID PRIMARY KEY,
    user_id UUID REFERENCES users(id),
    operation_type VARCHAR(50) NOT NULL, -- ej: 'create_org', 'assign_role'
    payload JSONB NOT NULL,
    status VARCHAR(20) DEFAULT 'pending', 
    attempts INT DEFAULT 0,
    next_retry_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW()
);
```

### 3. Ejecución Resiliente (Worker Event-Driven)
**DeepSeek Audit:** Usar alarmas en memoria (`time.AfterFunc`) es frágil si escalas tu backend a múltiples servidores (replicas), ya que los servidores no se coordinan entre sí y la memoria se pierde si hay un crash. 

Para lograr verdadera resiliencia B2B, usaremos un **Worker Event-Driven con PostgreSQL LISTEN/NOTIFY**:

1. **Trigger en PostgreSQL:** Cuando se inserta una fila en `pending_operations`, un trigger ejecuta `pg_notify('pending_operation', NEW.id)`.
2. **Worker escucha el canal:** El worker en Go usa `LISTEN pending_operation` y se despierta al instante cuando hay trabajo. Sin polling, sin consultar la tabla innecesariamente.
3. **Fallback de seguridad:** Si el worker se reinicia y perdió la conexión, un `LISTEN` reconecta automáticamente. Para eventos perdidos por crash, un heartbeat cada 60 segundos verifica tareas estancadas (solo como fallback, no como flujo principal).
4. **Idempotencia Real:** Cada intento contra Logto viajará con un `Idempotency-Key` único sacado del payload de la base de datos, evitando crear la misma organización dos veces si la red parpadea.
5. **Multi-Replica seguro:** `SELECT ... FOR UPDATE SKIP LOCKED` solo se usa como mecanismo de exclusión cuando múltiples workers reciben la misma notificación simultáneamente.

## 4. Endpoints Rediseñados e Idempotencia
- **`GET /v1/me`**: Retorna el `state` exacto del perfil (incluye `onboarding_state`).
- **`POST /v1/me/profile`**: Crea el perfil local base.
- **`POST /v1/onboarding`**: Inserta en `pending_operations` y retorna `202 Accepted` con `operation_id`. El worker processa en background.
- **`GET /v1/onboarding/{op_id}`**: Retorna el estado actual de una operación (para debugging, no para polling del frontend).
- **`POST /v1/auth/login`**: Login custom con Resource Owner Password Grant (reemplaza redirect a Logto).
- **`POST /v1/auth/register`**: Registro custom vía Management API de Logto.

## 5. Manejo Definitivo de Tokens
En el frontend, el proceso de autenticación solicitará tanto el `id_token` (para extraer Email/Nombre durante la creación del perfil) como el `access_token` (para la gestión de ramas y hubs). El validador en Go (`jwks.go`) será actualizado para no arrojar error si faltan claims de organización al leer el `id_token` inicial.

## 6. Login y Registro Custom (Reemplaza hosted UI de Logto)
Para eliminar la dependencia del hosted UI de Logto (que solo pide username + password sin email):

### Login (`POST /v1/auth/login`)
- Frontend: formularios de email + contraseña (no redirect a Logto)
- Backend: usa Resource Owner Password Grant (`grant_type=password`) contra Logto
- Retorna tokens OIDC (id_token + access_token)

### Registro (`POST /v1/auth/register`)
- Frontend: formularios de email + contraseña + nombre
- Backend: crea usuario vía Management API de Logto (`POST /api/users`)
- Auto-login después del registro

### Estados de Billing (Suspendido)
Agregar al `OnboardingState`:
```go
StateSuspended OnboardingState = "suspended" // No pago, hubs desactivados
```
- Cuando un usuario no paga, su `onboarding_state` cambia a `suspended`
- El middleware `RequireActive` verifica que el estado sea `active`
- Si está `suspended`, retorna 403 con mensaje "Tu cuenta está suspendida"
- Los hubs se desconectan automáticamente (el worker de tunnels verifica el estado)

## 7. Siguientes Pasos (Roadmap de Ejecución)
1. Ejecutar las migraciones SQL para crear `pending_operations` y agregar la columna `onboarding_state` a los usuarios.
2. Crear trigger en PostgreSQL para `pg_notify('pending_operation', NEW.id)` en `pending_operations`.
3. Codificar el `reconciliation_worker.go` con LISTEN/NOTIFY (sin polling).
4. Actualizar `onboarding.go` para encolar en `pending_operations` en vez de ejecutar síncrono.
5. Crear endpoint WebSocket `/v1/events/ws` para notificaciones al frontend.
6. Crear endpoints `POST /v1/auth/login` y `POST /v1/auth/register` para login custom.
7. Actualizar `GetMe` para retornar `onboarding_state` real.
8. Adaptar `Onboarding.svelte` para escuchar WebSocket y mostrar estados (`org_pending`, `org_failed`).
9. Agregar estado `suspended` para billing y middleware `RequireActive`.

---

## 7. Evolución y Actualizaciones Arquitectónicas (Aprobadas)

Tras un análisis profundo con Mimo y contrastar con el código actual, se definieron los siguientes ajustes al plan original:

### A. Granularidad de la Operación
La creación de empresa (`create_org`) será una **operación compuesta**. No se dividirán los pasos (crear org, asignar membresía, asignar owner) porque lógicamente son un flujo atómico. El worker reintentará la saga completa garantizando la idempotencia en Logto.

### B. Worker Secuencial Reutilizable
Se abstraerá el patrón de `SKIP LOCKED` que ya existe en `webhook_event_repo.go`. Sin embargo, a diferencia del webhook worker (que lanza múltiples *goroutines*), el worker de onboarding procesará las operaciones de forma **secuencial** para no saturar al proveedor de identidad (Logto) con picos de peticiones simultáneas.

### C. Estrategia de Migración de Estados (Sin Downtime)
Para pasar de los estados antiguos (`registered`/`pending`) a los nuevos (`new`, `profile_setup`, `org_pending`, etc.):
1. En la migración SQL, `organization_id` pasará a ser *nullable* (DROP NOT NULL) y se agregará la columna `onboarding_state` con valor por defecto `active` para los usuarios antiguos.
2. Se agregará un índice `UNIQUE (user_id)` condicionado a estados pendientes para evitar colisiones por doble click.

### D. Tiempo Real: WebSockets dedicados para el Frontend
Se descarta el uso de HTTP Polling para notificar al cliente. En su lugar:
- **No se reutilizará el canal WS de Hubs:** El endpoint `/ws` actual usa criptografía Ed25519 y es exclusivo para dispositivos físicos. Mezclar ahí clientes frontend representa un riesgo operativo.
- **Nuevo Canal Exclusivo:** Se creará un endpoint nuevo (ej. `/v1/events/ws`) autenticado por tokens JWT estándar. El frontend se conecta al WebSocket y recibe notificaciones instantáneas:
  - `org_created` → onboarding completado
  - `org_failed` → error en Logto, requiere retry
  - `operation_progress` → actualización del estado de la operación
- **Flujo event-driven completo:** PostgreSQL LISTEN/NOTIFY → Worker → WebSocket → Frontend. Sin polling en ninguna capa.
