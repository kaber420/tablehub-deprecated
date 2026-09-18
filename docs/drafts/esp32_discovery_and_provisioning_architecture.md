# Arquitectura de Descubrimiento Multicapa y Aprovisionamiento ESP32 (v0.1.x Prototipo & v0.2.x Prod)

**Fecha:** 2026-07-21  
**Módulo:** Firmware ESP32 & Hub Network Discovery  
**Estado:** Especificación Técnica & Arquitectura de Resiliencia  

---

## 1. Contexto y Problema

En entornos de restauración reales, la topología de red varía drásticamente:
* **Restaurantes Pequeños:** Router doméstico sin aislamiento de clientes; mDNS funciona perfectamente.
* **Restaurantes Medianos / Cadenas:** Routers empresariales (Ubiquiti/Mikrotik) con **Aislamiento de Clientes (Client Isolation)** y bloqueo de tráfico Multicast (mDNS).
* **Restaurantes Grandes / Complejos:** Redes ruteadas con múltiples subredes/VLANs (ej. Hub en `192.168.1.X` y Puntos de Acceso Wi-Fi en `192.168.10.X`), donde el tráfico mDNS/Multicast se descarta en el gateway.

Para garantizar que el ESP32 **siempre** encuentre al Hub local bajo cualquier condición de red sin intervención técnica compleja, se especifica una **Arquitectura de Descubrimiento en Cascada (Multi-Fallback)**.

---

## 2. Estrategia de Descubrimiento en Cascada (Fallback Cascade)

Cuando el ESP32 se conecta a la red Wi-Fi del establecimiento, ejecuta la secuencia de descubrimiento en orden jerárquico:

```
                  +-----------------------------------+
                  |   ESP32 Conectado al Wi-Fi Local   |
                  +-----------------+-----------------+
                                    |
                                    v
                  +-----------------------------------+
                  |  Nivel 1: IP Estática o Guardada  |
                  |  ¿Responde el Hub en esta IP?     |
                  +--------+-----------------+--------+
                           | No              | Sí
                           v                 v
                  +-------------------+    +--------------------+
                  | Nivel 2: mDNS     |    | ¡Conexión Exitosa! |
                  | ("tablehub.local")|    | Establecer MQTTS   |
                  +--------+----------+    +--------------------+
                           | No              ^
                           v                 | Sí
                  +-------------------+      |
                  | Nivel 3: UDP      |------+
                  | Broadcast (9999)  |
                  +--------+----------+
                           | No
                           v
                  +-----------------------------------+
                  | Nivel 4: Fallback de Error        |
                  | Reabrir AP o Mostrar "Sin Hub"    |
                  +-----------------------------------+
```

---

### Detalles de cada Nivel de Descubrimiento

### 📍 Nivel 1: IP Directa / Estática / Hub AP (Tiempo de resolución: ~100ms)
* **Mecanismo:** El ESP32 intenta conectarse primero a la IP explícita guardada en su memoria NVS (configurada opcionalmente durante el aprovisionamiento web) o a la IP predeterminada si el Hub actúa como su propio AP (`192.168.4.1`).
* **Ventaja:** Conexión casi instantánea sin saturar la red con peticiones de búsqueda.

### 📍 Nivel 2: mDNS - Multicast DNS (Tiempo de resolución: ~1 - 2s)
* **Mecanismo:** El ESP32 realiza una consulta mDNS buscando el servicio `_tablehub._tcp.local` o el host `tablehub.local`.
* **Ventaja:** No requiere saber la IP del Hub previamente.
* **Limitación:** Falla si el router tiene *Client Isolation* activo o bloquea el puerto UDP `5353`.

