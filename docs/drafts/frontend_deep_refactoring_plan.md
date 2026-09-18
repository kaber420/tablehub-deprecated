# Plan de Refactorización Profunda: Frontend a Backend

Este documento detalla el plan para resolver los anti-patrones arquitectónicos identificados en el frontend (`hub/web/src/`), delegando la lógica de negocio, validaciones pesadas y orquestación de red hacia el servidor backend (Maitre Hub).

## 1. Eliminar WebSocket Duplicado (Alta Prioridad)
**Problema:** `DevicesView.svelte` mantiene su propia conexión a `/ws/local`, su propio estado interno de dispositivos y su lógica de reconexión, duplicando a `websocket.svelte.js` y `devices.svelte.js`.
**Acciones:**
- Eliminar la inicialización de WebSocket en `DevicesView.svelte`.
- Importar y consumir reactivamente la store global: `devicesState.devices`.
- Asegurar que cualquier acción realizada en esta vista refleje inmediatamente en el estado global.

## 2. Refactorización de Usuarios y Atomicidad (Alta Prioridad)
**Problema:** `UsersView.svelte` hace dos peticiones en cadena (`PUT /api/users/:id` y `PUT /api/users/:id/pin`) para guardar un empleado, lo que viola el principio de atomicidad transaccional.
**Acciones:**
- **Backend:** Modificar la ruta `/api/users/:id` (o crear una nueva) para que acepte tanto los datos del perfil (nombre, rol, etc.) como el nuevo PIN opcional en el mismo JSON payload, usando una única transacción de base de datos.
- **Frontend:** Consolidar la acción de guardado en `UsersView.svelte` en un solo llamado API.

## 3. Delegación de Reglas de Alerta al Backend (Alta Prioridad)
**Problema:** En `devices.svelte.js`, el getter `deviceAlerts` tiene "hardcodeadas" reglas de negocio (batería ≤20%, señal ≤-85dBm) para crear alertas, impidiendo que el backend controle los umbrales globalmente.
**Acciones:**
- **Backend:** Implementar una nueva API (`GET /api/alerts`) o emitir alertas pre-empaquetadas a través de WebSocket cuando el servidor detecte niveles críticos.
- **Frontend:** Limpiar el getter de alertas, limitándose simplemente a renderizar el arreglo de alertas provisto por el backend.

## 4. Filtros y Búsquedas del Lado del Servidor (Media Prioridad)
**Problema:** En lugar de pedir al backend un resumen, el cliente web descarga todos los dispositivos y los filtra constantemente en el navegador.
**Acciones:**
- **Backend:** Expandir el endpoint `/api/devices` o crear `/api/devices/summary` para que el servidor entregue los contadores (activos, inactivos) procesados por SQLite.
- **Frontend:** Aprovechar queries directas `?status=active` al obtener vistas específicas, reduciendo la carga de CPU y memoria en el navegador.

## 5. Normalización de Tópicos y JSON (Media Prioridad)
**Problema:** `LiveEventsTunnel.svelte` tiene que adivinar las propiedades de un evento `order` con lógicas como `order.table_number || order.table_name || order.table`.
**Acciones:**
- **Backend:** Asegurar que el bridge MQTT y el Webhook normalicen los objetos de JSON hacia un formato canónico estricto antes de emitirlos al Frontend.
- **Frontend:** Eliminar los condicionales, asumiendo un objeto siempre estructurado (ej. usar solo `order.table_number`).

---
> **Nota de Implementación**: Al ejecutar estas fases, se respetará estrictamente la política actual de Svelte 5 (Runes) para estado cliente, priorizando siempre a MQTT y al backend de Go como única fuente de la verdad.
