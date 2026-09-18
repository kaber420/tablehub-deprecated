# Plan de Implementación: Fase 6 — Sistema de Webhooks (Refactor TastyIgniter)

## Contexto

El servidor cloud de Tablehub actúa como intermediario entre plataformas de pedidos online (TastyIgniter, UberEats, etc.) y los Hubs físicos en los restaurantes. Actualmente el flujo de webhooks solo existe en `main_legacy.go` como un prototipo hardcodeado. Esta fase lo refactoriza a una arquitectura limpia, extensible, y con validación contra la base de datos.

### Comportamiento Legacy (`main_legacy.go`)
1. `POST /v1/webhooks/tastyigniter/{restaurant_id}` recibe un webhook.
2. Valida el token `X-TastyIgniter-Token` contra un valor hardcodeado (`secret_tasty_token_123`).
3. Busca el túnel activo por `restaurant_id` en `sync.Map`.
4. Envía el payload como `TunnelMessage{event: "order.created"}` al hub vía WebSocket.
5. Responde `{"status":"delivered"}` o error.

### Qué cambia en esta fase
- **Tokens validados contra DB** (`restaurant.Settings.WebhookToken`) — ya NO hardcodeados.
- **Provider pattern extensible** — Parsers desacoplados por proveedor (TastyIgniter hoy, UberEats mañana).
- **Retry con backoff** — 2 intentos con 1s entre cada uno antes de reportar fallo.
- **Logging en DB** — Persistencia de eventos webhook para auditoría.
- **Event type dinámico** — El tipo de evento se extrae del payload del proveedor, no se fuerza `order.created`.

---

## Proposed Changes

### 1. Migración SQL

#### [NEW] `cloud/internal/infrastructure/db/migrations/002_webhook_events.sql`
- Tabla `webhook_events` para auditoría y logging:
  ```sql
  CREATE TABLE webhook_events (
      id           UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
      restaurant_id UUID NOT NULL REFERENCES restaurants(id) ON DELETE CASCADE,
      provider     TEXT NOT NULL,          -- 'tastyigniter', 'ubereats', etc.
      event_type   TEXT NOT NULL,          -- 'order.created', 'order.updated', etc.
      payload      JSONB NOT NULL,         -- raw payload del webhook
      status       TEXT NOT NULL DEFAULT 'received'
                   CHECK (status IN ('received', 'delivered', 'failed', 'hub_offline')),
      error_detail TEXT,                   -- detalle del error si falló
      received_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
      delivered_at TIMESTAMPTZ            -- NULL hasta que el hub confirme recepción
  );
  -- Índices
  CREATE INDEX idx_webhook_events_restaurant ON webhook_events(restaurant_id);
  CREATE INDEX idx_webhook_events_status ON webhook_events(status);
  CREATE INDEX idx_webhook_events_received ON webhook_events(received_at);
  ```

---

### 2. Capa de Dominio (`cloud/internal/domain/webhook/`)

#### [NEW] `webhook.go`
- **`WebhookConfig`** — Configuración de webhook por restaurante:
  - `Provider string` (e.g. "tastyigniter")
  - `Secret string` (token de autenticación)
  - `RestaurantID types.RestaurantID`
  
- **`WebhookEvent`** — Registro de un evento webhook:
  - `ID`, `RestaurantID`, `Provider`, `EventType`, `Payload json.RawMessage`
  - `Status` (`received`, `delivered`, `failed`, `hub_offline`)
  - `ErrorDetail`, `ReceivedAt`, `DeliveredAt`

- **`EventStatus`** — Enum: `StatusReceived`, `StatusDelivered`, `StatusFailed`, `StatusHubOffline`

#### [NEW] `errors.go`
- `ErrInvalidToken` — Token de webhook no coincide con DB
- `ErrUnsupportedProvider` — Proveedor no registrado
- `ErrEventNotFound` — Evento no encontrado

#### [NEW] `repository.go`
- **`EventRepository`** interface:
  - `LogEvent(ctx, event *WebhookEvent) error`
  - `UpdateEventStatus(ctx, eventID, status, errorDetail) error`
  - `GetEventsByRestaurant(ctx, restaurantID, limit, offset) ([]*WebhookEvent, error)`

---

### 3. Capa de Infraestructura

#### [NEW] `cloud/internal/infrastructure/db/webhook_event_repo.go`
- Implementación PostgreSQL de `webhook.EventRepository`.
- `LogEvent`: INSERT con RETURNING id.
- `UpdateEventStatus`: UPDATE con timestamp de delivery.

#### [NEW] `cloud/internal/infrastructure/webhook/provider.go`
- **`Provider`** interface:
  ```go
  type Provider interface {
      Name() string
      ValidateRequest(r *http.Request, secret string) error
      ParsePayload(r *http.Request) (eventType string, payload json.RawMessage, err error)
  }
  ```
- **`Registry`** struct — Mapa de proveedores registrados:
  - `Register(provider Provider)`
  - `Get(name string) (Provider, error)`

#### [NEW] `cloud/internal/infrastructure/webhook/tastyigniter.go`
- Implementa `Provider` para TastyIgniter:
  - `Name()` → `"tastyigniter"`
  - `ValidateRequest()` — Extrae token de `X-TastyIgniter-Token` header o query param `token`. Compara con secret.
  - `ParsePayload()` — Decodifica JSON body. Extrae `event_type` del payload o usa fallback `"order.created"`.

