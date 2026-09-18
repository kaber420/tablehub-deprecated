# Phase 2: Descomposición de UI en Componentes

Este plan detalla los pasos para la Fase 2 de la refactorización de `DashboardView.svelte`. En la Fase 1, logramos centralizar el estado (`devices.svelte.js` y `websocket.svelte.js`) y limpiar el CSS global. Ahora, el objetivo es dividir la enorme interfaz de `DashboardView.svelte` en múltiples componentes pequeños, reutilizables y limpios.

## Objetivos Principales
1. **Componentización**: Extraer secciones funcionales a sus propios archivos `.svelte`.
2. **Encapsulamiento de CSS**: Cada componente se llevará consigo su propio bloque `<style>` de forma local.
3. **UX Mejorada (Sin Bloqueos)**: Eliminar los anticuados y bloqueantes `prompt()` de JavaScript.

## Componentes a Crear

Se creará una nueva carpeta `src/lib/dashboard/` (o se añadirán en `src/lib/`) para alojar estos nuevos componentes:

### 1. `KpiCards.svelte`
- **Responsabilidad**: Renderizar las cuatro tarjetas superiores (Estado del Hub, Dispositivos Activos, Alertas Activas, Personal de Turno).
- **Dependencias**: Leerá de `devicesState` y `statusData`.

### 2. `ServerStatus.svelte`
- **Responsabilidad**: Mostrar las métricas de la base de datos, broker NATS, JetStream, y puertos.
- **Dependencias**: Recibirá la prop `statusData` (o la gestionará localmente).

### 3. `SystemAlerts.svelte`
- **Responsabilidad**: Listar dinámicamente las alertas (desconexión, batería, WiFi) provenientes del estado centralizado.
- **Dependencias**: Consumirá `devicesState.deviceAlerts`.

### 4. `LiveEventsTunnel.svelte`
- **Responsabilidad**: Mostrar el flujo de eventos en tiempo real.
- **Dependencias**: Consumirá `wsState.events` y ejecutará `wsState.clearTunnelLogs()`.

### 5. `ApprovalModal.svelte`
- **Responsabilidad**: Reemplazar la función `quickApprove()` que actualmente usa `prompt()`.
- **Comportamiento**: Un modal nativo en Svelte que solicitará el "Número de Mesa" o "Ubicación" cuando se quiera aprobar un nuevo dispositivo. Mostrará y ocultará su estado (`showModal = $state(false)`).

## Estructura Final de `DashboardView.svelte`

Tras esta fase, `DashboardView.svelte` quedará reducido a un mero orquestador estructural (aproximadamente 30-50 líneas).
