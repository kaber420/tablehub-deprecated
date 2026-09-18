# Plan de Implementación: Cloud SaaS — Hito 1 (Túneles y Gestión de Flotas)

> **Metodología**: Claude Opus orquesta y escribe código. DeepSeek y Mimo (vía OpenCode CLI) ejecutan tests, revisan código, e investigan errores.
> **Estimación total**: ~12 días de desarrollo
> **Fecha de inicio**: 2026-07-10

---

## 🎭 Flujo de Trabajo Multi-Agente

```
┌───────────────────────────────────────────────────────┐
│  1. Claude Opus: Escribe código de la fase            │
│                  ↓                                    │
│  2. DeepSeek: go build + go vet + go test -race       │
│               Reporta errores (archivo + línea)       │
│                  ↓                                    │
│  3. Claude Opus: Corrige errores reportados           │
│                  ↓                                    │
│  4. Mimo: Revisa arquitectura y calidad               │
│           Solo sugiere, NO modifica                   │
│                  ↓                                    │
│  5. Claude Opus: Aplica mejoras válidas               │
│                  ↓                                    │
│  ✅ Fase completada → Siguiente fase                  │
└───────────────────────────────────────────────────────┘
```

### Comandos de Delegación
```bash
# Tests y revisión rápida
opencode run -m opencode/deepseek-v4-flash-free "TAREA: ..."

# Investigación profunda
opencode run -m opencode/mimo-v2.5-free "TAREA: ..."

# Análisis con agente plan
opencode run --agent plan -m opencode/mimo-v2.5-free "TAREA: ..."
```

---

## ✅ Fase 1: Scaffolding y Configuración — COMPLETADA

**Duración real**: ~30 minutos

### Archivos creados
| Archivo | Descripción |
|---|---|
| `cloud/cmd/server/main.go` | Entry point con chi router, health check, graceful shutdown |
| `cloud/internal/config/config.go` | Configuración vía env vars (PORT, DATABASE_URL, ENV, LOG_LEVEL) |
| `cloud/internal/infrastructure/observability/logger.go` | zerolog con modo dev (colores) / prod (JSON) |
| `cloud/pkg/crypto/ed25519.go` | Utilidades Ed25519 compartibles (GenerateChallenge, VerifySignature) |
| `cloud/pkg/types/id.go` | Tipos UUID (RestaurantID, HubID) |
| `cloud/Makefile` | Targets: build, run, test, vet, db-up, db-down, dev |
| `cloud/docker-compose.yml` | PostgreSQL 16 Alpine para desarrollo |
| `cloud/go.mod` | Dependencias: chi, pgx, zerolog, gorilla/websocket, uuid |

### Verificación
- [x] `go build ./cmd/server/` — Compila sin errores
- [x] `go mod tidy` — Dependencias resueltas
- [x] Estructura Clean Architecture creada (domain, application, infrastructure, interfaces)
- [x] `main_legacy.go` preservado como referencia del mock original
- [/] DeepSeek: Revisión de código (en progreso)

---

## 📋 Fase 2: Base de Datos (PostgreSQL + Migraciones)
**Estimación**: ~2 días

### Claude Opus escribe:
- [ ] `cloud/internal/infrastructure/db/postgres.go` — Pool de conexiones `pgxpool`
- [ ] `cloud/internal/infrastructure/db/migrations/001_init.sql` — Tablas:
  ```sql
  restaurants (id UUID PK, slug UNIQUE, plan, settings JSONB, created_at)
  hub_registry (restaurant_id FK, hub_id, public_key, connection_status, last_seen, metadata JSONB)
  ```
- [ ] `cloud/internal/domain/restaurant/restaurant.go` — Entidad Restaurant
- [ ] `cloud/internal/domain/restaurant/errors.go` — ErrNotFound, ErrDuplicateSlug
- [ ] `cloud/internal/domain/hub/hub.go` — Entidad Hub
- [ ] `cloud/internal/domain/hub/errors.go` — ErrHubNotFound, ErrHubOffline
- [ ] `cloud/internal/infrastructure/db/restaurant_repo.go` — CRUD (Create, GetByID, GetBySlug, List, Update)
- [ ] `cloud/internal/infrastructure/db/hub_repo.go` — Register, UpdateStatus, GetOnline, GetByRestaurant

### DeepSeek verifica:
```bash
opencode run -m opencode/deepseek-v4-flash-free "
En cloud/ del proyecto tablehub:
1. Ejecuta 'make db-up' para levantar PostgreSQL
2. Aplica las migraciones SQL contra la DB
3. Ejecuta 'go test ./internal/infrastructure/db/...' -v
4. Verifica CRUD de restaurants y hubs
5. Reporta errores SQL, conexión, o tests fallidos
"
```

