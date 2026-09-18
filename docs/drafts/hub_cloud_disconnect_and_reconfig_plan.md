# Plan: Desconexión y Reconfiguración del Hub con la Nube (SaaS)

## Problema Actual
Al configurar el Hub por primera vez con el archivo de conexión (`.thub`), el Hub guarda el estado de la conexión de forma permanente en su base de datos local (SQLite `data/tablehub.db`). Concretamente guarda:
- `cloud_url`
- `restaurant_id`
- `hub_id`
- `cloud_enabled`
- Además, mantiene las llaves criptográficas (`hub_cloud.key`).

Actualmente **no hay un mecanismo (ni un endpoint ni un botón en la UI)** que limpie esta configuración. Solo existe un "toggle" para pausar la conexión, pero el Hub sigue atado a la cuenta original. Esto impide poder conectar el Hub a una cuenta diferente.

## Propuesta de Mejora y UI Intuitiva

### 1. Cambios en el Backend (Hub API)
- **Endpoint de Desconexión:** Crear `POST /api/settings/cloud/disconnect` en `hub/internal/web/settings.go`.
- **Lógica:**
  - Borrar de la tabla `settings`: `cloud_url`, `restaurant_id`, `hub_id`.
  - Asegurar que `cloud_enabled` quede en `false` o también se borre.
  - Cerrar la conexión NATS activa de forma segura (ej. `events.CloseClient()`).
  - Opcional pero recomendado: Permitir borrar o ignorar la clave privada antigua en este proceso, o sobreescribirla de manera segura en el próximo setup.

### 2. Cambios en el Frontend (Hub UI)
Mejorar la interfaz de configuración de la Nube para que sea más intuitiva (en `hub/web/src/lib/ConfigurationView.svelte` u otros componentes relevantes).

- **Estado 1: No Conectado**
  - Implementar una interfaz amigable de **Drag & Drop** (Arrastrar y Soltar) para subir el archivo `.thub`. Esto es mucho más intuitivo que tener que pegar JSON manualmente.

- **Estado 2: Conectado**
  - Mostrar claramente a qué restaurante y URL está conectado el Hub.
  - Añadir un botón rojo prominente: **"Desconectar de la Nube"** o **"Cambiar Cuenta"**.
  - Al pulsar el botón, mostrar un diálogo de confirmación (SweetAlert o modal propio) avisando que esto desvinculará el Hub del restaurante actual. Si se confirma, llamar al endpoint `/api/settings/cloud/disconnect`.

## Plan de Ejecución
1. Implementar `POST /api/settings/cloud/disconnect` en Go.
2. Añadir botón y modal de desconexión en el Frontend Svelte.
3. Probar el flujo: Conectar -> Desconectar -> Verificar BD limpia y conexión NATS cerrada -> Reconectar con nuevo archivo.
4. Mejorar la UI del setup para usar "Drag & Drop".

## Evolución y Diseño por Tabs para la Configuración (UI/UX)

Para mantener una interfaz limpia y modular (similar al panel de la nube con pestañas/tabs), la vista de **Configuración del Hub** se organizará en las siguientes pestañas:

### 1. Tab: Estado y Conexión Nube (SaaS)
* **Badge / Indicador de Estado en Vivo:**
  - 🟢 **Conectado:** Muestra un indicador verde con pulso animado, ping/latencia actual y hora de la última sincronización de eventos.
  - 🔴 **Desconectado / Pausado:** Badge rojo/amarillo aclarando la causa (sin internet, pausado por usuario, o sin credenciales).
* **Ficha del Restaurante Vinculado:**
  - **Restaurante ID:** Identificador de la sucursal/restaurante en la nube.
  - **Servidor Cloud (NATS URL):** Dirección del cluster SaaS.
  - **Hub Public Key:** Llave pública Ed25519 con botón de copia rápida (útil para auditoría).
* **Acciones de Conexión:**
  - Switch Toggle "Activar/Pausar Conexión Nube".
  - Botón "Desconectar Hub / Cambiar Cuenta" (Abre modal de confirmación y ejecuta la limpieza total de credenciales).

### 2. Tab: Vincular Nueva Cuenta (Dropzone / Setup)
*(Visión cuando el Hub está desvinculado)*
* **Área de Drag & Drop para `.thub`:**
  - Dropzone interactiva con animación al arrastrar el archivo de credenciales descargado del panel Cloud.
  - Alternativa: Botón "Examinar archivo" o pegar la clave token manualmente.
* **Validador Automático:**
  - Al soltar el archivo, valida la estructura del token, extrae los parámetros y prueba la latencia con la Nube antes de guardar.

### 3. Tab: Sistema Local y Red
* Configuración del puerto HTTP local (ej. 8080).
* Puertos NATS / MQTT locales para impresoras y comanderas de meseros.
* Estado del almacenamiento SQLite local (`tablehub.db`).

### 4. Tab: Autenticación y Administración
* Opción para cambiar contraseña del panel de administración del Hub.
* Clave de emergencia / restablecimiento.

## Auditoría Técnica con OpenCode y Plan de Corrección Definitivo

Mediante la herramienta `opencode` (modelo DeepSeek-v4-flash-free), se realizó una auditoría profunda del flujo de desconexión y re-aprovisionamiento para identificar por qué la desconexión fallaba al intentar conectar con otra cuenta.

### Hallazgos de la Auditoría:

1. **Fuga del Socket WebSocket Activo (`hub/internal/web/server.go`)**:
   - `HandleCloudDisconnect` llamaba a `RestartCloudConnection()`, cancelando el contexto (`cloudCancel()`). Sin embargo, el goroutine `handleCloudSession` estaba bloqueado en el bucle de lectura `conn.ReadMessage()` y **no cerraba inmediatamente la conexión física `conn.Close()`**. 
   - La conexión con la nube continuaba viva en segundo plano durante 90 segundos después de hacer clic en "Desconectar".

