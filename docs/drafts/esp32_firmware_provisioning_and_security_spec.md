# Especificación Técnica: Aprovisionamiento y Seguridad del Firmware ESP32

**Fecha:** 2026-07-21  
**Dispositivo Destino:** ESP32-S3 (Guition JC3248W535 - 3.5" 320x480 TFT Touch Capacitivo)  
**Proyecto:** TableHub2 - Módulo Firmware & Comunicación Local  

---

## 1. Resumen Ejecutivo y Objetivos

Este documento especifica la arquitectura de aprovisionamiento, seguridad de red y modelo de comunicación para los dispositivos de mesa basados en ESP32-S3.

El firmware tiene tres responsabilidades principales:
1. **Aprovisionamiento Wi-Fi Seguro:** Enlazar el dispositivo a la red Wi-Fi local del restaurante de forma cifrada y sin transmitir credenciales en texto plano.
2. **Autenticación con el Hub:** Conectarse al backend local (Hub) mediante un token de uso único y establecer un canal MQTTS aislado y seguro.
3. **Interfaz de Servicio (LVGL):** Mostrar las órdenes de la mesa y permitir acciones rápidas (Llamar Mesero, Pedir Cuenta), con lógica de escalamiento gestionada por el Hub.

---

## 2. Arquitectura de Aprovisionamiento Wi-Fi (DEPRECADA)

> [!WARNING]
> **ATENCIÓN: Esta sección (Aprovisionamiento por AP y App Kotlin) ha sido DEPRECADA.** 
> Se ha decidido unificar la estrategia del Hub (`.thub`) con la autenticación del ESP32. El nuevo flujo **elimina por completo el Access Point** y utiliza una **MicroSD con un archivo cifrado (`tablehub.enc`) y un teclado numérico en pantalla (PIN Pad)**.
> 
> 👉 **Para ver la nueva arquitectura de aprovisionamiento, consulta el plan oficial:** 
> [Plan de Aprovisionamiento Seguro por PIN y MicroSD](file:///home/kaber420/Documentos/proyectos/tablehub2/docs/drafts/provisioning_pin_plan.md)

Para propósitos de archivo histórico, el plan original basado en Protocomm era el siguiente:

Para evitar la interceptación de credenciales Wi-Fi por el aire en redes abiertas, se implementa el estándar **ESP Provisioning (Protocomm - Security 1)** con cifrado de curva elíptica y prueba de posesión por PIN.

```
+------------------+                    +--------------------+
|  ESP32-S3 (AP)   |                    | App Kotlin (Mesero)|
|  SSID: TableHub- |                    |                    |
|        Setup-A1B2|                    |                    |
|  PIN:  [ 8492 ]  |                    |                    |
+--------+---------+                    +---------+----------+
         |                                        |
         |  1. Escaneo AP & Conexión WPA2/Abierto |
         |<---------------------------------------+
         |                                        |
         |  2. Handshake ECDH (Curva Curve25519)  |
         |     usando PIN '8492' como Sal         |
         |<======================================>|
         |  [Clave Simétrica AES-256 Generada]    |
         |                                        |
         |  3. POST HTTP (Payload Cifrado AES)    |
         |     {SSID, Password, OneTimeToken}     |
         |<---------------------------------------+
         |                                        |
         |  4. Cierre de AP & Conexión a Wi-Fi    |
         v                                        v
```

### 2.1. Componentes del Modo Aprovisionamiento (AP)
* **SSID del AP:** `TableHub-Setup-XXXX` (donde `XXXX` son los últimos 4 dígitos de la MAC del ESP32).
* **Pantalla de Estado (LVGL):**
  * Título: `TableHub | Configuración Inicial`
  * Nombre del AP visible en pantalla.
  * **PIN de Seguridad (4 o 6 dígitos):** Generado de forma aleatoria al arrancar en modo AP (ej. `8492`).
  * Instrucciones visuales para el operador de la App Kotlin.

### 2.2. Proceso Criptográfico (Handshake ECDH + AES-256)
1. **Acuerdo de Claves (ECDH):** La App Kotlin y el ESP32 intercambian llaves públicas efímeras.
2. **Prueba de Posesión (PoP):** El PIN visible en la pantalla actúa como derivador de clave (PBKDF2/HMAC) para autenticar la negociación.
3. **Canal Seguro:** Se deriva una clave simétrica **AES-GCM de 256 bits** válida solo para esa sesión.
4. **Envío de Credenciales:** La App transmite el payload JSON cifrado. Nadie que escuche los paquetes en el aire con un sniffer Wi-Fi puede descifrar la clave del Wi-Fi ni el token.

---

## 3. Enlace y Seguridad ESP32 ↔ Hub Local (NATS MQTT)

Una vez que el ESP32 se conecta al Wi-Fi del restaurante, debe identificarse y conectarse con el Hub local.

### 3.1. Enlace por Token de Uso Único (One-Time Registration Token)
1. Durante el aprovisionamiento, la App Kotlin (autenticada con el Hub) solicita un `OneTimeToken` al Hub para la mesa deseada (ej. `Mesa 4`).
2. La App incluye este `OneTimeToken` dentro del paquete cifrado que le envía al ESP32.
3. Al conectarse al Wi-Fi, el ESP32 busca al Hub (vía mDNS o IP suministrada) y realiza la petición de registro con dicho token.
4. El Hub valida el `OneTimeToken`, **lo inhabilita/destruye inmediatamente** y le otorga al ESP32 sus credenciales permanentes de dispositivo (Token NKEYS o usuario/password MQTT exclusivo).

### 3.2. Protocolo de Transporte y Cifrado
* **Protocolo:** MQTT 3.1.1 / 5.0 sobre **TCP Puro** (Puerto `8883`). *Nota: Se evita WebSockets en el ESP32 para minimizar uso de memoria RAM y overhead de CPU.*
* **Cifrado de Capa de Transporte:** **MQTTS (TLS 1.2/1.3)** con el certificado CA del Hub cargado en el ESP32.
* **Broker:** NATS (módulo nativo `mqtt` habilitado).

### 3.3. Autorización y Aislamiento por Mesa (NATS ACLs)
NATS restringe los permisos de publicación y suscripción para cada ESP32 de forma estricta:
* **Suscripción permitida (Mesa N):** Únicamente a `table/{N}/order` y `table/{N}/status`.
* **Publicación permitida (Mesa N):** Únicamente a `table/{N}/action`.
* **Resultado de Seguridad:** Incluso si un dispositivo en la red es comprometido, no puede leer el tráfico de otras mesas ni enviar comandos no autorizados a la red del restaurante.

---

## 4. Flujo de Negocio: Llamada a Mesero y Escalamiento

```
+---------------+              +--------------+              +------------------+
|  ESP32 (Mesa) |              |  Hub (Go)    |              | APK Mesero Asig. |
+-------+-------+              +------+-------+              +--------+---------+
        |                             |                               |
        | 1. MQTT: "call_waiter"      |                               |
        +---------------------------->|                               |
        |                             | 2. Notificación Directa       |
        |                             +------------------------------->|
        |                             |                               |
        |                             | 3. Inicia Timer 5 min         |
        |                             |    (Goroutine memory timer)   |
        |                             |                               |
        |                             |--- Temp. 5 min expirado ----->|
        |                             |                               | (No respondió)
        |                             | 4. Broadcast Notificación     |
        |                             |    a TODOS los meseros        |
        |                             +------------------------------>| (Cualquier mesero
        |                             |                               |  puede acudir)
```

1. **Invocación:** El usuario presiona "Llamar Mesero" en la pantalla LVGL del ESP32.
2. **Mensaje:** El ESP32 publica en `table/{ID}/action` el payload `{"action": "call_waiter"}`.
3. **Despacho Primario:** El Hub local consulta la base de datos, identifica al mesero asignado a la mesa y le envía una notificación prioritaria a su APK (vía WebSockets / Push local).
4. **Temporizador de Escalamiento:** El Hub arranca un temporizador en memoria (ej. 5 minutos).
5. **Resolución / Escalamiento:**
   * **Caso A (Atendido a tiempo):** El mesero asignado presiona "Atender" en la APK. El Hub cancela el temporizador y notifica al ESP32 para cambiar el estado visual.
   * **Caso B (Timeout de 5 min):** Si el temporizador expira sin respuesta del mesero asignado, el Hub emite un evento general a **todas las APKs de los meseros conectados** indicando *"Mesa X requiere atención urgente"*.

---

## 5. Resumen de Tecnologías y Librerías

| Componente | Tecnología / Librería |
| :--- | :--- |
| **Hardware** | ESP32-S3 (Guition JC3248W535 - 3.5" TFT 320x480 Capacitivo) |
| **Framework ESP32** | PlatformIO + Arduino / ESP-IDF |
| **Interfaz Gráfica** | LVGL v9.x (con driver TFT_eSPI / LovyanGFX / ESP-Host) |
| **Aprovisionamiento** | `wifi_provisioning` (Protocomm Security 1 - ECDH + AES-256) |
| **App Móvil (Android)** | Kotlin + `esp-idf-provisioning-android` SDK |
| **Cliente MQTT ESP32** | `PubSubClient` o `async-mqtt-client` sobre TLS (`WiFiClientSecure`) |
| **Broker Mensajería** | NATS Server (Plugin MQTT activo en puerto TLS 8883) |
