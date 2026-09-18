# Plan de Aseguramiento de Conectividad Bidireccional

> **Contexto:** Análisis de la arquitectura de comunicación entre el Hub (ESP32 -> Hub Local) y la Nube (Hub Local -> Cloud SaaS), identificando cuellos de botella, asimetrías de protocolo y puntos de ruptura que comprometen la conectividad ininterrumpida.

---

## Índice de Problemas Detectados

| # | Problema | Gravedad | Componente |
|---|----------|----------|------------|
| 1 | Asimetría de payload de autenticación Hub↔Cloud | **CRÍTICO** | Hub + Cloud |
| 2 | Coalescencia incorrecta en writePump del Cloud | **CRÍTICO** | Cloud |
| 3 | Heartbeat asimétrico (WebSocket PING vs app-level "ping") | **ALTO** | Hub + Cloud |
| 4 | Sin heartbeat ni read deadline en el Hub | **ALTO** | Hub |
| 5 | MaxMessageSize (4KB) insuficiente para payloads batch | **MEDIO** | Cloud |
| 6 | Sin protección replay en túnel Hub↔Cloud | **MEDIO** | Hub + Cloud |
| 7 | Sin notificación de aprobación al ESP32 tras provisionar | **MEDIO** | Hub + Firmware |
| 8 | Firmware ESP32 sin capa de red implementada | **ALTO** | Firmware |
| 9 | Sin reenvío de eventos Cloud→ESP32 por MQTT | **ALTO** | Hub |
| 10 | SendBuffer desbordable sin backpressure al emisor | **BAJO** | Cloud |

---

## 1. Asimetría de Payload de Autenticación (CRÍTICO)

### Diagnóstico

El Hub (`hub/internal/web/server.go:268-272`) envía:

```go
type AuthResponsePayload struct {
    RestaurantID string `json:"restaurant_id"`
    PublicKey    string `json:"public_key"`    // ← Hub espera validación por clave
    Signature    string `json:"signature"`     // ← nombre legacy
}
```

La Nube nueva (`cloud/internal/infrastructure/ws/types.go:13-17`) espera:

```go
type AuthResponsePayload struct {
    RestaurantID string `json:"restaurant_id"`
    HubID        string `json:"hub_id"`          // ← Cloud espera ID lógico
    SignatureHex string `json:"signature_hex"`   // ← nombre nuevo
}
```

**Consecuencia:** El Hub NO PUEDE autenticarse contra la nueva Cloud. El campo `hub_id` llega vacío, la validación `authPayload.HubID == ""` en `cloud/internal/infrastructure/ws/auth.go:92` corta la conexión inmediatamente.

### Plan de Acción

#### Hub (lado del Hub local) — `hub/internal/web/server.go`

1. **Añadir `HubID` al `AuthResponsePayload`:**
   - Leer el `hub_id` desde `db.GetSetting("hub_id")`.
   - Si no existe, generar un UUID v4 y persistirlo.
   - Incluir `hub_id` en el response.

2. **Renombrar `signature` → `signature_hex`:**
   - El campo se sigue llamando `signature` en el Hub. Cambiar a `signature_hex`.

3. **Mantener `public_key` para compatibilidad reversa:**
   - El Cloud legacy aún lo necesita. Enviar ambos campos durante período de transición.

```go
type AuthResponsePayload struct {
    RestaurantID string `json:"restaurant_id"`
    HubID        string `json:"hub_id"`
    PublicKey    string `json:"public_key"`       // compat legacy
    SignatureHex string `json:"signature_hex"`   // nuevo nombre
    // Legacy alias opcional:
    Signature    string `json:"signature,omitempty"` // eliminar tras migración
}
```

4. **Persistir `hub_id` en SQLite:**
   - En `InitCloudKeys()` o en la primera conexión exitosa, generar y guardar `hub_id`.
   - Usar `db.SaveSetting("hub_id", hubID)`.
   - Tratarlo como identidad permanente del Hub (no cambiar a menos que se re-configure).

