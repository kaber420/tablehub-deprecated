# Roadmap de Desarrollo: Cloud (SaaS)

Este roadmap define las fases para el desarrollo de la plataforma centralizada en la nube que gestionará las flotas de Hubs de múltiples organizationes.

## 🏗️ Arquitectura de Datos (Jerarquía)
- **Organización (Cliente):** La entidad principal que paga la suscripción.
- **Sucursales (Branches):** Una organización puede tener múltiples sucursales físicas.
- **Hubs:** Una sucursal puede tener múltiples Hubs conectados.
> **Nota de desarrollo inicial:** Para facilitar y acelerar el desarrollo en las primeras fases, asumiremos una relación simple (1 Organización = 1 Sucursal = 1 Hub). Más adelante, cuando el sistema soporte múltiples Hubs por sucursal, se evaluarán mecanismos avanzados (como bases de datos distribuidas en los locales) para evitar colisiones.

## 📍 Hito 1: Túneles y Gestión de Flotas
**Objetivo:** Establecer una conexión segura y bidireccional con los Hubs locales.
- [ ] Servidor WebSocket de alto rendimiento para mantener miles de conexiones concurrentes.
- [ ] Autenticación de Hubs (Validación de firmas criptográficas Ed25519).
- [ ] Base de datos multitenant (aislamiento por organización).
- [ ] Generación y entrega segura del archivo de aprovisionamiento `.thub` (con credenciales, IP local, dominio y llaves públicas de firma del Hub).

## 📍 Hito 2: Frontend SaaS, Usuarios y Seguridad Base
**Objetivo:** Construir la interfaz web de la Nube y asegurar completamente el sistema para que funcione como un SaaS real.
- [ ] **Frontend Web (Svelte):** Inicializar proyecto Vite + Svelte en `cloud/web` (Dashboard, Login, Gestión de Hubs). *Importante:* Se deben reciclar los estilos (`app.css`) y componentes del Hub para mantener la identidad visual del ecosistema y acelerar el desarrollo.
- [ ] **Control de la API de Zitadel:** Integrar el backend de Go con la Management API de Zitadel para crear organizaciones y asignar "User Grants" (roles: `owner`, `admin`, `viewer`) de forma programática.
- [x] **Seguridad: Modelo "Solo Invitación":** El SaaS de Tablehub opera exclusivamente bajo un esquema B2B de "Solo Invitación". Se eliminó por completo el flujo de solicitudes de acceso (tabla `access_requests`, endpoints `/v1/access-requests` y pantalla "Unirse a Organización"), forzando a que las membresías de personal se manejen únicamente bajo invitación explícita del administrador.
- [x] **Seguridad mediante JWT Claims:** Decodificar la claim `urn:zitadel:iam:org:project:roles` en el middleware de Go para resolver el rol y organización del usuario al instante (bypass de base de datos en lecturas).
- [ ] Gestión de suscripciones y facturación (Stripe).
- [ ] Despliegue de actualizaciones de firmware en masa.

## 📍 Hito 3: Integraciones de Terceros (POS)
**Objetivo:** Conectar Tablehub con sistemas de Punto de Venta comerciales.
- [ ] API REST externa (Autenticada con API Keys).
- [ ] Sistema de Webhooks para notificar eventos a los sistemas de los organizationes (ej. "Mesa 4 llamó al mesero", "Mesa 12 pagó").
- [ ] Colas de mensajes (Redis/RabbitMQ) para manejar picos de tráfico en las integraciones.
