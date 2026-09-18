# Phase 4: Refactorización de Infraestructura, DB y Registro de WebSockets

> [!TIP]
> **Actualización:** Este plan integra los hallazgos de Mimo y DeepSeek para preparar la infraestructura base antes de implementar los Casos de Uso del Túnel. 

## Objetivo de la Fase
Preparar los cimientos del sistema resolviendo problemas de dependencias, corrigiendo timeouts del servidor, y adaptando el Registro de Hubs y Repositorios para soportar índices por restaurante y actualización de estados, sentando las bases para una Clean Architecture.

## 0. Pre-requisitos y Configuraciones Globales (`config.go` & `main.go`)
- **[MODIFY]** `config.go`
  - Añadir `AllowedOrigins []string` para configurar el CORS de las rutas administrativas.
  - Aumentar `MaxWSMessageSize int64` a `65536` para evitar bloqueos con payloads grandes (issue de provisioning).
- **[MODIFY]** `cmd/server/main.go`
  - Configurar `ReadTimeout: 0` y `WriteTimeout: 0` en el servidor HTTP para evitar que se cierren las conexiones WebSocket.
  - Implementar *Graceful Shutdown* en este orden: `HTTP server -> HubRegistry.Shutdown() -> pgxpool.Close()`.

## 1. Capa de Dominio e Infraestructura (Base de Datos)
- **[MODIFY]** `internal/domain/hub/hub.go`
  - Añadir `UpdateMetadata(id types.HubID, meta Metadata) error` a la interfaz del repositorio del Hub.
- **[MODIFY]** `internal/infrastructure/db/hub_repo.go`
  - Implementar el método `UpdateMetadata` para persistir la IP, versión de firmware, etc.
- **[NEW]** `internal/infrastructure/db/key_provider.go`
  - Crear el `DBKeyProvider` que implemente `ws.KeyProvider`, consultando `hubRepo.GetByID()` para la validación Ed25519 en la autenticación (reemplazando el Mock).

## 2. Refactor del Registro de WebSockets y Conexiones
- **[MODIFY]** `internal/interfaces/ws/hub_conn.go`
  - Añadir el campo `RestaurantID` al struct `HubConn` para guardar el ID del restaurante que llega tras la autenticación.
- **[MODIFY]** `internal/interfaces/ws/hub_registry.go`
  - Añadir un índice secundario para mapear `restaurantID -> hubID`.
  - Añadir el método `GetByRestaurantID(restaurantID string) (*HubConn, error)` para el futuro *ForwardMessage*.
  - Actualizar `Store()` para poblar el índice secundario.
  - Actualizar `Delete()` para limpiar ambos índices de forma segura.

> [!NOTE]
> La asimetría en el protocolo de autenticación (`hub_id` vs `public_key`) ya está manejada por código con un fallback del lado de la Nube, por lo que no es un bloqueante en esta fase. Solo requiere verificación en la DB local del Hub.
