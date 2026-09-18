# Arquitectura de Comunicación Local (Hub ↔ Dispositivos POS)

Este documento define la arquitectura de mensajería para la comunicación en tiempo real dentro del restaurante (LAN) entre el TableHub (Servidor Local) y los dispositivos cliente como teléfonos de meseros (Kotlin/Flutter) y pantallas de cocina (KDS).

## 1. El Motor Principal: NATS Local

En lugar de construir una API REST pesada o manejar WebSockets crudos manualmente, **TableHub utiliza su propia instancia embebida de NATS** como motor de mensajería (Pub/Sub) para la red local.

### Ventajas frente a sistemas comerciales (Toast, Square, etc.)
- **Sincronización Offline "Gratis" (JetStream):** Si un mesero pierde cobertura Wi-Fi y toma un pedido, la aplicación (en Kotlin o Flutter) encola el mensaje. Al recuperar la red, NATS JetStream sincroniza el estado instantáneamente sin conflictos complejos de bases de datos.
- **Conexión TCP Nativa:** Al usar apps nativas (Kotlin con `jnats` o Flutter con `dart_nats`), los dispositivos se conectan al puerto TCP crudo (4222) de NATS. Esto es mucho más rápido, estable y consume menos batería que WebSockets sobre HTTP.
- **Tiempo Real Verdadero:** Un cambio en una mesa se publica en `local.tables.status` y todos los dispositivos suscritos (cajeros, meseros, cocina) reciben la actualización en microsegundos, sin necesidad de hacer *polling* (peticiones repetitivas al servidor).

## 2. Seguridad y Aislamiento (Multitenancy en una sola instancia)

El Hub ejecuta **una sola instancia de NATS** (ahorrando RAM y CPU), pero aísla el tráfico de manera segura utilizando **Cuentas (Accounts)**.

### A. La Cuenta Cloud (`SYS_CLOUD`)
- Es la cuenta de máxima seguridad que mantiene el túnel (Leaf Node) hacia el servidor SaaS en la Nube.
- Utiliza cifrado Ed25519 (`hub_cloud.key`).
- Los dispositivos locales en el Wi-Fi del restaurante **no tienen acceso** a esta cuenta.

### B. La Cuenta Local (`POS_LOCAL`)
- Es la cuenta diseñada exclusivamente para los dispositivos de los meseros y la cocina.
- El Hub gestiona la autenticación de esta cuenta (ej. emitiendo usuarios temporales o JWTs locales a los teléfonos).
- Opera en un "sandbox". Los meseros solo pueden interactuar con temas como `local.orders.*` o `local.kitchen.*`.

## 3. El Puente de Datos (Imports & Exports)

Para que un pedido tomado por un mesero llegue a la Nube (para estadísticas del administrador), se usa una función nativa de NATS para cruzar fronteras de Cuentas de forma segura:

- **Exportación Controlada:** El Hub configura NATS para que los mensajes de `local.orders.completed` en la cuenta `POS_LOCAL` sean **importados** automáticamente por la cuenta `SYS_CLOUD`.
- De esta manera, el mesero nunca habla directamente con la nube, sino que el motor de mensajería promueve los mensajes de forma segura desde la LAN hacia el Leaf Node cuando hay conexión a internet disponible.

## 4. Requisitos de Implementación Pendientes

Para activar esta arquitectura en el Hub actual, es necesario:
1. Configurar `hub/internal/events/nats.go` para escuchar en la IP local (`0.0.0.0:4222`) además de `127.0.0.1`.
2. Implementar la configuración de Accounts (`SYS_CLOUD` y `POS_LOCAL`) en la instancia embebida.
3. Crear un endpoint en el Hub (`/api/local/auth`) que entregue las credenciales NATS locales a la app móvil de los meseros tras hacer login.
