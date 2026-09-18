# Especificación de Arquitectura: Encendido Instantáneo y Offline-First (ESP32-S3)

## 1. Visión General

Esta arquitectura define el modelo de funcionamiento de la mesa inteligente TableHub basada en el procesador dual-core **ESP32-S3**. 

El objetivo primordial es garantizar una **experiencia de usuario de gama alta con encendido instantáneo (0 segundos de espera)** y **resiliencia total ante fallos de red en el restaurante**.

---

## 2. Principios Fundamentales

### A. Separación de Responsabilidades: Interfaz Local vs. Transacciones de Red
- **La MicroSD es la Fuente de la Verdad para la Interfaz:** Todos los activos de la experiencia visual (el archivo de catálogo `/catalog_manifest.msgpack` y las imágenes comprimidas en `/assets`) residen físicamente en la tarjeta MicroSD.
- **La Red es un Canal de Transacción:** El WiFi y el protocolo MQTT se utilizan **exclusivamente** cuando el cliente ejecuta una acción transaccional en tiempo real (enviar pedido a cocina, pedir la cuenta o solicitar la presencia del mesero).

### B. Encendido Instantáneo (Instant-Boot Sequence)
- Al encender o reiniciar la mesa, el sistema **NO espera a la red WiFi ni a la conexión MQTT** para mostrar la interfaz gráfica.
- **Secuencia de Arranque:**
  1. **Inicio de Pantalla (Núcleo 1):** Lee inmediatamente el catálogo e imágenes locales de la MicroSD y despliega el Dashboard y el Menú en < 1 segundo.
  2. **Conexión de Red en Segundo Plano (Núcleo 0):** Inicializa el WiFi y la conexión MQTT de forma transparente sin interferir con la navegación táctil del usuario.

### C. Eliminación de Dependencias de Red Dinámicas (Depreciación de mDNS y Discovery)
- **Eliminación del Módulo Discovery:** El escaneo dinámico por UDP Broadcast y la librería `ESPmDNS` quedan retirados de la arquitectura por las siguientes razones:
  - Los routers empresariales en restaurantes suelen aplicar *Client Isolation* o filtrar tráfico Multicast/Broadcast por seguridad.
  - La búsqueda dinámica introduce retrasos innecesarios de 2 a 5 segundos durante el arranque.
- **Aprovisionamiento Determinista:** El archivo cifrado `tablehub.enc` grabado en la MicroSD suministra los datos exactos de red (SSID, Password, IP o host del Hub, puerto y número de mesa), logrando una conexión directa y sin incertidumbres.

---

## 3. Modelo de Concurrencia Dual-Core (FreeRTOS)

| Recurso | Asignación de Núcleo | Responsabilidad |
| :--- | :--- | :--- |
| **Núcleo 1 (Core 1)** | UI Thread (LVGL) | Decodificación de imágenes, renderizado gráfico a 60 FPS, gestión táctil y flujo de navegación. |
| **Núcleo 0 (Core 0)** | Network Task | Mantenimiento de la pila WiFi, reconexión MQTT y recepción de actualizaciones en segundo plano. |
| **Bus SPI (Hardware Shared)** | Mutex Lock (`lv_fs_spi_lock`) | Candado de hardware que arbitra el acceso a la tarjeta MicroSD entre hilos para evitar colisiones. |

---

## 4. Beneficios del Diseño

1. **Sensación Térmica de Velocidad:** La mesa responde como un dispositivo electrónico dedicado e instantáneo.
2. **Tolerancia a Fallos:** Si el WiFi se cae o el router tarda en asignar IP por DHCP, la pantalla jamás se congela ni se muestra en blanco; la navegación por el menú continúa estando 100% disponible.
3. **Optimización de Recursos:** Menor consumo de memoria RAM y Flash al retirar librerías de descubrimiento dinámico.

---

## 5. Evolución y Refinamiento Técnico de Hardware (Guition JC3248W535)

Con base en la evaluación de viabilidad realizada con **Mimo-2.5** y **DeepSeek-V4**, se integran las siguientes precisiones técnicas específicas para la placa Guition JC3248W535 (ESP32-S3 + AXS15231B 320x480 + Touch FT5336 + MicroSD):

### A. Clarificación de Buses de Hardware Independientes (QSPI vs SPI)
- **Pantalla (AXS15231B):** Utiliza un bus dedicado **QSPI (Quad SPI)** (CLK: 47, DATA: [21, 48, 40, 39], CS: 45).
- **Táctil (FT5336):** Utiliza bus **I2C** (SDA: 4, SCL: 8).
- **Tarjeta MicroSD:** Utiliza un bus **SPI estándar** dedicado.
- **Consecuencia de Hardware:** La pantalla y la MicroSD no comparten el bus físico SPI en esta placa. Por lo tanto, el mutex `lv_fs_spi_lock` no es requerido para arbitrar el bus entre pantalla y SD, lo que elimina cuellos de botella de contención de hardware directo en la pantalla.

### B. Estrategia de Memoria PSRAM y Framed Buffer (LVGL)
- **Presupuesto de Memoria:** A 320x480 de resolución (RGB565 16-bit), un buffer de frame requiere ~307 KB. Un esquema de doble búfer de LVGL asciende a **~614 KB**, excediendo la memoria SRAM interna del ESP32-S3 (~512 KB total disponible).
- **Asignación Obligatoria en PSRAM:**
  - Los buffers de dibujado y framebuffers de LVGL deben inicializarse explícitamente en **PSRAM externa**.
  - La memoria SRAM interna se reserva para stacks de tareas FreeRTOS de alta prioridad y colas de eventos con bajo jitter.
  - La pantalla AXS15231B requiere modo **Full-Refresh** en LVGL bajo QSPI.
  - Se establece un pool de **Caché LRU de activos en PSRAM** para almacenar en memoria las imágenes de menú de mayor frecuencia y evitar re-lecturas constantes desde la MicroSD durante la navegación del cliente.

### C. Optimización de Decodificación de Activos Visuales
- Dado que el ESP32-S3 carece de decodificador JPEG por hardware, decodificar JPEGs por software en tiempo real en Core 1 puede generar caídas de cuadros (*stuttering*).
- **Formato Recomendado:** Preferir assets almacenados en la MicroSD en formato **RAW RGB565** o comprimidos con **QOI / TJpgDec**.
- **Desacoplamiento de Carga:** En caso de requerir decodificación pesada, la tarea de descompresión se procesa en **Core 0 (Worker Task)** notificando a Core 1 mediante punteros en PSRAM.

### D. Secuencia de Arranque (Splash Screen Inmediato)
- Para garantizar una sensación térmica de encendido instantáneo (0.0s), se implementa un **Splash Screen ligero precargado en la memoria Flash NVS/SPIFFS interna**.
- Al recibir energía, Core 1 renderiza de inmediato la pantalla de inicio desde la Flash interna (0s) mientras en segundo plano monta el sistema de archivos FatFS en la MicroSD, descifra `tablehub.enc` y deserializa `/catalog_manifest.msgpack` (~750ms - 1s).

### E. Pila de Software y Drivers
- Usar como driver de pantalla `Arduino_GFX` (v1.5.0+) o `esp_lcd` nativo de ESP-IDF (evitando `TFT_eSPI` por falta de soporte QSPI para AXS15231B).
- Cifrado de `tablehub.enc` utilizando `mbedtls` (AES-256-GCM) con clave derivada del Hardware Unique ID / MAC del ESP32-S3.

