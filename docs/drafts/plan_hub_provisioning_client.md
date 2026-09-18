# Plan de Implementación Arquitectónica: Aprovisionamiento de Hubs (Client Dashboard)

**Objetivo:** Extender el panel de cliente (`Devices.svelte`) para permitir la generación y descarga segura del archivo de provisión `.thub`, garantizando una experiencia de usuario (UX) premium, manejo de errores robusto y el cumplimiento estricto de la arquitectura de seguridad existente.

---

## 1. Arquitectura de Seguridad y Endpoints

### 1.1 Consumo del Endpoint Existente
El endpoint a consumir es `POST /v1/orgs/{orgID}/hubs/provision`.
- **Autenticación:** El frontend debe incluir el Bearer token (JWT) del usuario actual en los headers. El middleware del backend (`middleware.go`) validará que el usuario pertenezca a la organización `orgID` antes de generar el token.
- **Protección de Datos:** La llave privada nunca se genera en la Nube. La Nube solo devuelve el `bootstrap_token` de un solo uso dentro de un archivo binario/JSON estructurado (`tablehub.thub`).
- **Idempotencia y Riesgos:** Generar un archivo `.thub` crea un registro en BD (`hub_registry`) esperando activación. El frontend debe prevenir múltiples clics accidentales (bloqueo de botón durante la petición) para evitar registros huérfanos.

### 1.2 Extensión de `api.js`
Actualmente `apiFetch` está optimizado para JSON. Crearemos un cliente robusto para Blob/Archivos.

**Archivo:** `cloud/web/src/lib/api.js`
```javascript
export async function downloadHubProvision(orgID) {
    // 1. Obtener token del auth_store
    // 2. Realizar petición POST con headers adecuados
    // 3. Manejo de errores dual: 
    //    - Si res.ok == false, el backend puede devolver JSON (ej. {"error": "unauthorized"}).
    //      Hay que leer res.text() o res.json() para mostrar un Toast útil.
    //    - Si res.ok == true, extraer res.blob().
    // 4. Forzar descarga dinámica usando `URL.createObjectURL(blob)` y un elemento <a> invisible.
}
```

---

## 2. Diseño de Interfaz y Experiencia de Usuario (UI/UX)

No podemos simplemente descargar un archivo sin contexto. El usuario final (dueño del restaurante) necesita saber qué es el archivo `.thub` y qué hacer con él.

### 2.1 Modificaciones en `Devices.svelte`
- **Ubicación del Botón:** En la cabecera `toolbar`, alineado a la derecha, agregaremos un botón primario con icono (ej. `DownloadCloud` o `Plus` de Lucide Svelte).
- **Estilo de Botón:** Usaremos una clase `.btn-primary` con gradiente o color de acento para destacar la acción ("Nuevo Hub Físico").

### 2.2 Modal de Provisión (UX Flow)
En lugar de iniciar la descarga directa al hacer clic en el toolbar, abriremos un modal instructivo.

**Flujo del Modal:**
1. **Pantalla Inicial:** Muestra un breve texto explicando: *"Estás a punto de registrar un nuevo Servidor Hub para tu restaurante. Se descargará un archivo de credenciales de un solo uso."*
2. **Acción de Generar:** Botón "Generar y Descargar Credenciales". Al hacer clic, el botón entra en estado `loading` (spinner) y se bloquea.
3. **Manejo de Respuesta:**
   - **Éxito:** El modal cambia a un estado de éxito (icono verde), el archivo se descarga automáticamente, y las instrucciones cambian a: *"Archivo descargado. Carga este archivo en la interfaz de configuración de tu servidor físico. Este modal se cerrará automáticamente en 5s."*
   - **Error:** Se muestra una alerta en rojo dentro del modal indicando el motivo (ej. "No tienes permisos" o "Fallo de conexión").

---

## 3. Manejo de Estados y Sincronización

### 3.1 Actualización de la Grilla de Dispositivos
- Cuando la generación es exitosa, el backend ya creó el esqueleto del Hub en la base de datos (con `connection_status: offline` y `administrative_status: activo`).
- El frontend debe llamar a `loadData()` de forma silente en el background justo después de la descarga exitosa.
- El usuario verá aparecer instantáneamente una nueva tarjeta (`hub-card`) en la vista, con el estado "OFFLINE" y un indicador visual de que está "Esperando activación".

---

## 4. Criterios de Aceptación (DoD)

### Seguridad y API
- [ ] La función de descarga en `api.js` propaga el token JWT correctamente.
- [ ] Los errores de red o HTTP 4xx/5xx no rompen la UI, sino que se atrapan y muestran de forma amigable.

### UI / UX
- [ ] El botón encaja perfectamente con el diseño del `toolbar` (padding, bordes, hover states).
- [ ] El Modal previene interacciones accidentales (fondo bloqueado, botón deshabilitado durante la carga).
- [ ] El mensaje post-descarga es claro respecto a qué debe hacer el usuario con el archivo `.thub`.

### Funcionalidad
- [ ] El archivo descargado mantiene la extensión `.thub` y el nombre `tablehub.thub` (o el nombre dictado por el header `Content-Disposition`).
- [ ] La grilla de `hubs-grid` refleja el nuevo Hub inmediatamente tras cerrar el modal.
