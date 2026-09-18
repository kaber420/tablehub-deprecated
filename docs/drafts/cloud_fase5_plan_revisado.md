# Plan de Implementación: Fase 5 (Túneles Cloud)

Este documento detalla la arquitectura y los pasos para implementar los Casos de Uso del Túnel y Enrutamiento (Connect/Disconnect/Forward) del servicio Cloud de Tablehub. Se basa en el borrador original, integrando todas las correcciones arquitectónicas de **Mimo** y los hallazgos críticos de **DeepSeek** sobre concurrencia y abstracciones.

## ⚠️ User Review Required
> [!IMPORTANT]
> **Cambios Clave Tras la Revisión de Mimo y DeepSeek:** 
> 1. **Prevención de Race Conditions (DeepSeek):** El `defer` dentro del websocket (`readPump`) ya NO eliminará la conexión del registro, solo cerrará el socket. La goroutine supervisora del Handler será la única encargada de orquestar la desconexión completa en DB y Memoria.
> 2. **Inversión de Dependencias Estricta (DeepSeek):** La capa de aplicación no dependerá de `*ws.HubConn` ni de `ws.TunnelMessage`. Se usarán interfaces `Sender` y `MessageHandler` para abstraer el transporte.
> 3. **Graceful Shutdown (DeepSeek):** Al apagar el servidor, se garantizará que todos los Hubs pasen a estado `offline` en la base de datos.
> 4. **Rate Limiting (DeepSeek):** Se usará `github.com/go-chi/httprate` o `chi.middleware.Throttle` para la API REST, dejando `x/time/rate` exclusivamente para el handshake del WebSocket.

---

## Proposed Changes

### Capa de Dominio e Infraestructura
*Aprovecharemos `hub.Repository` y la implementación actual en Postgres.*
- **[MODIFY]** `cloud/internal/infrastructure/db/hub_repo.go`
  - Añadir soporte para `context.Context` en los métodos del repositorio (como `UpdateStatus`) para permitir cancelación y timeouts (Issue #6 de DeepSeek).

---

### Capa de Aplicación (`cloud/internal/application/tunnel/`)
Contendrá las abstracciones estrictas para el transporte y la orquestación de negocio.

#### [NEW] `types.go`
- Define la estructura de datos `TunnelMessage` para que pertenezca a la aplicación, no a la infraestructura WS.

#### [NEW] `connection_registry.go`
- Define la interfaz `Sender { Send(msg TunnelMessage) error }`.
- Define la interfaz `ConnectionRegistry { Store, Get, GetByRestaurantID, Delete, Shutdown }`.

#### [NEW] `connect.go`
- Implementa `ConnectHubUseCase`. Recibe `ConnectionRegistry` y `hub.Repository`.
- **Flujo:** Registra el `Sender` (`Registry.Store`), actualiza la DB a `online` y guarda la última hora de conexión respetando el contexto.

#### [NEW] `disconnect.go`
- Implementa `DisconnectHubUseCase`.
- **Flujo:** Elimina del registro (`Registry.Delete`) y actualiza el estado en DB a `offline`.

#### [NEW] `forward.go`
- Implementa `ForwardMessageUseCase` y la interfaz `MessageHandler { HandleMessage(hubID, msg) }`.
- **Flujo:** Recibe un mensaje (desde WS), busca el `Sender` del restaurante destino (`Registry.GetByRestaurantID`) y lo enruta.

---

### Capa de Interfaces (`cloud/internal/interfaces/`)
Gestión del transporte (WebSocket / HTTP).

#### [MODIFY] `ws/hub_conn.go`
- Implementa la interfaz `Sender` de la aplicación.
- Modificar el método `Start()` para devolver un canal `<-chan struct{}` que avise cuando termine el `readPump`.
- Eliminar `registry.Delete` del `defer` de `readPump`. El `defer` solo debe hacer `c.Close()`.
- Añadir el `MessageHandler` por constructor para enviar los mensajes entrantes a la capa de aplicación sin acoplamiento.

#### [MODIFY] `ws/handler.go`
- Será el orquestador principal del ciclo de vida del socket:
  1. Autentica y hace Upgrade.
  2. Crea el `HubConn` (inyectando el `MessageHandler`).
  3. Llama a `ConnectHubUseCase.Execute(...)` pasando el `HubConn` (como `Sender`).
  4. Lanza la goroutine supervisora que escucha el `<-done` de `HubConn.Start()`. Al recibir la señal, ejecuta de forma segura `DisconnectHubUseCase.Execute(...)` evitando carreras.

#### [NEW] `rest/router.go`
- Setup del enrutador usando `go-chi/chi/v5`.
- Middlewares: Logging, Recovery y Rate Limiting (usando las herramientas nativas de Chi para REST).
- `CheckOrigin = true` para la ruta WebSocket; Allowlist para REST.

#### [NEW] `rest/hub_handler.go`
- Endpoints `GET /health`, `GET /v1/hubs`, `GET /v1/hubs/{id}`.

---

## Verification Plan

### Pruebas Unitarias Automáticas
- `go test ./internal/application/tunnel/...` con Mocks inyectando `ConnectionRegistry` y `hub.Repository`. Validaremos el flujo de abstracción `Sender`.

### Verificación End-to-End
1. Iniciar servidor Cloud. Simular conexión de Hub.
2. Comprobar que pasa a `online` en BD.
3. Forzar un cierre abrupto del cliente. Comprobar que la goroutine supervisora cambia a `offline` correctamente.
4. Mandar señal de `SIGTERM` (Ctrl+C) al servidor y comprobar que todos los Hubs conectados pasan a `offline` en la BD (Graceful Shutdown).