#### Cloud (lado del SaaS) — `cloud/internal/infrastructure/ws/auth.go`

5. **Aceptar ambos formatos durante transición:**
   - Si `authPayload.HubID` está vacío, intentar resolver por `authPayload.PublicKey`.
   - En `KeyProvider`, añadir método `GetByPublicKey(pubKeyHex string) ([]byte, error)`.

6. **Validación estricta tras migración:**
   - Una vez que todos los Hubs envíen `hub_id`, eliminar el fallback por `public_key`.

---

## 2. Coalescencia Incorrecta en writePump del Cloud (CRÍTICO)

### Diagnóstico

En `cloud/internal/infrastructure/ws/hub_conn.go:92-103`:

```go
w, err := c.conn.NextWriter(websocket.TextMessage)
// ...
n := len(c.send)
for i := 0; i < n; i++ {
    w.Write([]byte{'\n'})
    w.Write(<-c.send)
}
```

Esto escribe múltiples mensajes JSON separados por `\n` en **un solo frame WebSocket**. El Hub receptor lee con `conn.ReadMessage()` que devuelve **el frame completo** — no puede parsear el JSON concatenado. **Todo mensaje coalescido causa un parse error en el Hub.**

### Plan de Acción

7. **Opción A (recomendada): Enviar cada mensaje en su propio frame:**
```go
case message, ok := <-c.send:
    c.conn.SetWriteDeadline(time.Now().Add(c.config.WriteWait))
    if !ok {
        c.conn.WriteMessage(websocket.CloseMessage, []byte{})
        return
    }
    if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
        return
    }
```

8. **Opción B (si se necesita coalescencia por throughput):**
   - Cambiar a `websocket.BinaryMessage` con un delimitador explícito (longitud prefijada o CBOR).
   - O implementar un `sync.Mutex` simple alrededor de `WriteMessage`.

---

## 3. Heartbeat Asimétrico (ALTO)

### Diagnóstico

- **Cloud** (`hub_conn.go:111-128`): envía WebSocket-level PING frames cada 54s. El runtime Go responde PONG automáticamente. Read deadline: 60s.
- **Hub** (`server.go:337-342`): espera un mensaje de aplicación con `event: "ping"` y responde `event: "pong"`. **Nunca recibe estos mensajes** porque el Cloud manda PINGs de nivel WebSocket, no de aplicación.
- **Neto:** el Hub no recibe latidos, y el Cloud cree que el Hub está muerto si no responde PONG (pero el runtime Go lo maneja automágicamente).

### Plan de Acción

9. **Cloud: enviar heartbeats también como mensajes de aplicación:**
   - En `heartbeat()`, después del `WriteMessage(websocket.PingMessage, nil)`, enviar también un `TunnelMessage{Event: "ping"}`.

10. **Hub: unificar manejo de heartbeats:**
    - Configurar `conn.SetPongHandler()` para actualizar deadline.
    - Configurar `conn.SetReadDeadline()` con un valor razonable (ej. 90s).
    - Eliminar el case "ping" manual en el read loop, o mantenerlo como respaldo.

11. **Hub: añadir heartbeat propio:**
    - Implementar una goroutine `heartbeat()` que envíe `TunnelMessage{Event: "ping"}` periódicamente (ej. 30s), por si el Cloud deja de iniciar.
    - El Cloud debe responder con `event: "pong"`.

---

## 4. Sin Heartbeat ni Read Deadline en el Hub (ALTO)

### Diagnóstico

El Hub (`server.go:304-344`) no configura:
- `SetReadDeadline()` — una conexión silenciosa rota nunca se detecta.
- `SetPongHandler()` — no responde a PINGs de nivel WebSocket.
- Sin heartbeat saliente — si el Cloud no envía nada, el Hub no detecta la pérdida.

### Plan de Acción