---

### 4. Capa de Aplicación (`cloud/internal/application/webhook/`)

#### [NEW] `receive_webhook.go`
- **`ReceiveWebhookUseCase`** — Orquesta la recepción:
  1. Busca el `Restaurant` en DB por ID → obtiene `Settings.WebhookToken`.
  2. Valida el token vía el `Provider` correspondiente.
  3. Parsea el payload usando el `Provider`.
  4. Persiste el evento en DB con estado `received`.
  5. Retorna el `WebhookEvent` creado.
  
- Dependencias inyectadas:
  - `restaurant.Repository`
  - `webhook.EventRepository`
  - `webhookinfra.Registry` (Provider Registry)

#### [NEW] `dispatch_webhook.go`
- **`DispatchWebhookUseCase`** — Orquesta el envío al Hub:
  1. Busca el túnel activo por `restaurantID` en el `ConnectionRegistry`.
  2. Construye un `TunnelMessage` con el `event_type` y `payload`.
  3. Envía al Hub vía `Sender.Send()`.
  4. **Retry**: Si falla, espera 1s y reintenta (máximo 2 intentos).
  5. Actualiza el estado del evento en DB (`delivered` o `failed`).
  6. Si el hub está offline → estado `hub_offline`.
  
- Dependencias inyectadas:
  - `tunnel.ConnectionRegistry`
  - `webhook.EventRepository`

---

### 5. Capa de Interfaces

#### [NEW] `cloud/internal/interfaces/rest/webhook_handler.go`
- **`WebhookHandler`** — HTTP handler genérico:
  - Endpoint: `POST /v1/webhooks/{provider}/{restaurant_id}`
  - Extrae `provider` y `restaurant_id` de los URL params.
  - Llama a `ReceiveWebhookUseCase.Execute()` → luego `DispatchWebhookUseCase.Execute()`.
  - Responde `200 {"status":"delivered"}` o código de error apropiado.

#### [MODIFY] `cloud/internal/interfaces/rest/router.go`
- Añadir ruta: `r.Post("/v1/webhooks/{provider}/{restaurant_id}", webhookHandler.HandleWebhook)`

#### [MODIFY] `cloud/cmd/server/main.go`
- Inicializar:
  - `webhookinfra.Registry` con `tastyigniter.Provider` registrado.
  - `db.WebhookEventRepo`
  - `ReceiveWebhookUseCase` y `DispatchWebhookUseCase`
  - `WebhookHandler`
- Pasar `WebhookHandler` al router.

---

## Diagrama de Flujo

```
POST /v1/webhooks/tastyigniter/abc-123
    │
    ▼
┌─────────────────────────────┐
│  WebhookHandler             │
│  Extrae: provider, restID   │
└──────────┬──────────────────┘
           │
           ▼
┌─────────────────────────────┐
│  ReceiveWebhookUseCase      │
│  1. Restaurant.GetByID()    │
│  2. Provider.Validate()     │
│  3. Provider.Parse()        │
│  4. EventRepo.LogEvent()    │
└──────────┬──────────────────┘
           │
           ▼
┌─────────────────────────────┐
│  DispatchWebhookUseCase     │
│  1. Registry.GetByRestID()  │
│  2. Sender.Send(msg)        │
│  3. Retry (1s, 2 intentos)  │
│  4. EventRepo.UpdateStatus()│
└──────────┬──────────────────┘
           │
           ▼
  Response: 200 {"status":"delivered"}
            503 "Hub offline"
            401 "Invalid token"
```

---

## Verification Plan

### Pruebas Unitarias
- `go test ./internal/domain/webhook/...` — Validación de tipos y enums.
- `go test ./internal/application/webhook/...` — Mocks de Repository, ConnectionRegistry, Provider.
  - Test: Token válido → delivered
  - Test: Token inválido → 401
  - Test: Hub offline → status hub_offline
  - Test: Retry → primer intento falla, segundo éxito
  - Test: Proveedor desconocido → 400

### Verificación End-to-End
1. Levantar servidor.
2. `POST /v1/webhooks/tastyigniter/{id}` con token válido e inválido.
3. Verificar que el hub recibe el mensaje vía túnel.
4. Verificar que `webhook_events` tiene el registro en DB.
5. Comparar comportamiento con `main_legacy.go`.

### Compatibilidad con Legacy
- El endpoint `POST /v1/webhooks/tastyigniter/{restaurant_id}` debe funcionar idéntico al legacy, con la diferencia de que el token se valida contra DB.

---

## Open Questions

> [!IMPORTANT]
> **¿Webhook Token por Proveedor?** El schema actual tiene un único `webhook_token` en `restaurant.Settings`. ¿Debería haber un token diferente por proveedor (e.g. un token para TastyIgniter, otro para UberEats)? Por ahora asumo un único token genérico, pero esto podría evolucionar.

> [!NOTE]
> **Offline Queue** — El plan original menciona "encolar si hub offline" como fase futura. En esta implementación, si el hub está offline, el evento se marca como `hub_offline` en la DB pero NO se encola. La cola offline es tema de una fase posterior.
