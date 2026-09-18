# Especificación Técnica: Aprovisionamiento y Seguridad del Firmware ESP32 (MicroSD + PIN)

**Fecha:** 2026-07-26
**Dispositivo Destino:** ESP32-S3 (Guition JC3248W535 - 3.5" 320x480 TFT Touch Capacitivo)  
**Proyecto:** TableHub2 - Módulo Firmware & Comunicación Local  

---

## 1. Resumen Ejecutivo y Objetivos

Este documento reemplaza la arquitectura anterior basada en Protocomm/Access Point. Define la nueva arquitectura de aprovisionamiento "Offline" para los dispositivos de mesa ESP32-S3.

El firmware tiene las siguientes responsabilidades en su ciclo de vida inicial:
1. **Aprovisionamiento Wi-Fi Seguro (Offline):** Enlazar el dispositivo a la red Wi-Fi local leyendo un archivo cifrado (`tablehub.enc`) desde una tarjeta MicroSD, desencriptándolo mediante un PIN ingresado físicamente en la pantalla táctil.
2. **Autenticación con el Hub:** Conectarse al backend local (Hub) mediante un token de uso único (contenido dentro del archivo cifrado) y establecer un canal MQTTS aislado.
3. **Erradicación de Vectores de Ataque:** Al eliminar el Access Point (AP), se mitigan ataques de intermediario (MitM), fuerza bruta sobre Wi-Fi abierto y suplantación del dispositivo durante el aprovisionamiento.

---

## 2. Arquitectura de Aprovisionamiento (MicroSD + PIN Físico)

El flujo se compone de dos fases: la generación del archivo (en el servidor/Hub) y el consumo del archivo (en el ESP32).

```
+------------------+                    +--------------------+
|   Hub Local (Go) |                    |  ESP32-S3 (Mesa)   |
|                  |                    |                    |
| Genera Archivo:  |                    |                    |
| [tablehub.enc]   |                    |                    |
+--------+---------+                    +---------+----------+
         |                                        |
         |  1. Exporta archivo cifrado a MicroSD  |
         +--------------------------------------->|
                                                  |
                                                  | 2. Usuario inserta MicroSD
                                                  | 3. ESP32 arranca y detecta SD
                                                  | 4. Interfaz LVGL pide PIN
                                                  | 5. Desencriptación AES local
                                                  | 6. Borrado seguro de SD
                                                  | 7. Conexión Wi-Fi & MQTT
```

### 2.1. Estructura y Generación del Archivo (En el Hub)
* El administrador, desde el panel Svelte del Hub Local, selecciona la mesa que desea aprovisionar (ej. "Mesa 4").
* El Hub empaqueta un JSON (Payload en texto plano) con la siguiente estructura:
  ```json
  {
    "wifi_ssid": "Restaurante_Local",
    "wifi_pass": "P@ssw0rd123",
    "hub_ip": "192.168.1.100",
    "mqtt_port": 8883,
    "bootstrap_token": "abc123xyz-one-time-token",
    "table_id": 4
  }
  ```
* **Cifrado (AES-256-GCM):** El Hub genera o solicita un PIN numérico (ej. 4 o 6 dígitos). Utiliza una función de derivación de claves fuerte (PBKDF2 o Argon2) usando el PIN como secreto, para generar la llave AES-256.
* El Payload se cifra y el archivo resultante binario (`tablehub.enc`) se guarda en la MicroSD insertada en la PC del administrador.

### 2.2. Flujo de Desbloqueo en el Dispositivo (ESP32)

1. **Detección de Hardware:** En el método `setup()`, el ESP32 inicializa el bus SPI/SD_MMC. Si no encuentra configuración previa en su memoria NVS, busca el archivo `tablehub.enc` en la raíz de la SD.
2. **Pantalla de Seguridad (LVGL - `SetupPinView`):** 
   * Si el archivo existe, la interfaz muestra un teclado numérico táctil (PIN Pad) a pantalla completa.
   * Se bloquean todas las demás vistas del UIManager.
3. **Verificación Local (mbedtls):** 
   * El usuario ingresa el PIN proporcionado por el administrador.
   * El ESP32 deriva la llave (PBKDF2) y usa el motor AES acelerado por hardware para intentar desencriptar `tablehub.enc`.
4. **Validación y Acción:**
   * **Error:** Si la etiqueta GCM o el padding fallan, se muestra un error visual (Toast rojo) y se limpia el PIN pad.
   * **Éxito:** El JSON es parseado (`ArduinoJson`). Las credenciales (`wifi_ssid`, `wifi_pass`, etc.) se guardan en la memoria NVS cifrada del ESP32.
5. **Autodestrucción:** Por seguridad, el ESP32 borra (`SD.remove()`) el archivo `tablehub.enc` de la tarjeta para evitar que alguien extraiga la MicroSD y saque la contraseña del Wi-Fi en otra máquina.
6. **Reinicio/Conexión:** El ESP32 aplica la configuración, conecta al Wi-Fi y procede al Handshake MQTTS con el Hub usando el `bootstrap_token`.

---

## 3. Integración con Base de Código Existente

Para llevar esto a cabo sin romper la arquitectura, se realizarán las siguientes modificaciones en el firmware:

### A. Modificaciones de UI (Carpetas `src/UI/Views/`)
* Eliminar cualquier referencia a pantallas de "Modo AP".
* Crear `SetupPinView.cpp / .h`:
  * Matriz de botones (`lv_btnmatrix`) para los números 0-9, Backspace y Enter.
  * Área de texto superior para mostrar los asteriscos (`****`).
  * Llamada al `ConfigManager::getInstance().attemptProvisioning(pin_str)`.

### B. Modificaciones de Red y Configuración (`src/Network/ConfigManager`)
* Importar `<SD.h>` o `<SD_MMC.h>`.
* Importar `mbedtls/aes.h`, `mbedtls/gcm.h` y `mbedtls/pkcs5.h` (para PBKDF2).
* Nueva función `attemptProvisioning`:
  * Abre archivo, lee binario.
  * Deriva llave -> Desencripta -> Parsea JSON.
  * Guarda en NVS y retorna `true`.

### C. Eliminación de Componentes Obsoletos
* Se borrará el archivo `src/Network/ProvisioningAP.cpp / .h` por completo, ya que la directiva es **eliminar el Access Point (AP)** y erradicar esta vulnerabilidad.

---

## 4. Tareas de Desarrollo Inmediatas

- [ ] 1. Eliminar `ProvisioningAP` del árbol de código.
- [ ] 2. Crear documento de UI `SetupPinView` con el teclado numérico en LVGL.
- [ ] 3. Programar el parseo de MicroSD (`SD_MMC`) adaptado a los pines del JC3248W535.
- [ ] 4. Implementar el motor criptográfico en el `ConfigManager` con `mbedtls`.
