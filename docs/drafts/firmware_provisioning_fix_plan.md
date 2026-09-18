# Plan de Corrección: Flujo de Aprovisionamiento en Firmware

Este documento detalla los fallos encontrados en el flujo de aprovisionamiento offline del ESP32 y el plan exacto para solucionarlos.

## 1. Problemas Identificados

1. **Ignorar la SD por configuración previa:** En `main.cpp`, si el dispositivo tiene una configuración guardada (`g_isConfigured = true`), el arranque salta por completo la verificación de la memoria SD. Si el usuario inserta la SD para re-aprovisionar, el archivo es ignorado. Además, al saltarse esto, **la SD nunca se montaba** para el caché de imágenes posterior si la mesa ya estaba configurada.
2. **Incompatibilidad de `table_id`:** El hub genera un UUID (string) para el ID de la mesa. El ESP32 lo espera como un `int`, provocando que al leer el JSON asigne siempre `0` o genere corrupción de memoria en futuras lecturas.
3. **Ciclo de vida de `SD_MMC` roto:** Las llamadas a `SD_MMC.begin()` y `SD_MMC.end()` estaban mal repartidas y causaban cierres prematuros.
4. **Falta de información (Diagnóstico UI):** Si algo falla en `attemptProvisioning()` (contraseña incorrecta, archivo corrupto, error criptográfico), la UI siempre muestra un mensaje genérico.
5. **Comportamiento UI para Dispositivos Nuevos:** Actualmente, si no está aprovisionado, solo muestra un texto flotante básico. Se necesita una ventana/modal formal que instruya al usuario.

## 2. Plan de Implementación C++

### A. Montaje Global de la SD y Mejoras en UI (`main.cpp`)
- La memoria MicroSD **se usará para caché de imágenes (menú) y aprovisionamiento**, por lo tanto, el sistema realizará un `SD_MMC.begin()` de forma **global y permanente** al principio del arranque (`setup()`), independientemente del estado de la configuración. **Nunca se llamará a `SD_MMC.end()`**, dejando el sistema de archivos siempre accesible para LVGL y la caché.
- Inmediatamente después de montarla, el arranque comprobará una sola vez si existe `/tablehub.enc`.
  - **Si existe y el dispositivo NO está configurado**: Lanza directamente la ventana de `SetupPinView` para pedir el PIN.
  - **Si existe y el dispositivo SÍ está configurado**: Lanza una ventana de **confirmación**: *"Se ha detectado un archivo de aprovisionamiento. ¿Deseas re-configurar esta mesa?"*. 
    - **Aceptar**: Lanza `SetupPinView` para pedir el PIN.
    - **Cancelar**: Ignora el archivo y continúa el arranque normal de Wi-Fi/MQTT (no vuelve a gastar ciclos leyendo el `.enc`).
  - **Si no existe y SÍ está configurado**: Continúa el arranque normal de Wi-Fi/MQTT.
  - **Si no existe y NO está configurado**: Muestra una **ventana/modal de instrucción** que indique: *"Dispositivo no configurado. Inserta la memoria MicroSD con el archivo tablehub.enc y reinicia"*.
- Como el proceso de "arranque normal" ahora puede ser disparado desde el botón de "Cancelar" del modal, el código de conexión a Wi-Fi y carga del Dashboard se extraerá a una función independiente (Ej: `startNormalBoot()`).

### B. Corrección de Tipos y Migración (`ConfigManager.h` y `ConfigManager.cpp`)
- Modificar ambos bloques del `#ifdef ARDUINO` en `ConfigManager.h` para que `tableId` sea `String` (o `std::string` en nativo).
- Cambiar el mock del Emulador en `ConfigManager.cpp:240` para asignar un string (`"mesa-mock"` en lugar de `1`).
- En la lectura (`loadConfig`), para **evitar corromper** a los dispositivos viejos que lo tengan guardado como entero, se usará `getString("table_id", "")`. 
- Al parsear el JSON usar `doc["table_id"].as<const char*>()`.

### C. Aprovisionamiento seguro (`ConfigManager.cpp`)
- Dado que `main.cpp` ya dejó la tarjeta inicializada (`SD_MMC.begin()`), `ConfigManager::attemptProvisioning()` ya **NO** intentará montar ni desmontar la tarjeta. Se limitará a leer `/tablehub.enc` y, si el descifrado y configuración son exitosos, **eliminará el archivo** (`SD_MMC.remove("/tablehub.enc")`). Así aseguramos que en futuros reinicios ya no se gasten ciclos en leerlo ni vuelva a preguntar nada.
- Si el usuario falla el PIN (error de descifrado), el archivo no se elimina y el usuario simplemente puede volver a intentarlo.

### D. Diagnóstico de Errores en UI y Reinicio
- Modificar la firma de `attemptProvisioning` a: `bool attemptProvisioning(const std::string& pin, String& errorMessage);` (o `std::string&` nativo).
- Asignar el mensaje real a `errorMessage` según el fallo (Ej: "El archivo tablehub.enc no existe.", "PIN incorrecto o archivo corrupto.", etc.).
- En `SetupPinView.cpp`, si el aprovisionamiento es exitoso, mostrar el mensaje de "Rebooting..." y llamar a `ESP.restart()` para que el sistema arranque desde cero con las nuevas credenciales, matando así limpiamente cualquier hilo Wi-Fi/MQTT anterior que pudiera haber existido.

## 3. Criterios de Aceptación
1. Una mesa nueva debe mostrar un modal claro pidiendo la SD.
2. Al insertar una SD con `tablehub.enc` y arrancar, la mesa DEBE pedir el PIN (SetupPinView).
3. Si el usuario se equivoca de PIN, puede volver a intentarlo.
4. El ID de la mesa (`table_id`) debe guardarse como texto (UUID).
5. Tras el aprovisionamiento correcto, la mesa se reinicia automáticamente y carga la red configurada.
6. La memoria SD queda montada globalmente para el sistema de imágenes futuro.
