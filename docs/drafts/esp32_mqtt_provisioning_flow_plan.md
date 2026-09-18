# Especificación Técnica y Plan: Flujo de Aprovisionamiento MQTT y Token (ESP32 <-> Hub)

**Fecha:** 2026-07-26  
**Dispositivo Destino:** ESP32-S3 (Guition JC3248W535 - 3.5" 320x480 TFT Touch Capacitivo)  
**Proyecto:** TableHub2 - Módulo de Comunicación MQTT / JetStream  

---

## 1. Contexto y Problema Diagnosticado

Durante el aprovisionamiento por MicroSD:
1. El Hub genera `tablehub.enc` cifrado con AES-256-GCM a partir de un PIN numérico.
2. Dentro del payload de `tablehub.enc`, el Hub incluye un `bootstrap_token` único (UUIDv4) registrado en la base de datos SQLite.
3. El ESP32 desencripta el archivo, guarda las credenciales en NVS y borra el archivo de la tarjeta SD.
4. **Problema Diagnosticado:** Al arrancar, el ESP32 no transmitía el `bootstrap_token` al Hub por MQTT debido a dos razones:
   - Los tópicos MQTT en el ESP32 carecían del prefijo obligatorio **`tablehub/`** (`tablehub/device/...`), por lo que NATS JetStream descartaba los paquetes.
   - El manejador de JetStream en el Hub (`jetstream.go`) no estaba validando la firma del `bootstrap_token` enviada por MQTT para auto-activar la mesa en la base de datos.

---

## 2. Protocolo de Handshake MQTT con Token de Aprovisionamiento

### Diagrama de Secuencia

```
+---------------+                +------------------+                +------------------+
| ESP32 (Mesa)  |                | Broker NATS MQTT |                | Hub Server (Go)  |
+-------+-------+                +--------+---------+                +--------+---------+
        |                                 |                                   |
        | 1. Conexión MQTT (Port 1883)    |                                   |
        +-------------------------------->|                                   |
        |                                 |                                   |
        | 2. Subscribe "tablehub/device/{MAC}/state"                          |
        +-------------------------------->|                                   |
        |                                 |                                   |
        | 3. Publish "tablehub/device/provision"                              |
        |    Payload: { mac, table_id, bootstrap_token, ip, type }           |
        +-------------------------------->+---------------------------------->|
        |                                 |                                   | (Valida bootstrap_token en DB)
        |                                 |                                   | (Asigna mesa y cambia status='active')
        |                                 |                                   | (Invalida bootstrap_token)
        |                                 |                                   |
        | 4. Publish "tablehub/device/{MAC}/state"                            |
        |<--------------------------------+-----------------------------------|
        |    Payload: { status: "active", table_id: "4" }                     |
        |                                 |                                   |
        | 5. Telemetría Periódica (Cada 10s)                                 |
        |    Publish "tablehub/device/{MAC}/status"                           |
        +-------------------------------->+---------------------------------->|
```

---

## 3. Especificación de Tópicos MQTT y Payloads

### 3.1. Solicitud de Registro con Token (`tablehub/device/provision`)
* **Dirección:** ESP32 -> Hub Server
* **Tópico MQTT:** `tablehub/device/provision`
* **Subject NATS:** `tablehub.device.provision`
* **Payload JSON:**
  ```json
  {
    "mac": "14:C1:9F:4D:7F:D8",
    "table_id": "4",
    "bootstrap_token": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
    "ip": "192.168.1.50",
    "type": "table_pad"
  }
  ```

### 3.2. Respuesta del Hub y Activación (`tablehub/device/{MAC}/state`)
* **Dirección:** Hub Server -> ESP32
* **Tópico MQTT:** `tablehub/device/14:C1:9F:4D:7F:D8/state`
* **Subject NATS:** `tablehub.device.14:C1:9F:4D:7F:D8.state`
* **Payload JSON:**
  ```json
  {
    "status": "active",
    "table_id": "4",
    "message": "Dispositivo activado exitosamente"
  }
  ```

### 3.3. Telemetría y Estado (`tablehub/device/{MAC}/status`)
* **Dirección:** ESP32 -> Hub Server
* **Tópico MQTT:** `tablehub/device/14:C1:9F:4D:7F:D8/status`
* **Subject NATS:** `tablehub.device.14:C1:9F:4D:7F:D8.status`
* **Payload JSON:**
  ```json
  {
    "mac": "14:C1:9F:4D:7F:D8",
    "table": "4",
    "ip": "192.168.1.50",
    "battery": 85,
    "wifi": -55,
    "status": "active"
  }
  ```

---

## 4. Cambios Requeridos en el Código

### A. Firmware ESP32
1. **[MQTTService.h / .cpp](file:///home/kaber420/Documentos/proyectos/tablehub2/firmware/src/Network/MQTTService.h):**
   - Corregir el prefijo de los tópicos a `tablehub/...`.
   - Guardar y enviar `bootstrap_token` al conectar por primera vez al servidor.
   - Eliminar el retardo de 15s en la conexión inicial al arrancar.
2. **[ConfigManager.h / .cpp](file:///home/kaber420/Documentos/proyectos/tablehub2/firmware/src/Network/ConfigManager.h):**
   - Asegurar que `bootstrapToken` persista en NVS (`Preferences`) junto con `wifiSsid`, `wifiPass`, `hubIp`, `mqttPort` y `tableId`.

### B. Hub Backend (Go)
1. **[hub/internal/events/jetstream.go](file:///home/kaber420/Documentos/proyectos/tablehub2/hub/internal/events/jetstream.go):**
   - En `processDeviceProvision`: Si se recibe `bootstrap_token`, validar contra la tabla `bootstrap_tokens` en SQLite.
   - Si la validación es correcta, actualizar el dispositivo a estado `active` en la base de datos `devices` e inhabilitar el `bootstrap_token` usado.
   - Publicar el evento de confirmación `tablehub/device/{MAC}/state` al cliente MQTT.

---

## 5. Plan de Verificación

1. **Generación de Archivo en Hub:**
   - Crear un archivo `tablehub.enc` para la Mesa 1 con PIN `1234`.
2. **Desbloqueo en ESP32:**
   - Insertar MicroSD, ingresar PIN `1234` en `SetupPinView`.
3. **Validación de Registro:**
   - Verificar en los logs del Hub (`log.Printf`) que se recibe `tablehub.device.provision` con el `bootstrap_token` correcto.
   - Confirmar que la mesa pasa inmediatamente a estado `active` en el panel de administración Svelte.

---

## 6. Mecanismos de Seguridad y Protección de Datos MQTT

Para evitar que un usuario no autorizado escuche el tráfico o inyecte transacciones falsas en la red del restaurante:

1. **Cifrado de Transporte (MQTTS / TLS):**
   - El canal de comunicación MQTT se ejecuta cifrado mediante TLS (MQTTS en puerto 8883) o encapsulado dentro de una red Wi-Fi privada/aislada (VLAN de servicio).

2. **Firma Criptográfica Asimétrica (Ed25519) por Mensaje:**
   - Cada evento o cambio de estado enviado por los dispositivos puede incluir una firma digital en formato hexadecimal (`signature`).
   - El Hub verifica en `verifyDeviceSignature()` la firma utilizando la clave pública (`public_key`) registrada para esa dirección MAC en SQLite. Si un atacante intercepta o intenta alterar un paquete, la firma no coincidirá y el Hub lo descarta.

3. **Protección contra Replay Attacks (Timestamp Strict Window):**
   - Todos los payloads llevan una marca de tiempo Unix (`timestamp`). En `verifyTimestamp()`, el Hub rechaza cualquier mensaje con una desviación mayor a 5 minutos (300 segundos).

4. **Filtrado por Estado en el Backend (RBAC de Dispositivo):**
   - Si un dispositivo está en estado `unprovisioned` o `blocked`, el servidor rechaza automáticamente cualquier solicitud de negocio (`processDeviceEventSecure` exige `status == 'active'`).

