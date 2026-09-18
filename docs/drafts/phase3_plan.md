# Phase 3: Delegación de Estado a MQTT (Cero Lógica en Frontend)

Este plan detalla los pasos para la Fase 3 de la refactorización de `DashboardView.svelte` y la arquitectura del proyecto, enfocándose en la delegación total del control de presencia (online/offline) de los dispositivos hacia el Broker MQTT nativo y el servidor backend.

## Objetivos Principales
1. **Cliente Frontend 100% Pasivo**: Svelte se limitará a mutar el estado de dispositivos basándose estrictamente en eventos WebSocket del backend. No debe calcular inactividad por sí mismo.
2. **Eliminación Absoluta de Relojes de Presencia**: Asegurar que ningún módulo (`clock.svelte.js` o `setInterval`) muten el estado a `offline` dentro del navegador.
3. **Delegación Nativa a MQTT**: Utilizar las características de protocolo IoT nativas (Keep-Alive y Last Will and Testament) para manejar desconexiones imprevistas.

## Análisis de la Situación Actual
Gracias a las Fases 1 y 2, el frontend ya está centralizado y componentezado. El estado reactivo de Svelte (`devices.svelte.js`) maneja correctamente la reactividad si un dispositivo cambia su `status` a `'offline'`. No obstante, el sistema actualmente carece de una política clara sobre *quién* decide que está desconectado.

## Tareas a Realizar

### 1. Backend/IoT: Implementación de Last Will and Testament (LWT)
- **Modificación en el Cliente IoT**: Al conectar al broker MQTT, el dispositivo debe especificar un Testamento (LWT).
  - Tópico sugerido: `telemetry/lwt` o similar.
  - Payload sugerido: `{"mac": "XX:XX:XX", "status": "offline"}`.
- **Keep-Alive**: Configurar el margen del protocolo (ej. 15, 30 o 60 segundos). Si el dispositivo no envía un PING, el broker MQTT despachará el Testamento automáticamente a todos los suscriptores.

### 2. Backend: Traducción y Reenvío hacia el Frontend (WebSocket)
- El Maitre Hub backend (suscrito al broker MQTT) debe interceptar estos mensajes LWT.
- Al recibirlos, debe despachar inmediatamente un evento a través del túnel de WebSocket hacia la UI:
  ```json
  {
    "event": "device.status_update",
    "payload": {
      "mac": "XX:XX:XX:XX:XX:XX",
      "status": "offline"
    }
  }
  ```

### 3. Frontend: Auditoría Final y Limpieza
- En `src/lib/websocket.svelte.js`, la recepción de `device.status_update` con `status: 'offline'` actualizará automáticamente la store de dispositivos, detonando reactivamente la alerta en `SystemAlerts.svelte`.
- Realizar un `grep` exhaustivo en `src/lib/` para asegurarse de que no quede ningún `setInterval` ni función que modifique `status = 'offline'` localmente comparando un campo `last_seen`. (El campo `last_seen` puede permanecer como algo meramente visual o informativo).

## Plan de Verificación
1. **Simulación de Desconexión Abrupta**: Apagar el Wi-Fi o desconectar físicamente de la corriente un dispositivo en estado `active`.
2. **Espera de Keep-Alive**: Esperar el tiempo especificado de expiración en el MQTT.
3. **Observación Pasiva**: Observar el Dashboard (sin recargar y sin hacer ninguna acción). El backend enviará el LWT a través de WebSocket y el dispositivo pasará de `active` a `offline` automáticamente, generando la alerta roja en la UI.
