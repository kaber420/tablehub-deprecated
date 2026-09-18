# Plan de Corrección: Estabilización del Hardware (MJPEG)

*Diagnóstico y plan generado en conjunto con DeepSeek-v4.*

## Diagnóstico Real de Congelamiento
Al implementar el diseño técnico original del reproductor MJPEG, la placa sufrió de asfixia y congelamientos totales. Las causas reales dictaminadas son:

1. **Asfixia de LVGL (Core 1):** La tarea `decoderTask` fue asignada erróneamente al Núcleo 1 (el mismo que usa LVGL) con la misma prioridad. Al ser la decodificación JPEG (`TJpgDec`) un proceso síncrono y bloqueante, saturó el núcleo y ahogó la interfaz gráfica por completo.
2. **Colisión de Bus SPI:** El Núcleo 0 (reproductor de video) y el Núcleo 1 (gestor de activos LVFS de LVGL) leen la MicroSD al mismo tiempo sin ningún mecanismo de sincronización. Al compartir el mismo bus SPI sin protección, causan corrupción de memoria y congelamientos duros de hardware.
3. **Bloqueo Bruto en Transición:** La función `stop()` del reproductor introdujo un `delay(100)` bloqueante. Esto congela la interfaz por 100ms exactamente en el momento más crítico: cuando el usuario toca la pantalla para salir del salvapantallas.

## Plan de Acción (Corrección Definitiva)

### 1. Aislamiento de Núcleos (Core Isolation)
- **Modificación en `MJPEGPlayer.cpp`:** Cambiar la asignación de `decoderTask` para que corra exclusiva y completamente en el **Core 0** (el mismo núcleo donde opera el lector de SD). 
- **Objetivo:** Liberar el Core 1 al 100% para que LVGL procese toques de pantalla y dibuje la interfaz a 60FPS sin interrupciones del video.

### 2. Implementar Candado de Hardware (SPI Mutex)
- **Modificación en `main.cpp`:** Declarar un `SPI_MUTEX` global usando `xSemaphoreCreateMutex()`.
- **Modificación en `MJPEGPlayer.cpp` y `LVFS_Driver.cpp`:** Cualquier núcleo o tarea que necesite acceder a la MicroSD (ya sea LVGL para cargar íconos, o el reproductor para cargar video) deberá solicitar este Mutex antes de llamar a `SD.open()` o `read()`.
- **Objetivo:** Prevenir colisiones físicas en el hardware y corrupción de memoria.

### 3. Reemplazar Tiempos Muertos (Delays)
- **Modificación en `MJPEGPlayer.cpp`:** Eliminar el `delay(100)` de la función `stop()` y reemplazarlo por mecanismos nativos asíncronos (`xTaskNotifyWait` o *flags* no bloqueantes).
- **Objetivo:** Detener los hilos de manera limpia e instantánea sin congelar el sistema al tocar la pantalla.

### 4. Sincronización de Trama (Frame Sync)
- **Objetivo:** El decodificador (Core 0) esperará a que LVGL (Core 1) termine de dibujar el frame actual en pantalla mediante un semáforo, antes de lanzar y sobreescribir el siguiente fotograma, evitando cuellos de botella de memoria y pérdida de frames.

---
**Nota para Desarrollo:** Si este plan es aprobado, se procederá a implementar únicamente estos cambios para estabilizar el sistema de video.

---

## Evolución (2026-07-29)

### Auditoría Completa y Reescritura

El plan original de 4 puntos fue auditado en profundidad. Se encontraron **7 bugs fundamentales** (vs. los 3 diagnosticados originalmente). Los bugs adicionales descubiertos fueron:

4. **Sobredimensionamiento de Buffers:** Triple buffer (3×307KB = 921KB) reducido a doble buffer (2×307KB = 614KB). Ahorro de ~307KB PSRAM.
5. **Sin loop del video:** `sdReaderTask` moría al EOF. Se implementó loop automático.
6. **Delays innecesarios:** `vTaskDelay(5ms)` y `vTaskDelay(2ms)` eliminados del reader; target FPS ajustado de 20→15 FPS.
7. **Variable estática `lastDisplayedFrameIdx`:** Movida a miembro de clase para resetear entre sesiones.

### Archivos Modificados
- `Video/MJPEGPlayer.h` — Reducción de buffers, mutex SPI, flags de stop, miembro lastDisplayedIdx
- `Video/MJPEGPlayer.cpp` — Reescritura completa: core isolation, SPI mutex, stop limpio, loop, delays eliminados
- `Core/LVFS_Driver.h` — Declaración de `lv_fs_set_spi_mutex()`
- `Core/LVFS_Driver.cpp` — Todas las operaciones SD wrapeadas con mutex SPI
- `main.cpp` — Creación de mutex global e inyección a LVFS y MJPEGPlayer