12. **Añadir configuración de deadlines en `handleCloudSession()`:**
```go
conn.SetReadDeadline(time.Now().Add(90 * time.Second))
conn.SetPongHandler(func(string) error {
    conn.SetReadDeadline(time.Now().Add(90 * time.Second))
    return nil
})
```

13. **Implementar goroutine de heartbeat en el Hub:**
   - Similar a `HubConn.heartbeat()` del Cloud.
   - Enviar `TunnelMessage{Event: "ping"}` cada 30s.
   - Si falla el write, cerrar sesión y forzar reconexión.

---

## 5. MaxMessageSize Insuficiente (MEDIO)

### Diagnóstico

```go
MaxMessageSize: 4096  // cloud/internal/infrastructure/ws/config.go:18
```

4KB es ajustado para payloads de provisioning batch o logs de sync.

### Plan de Acción

14. **Aumentar a 64KB (o parametrizable por restaurante/plan):**
```go
MaxMessageSize: 65536
```

15. **Si un mensaje excede el límite, partir en chunks con un protocolo de reassembly:**
   - Añadir campo `chunk_id`, `chunk_total`, `chunk_index` opcionales en `TunnelMessage`.
   - Descartar (log warning) en lugar de crash si se excede el límite.

---

## 6. Sin Protección Replay en Túnel Hub↔Cloud (MEDIO)

### Diagnóstico

El Hub→dispositivo tiene protección replay (timestamp ±5 min en `verifyTimestamp()`), pero el túnel Hub↔Cloud **no tiene ninguna protección** después del handshake inicial.

### Plan de Acción

16. **Añadir timestamp a todos los `TunnelMessage` y validarlo en ambos lados:**
   - En Cloud (`hub_conn.go:readPump`): verificar `|now - msg.Timestamp| < tolerance`.
   - En Hub (`server.go:handleCloudSession` read loop): verificar simétricamente.
   - Tolerancia sugerida: ±30s (relojes NTP-sync en ambos lados).

17. **Protección adicional (opcional pero recomendada):**
   - Refrescar el challenge Ed25519 periódicamente (ej. cada 24h) sin reconectar el WS.
   - O usar un nonce rotatorio en cada mensaje.

---

## 7. Sin Notificación de Aprobación al ESP32 (MEDIO)

### Diagnóstico

Cuando el admin aprueba un dispositivo (`HandleApproveDevice`), el Hub actualiza SQLite pero **no notifica al ESP32**. El ESP32 sigue en estado "unprovisioned" hasta que reinicia o re-envía un provision request.

### Plan de Acción

18. **Publicar comando de aprobación en MQTT:**
   - Tras aprobar en `hub/internal/devices/handlers.go`, publicar en `tablehub/device/{mac}/command`:
   ```json
   {"command": "provision_approved", "assigned_table": "Mesa 5"}
   ```

19. **Firmware (futuro):** suscribirse a `tablehub/device/{mac}/command` y reaccionar al comando `provision_approved`:
   - Guardar `assigned_table` en NVS.
   - Activar funcionalidad completa.

---

## 8. Firmware ESP32 Sin Capa de Red (ALTO)

### Diagnóstico

El firmware (`firmware/src/`) solo contiene UI con LVGL. No hay:
- WiFi driver
- MQTT client
- Provisioning logic
- Deep sleep

### Plan de Acción

20. **Implementar Hito 1 del roadmap del firmware:**
    - **WiFi Provisioning:** usar WiFiManager o similar para captive portal en primera config.
    - **MQTT Client:** usar `esp-mqtt` (componente oficial ESP-IDF).
    - **Conectar al Hub:** `mqtt://<hub-ip>:1883`.
    - **Topicos MQTT:**

    | Topic | QoS | Dirección | Frecuencia |
    |-------|-----|-----------|------------|
    | `tablehub/device/provision` | 1 | ESP32 → Hub | 1 vez al inicio |
    | `tablehub/device/{mac}/status` | 0 | ESP32 → Hub | Cada 60s |
    | `tablehub/device/{mac}/event` | 1 | ESP32 → Hub | Bajo demanda |
    | `tablehub/device/{mac}/command` | 1 | Hub → ESP32 | Bajo demanda |
    | `tablehub/table/{id}/order` | 1 | Hub → ESP32 | Bajo demanda |

