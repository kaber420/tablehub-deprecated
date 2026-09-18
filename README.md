# Tablehub (Deprecated)

> ⚠️ **STATUS: DEPRECATED**  
> Este repositorio ha sido formalmente deprecado y se conserva como snapshot histórico y de referencia arquitectónica.  
> Toda su funcionalidad y lógica está siendo migrada y reemplazada como programas nativos dentro de **cbdos v0.3.0**.

---

## 📌 Descripción del Proyecto
**Tablehub** fue diseñado como un ecosistema integral para restaurantes y puntos de venta (POS) inteligentes, combinando hardware embebido, servidores locales offline-first y servicios en la nube:

- **Firmware (`/firmware`):** Diseñado para pantallas táctiles Guition / AXS15231B JC3248W535 (ESP32-S3, 3.5", 320x480) utilizando LVGL v8, comunicación MQTT y sincronización de estado de pedidos en tiempo real.
- **Hub Local (`/hub`):** Servidor central escrito en Go con NATS / JetStream embebido y SQLite, permitiendo operación 100% offline dentro de la red local del restaurante.
- **Cloud (`/cloud`):** Backend y paneles para gestión multi-sucursal y sincronización centralizada.
- **Docker (`/docker`):** Recetas y scripts de despliegue para servicios de terceros como **TastyIgniter** (con webhook bridge).
- **Documentación (`/docs`):** Especificaciones completas de arquitectura, seguridad y flujos de provisionamiento.

---

## 🚀 Despliegue de TastyIgniter (Docker)
Para levantar el contenedor y descargar TastyIgniter en un entorno local:
```bash
cd docker/tastyigniter
./install.sh
```
El script inicializa automáticamente MySQL, descarga el framework mediante Composer y configura las migraciones iniciales.