### Mimo revisa:
```bash
opencode run -m opencode/mimo-v2.5-free "
Revisa migrations/001_init.sql en cloud/:
1. ¿Schema multi-tenant correcto?
2. ¿Faltan índices importantes?
3. ¿Tipos de datos óptimos?
Solo sugiere, NO modifiques.
"
```

### Criterio de éxito
- PostgreSQL corriendo en Docker
- Migraciones ejecutadas sin error
- CRUD de restaurants y hubs pasa tests

---

## 📋 Fase 3: WebSocket Manager (Sharded Registry + Auth Ed25519)
**Estimación**: ~3 días

### Claude Opus escribe:
- [ ] `cloud/internal/infrastructure/ws/hub_conn.go` — Wrapper de conexión:
  - `writePump()` — Goroutine dedicada con canal bufferizado
  - `SendMessage(msg)` — Non-blocking send con overflow protection
  - Métricas por conexión (MessagesSent, BytesTransferred, etc.)
- [ ] `cloud/internal/infrastructure/ws/hub_registry.go` — Sharded map (256 shards):
  - `Store(restaurantID, conn)`, `Get(restaurantID)`, `Delete(restaurantID)`
  - `Count()`, `ForEach(fn)` para admin
- [ ] `cloud/internal/infrastructure/ws/auth.go` — Challenge-response Ed25519:
  - Migrado de `main_legacy.go`
  - Verificación contra DB (public_key registrada)
- [ ] `cloud/internal/infrastructure/ws/heartbeat.go` — Ping/Pong:
  - Jitter por hub (evita picos de tráfico)
  - Timeout configurable (default 90s sin pong → desconexión)
- [ ] `cloud/internal/infrastructure/ws/handler.go` — HTTP upgrade handler:
  - Integración con chi router
  - Logging estructurado por conexión

### DeepSeek verifica:
```bash
opencode run -m opencode/deepseek-v4-flash-free "
En cloud/ del proyecto tablehub:
1. Ejecuta 'go build ./...'
2. Ejecuta 'go test ./internal/infrastructure/ws/...' -v -race
3. Verifica CERO data races con el flag -race
4. Revisa hub_registry.go: ¿thread-safe el sharding con 256 shards?
5. Revisa auth.go: ¿flujo Ed25519 idéntico al main_legacy.go?
6. Reporta TODOS los errores con archivo + línea
"
```

### Criterio de éxito
- Registry pasa tests de concurrencia (100 goroutines simultáneas)
- Auth Ed25519 funciona idéntico al mock original
- Zero data races con `-race`

---

## 📋 Fase 4: Aseguramiento de Conectividad (Hotfixes)
**Estimación**: ~1 día

### Claude Opus escribe:
- [ ] `cloud/internal/infrastructure/ws/types.go` — Ajustar `AuthResponsePayload` para que coincida con el firmware (`public_key` en lugar de `hub_id`).
- [ ] `cloud/internal/infrastructure/ws/hub_conn.go` — Modificar `writePump` para asegurar que envía 1 mensaje JSON por frame WebSocket sin coalescing (`\n`).
- [ ] `cloud/internal/infrastructure/ws/heartbeat.go` — Sincronizar el Heartbeat. Evitar el ping nativo de WS si el firmware no lo soporta, o forzar al firmware a usar el ping de aplicación (`event: "ping"`).

### DeepSeek verifica:
```bash
opencode run -m opencode/deepseek-v4-flash-free "
Verifica que las asimetrías de conectividad fueron resueltas en cloud/
1. ¿El Payload de Auth coincide con el Hub?
2. ¿El coalescing de mensajes fue removido del writePump?
3. ¿El heartbeat está sincronizado?
"
```

### Criterio de éxito
- El parser JSON del Hub físico no se rompe por múltiples tramas unidas.
- La autenticación no falla por asimetría de llaves.

---

## 📋 Fase 5: Casos de Uso del Túnel (Connect/Disconnect/Forward)
**Estimación**: ~2 días

### Claude Opus escribe:
- [ ] `cloud/internal/application/tunnel/connect_hub.go` — ConnectHubUseCase:
  - Valida auth → Busca en DB → Registry.Store → Actualiza status en DB → Log
- [ ] `cloud/internal/application/tunnel/disconnect_hub.go` — DisconnectHubUseCase:
  - Registry.Delete → DB status='offline' → Log de desconexión
- [ ] `cloud/internal/application/tunnel/forward_message.go` — ForwardMessageUseCase:
  - Lookup restaurantID → Obtiene conexión → SendMessage → Log
- [ ] `cloud/internal/interfaces/rest/router.go` — Router chi con todas las rutas:
  - `GET /health` — Health check
  - `GET /v1/hubs` — Lista de hubs (con filtros)
  - `GET /v1/hubs/{id}` — Hub específico
  - `WS /v1/hub/connect` — WebSocket upgrade para túneles