21. **Payload mínimo firmado (Ed25519):**
    - El ESP32 debe generar un par Ed25519 en primera ejecución.
    - Guardar private key en NVS (flash encriptada).
    - Firmar todos los payloads de status y event.

22. **Manejo de reconexión WiFi/MQTT:**
    - Implementar backoff exponencial (1s → 30s max).
    - Re-enviar provision request tras reconectar.
    - Buffer de eventos offline para re-enviar al reconectar.

---

## 9. Sin Reenvío de Eventos Cloud→ESP32 por MQTT (ALTO)

### Diagnóstico

Cuando el Cloud recibe un webhook de POS (`order.created`) y lo reenvía por el túnel al Hub, el Hub lo procesa en `handleCloudSession:321-335` y lo publica en NATS:

```go
events.Publish(fmt.Sprintf("tablehub.table.%s.order", tbl), event.Payload)
```

Pero NATS no es MQTT. Aunque el Hub tiene un bridge MQTT→NATS en el servidor embebido, la **dirección inversa** (NATS→MQTT) no está garantizada para todos los topics.

### Plan de Acción

23. **Verificar que el bridge MQTT de NATS Server publica en ambos sentidos:**
    - NATS Server embebido tiene MQTT bridge integrado. Verificar que `tablehub.table.{id}.order` se publica como MQTT `tablehub/table/{id}/order` a todos los suscriptores MQTT.
    - Si no funciona (es unidireccional por defecto), implementar un **forwarder explícito**:

24. **Implementar `MQTTForwarder` en el Hub:**
    - En `hub/internal/events/`, crear `mqtt_forward.go`.
    - Suscribirse a NATS subjects `tablehub.table.>.order` y `tablehub.device.>.command`.
    - Publicar en el broker MQTT embebido usando un cliente MQTT interno (no el bridge).
    - Alternativa: usar `nats: nats_mqtt_prefix` + JetStream para topología pub/sub bidireccional.

---

## 10. SendBuffer Desbordable sin Backpressure (BAJO)

### Diagnóstico

```go
send: make(chan []byte, config.SendBufferSize)  // 256 slots
```

Si el Cloud envía mensajes más rápido de lo que el Hub los consume (ej. múltiples órdenes simultáneas contra un Hub con latencia alta), `Send()` devuelve `context.DeadlineExceeded` y el mensaje se pierde.

### Plan de Acción

25. **Implementar backpressure en lugar de drop silencioso:**
    - En `forward_message.go` (cloud_fase4), si `Send()` falla, reintentar con backoff corto (100ms, máx 3 intentos).
    - Registrar métrica de buffer lleno y alertar.

26. **Aumentar `SendBufferSize` a 1024:**
```go
SendBufferSize: 1024
```

---

## Resumen de Archivos a Modificar

### Hub (`hub/`)

| Archivo | Cambios |
|---------|---------|
| `internal/web/server.go` | Arreglar `AuthResponsePayload` (añadir `hub_id`, `signature_hex`); añadir `SetReadDeadline`/`SetPongHandler`; añadir heartbeat goroutine; eliminar coalescencia incorrecta (no aplica, es server-side del hub) |
| `internal/web/settings.go` | Persistir `hub_id` en `InitCloudKeys()`; exponer en `CloudSettingsPayload` |
| `internal/events/mqtt_forward.go` **(NUEVO)** | Forwarder NATS→MQTT para comandos y órdenes |
| `internal/devices/handlers.go` | Publicar comando `provision_approved` por MQTT tras aprobar |
| `internal/db/sqlite.go` | Verificar que la tabla `settings` soporte `hub_id` (genérico, ya soporta) |

### Cloud (`cloud/`)

