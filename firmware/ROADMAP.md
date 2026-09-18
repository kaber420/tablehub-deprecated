# Roadmap de Desarrollo: Firmware (ESP32)

Este roadmap define las fases de desarrollo para el hardware local de Tablehub (los dispositivos en las mesas).

## 📍 Hito 1: Infraestructura Base y Aprovisionamiento
**Objetivo:** Permitir que los dispositivos se conecten a la red del restaurante sin necesidad de reprogramarlos por cable.
- [ ] Implementar administrador de WiFi (WiFiManager o Improv via BLE) para configuración desde el móvil.
- [ ] Configurar cliente MQTT (ej. `PubSubClient` o `async-mqtt-client`).
- [ ] Implementar reconexión automática asíncrona asegurando QoS 1 para no perder mensajes críticos.
- [ ] Diseño de la estructura de Tópicos MQTT (ej. MAC address para aprovisionamiento, luego ID lógico: `table/{id}/order`).
- [ ] El dispositivo no manejará colas complejas, confiará en el broker local del Hub (NATS JetStream) para el enrutamiento y persistencia.

## 📍 Hito 2: Periféricos y UI Física
**Objetivo:** Interacción básica de hardware.
- [ ] Integración de pantalla Touch (TFT/LCD) y envío de órdenes en payloads JSON vía MQTT.
- [ ] Integración del lector RFID/NFC (MFRC522 o PN532).
- [ ] Control de LEDs (WS2812B/NeoPixels) para feedback visual de estados enviados por el Hub (Libre, Ocupada, Llamando Mesero).
- [ ] Manejo de botones físicos (antirrebote/debounce).

## 📍 Hito 3: Lógica de Negocio y Energía
**Objetivo:** Comportamiento inteligente y autonomía.
- [ ] Máquina de estados (Libre -> Tag Detectado -> Esperando confirmación del Hub -> Ocupada).
- [ ] Deep Sleep y optimización de consumo de batería.
- [ ] Actualizaciones Over-The-Air (OTA) enviadas desde el Hub.
