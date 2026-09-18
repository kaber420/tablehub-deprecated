# Recomendaciones de Seguridad (Tablehub)

Este documento detalla las medidas de seguridad recomendadas para proteger el sistema Tablehub local (Maitre Hub), el tráfico de red de los periféricos IoT (ESP32) y la autenticación administrativa.

---

## 1. Seguridad en el Servidor (Maitre Hub)

### 1.1. Extraer la Clave Privada de la Nube de SQLite
* **Problema:** Actualmente, la `private_key` (usada para negociar la sesión segura del túnel Ed25519 con el SaaS) se guarda en la tabla `settings` de SQLite. Si la base de datos se ve comprometida o filtrada, la clave privada queda expuesta.
* **Medida:**
  * Almacenar la clave en un archivo físico en el host (ej. `./data/hub_cloud.key`) con permisos restringidos de sistema operativo (`chmod 600`), de forma similar a como se maneja actualmente `./data/hub_secret.key` para los JWT.
  * O bien, cargarla exclusivamente a través de variables de entorno del sistema (`CLOUD_PRIVATE_KEY`) que no se escriban en disco.

### 1.2. Habilitar HTTPS en el Panel de Administración Local
* **Problema:** El servidor web se comunica sobre HTTP estándar. Si un administrador accede al panel web desde otro dispositivo en la misma red Wi-Fi, la cookie de sesión (`token`) y los datos viajan en texto plano.
* **Medida:**
  * Configurar `http.ListenAndServeTLS` en `hub/internal/web/server.go`.
  * Generar certificados SSL/TLS de confianza local usando herramientas como **mkcert** para el despliegue del restaurante.
  * Cambiar la configuración de la cookie JWT en `auth.go` a `Secure: true`.

### 1.3. Restricción de Origen en WebSockets (CSWSH)
* **Problema:** En `server.go`, la función `CheckOrigin` del Websocket Upgrader retorna `true` indiscriminadamente. Esto expone el servidor local a ataques de *Cross-Site WebSocket Hijacking*.
* **Medida:**
  * Modificar `CheckOrigin` para validar que la cabecera `Origin` coincida estrictamente con el host y puerto local configurados para el Maitre Hub.

---

## 2. Seguridad en la Red y Comunicaciones Locales (NATS / MQTT)

### 2.1. Habilitar Autenticación en NATS/MQTT
* **Problema:** El broker NATS y el puente MQTT están expuestos en la red local (`0.0.0.0`) y aceptan conexiones de cualquier dispositivo sin credenciales. Aunque la integridad de los mensajes está protegida por firmas Ed25519, un cliente malicioso en la red Wi-Fi podría suscribirse a los tópicos y escuchar toda la telemetría del restaurante.
* **Medida:**
  * Configurar autenticación en el servidor NATS embebido (`server.Options`).
  * **Flujo Dinámico:** Durante el flujo de aprovisionamiento, generar un token o usuario/contraseña aleatorio único para el dispositivo aprobado. Comunicarlo al ESP32 por un canal seguro (ej: AP temporal protegido) para que lo use en la autenticación del cliente MQTT.

### 2.2. Cifrado Ligero con CoAP + DTLS (Session Resumption)
* **Problema:** Enviar datos por MQTT local en texto plano permite que un atacante capture y escuche el tráfico de red (sniffing). Sin embargo, habilitar cifrado estándar TLS (MQTTS) sobre TCP requiere un handshake pesado de ~1.5 a 2 segundos en el ESP32, lo que drenaría la batería rápidamente.
* **Medida:**
  * Evaluar la migración de MQTT a **CoAP (Constrained Application Protocol)** sobre UDP.
  * Implementar **DTLS (Datagram TLS)** con soporte para **Session Resumption** (Reanudación de Sesión mediante Session Tickets).
  * **Funcionamiento:** En la primera conexión, el ESP32 negocia las claves en el handshake completo y almacena un boleto cifrado (*Session Ticket*). En transmisiones posteriores, despierta de *Deep Sleep*, envía el ticket y reanuda el canal cifrado en milisegundos con un solo viaje de ida y vuelta, logrando cifrado total de extremo a extremo sin penalización de batería.

### 2.3. Aislamiento mediante Red Wi-Fi Dedicada
* **Problema:** Si los clientes del restaurante y las terminales ESP32/Maitre Hub comparten la misma red Wi-Fi pública, cualquiera podría intentar escanear puertos o realizar ataques de denegación de servicio (DoS) contra el Hub.
* **Medida:**
  * Configurar una **red Wi-Fi dedicada y exclusiva** (un SSID y router/AP separado) únicamente para los dispositivos de Tablehub (Maitre Hub y ESP32).
  * Esta red debe tener activado el aislamiento de clientes (AP Isolation) y no debe tener salida a internet directa ni visibilidad desde la red Wi-Fi pública de los clientes, previniendo que cualquier comensal vea el tráfico local o interactúe con el Hub.

---

## 3. Seguridad Física y del Hardware (ESP32)

### 3.1. Cifrado de Memoria Flash (Flash Encryption)
* **Problema:** Un atacante con acceso físico a un TablePad de una mesa puede desoldar el chip flash o usar el puerto serial (UART) para leer el firmware y extraer la clave privada Ed25519 de su almacenamiento NVS.
* **Medida:**
  * Activar **Flash Encryption** en el menú de configuración de ESP-IDF/Arduino. Esto cifra todo el firmware y los datos almacenados en la partición NVS mediante una clave de hardware única quemada en los eFuses del ESP32.

### 3.2. Arranque Seguro (Secure Boot)
* **Problema:** Alguien podría flashear un firmware modificado en el dispositivo para espiar el tráfico o realizar peticiones fraudulentas en el local.
* **Medida:**
  * Habilitar **Secure Boot** en el ESP32 para verificar criptográficamente que la firma del cargador de arranque (bootloader) y del firmware coincidan con las claves autorizadas del restaurante antes de iniciar.

### 3.3. Desactivar Puertos de Depuración en Producción
* **Problema:** Los pines JTAG y el puerto serial permiten depurar y leer el estado en vivo del microcontrolador.
* **Medida:**
  * Quemar físicamente los fusibles eFuse de depuración en el chip ESP32 (`DISABLE_JTAG` y `DISABLE_DL_DECRYPT`) al programar la versión final de producción.