- [ ] `cloud/internal/interfaces/rest/hub_handler.go` — Handlers REST para hubs

### DeepSeek verifica:
```bash
opencode run -m opencode/deepseek-v4-flash-free "
En cloud/ del proyecto tablehub:
1. Ejecuta 'go test ./internal/application/tunnel/...' -v
2. Ejecuta 'go test ./internal/interfaces/rest/...' -v
3. Levanta el servidor y prueba:
   - GET http://localhost:9000/health
   - GET http://localhost:9000/v1/hubs
4. Reporta errores o respuestas inesperadas
"
```

### Criterio de éxito
- API REST responde correctamente
- Flujo connect→forward→disconnect funciona end-to-end

---

## 📋 Fase 6: Sistema de Webhooks (Refactor TastyIgniter)
**Estimación**: ~2 días

### Claude Opus escribe:
- [ ] `cloud/internal/domain/webhook/webhook.go` — Tipos:
  - `WebhookConfig` (provider, secret, restaurantID)
  - `WebhookEvent` (event_type, payload, received_at)
- [ ] `cloud/internal/application/webhook/receive_webhook.go` — ReceiveWebhookUseCase:
  - Validar token contra DB (no hardcodeado)
  - Parsear payload según provider
  - Log en DB
- [ ] `cloud/internal/application/webhook/dispatch_webhook.go` — DispatchWebhookUseCase:
  - Buscar túnel activo → Forward al hub
  - Retry (2 intentos, 1s entre cada uno)
  - Si hub offline → Encolar (Fase futura: Offline Queue)
- [ ] `cloud/internal/infrastructure/webhook/tastyigniter.go` — Parser específico TastyIgniter
- [ ] `cloud/internal/interfaces/rest/webhook_handler.go` — Endpoint genérico:
  - `POST /v1/webhooks/{provider}/{restaurant_id}`

### DeepSeek verifica:
```bash
opencode run -m opencode/deepseek-v4-flash-free "
En cloud/ del proyecto tablehub:
1. Ejecuta 'go test ./internal/application/webhook/...' -v
2. Compara el comportamiento con main_legacy.go:
   - ¿El endpoint POST /v1/webhooks/tastyigniter/{id} funciona igual?
   - ¿La validación de token es correcta?
3. Reporta diferencias o errores
"
```

### Criterio de éxito
- Webhook TastyIgniter funciona idéntico al mock original
- Tokens validados contra DB (no hardcodeados)
- Logs de webhooks persistidos en PostgreSQL

---

## 📋 Fase 7: Observabilidad + Graceful Shutdown
**Estimación**: ~1 día

### Claude Opus escribe:
- [ ] Mejorar `cmd/server/main.go`:
  - Drain de conexiones WebSocket antes de cerrar
  - Logging de métricas al shutdown
- [ ] `cloud/internal/interfaces/rest/admin_handler.go`:
  - `GET /v1/admin/stats` — JSON con:
    ```json
    {
      "hubs_online": 42,
      "hubs_total": 150,
      "webhooks_processed_24h": 1234,
      "uptime_seconds": 86400
    }
    ```
  - `GET /v1/admin/tunnels` — Lista de conexiones activas (debug)

### DeepSeek verifica:
```bash
opencode run -m opencode/deepseek-v4-flash-free "
En cloud/ del proyecto tablehub:
1. Levanta el servidor con 'go run ./cmd/server/'
2. Verifica logs JSON estructurado
3. Envía SIGTERM y verifica:
   - ¿Cierra conexiones gracefully?
   - ¿Termina en <30s?
4. Prueba GET /v1/admin/stats
5. Reporta problemas
"
```

### Criterio de éxito
- Shutdown graceful en <30 segundos
- Logs JSON con `restaurant_id` contextualizados
- Stats endpoint funcional

---

## 📋 Fase 8: Integración Final + Test End-to-End
**Estimación**: ~1 día

### DeepSeek ejecuta E2E:
```bash
opencode run -m opencode/deepseek-v4-flash-free "
PRUEBA END-TO-END del cloud de tablehub:
1. Levanta PostgreSQL con docker-compose
2. Ejecuta migraciones
3. Levanta cloud server en :9000
4. Ejecuta 'go test ./...' -v -race -count=1
5. Prueba flujo completo:
   a. Registra restaurante vía API
   b. Conecta hub simulado vía WebSocket
   c. Verifica auth Ed25519
   d. Envía webhook TastyIgniter
   e. Verifica que llegó al hub vía túnel
   f. Desconecta hub
   g. Verifica cleanup en DB
6. Reporta resultado de CADA paso
"
```

