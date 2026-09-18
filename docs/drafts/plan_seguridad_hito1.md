# Plan de Seguridad: Cierre Hito 1 (TableHub)

Este documento describe la arquitectura de los cambios definitivos para resolver las vulnerabilidades críticas reportadas en el sistema.

## Pregunta Abierta
- **Generación de ID del Hub**: En el `Hub` local, cuando se ejecute por primera vez y no tenga un `HubID` configurado, proponemos generar un UUID v4 (`hub-1234...`) y guardarlo de forma permanente en su base de datos local SQLite. ¿Estás de acuerdo con utilizar este estándar para la identificación única de los dispositivos físicos?
  *(Nota sobre fallos: Si la base de datos se corrompe y se pierde el UUID, el Hub generará uno nuevo y deberá volver a ser registrado, actuando como un dispositivo totalmente nuevo, lo que asegura que un Hub corrupto no pueda usurpar sesiones).*

---

## Cambios Propuestos

### 1. Eliminar Código Legacy Inseguro
El archivo `cloud/main_legacy.go` será eliminado completamente del repositorio. Esto suprime los secretos quemados ("secret_tasty_token_123") y código obsoleto que ya no se alinea con la Clean Architecture que hemos adoptado.

### 2. Prevención de Spoofing de Identidad (Cloud)
**Archivo a modificar:** `cloud/internal/interfaces/ws/auth.go`
- Eliminaremos el "fallback" automático donde el servidor asume que si no recibe un `HubID`, el ID es igual a la llave pública.
- Añadiremos una validación estricta: Si `authPayload.HubID == ""`, el servidor registrará (log) el intento fallido de conexión por carecer de identidad.
- El servidor enviará un código de cierre de WebSocket específico para fallo de seguridad (ej: `4001 Unauthorized`) con la razón "missing hub_id" y desconectará inmediatamente al cliente (nota: como no existen clientes heredados/legacy en producción actual, el rechazo será total y estricto).

### 3. Protección de Origen de WebSockets (CORS)
**Archivo a modificar:** `cloud/internal/interfaces/ws/handler.go`
- Actualizaremos la configuración `CheckOrigin` del `upgrader`. En vez de devolver siempre `true`, verificará el encabezado `Origin` de la solicitud entrante contra una lista controlada.
- Si el origen no está dentro de la lista de permitidos (ej. `localhost:3000` para desarrollo, y los dominios de producción), la conexión será bloqueada.
- Se implementará **Logging de Seguridad** para registrar intentos fallidos de conexión por violación de CORS, lo que servirá para la auditoría y monitoreo.

### 4. Generación Autónoma de Identidad (Local Hub)
**Archivo a modificar:** `hub/internal/web/server.go`
- En la función `InitCloudKeys()`, el sistema verificará si existe el parámetro `hub_id` en la base de datos SQLite.
- Si no existe (primer arranque de un Hub físico), el programa generará un UUID usando `github.com/google/uuid`, lo guardará con la clave `hub_id` y lo utilizará en todos los futuros intentos de conexión (payload de autenticación).
- Se añadirá una verificación de integridad al iniciar, para asegurar que el `hub_id` leído cumpla el formato de UUID.

---

## 5. Auditoría, Monitoreo y Fallbacks
- **Auditoría:** Todos los rechazos de WebSockets (por CORS o por falta de HubID) generarán logs estructurados tipo `WARN` o `ERROR` para detección de posibles ataques.
- **Fallbacks:** Si el proceso de generación de llaves falla localmente o el cloud no autentica, el proceso se abortará en vez de continuar de manera degradada.
- *(Nota: No hay necesidad de plan de migración para clientes porque se asume que todos los clientes conectados a futuro adoptarán este flujo desde cero).*

---

## Verificación

1. **Reinicio de Entorno:**
   Usaremos nuestro nuevo `dev-cli` para borrar la caché, reiniciar la DB y probar el sistema con la nueva estricta configuración.
   
2. **Prueba End-to-End:**
   Verificaremos que al iniciar el Hub por primera vez, este loguea "Generando nuevo Hub ID...", y que el servidor Cloud lo autentica exitosamente bajo su nuevo ID seguro en vez de rechazarlo.
