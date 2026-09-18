# Phase 5: Casos de Uso del Túnel y Enrutamiento (Connect/Disconnect/Forward)

> [!TIP]
> **Actualización:** Este plan implementa la lógica de aplicación basándose en la Clean Architecture y puertos sugeridos por DeepSeek, apoyándose en la infraestructura construida en la Fase 4.

## Objetivo de la Fase
Coordinar la autenticación del WebSocket con la base de datos a través de Casos de Uso independientes, invirtiendo dependencias, y exponer las rutas de conexión y API REST.

## 1. Puertos de Aplicación (Clean Architecture)
- **[NEW]** `internal/application/tunnel/port_registry.go`
  - Interfaz abstracta para el registro de WebSockets: `HubRegistry { Store, Get, GetByRestaurantID, Delete }`.
- **[NEW]** `internal/application/tunnel/port_repository.go`
  - Interfaz abstracta (wrapper) para acceder a los datos del hub, desacoplando los Use Cases de la infraestructura física.

## 2. Casos de Uso (`internal/application/tunnel/`)
- **[NEW]** `connect_hub.go` (ConnectHubUseCase)
  - Orquesta: recibe `AuthResult` ya validado -> `Registry.Store` -> `HubRepo.UpdateStatus(online)` -> Actualiza Metadatos.
- **[NEW]** `disconnect_hub.go` (DisconnectHubUseCase)
  - Orquesta: `Registry.Delete` -> `HubRepo.UpdateStatus(offline)`.
- **[NEW]** `forward_message.go` (ForwardMessageUseCase)
  - Busca la conexión usando `Registry.GetByRestaurantID(restaurantID)` y envía el mensaje (`SendMessage`).

## 3. Capa de Interfaces (`internal/interfaces/`)
- **[MODIFY]** `internal/interfaces/ws/handler.go`
  - Inyectar `ConnectHubUseCase` y el `DBKeyProvider`.
  - El upgrader delega a `ConnectHubUseCase.Execute(...)` luego de autenticar.
- **[MODIFY]** `internal/interfaces/ws/hub_conn.go`
  - En lugar de usar un callback `OnDisconnect` problemático, propagar un `context.CancelFunc` o un canal que señalice a una goroutine superior para ejecutar el `DisconnectHubUseCase` y evitar *race conditions*.
- **[NEW]** `internal/interfaces/rest/router.go`
  - Setup de `go-chi/chi/v5` con middlewares de CORS (usando `AllowedOrigins` de config) y Rate Limit.
  - El upgrader WS en `/v1/hub/connect` permitirá `CheckOrigin = true`, ya que el Hub físico no envía origin, mientras que el resto de rutas administrativas usarán el allowlist.
- **[NEW]** `internal/interfaces/rest/hub_handler.go`
  - Rutas: `GET /health`, `GET /v1/hubs`, `GET /v1/hubs/{id}`.

## Plan de Verificación

### Pruebas Unitarias
- **[NEW]** `go test ./internal/application/tunnel/... -v` con Mocks de los puertos.
- **[NEW]** `go test ./internal/interfaces/rest/... -v` usando `httptest`.

### Pruebas de Integración y End-to-End
- `GET http://localhost:9000/health`
- Validación de que al desconectar el socket, el status en PostgreSQL cambia a `offline`.
