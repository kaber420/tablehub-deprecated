# Especificación Técnica: Gestión Persistente de Mesas y Re-generación de Archivo MicroSD (`tablehub.enc`)

**Fecha:** 2026-07-26  
**Proyecto:** TableHub2 - Módulo de Backend Hub & Gestión de Dispositivos  

---

## 1. Contexto y Requisito de Usuario

### Problema Actual
Actualmente, la generación del archivo `tablehub.enc` se realiza de forma efímera e independiente ("al vuelo"). Si el usuario descarga el archivo y la tarjeta MicroSD se pierde o se corrompe antes de insertarla en el ESP32, se pierde el rastro de la mesa en el sistema y no existe una entidad registrada en el Hub para esa mesa.

### Solución Diseñada
Transformar la gestión de dispositivos en un modelo **Persistente basado en Entidad Mesa**:
1. **Creación Previa de la Entidad:** El usuario crea y gestiona sus mesas/dispositivos en el servidor Hub (ej. "Mesa 1", "Mesa 2", "Barra 3"). La mesa queda guardada en la base de datos SQLite con un `bootstrap_token` asignado.
2. **Generación Reutilizable a la Demanda:** El administrador puede presionar el botón **"Generar / Descargar `tablehub.enc`"** en cualquier momento para cualquier mesa registrada, tantas veces como sea necesario (si pierde la MicroSD o desea reconfigurar).
3. **Adopción Transparente:** La mesa permanece en estado `pending_provision` (o `unprovisioned`) hasta que el ESP32 físicamente ingrese el PIN de esa MicroSD y efectúe el Handshake MQTT, momento en el cual la mesa pasa a `active` vinculando la dirección MAC del chip.

---

## 2. Flujo de Arquitectura y Ciclo de Vida

```
+-----------------------------------------------------------------------------------+
| 1. Panel Hub Web (Svelte)                                                         |
|    - Administrador crea "Mesa 5"                                                 |
|    - Hub asigna bootstrap_token en SQLite (table_number: "5")                     |
|    - Estado inicial: 'unprovisioned'                                             |
+-----------------------------------------+-----------------------------------------+
                                          |
                                          v
+-----------------------------------------------------------------------------------+
| 2. Re-generación de Archivo a la Demanda                                          |
|    - Administrador selecciona "Mesa 5" -> Clic en "Descargar tablehub.enc"       |
|    - Ingresa PIN y selecciona Red Wi-Fi                                           |
|    - Hub cifra el payload usando el bootstrap_token existente                     |
|    - Descarga el archivo (se puede repetir N veces sin perder el registro)       |
+-----------------------------------------+-----------------------------------------+
                                          |
                                          v
+-----------------------------------------------------------------------------------+
| 3. Adopción y Activación (ESP32)                                                  |
|    - ESP32 descifra MicroSD con PIN e ingresa al Wi-Fi                            |
|    - Envía MQTT `tablehub/device/provision` con bootstrap_token + MAC             |
|    - Hub asocia la MAC a "Mesa 5" y cambia estado a 'active'                     |
+-----------------------------------------------------------------------------------+
```

---

## 3. Cambios en API y Base de Datos (Go Backend)

### 3.1. Base de Datos SQLite (`hub/internal/db/`)
* **Tabla `devices`:**
  - Garantizar que las mesas registradas manualmente tengan `status = 'unprovisioned'` o `status = 'pending_provision'`.
  - Vincular la dirección MAC una vez recibida en el paquete de adopción MQTT.
* **Tabla `bootstrap_tokens`:**
  - Mantener la relación `table_id <-> token`. El token NO se elimina si la mesa se reconfigura; sólo se actualiza o reutiliza de forma segura.

### 3.2. Endpoints HTTP Backend (`hub/internal/devices/`)
1. **`POST /api/devices/create` (Nueva Mesa/Dispositivo):**
   - Registra la mesa en la base de datos `devices` con estado `unprovisioned` y genera su `bootstrap_token`.
2. **`POST /api/devices/provision-file` (Generar `tablehub.enc`):**
   - Acepta `table_id`, `pin`, `wifi_network_id` (o SSID/Pass).
   - Busca el `bootstrap_token` existente de `table_id`.
   - Genera el binario AES-256-GCM cifrado y retorna el archivo para descarga.
   - Permite descargas ilimitadas para la misma mesa.

---

## 4. Cambios en la Interfaz Web (Svelte Frontend)

### `hub/web/src/lib/DevicesView.svelte` & `hub/web/src/lib/devices.svelte.js`
1. **Tarjeta/Fila de Mesa:** Cada mesa registrada mostrará el botón de acción: **"Generar Archivo de Aprovisionamiento (`tablehub.enc`)"**.
2. **Modal de Cifrado:** Al hacer clic en una mesa, se abre el modal solicitando el PIN para generar y cifrar el archivo de esa mesa específica.
3. **Indicador de Estado de Adopción:**
   - 🟡 **Amarillo / Pendiente:** "Esperando Aprovisionamiento de Dispositivo" (`unprovisioned`).
   - 🟢 **Verde / Activo:** "Dispositivo Enlazado y Activo" (`active` con MAC y señal Wi-Fi).

---

## 5. Plan de Verificación

1. **Crear Mesa en Panel Web:** Crear "Mesa 10" en el panel. Verificar que aparece en la lista como `Pendiente de Aprovisionamiento`.
2. **Descargar Archivo Múltiples Veces:** Generar `tablehub.enc` para "Mesa 10". Borrar el archivo localmente y volverlo a generar. Verificar que ambos archivos son válidos y usan el mismo `bootstrap_token`.
3. **Prueba en Hardware ESP32:** Cargar el archivo en la MicroSD, ingresar el PIN en la pantalla. Verificar que la "Mesa 10" pasa a color Verde (`active`) vinculando la MAC del ESP32 en tiempo real.