### Mimo hace revisión final:
```bash
opencode run --agent plan -m opencode/mimo-v2.5-free "
Revisa TODO el código en cloud/ del proyecto tablehub:
1. ¿Clean Architecture bien implementada?
2. ¿Dependencias invertidas (domain importando infrastructure)?
3. ¿Interfaces/ports bien definidos?
4. ¿Code smells o anti-patterns?
5. ¿Tests cubren edge cases (auth fallida, hub offline, DB caída)?
6. Score de calidad del 1 al 10 con justificación
7. Las 3 mejoras más importantes pendientes
NO modifiques ningún archivo. Solo reporta.
"
```

### Criterio de éxito
- Todos los tests pasan con `-race`
- Flujo E2E completo funciona
- Score de calidad ≥ 7/10
- Zero data races

---

## 📊 Resumen de Progreso

| Fase | Descripción | Estado | Estimación |
|---|---|---|---|
| 1 | Scaffolding + Config | ✅ **COMPLETADA** | ~30 min |
| 2 | PostgreSQL + Migraciones | ✅ **COMPLETADA** | ~2 días |
| 3 | WebSocket Manager | ✅ **COMPLETADA** | ~3 días |
| 4 | Conectividad (Hotfixes) | ⬜ Pendiente | ~1 día |
| 5 | Tunnel Use Cases | ⬜ Pendiente | ~2 días |
| 6 | Webhooks Refactor | ⬜ Pendiente | ~2 días |
| 7 | Observabilidad | ⬜ Pendiente | ~1 día |
| 8 | Integración Final | ⬜ Pendiente | ~1 día |
| | **TOTAL** | **3/8 fases** | **~12 días** |

---

## 📁 Mapa Completo de Archivos

### Ya Creados (Fase 1) ✅
```
cloud/
├── cmd/server/main.go                              ✅
├── internal/
│   ├── config/config.go                            ✅
│   └── infrastructure/observability/logger.go      ✅
├── pkg/
│   ├── crypto/ed25519.go                           ✅
│   └── types/id.go                                 ✅
├── main_legacy.go                                  ✅ (referencia)
├── go.mod                                          ✅
├── go.sum                                          ✅
├── Makefile                                        ✅
└── docker-compose.yml                              ✅
```

### Por Crear (Fases 2-7)
```
cloud/
├── internal/
│   ├── domain/
│   │   ├── hub/hub.go                              📋 Fase 2
│   │   ├── hub/errors.go                           📋 Fase 2
│   │   ├── restaurant/restaurant.go                📋 Fase 2
│   │   ├── restaurant/errors.go                    📋 Fase 2
│   │   └── webhook/webhook.go                      📋 Fase 5
│   ├── application/
│   │   ├── tunnel/connect_hub.go                   📋 Fase 4
│   │   ├── tunnel/disconnect_hub.go                📋 Fase 4
│   │   ├── tunnel/forward_message.go               📋 Fase 4
│   │   ├── webhook/receive_webhook.go              📋 Fase 5
│   │   └── webhook/dispatch_webhook.go             📋 Fase 5
│   ├── infrastructure/
│   │   ├── db/postgres.go                          📋 Fase 2
│   │   ├── db/migrations/001_init.sql              📋 Fase 2
│   │   ├── db/restaurant_repo.go                   📋 Fase 2
│   │   ├── db/hub_repo.go                          📋 Fase 2
│   │   ├── ws/hub_conn.go                          📋 Fase 3
│   │   ├── ws/hub_registry.go                      📋 Fase 3
│   │   ├── ws/auth.go                              📋 Fase 3
│   │   ├── ws/heartbeat.go                         📋 Fase 3
│   │   ├── ws/handler.go                           📋 Fase 3
│   │   └── webhook/tastyigniter.go                 📋 Fase 5
│   └── interfaces/
│       ├── rest/router.go                          📋 Fase 4
│       ├── rest/hub_handler.go                     📋 Fase 4
│       ├── rest/webhook_handler.go                 📋 Fase 5
│       └── rest/admin_handler.go                   📋 Fase 6
```

---

## 🔗 Documentos Relacionados
- [Roadmap General](file:///home/kaber420/Documentos/proyectos/tablehub/drafts/ROADMAP.md)
- [Roadmap Cloud](file:///home/kaber420/Documentos/proyectos/tablehub/cloud/ROADMAP.md)
- [Guía de Desarrollo Cloud (Investigación IA)](file:///home/kaber420/.gemini/antigravity-ide/brain/ac7f4838-6dbc-42a5-b837-fb8d25e38241/cloud_development_guide.md)
- [Arquitectura de Dispositivos](file:///home/kaber420/Documentos/proyectos/tablehub/drafts/devices_architecture_plan.md)
- [Recomendaciones de Seguridad](file:///home/kaber420/Documentos/proyectos/tablehub/drafts/security_recommendations.md)
