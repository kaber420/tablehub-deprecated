# Plan de Implementación: Fase 7 — Observabilidad y Colas Offline (ACK & Retry Persistente)

## Contexto

Esta fase abarca dos objetivos críticos de infraestructura y resiliencia:
1. **Observabilidad (Admin API)**: Añadir endpoints para monitorear el estado del clúster (túneles activos, estadísticas globales) en tiempo real.
2. **Cola Offline de Webhooks**: En la Fase 6, si un webhook llega y el Hub está offline, lo marcamos como `hub_offline` pero se pierde. En esta fase implementaremos un proceso asíncrono (worker) que reintente periódicamente el envío de estos eventos rezagados en cuanto el Hub se reconecte (Retry Persistente) e implementaremos el reconocimiento `ACK` real desde el Hub.

---

## Cambios Propuestos

### 1. Observabilidad y Stats (Admin API)

#### [NEW] `cloud/internal/interfaces/rest/admin_handler.go`
Creación de endpoints de administración:
- `GET /v1/admin/stats`
  - Retorna JSON con: `hubs_online`, `hubs_total`, `webhooks_processed_24h`, `uptime_seconds`.
  - Requerirá extender los repositorios o agregar queries ligeras (`COUNT(*)`).
- `GET /v1/admin/tunnels`
  - Retorna una lista con el ID de los restaurantes que tienen un túnel actualmente conectado, para debugging.

#### [MODIFY] `cloud/internal/interfaces/rest/router.go`
- Añadir el `AdminHandler` y registrar sus rutas (`/v1/admin/...`). Opcionalmente protegerlas con Auth básico.

---

### 2. Reconocimiento (ACK) y Retry Persistente (Offline Queue)

Actualmente la Fase 6 confía en que el `Send` del WebSocket no devuelva error. Si el Hub físico se desconecta sin cerrar el socket (ej. corte de energía), `Send` podría tener éxito en el buffer de red, pero el Hub nunca procesar el ticket.

#### [NEW] Patrón ACK en Webhooks
- El Hub debe enviar un mensaje `{ "event": "webhook.ack", "payload": { "event_id": "UUID" } }` cuando procesa el ticket.
- **Cambio en `HubConn` o `DispatchWebhookUseCase`**: Cambiaremos la definición de éxito. Un webhook estará en estado `pending_ack` y solo pasará a `delivered` cuando se reciba el mensaje `webhook.ack` desde el túnel.

#### [NEW] Background Worker de Reintentos
- Un worker (`cloud/internal/application/webhook/retry_worker.go`) que corra cada X minutos.
- Busca en la BD eventos `webhook_events` con estado `hub_offline` o `pending_ack` que hayan expirado.
- Intenta reenviarlos a través del `DispatchWebhookUseCase`.

---

## Plan de Ejecución (Modelos IA)

Para asegurar la máxima calidad de este diseño arquitectónico y de tolerancia a fallos distribuidos:
1. **Mimo**: Revisará este plan para sugerir mejoras arquitectónicas (ej. ¿Es mejor usar una goroutine para la cola de reintentos o delegar a una tabla de BD que sirva de cola?).
2. **DeepSeek**: Analizará posibles condiciones de carrera (data races) en el worker en background y si la integración del mensaje `webhook.ack` a través de WebSocket podría causar leaks de memoria por canales bloqueados.

---

## Verification Plan
1. Ejecutar tests unitarios.
2. Levantar servidor y verificar `/v1/admin/stats`.
3. Simular caída de hub, enviar webhook, reconectar hub y verificar que el reintento persistente procesa el webhook.
