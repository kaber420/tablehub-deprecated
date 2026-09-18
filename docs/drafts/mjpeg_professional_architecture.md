# Arquitectura Profesional: Reproductor MJPEG Acelerado en ESP32-S3

*Documento técnico elaborado a partir de las revisiones de DeepSeek-V4-Flash y Mimo-v2.5.*

## 1. El Problema de la Implementación Ingenua
Un enfoque básico (leer SD -> decodificar -> dibujar con un Mutex) fracasaría estrepitosamente en el ESP32-S3 con LVGL 9.5 debido a:
- **Latencia I/O:** El bus SPI de la SD a 4MHz (por defecto) solo da ~1.5 MB/s, lo cual asfixiaría archivos `.mjpeg` pesados.
- **Overhead Gráfico:** `lv_canvas` añade una carga de renderizado masiva, inviable para 30fps.
- **Kernel Panics:** LVGL no es thread-safe. Usar Mutex desde otra tarea interrumpe el ciclo principal y corrompe la memoria.
- **Memory Leaks / Crashes:** Liberar memoria (`heap_caps_free`) sin coordinar la detención exacta de las tareas asíncronas provoca fallos fatales de *use-after-free*.

---

## 2. Solución: Pipeline de 3 Etapas con Búferes Circulares (Ring Buffers)

La arquitectura debe desacoplar completamente la lectura de la tarjeta SD, la decodificación de la CPU (SIMD), y el renderizado en pantalla, utilizando **3 etapas conectadas por colas (`xQueue`)**.

### Etapa 1: Lector SD Dedicado (Task en Core 1 o Core 0)
- **Función:** Leer fragmentos de 16KB-32KB directamente del archivo en la SD a la PSRAM lo más rápido posible.
- **Mejora Crítica:** Se debe forzar el bus SPI a **20MHz** durante la inicialización de la SD (`SD.begin(CS_PIN, spi, 20000000);`) para lograr ~5-6 MB/s y soportar los archivos `.mjpeg` existentes sin cortes.
- **Salida:** Alimenta un Ring Buffer de fragmentos "crudos".

### Etapa 2: Decodificador JPEG (Task en Core 1)
- **Función:** Tomar los fragmentos JPEG del Ring Buffer y alimentarlos a un decodificador altamente optimizado (`esp_jpeg` o `TJpg_Decoder` con `LV_USE_SJPG=1`).
- **Memoria PSRAM:** Se asignarán 3 buffers de fotograma completo en PSRAM (320x480 RGB565 = ~307KB × 3 = ~921KB). Usar un esquema de Triple Buffer asegura que el decodificador nunca espere a la pantalla.
- **Salida:** Envía punteros (referencias de memoria) a través de un `xQueue` hacia LVGL, sin copiar los datos masivos.

### Etapa 3: Motor Gráfico LVGL (Core 1 - Bucle Principal)
- **Función:** Mostrar los fotogramas en pantalla.
- **Implementación Segura:** En LVGL 9.5, se utilizará el widget `lv_image` configurado con `LV_IMG_SRC_VARIABLE`. 
- **Thread-Safety:** En lugar de usar un Mutex, la tarea de decodificación invocará `lv_async_call()` para notificar a LVGL de forma segura que un nuevo fotograma está listo, o LVGL revisará el `xQueue` en cada ciclo de su `lv_timer_handler`.

---

## 3. Requisitos de Memoria Exactos (PSRAM Obligatorio)

La placa JC3248W535 cuenta con 8MB de PSRAM. Se reservarán aproximadamente:
- **Buffers de lectura SD (Read-ahead):** 64 KB (4 chunks de 16KB).
- **Memoria de trabajo del Decodificador JPEG:** ~32 KB.
- **Triple Buffer de Fotogramas (RGB565):** ~921 KB.
- **Total Dedicado al Video:** ~1.01 MB (perfectamente asumible).
- **Regla:** Toda asignación se hará con `heap_caps_malloc(size, MALLOC_CAP_SPIRAM)`.

---

## 4. Estrategia de Limpieza Segura (Teardown)
Al dispararse el evento de que el salvapantallas debe ocultarse (`LV_EVENT_SCREEN_UNLOADED`), la secuencia debe ser:
1. Enviar una señal atómica (`TaskNotify` o `EventGroup`) indicando aborto a la Etapa 1 y Etapa 2.
2. Esperar (timeout controlado) a que ambas tareas confirmen su terminación (`vTaskDelete`).
3. Ejecutar `fclose()` sobre el archivo MJPEG.
4. Realizar `heap_caps_free()` sobre los Ring Buffers y los buffers de fotogramas. Solo así se evita corromper la SD o causar Kernel Panics.

---

## 5. Próximos Pasos en el Código

1. Modificar la inicialización del SPI en `LVFS_Driver` o `main` para maximizar el reloj a 20MHz.
2. Instalar/Habilitar un decodificador nativo (`esp_jpeg`).
3. Reemplazar completamente la clase actual `ScreensaverView` para alojar las tareas de FreeRTOS y el widget `lv_image`.