### 📍 Nivel 3: UDP Broadcast (Tiempo de resolución: ~1 - 3s)
* **Mecanismo:** El ESP32 envía un datagrama UDP de difusión amplia al puerto `9999` hacia `255.255.255.255` con la estructura:
  ```json
  {"cmd": "DISCOVER_HUB", "device_mac": "A1:B2:C3:D4:E5:F6"}
  ```
  El servicio Hub en Go escucha en la red local el puerto UDP `9999` y responde directamente a la IP del ESP32 con:
  ```json
  {"status": "IAM_HUB", "hub_ip": "192.168.1.50", "mqtt_port": 8883}
  ```
* **Ventaja:** Atraviesa restricciones de Client Isolation donde mDNS es descartado.

### 📍 Nivel 4: Modo Fallback (Sin Conexión al Hub)
* **Mecanismo:** Si tras 3 reintentos en todos los niveles no hay respuesta del Hub, el ESP32:
  1. Muestra en pantalla el aviso: *"Conectado a Wi-Fi, pero Hub no encontrado"*.
  2. Ofrece un botón de *"Reconfigurar Red"* o reabre el Access Point local tras 30 segundos.

---

## 3. Plan de Aprovisionamiento por Fases (Roadmap)

### 🚀 Fase v0.1.x (Prototipo Rápido - Desacoplado de la APK)
**Objetivo:** Permitir desarrollo e integración continua del firmware ESP32 y backend Go sin esperar el desarrollo de la APK en Kotlin.

* **Aprovisionamiento:** Modo Access Point (`TableHub-Setup-XXXX`) con Servidor Web HTTP local (`http://192.168.4.1` o `http://setup.local`).
* **Interfaz de Aprovisionamiento:** Formulario web simple con:
  * Desplegable de redes Wi-Fi escaneadas.
  * Campo de contraseña Wi-Fi.
  * Campo opcional: *IP del Hub* (para omitir descubrimiento automático).
* **Descubrimiento:** Implementación de Nivel 1 (IP Directa) + Nivel 2 (mDNS) + Nivel 3 (UDP Broadcast).

### 🚀 Fase v0.2.x (Producción - APK & Cifrado Avanzado)
**Objetivo:** Máxima seguridad y cero fricción en campo.

* **Aprovisionamiento:** Protocomm (Security 1) vía App Kotlin con cifrado ECDH + AES-256 autenticado con el PIN mostrado en la pantalla TFT.
* **Tokenización:** Intercambio de `OneTimeToken` de registro único suministrado por el Hub a la APK.
* **Mantenimiento Técnico Secundario:** El portal web (v0.1) se mantiene como método de emergencia por si no se dispone de la APK.

---

## 4. Redes Dedicadas vs. Redes Compartidas (Recomendaciones)

| Escenario | Topología | Rendimiento | Recomendación |
| :--- | :--- | :--- | :--- |
| **Opción A (Ideal)** | Red Wi-Fi exclusiva para TableHub (Router dedicado o Hub en modo AP) | ⭐️⭐️⭐️⭐️⭐️ (Latencia <10ms) | **Altamente Recomendada.** Aislar el hardware de comensales previene saturación. |
| **Opción B (Compartida)** | Wi-Fi existente del restaurante | ⭐️⭐️⭐️ (Sujeto a saturación) | Usar MQTTS con reconexión asíncrona QoS 1 y cascada de descubrimiento activa. |

---

## 5. Viabilidad y Carga de Hardware (ESP32-S3)

* **Potencia Disponible:** ESP32-S3 Dual-Core 240MHz, 8MB PSRAM.
* **Impacto en Recursos:** El consumo combinado de la pila mDNS, UDP Broadcast y el servidor HTTP de aprovisionamiento representa menos del **2% de CPU y RAM**.
* **Carga Principal:** El 90% de los recursos de la GPU/CPU se reservan para la renderización de la interfaz LVGL (320x480 TFT Touch a 30+ FPS).
* **Justificación de Diseño:** La cascada de descubrimiento no representa *overkill* de hardware; es una optimización de software de bajo costo que previene intervenciones de soporte técnico en campo.

