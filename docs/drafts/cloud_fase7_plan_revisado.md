# Plan de Implementación Revisado: Fase 7 (Observabilidad, ACKs y Offline Queue)

Este plan incorpora el feedback conjunto de **Mimo** (Arquitectura de BD y Colas) y **DeepSeek** (Concurrencia, Condiciones de Carrera y Fugas de Memoria).

## Resumen de Cambios

1. **Observabilidad (Admin API)**: Endpoints REST para métricas globales (`/v1/admin/stats` y `/v1/admin/tunnels`).
2. **Patrón ACK y Retry Persistente**: Garantizar entrega mediante reintentos robustos. Se usará la base de datos (PostgreSQL) como única fuente de verdad (Source of Truth) y cola transaccional, eliminando el uso de canales Go para evitar deadlocks y fugas de memoria.

---

## 🛠️ Correcciones Arquitectónicas (Feedback Aplicado)

### 1. Base de Datos como Cola (Sugerencia de Mimo & DeepSeek)
- **Cero Canales Go para ACKs**: Todo el estado reside en BD. El `readPump` solo actualiza la BD al recibir un ACK.
- **Transaccionalidad (`FOR UPDATE SKIP LOCKED`)**: Permite correr múltiples workers o instancias del servidor sin que colisionen al procesar los mismos eventos pendientes.
- **Nuevos Campos y Estados**:
  - `status`: Agregamos el estado `'expired'`. No usamos `pending_ack` como estado intermedio; en su lugar, usamos `status = 'delivered'` + `ack_deadline`.
  - `next_retry_at TIMESTAMPTZ`: Índice para que el worker encuentre rápido qué procesar.
  - `ack_deadline TIMESTAMPTZ`: Para detectar ACKs perdidos.

### 2. Prevención de Condiciones de Carrera (Sugerencia de DeepSeek)
- **Procesamiento Asíncrono de ACKs**: En `hub_conn.go`, la actualización a BD por el ACK se lanzará en una goroutine (`go AcknowledgeWebhook(...)`) o con un `context.WithTimeout` muy estricto para **no bloquear** el `readPump`.
- **Limpieza al Desconectar**: En `DisconnectHubUseCase`, todos los eventos con `ack_deadline` activo de ese Hub pasarán a `hub_offline` y su `next_retry_at` se ajustará para que el worker los recoja rápido.
- **Idempotencia en el Hub Físico**: Incluiremos el `event_id` en el JSON del túnel. Es **crítico** que el Hub ignore IDs duplicados en caso de que su ACK original se haya perdido en la red.
- **Push vs Pull (Opcional/Futuro)**: El worker puede tener un canal para despertarse instantáneamente cuando un Hub se conecta, mejorando la latencia frente al `time.Ticker` puro.

### 3. Graceful Shutdown del Worker
- El worker usará `sync.WaitGroup` en `main.go` y `select { case <-ctx.Done(): ... }` para garantizar que no haya fugas de goroutines ni queries interrumpidas corruptas al apagar el server.

---

## 📝 Plan de Ejecución (Archivos a Modificar)

### 1. Migración SQL
#### [NEW] `cloud/internal/infrastructure/db/migrations/003_retry_ack.sql`
- Añadir `ack_deadline`, `next_retry_at` y `expires_at`.
- Actualizar `webhook_events_status_check` para incluir `expired`.

### 2. Dominio y Repositorios
#### [MODIFY] `cloud/internal/domain/webhook/...` y `cloud/internal/infrastructure/db/webhook_event_repo.go`
- Métodos `GetPendingRetries` (con `SKIP LOCKED`).
- Método `AcknowledgeEvent` (limpia el deadline).
- Método `FailPendingACKs` (para cuando un hub se desconecta abruptamente).

### 3. Casos de Uso y WebSockets
#### [MODIFY] `cloud/internal/application/tunnel/disconnect_hub.go`
- Llamar a `FailPendingACKs` para liberar mensajes en tránsito del hub que se acaba de caer.
#### [MODIFY] `cloud/internal/interfaces/ws/hub_conn.go` o `forward.go`
- Procesar el `webhook.ack` de forma asíncrona.

### 4. Worker
#### [NEW] `cloud/internal/application/webhook/retry_worker.go`
- Goroutine atada al ciclo de vida de `main.go`. Busca en BD según `next_retry_at` o `ack_deadline` expirados.

### 5. API Admin
#### [NEW] `cloud/internal/interfaces/rest/admin_handler.go`
- `/v1/admin/stats` y `/v1/admin/tunnels`.

---

## 🙋 Open Questions

1. Para evitar cuellos de botella en la conexión WebSocket, procesaremos el ACK en una goroutine separada (fire-and-forget). ¿Estás de acuerdo con este enfoque?
2. Los ACKs requieren que el Hub envíe de vuelta el `event_id`. Debemos asegurarnos de que el equipo de Hardware / Firmware esté al tanto de este cambio. ¿Lo anoto en algún documento global de requerimientos o tú te encargas?
3. ¿Dejamos el endpoint `/v1/admin/stats` sin autenticación por ahora para facilitar el desarrollo, o le pongo un middleware básico?

## Verification Plan
1. Ejecutar `make db-up` y aplicar migraciones.
2. Inyectar webhooks y simular caídas del Hub.
3. Verificar que el Worker recoge los webhooks y aplica `FOR UPDATE SKIP LOCKED` correctamente.
4. Validar el apagado limpio (Graceful Shutdown) del worker.