2. **Persistencia del Archivo de Claves Criptográficas (`data/hub_cloud.key`)**:
   - `HandleCloudDisconnect` borraba `cloud_url` y `restaurant_id` en SQLite, pero **dejaba intacto en disco el archivo `data/hub_cloud.key`**.
   - Al intentar vincular el Hub con un nuevo archivo `.thub` de otra cuenta/restaurante, `InitCloudKeys()` detectaba que `data/hub_cloud.key` existía y **reutilizaba la clave privada antigua**, en lugar de generar una clave limpia. La Nube rechazaba el `auth_challenge` por discrepancia de identidad.

3. **Invalidez de Caché / Estado de Sesión en Aprovisionamiento (`settings.go`)**:
   - Al aprovisionar una nueva cuenta con `HandleProvisionUpload`, no se forzaba la regeneración de claves si la cuenta anterior se había desconectado.

### Plan de Acción de Corrección (Aprobación Requerida):

1. **Corrección en `HandleCloudDisconnect` (`hub/internal/web/settings.go` y `server.go`)**:
   - Forzar el cierre explícito de la conexión de red del WebSocket (`CloseCloudSocket()`).
   - Eliminar físicamente el archivo de claves `data/hub_cloud.key` y limpiar `public_key` en SQLite al presionar "Desconectar".
   - Limpiar variables en memoria `SetCloudConnected(false)`.

2. **Corrección en `InitCloudKeys` / `HandleProvisionUpload`**:
   - Al procesar un nuevo archivo `.thub`, si el `restaurant_id` o `hub_id` no coinciden con los guardados previamente, regenerar el archivo `data/hub_cloud.key` y la clave pública Ed25519 correspondiente.

3. **Pruebas de Verificación Exhaustivas**:
   - Conectar a Cuenta A con `.thub`.
   - Ejecutar Desconexión -> Confirmar borrado físico de `data/hub_cloud.key` y cierre inmediato del socket NATS.
   - Cargar `.thub` de Cuenta B -> Verificar generación de nuevas llaves y handshake Ed25519 exitoso con la Nube.

## Evolución: Estrategia del Ciclo de Vida de Claves Ed25519

### Análisis de los 2 Escenarios de Uso

1. **Re-aprovisionamiento / Actualización de `.thub` (Mismo Restaurante / Mismo Hub ID)**:
   - **Caso de uso**: Cambios de URL del servidor SaaS, renovación de certificados o actualización de tokens de bootstrap en el mismo restaurante.
   - **Estrategia**: **Conservar la clave privada (`hub_cloud.key`)**. Como el `hub_id` y el `organization_id` no cambian, mantener el mismo par Ed25519 evita tener que re-autorizar la clave pública en el dashboard SaaS. La reconexión es transparente e inmediata.

2. **Desconexión Total / Re-asignación a Otra Cuenta/Restaurante (Distinto Hub ID u Organización)**:
   - **Caso de uso**: El equipo físico del Hub se reasigna a otro cliente o sucursal.
   - **Estrategia**: **Limpiar / Regenerar la clave privada (`hub_cloud.key`)**. Al cambiar de tenant en el SaaS, se debe garantizar el principio de aislamiento multi-inquilino generando una nueva identidad Ed25519 para evitar que el nuevo restaurante use la identidad del anterior.

### Opciones en la Interfaz (UI/UX)
Para brindar flexibilidad total al usuario avanzado sin complicar al usuario convencional:
- **En la pestaña "Estado Nube" -> Sección Avanzada**:
  - Opción de "Rotar Claves de Seguridad Ed25519" con botón dedicado para casos de auditoría o clave comprometida.
- **En el Modal de Desconexión**:
  - Casilla opcional: `[x] Regenerar claves criptográficas (Recomendado si cambias de restaurante o sucursal)`.

## Actualización de Estado: Implementación y Verificación Completadas ✅

*(Por reglas del proyecto, las secciones iniciales del plan se conservan para mantener el historial de la propuesta inicial. A continuación se resume la implementación realizada)*.

### 1. Backend (`Go`) - COMPLETADO
- ✅ **Endpoint `/api/settings/cloud/disconnect`**: Implementado en `hub/internal/web/settings.go`.
- ✅ **Cierre Inmediato de Socket**: Implementadas funciones de gestión de conexión activa `SetActiveCloudConn` y `CloseActiveCloudConn` en `hub/internal/web/server.go`.
- ✅ **Borrado Físico de Claves**: `HandleCloudDisconnect` borra SQLite `public_key` y ejecuta `os.Remove("data/hub_cloud.key")`.
- ✅ **Limpieza en Re-aprovisionamiento**: `HandleProvisionUpload` detecta cambios de `hub_id` y limpia claves obsoletas automáticamente.
- ✅ **Estado Real en Vivo**: Se expuso `is_connected` a través de la API comprobando el handshake `auth_success`.

### 2. Frontend (`Svelte`) - COMPLETADO
- ✅ **Vista Tabulada (Subtabs)**: Refactorizado `ConfigurationView.svelte` con clases idénticas al tema visual del SaaS Cloud (`.subtabs-panel`, `.subtabs-navigation`, `.subtab-btn.active`).
- ✅ **Dropzone Drag & Drop**: Zona interactiva para soltar archivos `.thub`.
- ✅ **Botón y Modal de Desconexión**: Acción prominente en rojo con cuadro de diálogo modal de confirmación.
- ✅ **Indicador de Estado Real**: Visualización de estado en tiempo real (Conectado / Reconectando / Desvinculado).
