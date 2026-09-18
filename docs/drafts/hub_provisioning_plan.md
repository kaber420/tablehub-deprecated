# Plan de Implementación: Generador de Archivo de Aprovisionamiento (Hub)

Este plan describe cómo implementar en el Hub (Go + Svelte) la lógica para generar el archivo de configuración cifrado (`tablehub.enc`) requerido por las mesas (ESP32-S3) basado en la arquitectura Offline definida en `docs/drafts/provisioning_pin_plan.md`.

## 1. Decisiones de Diseño (Seguridad y Usabilidad)
Para evitar errores de tipeo y facilitar el aprovisionamiento de múltiples mesas, el Hub permitirá **guardar redes Wi-Fi (SSID y Password)** en su base de datos.
Al aprovisionar una mesa, el administrador podrá seleccionar una red previamente guardada o introducir una nueva (con opción a guardarla). Esto aporta flexibilidad dado que la infraestructura de red final de los usuarios puede variar (cableado vs Wi-Fi del Hub).

### A. Modelo de Datos para Redes Wi-Fi
Se crea una nueva tabla `wifi_networks` en SQLite con los siguientes campos:
- `id` (INTEGER PK AUTOINCREMENT)
- `ssid` (TEXT NOT NULL)
- `password` (TEXT NOT NULL)
- `created_at` (TIMESTAMP DEFAULT CURRENT_TIMESTAMP)

No se almacenan múltiples redes con el mismo SSID (se hace UPSERT por SSID).

### B. API REST para Redes Wi-Fi
Endpoints protegidos (requieren autenticación JWT):
- `GET /api/settings/wifi` — Lista todas las redes guardadas (solo SSID, sin password)
- `POST /api/settings/wifi` — Guarda o actualiza una red `{ssid, password}`
- `DELETE /api/settings/wifi?id=X` — Elimina una red por ID

El GET omite el campo `password` por seguridad; el password solo se usa al generar el archivo de aprovisionamiento.

## 2. Backend (Go - `hub/internal`)

### A. Base de Datos — Tabla `wifi_networks` + CRUD
Archivo: `hub/internal/db/wifi.go`
- `WifiNetwork` struct con `ID`, `SSID`, `Password`, `CreatedAt`
- `ListWifiNetworks() ([]WifiNetwork, error)` — lista todas las redes
- `SaveWifiNetwork(ssid, password string) error` — upsert por SSID
- `DeleteWifiNetwork(id int) error` — elimina por ID

### B. Handlers HTTP para Wi-Fi
Archivo: `hub/internal/devices/wifi.go`
- `HandleWifiNetworks(w, r)` — GET lista / POST guarda / DELETE elimina
- GET: retorna `[{id, ssid, created_at}]` (sin password)
- POST: recibe `{ssid, password}`, guarda, retorna la red guardada
- DELETE: recibe `?id=X`, elimina

### C. Ruta de API de Aprovisionamiento (Ya Implementada)
En `hub/internal/web/server.go`:
- Ruta protegida: `POST /api/devices/provision_file`
- Handler: `HandleGenerateProvisionFile` en `hub/internal/devices/provision.go`

El handler ya implementa:
1. Recibe `{table_id, wifi_ssid, wifi_pass, pin}` vía POST.
2. Obtiene IP local y puerto MQTT del Hub.
3. Genera `bootstrap_token` UUID v4, lo guarda en `bootstrap_tokens`.
4. Crea payload JSON, lo cifra con PBKDF2-HMAC-SHA256 + AES-256-GCM.
5. Devuelve los bytes concatenados como `tablehub.enc`.

Se modifica para aceptar también un `wifi_network_id` opcional, que permite usar credenciales guardadas en la DB.

### D. Nuevas Rutas en `server.go`
```go
http.HandleFunc("/api/settings/wifi", AuthMiddleware(devices.HandleWifiNetworks))
http.HandleFunc("/api/devices/provision_file", AuthMiddleware(devices.HandleGenerateProvisionFile))  // ya existe
```

## 3. Frontend (Svelte - `hub/web/src/lib`)

### A. Funciones API para Wi-Fi
Archivo: `hub/web/src/lib/api.js`
- `getWifiNetworks()` — GET /api/settings/wifi
- `saveWifiNetwork(ssid, password)` — POST /api/settings/wifi
- `deleteWifiNetwork(id)` — DELETE /api/settings/wifi?id=X

### B. Modal de Aprovisionamiento (`DevicesView.svelte`)
El modal existente se mejora con:
1. **Selector de Wi-Fi guardado**: Un `<select>` que lista las redes guardadas. Al seleccionar una, autocompleta SSID y Password.
2. **Campos manuales**: SSID y Password siguen siendo editables manualmente.
3. **Checkbox "Guardar red"**: Si se introducen SSID/Password manualmente, se puede guardar la red en la DB.
4. **Botón "Gestionar Wi-Fi"**: Abre un mini-modal o sección para ver/eliminar redes guardadas.
5. **Flujo de descarga**: Tras generar el archivo, se muestra el botón "Descargar tablehub.enc" (ya implementado).

### C. Gestión de Redes Wi-Fi
Pequeña sección dentro del modal de aprovisionamiento (o un modal separado "Gestionar Redes Wi-Fi") que permite:
- Ver lista de redes guardadas (SSID, fecha de creación)
- Eliminar una red
- El SSID seleccionado se usa automáticamente en el formulario de aprovisionamiento

## 4. Estado de Implementación
- [x] Endpoint Go que genera el JSON cifrado con PBKDF2/AES-GCM (`provision.go`)
- [x] Botón y modal en Svelte para datos y descarga del archivo (`DevicesView.svelte`)
- [x] Botón de descarga manual tras generar el archivo (no descarga automática)
- [ ] Tabla `wifi_networks` en SQLite + CRUD
- [ ] Handlers HTTP para listar/guardar/eliminar redes Wi-Fi
- [ ] Frontend: selector de Wi-Fi guardado en el modal de aprovisionamiento
- [ ] Frontend: opción de guardar nueva red desde el modal
- [ ] Probar compatibilidad criptográfica: Go (PBKDF2 + AES-256-GCM) ↔ ESP32 (mbedtls)
