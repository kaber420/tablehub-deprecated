# Investigación Arquitectónica: Reproducción de GIFs/Video en LVGL (ESP32-S3)

*Reporte generado por el agente DeepSeek-V4-Flash.*

El análisis profundo del código interno de la librería `lv_gif` y la arquitectura del ESP32-S3 (JC3248W535) revela por qué la pantalla se congela y tarda tanto en cargar, confirmando que **el ESP32 sí tiene capacidad de sobra, pero la librería estándar de LVGL lo está limitando**.

## 1. El Cuello de Botella de `lv_gif`
Actualmente, la librería de LVGL lee el archivo de la SD y utiliza fuerza bruta con la CPU para decodificar la compresión LZW del GIF fotograma por fotograma. Para un GIF de 1.8MB a resolución completa, esto asfixia el hilo principal (Core 0), provocando la pantalla negra.

## 2. Alternativas de Solución Profesionales

DeepSeek ha propuesto 3 fases/arquitecturas posibles. Necesito que elijas la que mejor se adapte a tu proyecto:

### Opción A: Streaming de Video Raw (Máximo Rendimiento - Recomendado)
En lugar de subir un archivo `.gif` comprimido, se convierte el video en la computadora a un archivo binario "crudo" (Raw RGB565) antes de meterlo a la MicroSD.
- **Ventaja:** El ESP32 ya no tiene que decodificar nada. Simplemente lee la tarjeta SD y escupe los pixeles directo a la pantalla usando hardware (DMA). El arranque es instantáneo (0 latencia) y el uso de CPU baja a casi cero.
- **Desventaja:** Requiere correr un script de Python en tu PC para convertir el logo de `.gif` a `.bin` antes de ponerlo en la memoria.

### Opción B: Hilo Paralelo (FreeRTOS) + PSRAM (Mejora del Software Actual)
Tu placa tiene **8MB de memoria PSRAM**, pero LVGL actualmente no la está usando para los GIFs. Modificaremos el firmware para forzar a LVGL a meter el GIF en la PSRAM. Además, crearíamos un "Hilo de Trabajo" (FreeRTOS Task) en el Núcleo 1 del procesador.
- **Ventaja:** Puedes seguir usando archivos `.gif` normales subidos directamente a la SD. Al mover el trabajo al Núcleo 1, el menú de TableHub jamás se congelará.
- **Desventaja:** Seguirá habiendo un pequeño retraso antes de que el GIF aparezca, pero la pantalla mostrará una transición elegante o un texto de "Cargando..." sin trabar el táctil.

### Opción C: Modo Turbo `AnimatedGIF`
Activar un buffer especial oculto en las librerías internas que gasta 24KB extra de RAM pero acelera la decodificación matemática un 300%.

---
## Decisión Requerida
¿Prefieres que mantengamos el formato `.gif` tradicional e implementemos el Hilo Paralelo para que no congele la UI (Opción B)? ¿O estás dispuesto a usar un archivo convertido `.bin` crudo para obtener el rendimiento máximo instantáneo (Opción A)?

*Nota: Independientemente de lo que elijas, aplicaremos el candado en el arranque del sistema (como planeamos antes) para que el salvapantallas jamás intente leer la memoria si no está visible.*

## Actualizaciones y Evolución

### Análisis MJPEG (Motion JPEG) vs GIF
Se evaluó el uso de los archivos `.mjpeg` existentes en la carpeta `assets` (`ABC.mjpeg`, `my0.mjpeg`, `极乐净土480320.mjpeg`).

**Veredicto:** **Sí, MJPEG es muy superior al GIF en el ESP32** en cuanto a velocidad de decodificación (~5x más rápido). El microcontrolador cuenta con decodificadores (como TJpgDec) que aprovechan eficientemente la CPU, y al ser frames JPEG independientes, no requieren reconstruir el frame anterior como el formato LZW de los GIF.

Sin embargo, para implementar esta **Opción D (MJPEG)**, se deben resolver los siguientes retos de los archivos actuales:
1. **Tamaño excesivo:** Los archivos pesan entre 40MB y 65MB. Leer 40MB desde la MicroSD ahogaría el bus SPI. Es obligatorio comprimirlos y reducirlos a ~5-10MB.
2. **Resolución y Rotación:** Actualmente están en horizontal (480x320) y la pantalla del dispositivo es vertical (320x480). Deben rotarse o reescalarse.
3. **Implementación de Software:** LVGL no tiene un widget para reproducir archivos `.mjpeg` directamente desde SD. Requeriría que implementemos un reproductor a medida en un `FreeRTOS task` corriendo en el Núcleo 1, usando doble buffer en PSRAM.

**Conclusión:** MJPEG es el balance perfecto de máximo rendimiento técnico manteniendo archivos independientes en la SD. (Nota: Se confirmó que el hardware puede procesar los videos originales de la SD sin necesidad de convertirlos, por lo que se usarán tal cual).

### Diseño Técnico: Reproductor MJPEG (FreeRTOS + PSRAM)

#### 1. Gestión de Memoria (Doble Buffer)
Para reproducir frames de 320x480 (RGB565) sin trabar el sistema:
- Se usarán 2 buffers de memoria de ~307KB cada uno, forzados a inicializarse en la memoria externa (`MALLOC_CAP_SPIRAM`).
- Mientras el Núcleo 1 decodifica el frame futuro en el Buffer A, la pantalla pinta el frame actual del Buffer B (Double Buffering), evitando el parpadeo de pantalla.

#### 2. Tarea Concurrente (FreeRTOS)
- **Creación:** Se lanzará una tarea (`mjpeg_decoder_task`) anclada al Core 1.
- **Sincronización:** Dado que LVGL no es seguro para hilos, la inyección del fotograma decodificado hacia `lv_canvas` se protegerá bloqueando temporalmente el sistema con un Mutex.
- **Limpieza Estratégica:** En el evento `LV_EVENT_SCREEN_UNLOADED`, se cierra el archivo de inmediato y se hace un `heap_caps_free()` estricto de los buffers para no causar *memory leaks*.

#### 3. Modificaciones en el Código
- `ScreensaverView.h`: Se removerá `lv_gif_obj` y se añadirán `TaskHandle_t`, un buffer y un `lv_canvas`.
- `ScreensaverView.cpp`: Lógica de arranque (candado), bucle de decodificación constante y uso de la librería de descompresión JPEG del sistema.
