# Plan de Refactorización de DashboardView

## 1. El Problema Actual
Actualmente, el archivo `DashboardView.svelte` consta de más de 1000 líneas de código. Esto se debe a que está actuando como un componente "monolítico" que asume demasiadas responsabilidades simultáneamente:
- **Lógica de red:** Gestión directa de la conexión WebSocket y fetching de APIs.
- **Gestión de estado:** Mantenimiento de listas de eventos, dispositivos, usuarios y cálculo reactivo de alertas.
- **Estructura visual (HTML):** Renderizado del encabezado, las tarjetas de KPIs, el estado del servidor, la lista de alertas y el túnel de eventos.
- **Diseño (CSS):** Una gran cantidad de CSS encapsulado (scoped) para dar estilo a todas estas secciones dispares.

Esta centralización dificulta la lectura, el mantenimiento y la reutilización de código.

## 2. Propuesta Arquitectónica

La refactorización se basa en dos pilares fundamentales: **Gestión de Estado Centralizada** y **Separación de Componentes de Presentación**.

### A. Centralización del Estado (Svelte Stores)
Extraer la lógica de red y estado a Svelte Stores. Los componentes visuales solo deben "consumir" estos almacenes y renderizar datos, no gestionar conexiones.

- Crear `src/lib/stores/websocketStore.js` (o similar): 
  - Se encargará de iniciar y mantener la conexión WebSocket.
  - Almacenará la lista de eventos (`events`) de forma reactiva.
- Crear `src/lib/stores/deviceStore.js` (opcional/recomendado):
  - Mantendrá el estado de los dispositivos y las alertas calculadas.

### B. Descomposición en Componentes (UI)
Dividir la interfaz en componentes más pequeños y enfocados (Dumb Components), pasándoles los datos necesarios mediante _props_ o suscribiéndolos directamente a los Stores.

1. **`KpiCards.svelte`**: Componente para renderizar las tarjetas superiores (Hub Status, Dispositivos, Alertas, Personal).
2. **`ServerStatus.svelte`**: Componente para la sección "Server Status" (Base de datos, NATS, Puertos, etc.).
3. **`SystemAlerts.svelte`**: Componente para listar las alertas del sistema basándose en el estado de los dispositivos.
4. **`LiveEventsTunnel.svelte`**: Componente exclusivo para renderizar la lista de eventos recibidos. Solo leerá del array de eventos y no tendrá lógica de conexión.

### C. Organización del CSS
Al dividir los componentes, cada archivo `.svelte` nuevo se llevará exclusivamente el bloque de CSS que le corresponde de la etiqueta `<style>`. Esto limpiará enormemente el archivo principal y hará que cada componente sea auto-contenido visualmente. Si se detectan más estilos comunes, se pueden mover a `app.css`.

## 3. Beneficios
- **Mantenibilidad:** Archivos de 100-200 líneas en lugar de 1000.
- **Rendimiento y Limpieza:** Se evita la duplicación de conexiones WebSocket en un futuro si otra vista necesita escuchar eventos.
- **Reusabilidad:** Componentes como `LiveEventsTunnel` o `SystemAlerts` podrían instanciarse en otras partes del Dashboard si fuera necesario.

---
## Evolución y Actualizaciones
*(Cualquier cambio futuro o extensión a este plan deberá añadirse debajo de esta sección, sin destruir el plan original).*

### 2026-07-20: Revisión Técnica Arquitectónica (Svelte 5)
Tras un análisis exhaustivo del código fuente (`DashboardView.svelte` y `package.json`), se han identificado lógicas críticas que el plan original omitía. Dado que el proyecto utiliza **Svelte 5**, la arquitectura debe modernizarse.

**Ajustes Clave al Plan:**

1. **Svelte 5 Runes en lugar de Stores Legacy:**
   - No se usarán `stores` tradicionales. En su lugar se crearán módulos `.svelte.js` utilizando `$state` y `$derived`.
   - **`websocket.svelte.js`**: Manejará la conexión y exportará `events = $state([])`.
   - **`devices.svelte.js`**: Exportará `devices = $state([])`, `deviceAlerts = $derived(...)` y métodos de actualización.
   - **Acoplamiento**: El módulo WebSocket importará funciones del módulo de dispositivos para mutar el estado directamente al recibir telemetría.

2. **Extracción del Reloj Reactivo:**
   - El motor de alertas depende de un `setInterval`. Se creará un módulo `clock.svelte.js` para mantener un reloj global centralizado.

3. **Eliminación del `prompt()` bloqueante:**
   - La función `quickApprove()` no puede vivir en un componente "tonto" porque usa un prompt del navegador. Se diseñará un nuevo componente `ApprovalModal.svelte` para manejar el ingreso de datos de forma asíncrona y elegante.

4. **Persistencia y Limpieza (LocalStorage):**
   - La función de `clearTunnelLogs` interactúa con el localStorage. Esta lógica se encapsulará en el módulo de estado `websocket.svelte.js`, no en el componente visual.

5. **Extracción de CSS Global Temprana:**
   - Clases utilitarias como `.text-success`, `.text-danger`, `.monospace`, y `.led-indicator` se moverán a `app.css` como primer paso antes de dividir los archivos.

### Decisión Arquitectónica Crítica: Delegación de Estado a MQTT (Cero Lógica en Frontend)
- **El Error de Diseño Original:** Anteriormente, el frontend evaluaba constantemente con un reloj local (`currentTime`) si los dispositivos habían superado un tiempo sin reportarse para marcarlos como desconectados. Esto era un error grave, ya que si se cierra el navegador, el sistema queda ciego y no procesa alertas.
- **La Solución Definitiva (Protocolo MQTT):** Se prohíbe el uso de relojes o lógica de monitoreo de estado en el frontend. Toda la delegación de presencia recaerá nativamente sobre **MQTT**.
  - Se delegará el rastreo de inactividad a los pings nativos del protocolo MQTT (`Keep-Alive`).
  - Se utilizará obligatoriamente la función **Last Will and Testament (LWT)** del Broker MQTT para que, al caerse la red de un dispositivo sin previo aviso, el Broker emita automáticamente un mensaje de `Offline`.
- **Impacto Arquitectónico:** El servidor y MQTT asumen su rol como la *única fuente de verdad*. El frontend (Svelte) se convierte en un cliente 100% pasivo que únicamente reacciona a los eventos validados por el backend, respetando la naturaleza real de los sistemas IoT.
