# Roadmap General del Proyecto (Tablehub)

Este documento sirve como índice central para coordinar el desarrollo de los tres componentes principales del ecosistema Tablehub. Cada módulo tiene su propio ciclo de vida y ruta de desarrollo detallada.

## Módulos Principales

### 1. 🖥️ [Maitre Hub (Servidor Local)](file:///home/kaber420/Documentos/proyectos/tablehub/hub/ROADMAP.md)
El cerebro de la operación en el restaurante. Se encarga de gestionar la red local MQTT, servir la interfaz web de administración para el gerente, guardar configuraciones, y mantener el túnel WebSocket activo con la nube.
- **Estado Actual:** Implementando Seguridad (Hito 1 completado).
- **👉 [Ver Roadmap del Hub](file:///home/kaber420/Documentos/proyectos/tablehub/hub/ROADMAP.md)**

### 2. ⚡ [Firmware (Dispositivos ESP32)](file:///home/kaber420/Documentos/proyectos/tablehub/firmware/ROADMAP.md)
El código en C++ que correrá en los microcontroladores distribuidos en las mesas. Se encarga del aprovisionamiento de WiFi por Bluetooth/AP, lectura de tarjetas RFID/NFC, control de LEDs de estado, y comunicación MQTT de baja latencia con el Hub.
- **Estado Actual:** Planeación inicial.
- **👉 [Ver Roadmap del Firmware](file:///home/kaber420/Documentos/proyectos/tablehub/firmware/ROADMAP.md)**

### 3. ☁️ [Cloud (SaaS Centralizado)](file:///home/kaber420/Documentos/proyectos/tablehub/cloud/ROADMAP.md)
La infraestructura en la nube que recibe las conexiones de miles de Hubs a nivel global, gestiona las suscripciones de los restaurantes, centraliza la analítica y permite integraciones con sistemas POS de terceros a través de Webhooks y APIs REST.
- **Estado Actual:** Planeación inicial.
- **👉 [Ver Roadmap del Cloud](file:///home/kaber420/Documentos/proyectos/tablehub/cloud/ROADMAP.md)**

---

*Nota Arquitectónica: Todos los módulos están diseñados para ser débilmente acoplados (loosely coupled). El Firmware solo conoce al Hub, y el Hub es el único puente autorizado para hablar con el Cloud. Si el Cloud cae, el Firmware y el Hub seguirán funcionando localmente sin interrupciones.*
