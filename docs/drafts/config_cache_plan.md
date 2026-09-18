# Plan de Optimización de Rendimiento: ConfigManager Cache

## Descripción del Problema
Actualmente, la pantalla tarda demasiado en cargar y mostrar la interfaz gráfica. Tras investigar, se identificó que la función `loop()` en `main.cpp` está llamando a `ConfigManager::getInstance().loadConfig(config)` miles de veces por segundo. Esta función abre, lee y cierra la memoria flash NVS del ESP32 en cada llamada, bloqueando el procesador e impidiendo que el motor gráfico (LVGL) renderice los cuadros a tiempo.

## Cambios Propuestos

El objetivo de este plan es evitar el acceso constante a la memoria Flash, introduciendo una pequeña memoria caché en RAM.

### 1. Componente: Administrador de Configuración (ConfigManager)

#### [MODIFY] `firmware/src/Network/ConfigManager.h`
- Agregar propiedades privadas para almacenar en memoria RAM el último estado leído:
  ```cpp
  private:
      DeviceConfig _cachedConfig;
      bool _isLoaded = false;
  ```

#### [MODIFY] `firmware/src/Network/ConfigManager.cpp`
- **En `loadConfig()`:** Verificar si `_isLoaded` es verdadero. Si lo es, devolver los valores directamente desde `_cachedConfig` sin consultar `preferences.begin()`. Si es falso, leer de la NVS normalmente y guardar los resultados en la caché.
- **En `saveConfig()`:** Tras guardar en NVS, actualizar también `_cachedConfig` y marcar `_isLoaded` como `true`.
- **En `clearConfig()`:** Reiniciar la caché y poner `_isLoaded` a `false`.

## Open Questions / User Review Required
> [!IMPORTANT]
> **Aprobación Requerida:** Este plan altera el modo en que el firmware maneja el acceso a su almacenamiento interno, lo cual afectará positivamente el rendimiento (velocidad de interfaz). Por favor, aprueba este documento para proceder con la ejecución de estos cambios.

## Plan de Verificación
1. **Verificación Automatizada:** Compilar el código con `pio run -e esp32` para confirmar que la lógica de caché no introduzca errores.
2. **Verificación Manual:** Subir el firmware modificado al ESP32 y constatar visualmente que la pantalla inicial (Dashboard o Screensaver) se carga instantáneamente sin los retrasos actuales.