| Archivo | Cambios |
|---------|---------|
| `internal/infrastructure/ws/types.go` | Añadir campo `PublicKey` opcional a `AuthResponsePayload` para compat transición |
| `internal/infrastructure/ws/auth.go` | Aceptar `public_key` como fallback si `hub_id` vacío |
| `internal/infrastructure/ws/hub_conn.go` | Eliminar coalescencia de writes; enviar cada mensaje en su propio frame; añadir heartbeats de aplicación; validar timestamps |
| `internal/infrastructure/ws/config.go` | Aumentar `MaxMessageSize` a 64KB; aumentar `SendBufferSize` a 1024 |
| `internal/application/tunnel/forward_message.go` **(NUEVO, cloud_fase4)** | Reintentar con backpressure |

### Firmware (`firmware/`)

| Archivo | Cambios |
|---------|---------|
| `src/Network/WiFiManager.cpp` **(NUEVO)** | WiFi provisioning con WiFiManager |
| `src/Network/MQTTClient.cpp` **(NUEVO)** | Cliente MQTT con reconexión y backoff |
| `src/Network/Crypto.cpp` **(NUEVO)** | Ed25519 key generation, signing, NVS storage |
| `src/Network/DeviceProvisioner.cpp` **(NUEVO)** | Provision request flow |
| `platformio.ini` | Añadir dependencias `esp-mqtt`, `WiFiManager`, `micro-ecc` o `libsodium` |

---

## Secuencia de Implementación Recomendada

### Fase 1 (Urgente — Bloqueante)

```
Semana 1:
  [Hub]   Arreglar asimetría AuthResponsePayload (#1)
  [Hub]   Añadir heartbeat y read deadlines (#3, #4)
  [Cloud] Arreglar coalescencia writePump (#2)
  [Cloud] Aceptar fallback public_key en auth (#1)
```

### Fase 2 (Media — Estabilidad)

```
Semana 2:
  [Cloud] Aumentar MaxMessageSize y SendBufferSize (#5, #10)
  [Cloud] Añadir heartbeats de aplicación (#3)
  [Hub]   Implementar heartbeat propio (#4)
  [Hub]   Validar timestamps en mensajes del túnel (#6)
```

### Fase 3 (Largo Plazo — Firmware)

```
Semanas 3-4:
  [Firmware] Implementar Hito 1: WiFi + MQTT (#8)
  [Hub]     Forwarder NATS→MQTT (#9)
  [Hub]     Notificar aprobación al ESP32 (#7)
  [Firmware] Generación y almacenamiento de clave Ed25519
```

### Fase 4 (Mejora Continua)

```
Semana 5+:
  [Cloud] Protección replay con desafíos periódicos (#6)
  [Hub]   Refresco de challenge Ed25519 periódico
  [TODOS] Tests E2E de reconexión: cortar WiFi, reiniciar Cloud, etc.
```

---

## Verificación

### Tests Existentes que Siguen Siendo Válidos

- `cloud/internal/infrastructure/ws/auth_test.go` — todas las pruebas de firma (ajustar si se modifica el payload)
- `cloud/internal/infrastructure/ws/registry_test.go` — concurrencia del registry

### Tests Nuevos Requeridos

| Test | Cobertura |
|------|-----------|
| Test hub auth con payload legacy (public_key) | Transición cloud |
| Test hub auth con payload nuevo (hub_id + signature_hex) | Transición cloud |
| Test writePump sin coalescencia | Cloud hub_conn |
| Test heartbeat con doble mecanismo | Cloud + Hub |
| Test reconexión con backoff | Hub ConnectToCloud |
| Test forwarder NATS→MQTT | Hub MQTT forward |
| Test provision→aprobación→notificación | Hub devices + MQTT |
| Test buffer lleno con backpressure | Cloud forward_message |

### Comandos de Verificación

```bash
# Hub
cd hub && go test ./internal/web/... ./internal/events/... -v

# Cloud
cd cloud && go test ./internal/infrastructure/ws/... -v

# Firmware (cuando exista)
cd firmware && pio test -e esp32dev
```
