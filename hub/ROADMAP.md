# Roadmap de Desarrollo: Tablehub

Este documento sirve como guía permanente para el desarrollo e implementación de las nuevas características del Hub de Tablehub.

---

## 📍 Hito 1: Sistema de Autenticación Real (Bcrypt + JWT)
**Objetivo:** Proteger la interfaz y las APIs del hub mediante contraseñas cifradas y tokens de sesión estandarizados. 
*(Para detalles técnicos precisos, consulta: [docs/AUTH_DESIGN.md](file:///home/kaber420/Documentos/proyectos/tablehub/docs/AUTH_DESIGN.md))*

### Tareas:
- [ ] **Backend (Go):**
  - Instalar dependencias (`golang.org/x/crypto/bcrypt`, `github.com/golang-jwt/jwt/v5`).
  - Implementar estado de "Setup" inicial si no existe hash de contraseña.
  - Almacenar el `jwt_secret` fuera de la base de datos (archivo seguro).
  - Crear endpoints `POST /api/setup` y `POST /api/login`.
  - Enviar el Token JWT usando **Cookies Seguras (HttpOnly)**.
  - Implementar middleware `AuthMiddleware` para validar la cookie en peticiones protegidas.
- [ ] **Frontend (Svelte):**
  - Crear el componente `SetupView.svelte` para el primer uso.
  - Crear el componente `LoginView.svelte`.
  - Configurar las llamadas `fetch` para que incluyan credenciales automáticamente (cookies).

---

## 📍 Hito 2: Configuración del Servidor Cloud y Seguridad Local
**Objetivo:** Permitir configurar desde el panel de administración a qué servidor SaaS se conecta el Hub (URL, ID, carga de archivos `.thub`), soportando reinicio en caliente, así como asegurar las conexiones inalámbricas locales.

### Tareas:
- [ ] **Backend (Go):**
  - Refactorizar `ConnectToCloud` usando `context.Context` para poder destruir y reiniciar la conexión WebSocket sin apagar el programa principal.
  - Soportar carga y almacenamiento persistente del archivo de aprovisionamiento `.thub` (contiene IP, dominio y llaves de la Nube).
  - Integrar certificados SSL locales (`TLS/HTTPS`) autofirmados únicos por Hub en el servidor web de Go y forzar cookies de sesión seguras (`Secure`).
  - Crear endpoint local `POST /api/local-auth/challenge` para firmar desafíos con la llave privada del Hub (verificación de identidad para APKs de meseros, mitigando mDNS spoofing).
  - Generación de llaves Ed25519 al iniciar el hub si no existen, para que siempre estén disponibles en la UI.
  - Crear endpoint `GET /api/settings/cloud` y `POST /api/settings/cloud` para validar y guardar configuración de forma manual.
- [ ] **Frontend (Svelte):**
  - Agregar sección de "Cloud Connection" en `SettingsView.svelte`.
  - Habilitar subida de archivo `.thub` para configuración rápida.
  - Mostrar estado visual de la conexión (Conectado / Desconectado).

---

## 📍 Siguientes Pasos Planificados

### Hito 3: Infraestructura de Mensajería y Eventos (NATS + MQTT)
- Importar y configurar `github.com/nats-io/nats-server/v2/server` embebido en la aplicación de Go.
- Habilitar el puerto 1883 para conexiones MQTT nativas hacia NATS.
- Inicializar JetStream y definir los "Streams" persistentes (Ej: `EVENTS.tables.*`).
- Crear un servicio en Go (`EventService`) que se suscriba a los tópicos de JetStream y maneje la lógica de negocio (recibir órdenes de los ESP32 y propagarlas a la UI/Cloud).

### Hito 4: Mapeo Visual del Restaurante (Floor Plan Builder)
- Crear un editor interactivo en Svelte (canvas o drag-and-drop con CSS) para dibujar el "croquis" del restaurante.
- Permitir la creación de zonas/salones (Ej: Terraza, Salón Principal, VIP).
- Herramientas simples para dibujar paredes, barras y obstáculos básicos.
- Arrastrar y soltar "Mesas" (representando a los ESP32) dentro del mapa, asociando la MAC del dispositivo a una mesa visual en el croquis.
- Vista de monitor en tiempo real: ver el mapa con las mesas cambiando de color según su estado (Libre, Ocupada, Llamando).

### Hito 5: Gestión de Usuarios (Staff)
- Crear base de datos de usuarios (meseros/admins).
- CRUD de usuarios en backend Go.
- Panel de gestión en Svelte.
- Autenticación mediante PIN numérico para meseros.

### Hito 4: App Nativa y Sistema Standalone Completo (Menu, Micro-POS y KDS Local)
**Objetivo:** Permitir el funcionamiento autónomo del Hub en locales sin conectividad a internet o que no usan POS externos, proveyendo administración de menú, comandas básicas y terminal de cocina.

- [ ] **Base de Datos y Modelado Local:** Implementar soporte SQLite local para almacenar el catálogo de menú, mesas, zonas/salones y pedidos.
- [ ] **Autenticación Local Offline (APK):**
  - Implementar hashing de contraseñas con **Bcrypt** para cuentas locales de meseros y administradores.
  - Endpoint `POST /api/local-login` para validar credenciales y emitir tokens JWT locales firmados con la llave de máquina efímera del Hub.
  - Soporte de ciclos de refresco de tokens (Access y Refresh Tokens) localmente en el Hub.
- [ ] **Módulo KDS (Kitchen Display System):** Desarrollar la interfaz web (KDS) en Svelte para cocina que enlista comandas locales en tiempo real.
- [ ] **Flujo de Transición Standalone a Nube (Cloud Upgrade):**
  - Desarrollar el disparador "Conectar a la Nube" que permita descargar el `.thub` final de internet.
  - Sincronizar automáticamente la base de datos local (mesas, zonas, productos) subiéndola a Tablehub Cloud.
  - Realizar el switch en caliente del WebSocket local para levantar el túnel saliente seguro hacia `wss://cloud.tablehub.com/ws`.
- [ ] **Cliente móvil (Kotlin APK):** Desarrollar e integrar cliente nativo para toma de pedidos y recepción de alertas offline conectándose directamente al WebSocket del Hub.

---

## 📍 Propuesta: Ciberseguridad y Optimización de Telemetría IoT (Seguridad Avanzada)
**Objetivo:** Proteger el canal de transporte local y el almacenamiento físico del hardware mediante cifrado de extremo a extremo, resguardo de claves y aislamiento.

- [ ] **Seguridad del Hub (Maitre Hub):**
  - **Aislamiento de la Clave del Túnel:** Migrar la `private_key` del túnel SaaS fuera de la base de datos SQLite y almacenarla como variable de entorno o en un archivo con permisos restrictivos (`0600`).
  - **Conexiones Seguras HTTPS:** Integrar certificados SSL locales (`TLS/HTTPS`) en el servidor web de Go y forzar cookies de sesión seguras (`Secure`).
  - **Restricción CORS/CSWSH:** Filtrar el origen de las conexiones WebSocket del panel administrador local.
- [ ] **Seguridad en la Red Local (NATS/MQTT):**
  - **Autenticación MQTT:** Implementar generación dinámica de credenciales por dispositivo durante el aprovisionamiento.
  - **Canal de Cifrado Ligero (CoAP + DTLS):** Evaluar el reemplazo de MQTT clásico por CoAP con cifrado DTLS y *Session Resumption* (Reanudación de Sesión mediante Session Tickets) para garantizar el cifrado en el aire sin agotar la batería de los ESP32.
- [ ] **Seguridad Física (ESP32):**
  - **Cifrado de Flash (Flash Encryption):** Habilitar el cifrado de firmware y la partición NVS del ESP32 en el arranque por hardware.
  - **Arranque Seguro (Secure Boot):** Validar la firma digital del firmware para evitar flasheos maliciosos de terceras personas.

