# Plan de Arquitectura e Implementación: Sincronización de Órdenes (TastyIgniter ↔ Hub ↔ ESP32)

Este documento es el blueprint técnico definitivo para los desarrolladores. Define los contratos de datos, la estructura de la base de datos, las modificaciones a nivel de código en Go (Maitre Hub y Cloud) y C++ (ESP32), y los esquemas JSON de los payloads MQTT.

---

## 1. Arquitectura de Tópicos MQTT (Jerarquía Desacoplada)

El sistema abandona el tópico único monolítico para adoptar una jerarquía semántica y escalable. El ESP32 se suscribirá a tópicos específicos según sus capacidades, lo que permite enrutar y procesar eventos (órdenes, branding, configuraciones) de forma aislada sin tener que parsear bloques de datos irrelevantes en cada mensaje.

### 1.1 Tópicos Definidos

*   **Configuración General:** `tablehub/device/{mac_address}/config`
    *   *Uso:* Brillo, idioma, timeout de pantalla.
*   **Branding Dinámico:** `tablehub/device/{mac_address}/branding`
    *   *Uso:* Colores corporativos (hex), URL de logos, fuentes.
*   **Telemetría (Uplink):** `tablehub/device/{mac_address}/status`
    *   *Uso:* El ESP32 reporta su nivel de batería, RSSI Wi-Fi y versión de firmware al Hub.
*   **Gestión de Órdenes:** `tablehub/device/{mac_address}/order/{order_id}/state`
    *   *Uso:* Actualizaciones granulares por cada pedido. El ESP32 se suscribirá usando el wildcard: `tablehub/device/{mac_address}/order/+/state`.

### 1.2 Esquemas JSON Esperados

**Payload para Órdenes (`/order/{order_id}/state`)**
```json
{
  "order_id": "12345",
  "table_number": "Mesa 12",
  "customer_name": "Juan Perez",
  "order_progress_pct": 75,
  "items": [
    {
      "id": "item_1",
      "name": "Hamburguesa Clásica",
      "status_label": "En Cocina"
    },
    {
      "id": "item_2",
      "name": "Papas Fritas",
      "status_label": "Listo"
    }
  ]
}
```

---

## 2. Modificaciones en el Backend (Go - Maitre Hub)

### 2.1 Búsqueda de Dispositivos (`hub/internal/db/devices.go`)
Para rutear el mensaje al dispositivo correcto, el Hub debe buscar la dirección MAC usando el número de mesa.

**Acción para Desarrollo:**
Crear la función `GetActiveDeviceByTable`:
```go
// GetActiveDeviceByTable devuelve la información del dispositivo asociado a una mesa específica.
func GetActiveDeviceByTable(tableNumber string) (*Device, error) {
    var d Device
    query := `SELECT mac_address, device_type, table_number, location_name, status 
              FROM devices 
              WHERE table_number = ? AND status = 'active' LIMIT 1`
    
    err := DB.QueryRow(query, tableNumber).Scan(&d.MacAddress, &d.DeviceType, &d.TableNumber, &d.LocationName, &d.Status)
    if err != nil {
        return nil, err
    }
    return &d, nil
}
```

### 2.2 Ingesta y Mapeo del Webhook (`hub/internal/web/server.go`)
El túnel websocket recibe el evento `order.created` o `order.updated`, pero actualmente no se extraen los platos (`items`) y la función ignora el payload de negocio.

**Acciones para Desarrollo:**
1.  **Interceptar evento en `handleCloudSession`:** Modificar el `switch event.Event` para reenviar los payloads de tipo "orden" a `ProcessCloudOrder`.
2.  **Extender el Struct:** Definir el modelo interno:
    ```go
    type NormalizedOrder struct {
        // ... campos existentes ...
        OrderProgressPct int                   `json:"order_progress_pct"`
        Items            []NormalizedOrderItem `json:"items"`
    }

    type NormalizedOrderItem struct {
        ID          string `json:"id"`
        Name        string `json:"name"`
        StatusLabel string `json:"status_label"`
    }
    ```
3.  **Lógica en `normalizeOrderPayload`:**
    *   Extraer el array de `items` (o `order_menu`) del JSON que envía TastyIgniter.
    *   Mantener el estado textual de cada plato en `status_label`.
    *   Calcular el porcentaje general de la orden (`order_progress_pct`) analizando cuántos ítems están listos o en preparación vs el total de ítems.

### 2.3 Publicación Directa a MQTT (`hub/internal/events/...`)
*(⚠️ Prevención de Loop: Para evitar loops infinitos donde el NATS-MQTT bridge capture sus propios mensajes, el puente genérico de `handleNatsEvent` **NO** procesará órdenes).*

**Acción para Desarrollo:**
Dentro de la función que ingiere la orden (ej. `ProcessCloudOrder`), una vez construido el JSON normalizado:
1. Buscar la MAC: `device, err := db.GetActiveDeviceByTable(tableNumber)`
2. Construir topic MQTT: `tablehub/device/{mac}/order/{order_id}/state`
3. **Publicar directo a MQTT**: `events.Publish(mqttTopic, orderJSONPayload)`
4. Eliminar el viejo `nc.Publish` hacia `tablehub.table.<N>.order` para evitar duplicación.
5. Actualizar la caché en memoria (ver sección 2.4).

### 2.4 Resiliencia: Cache de Órdenes en Memoria (`hub/internal/state/orders.go`)
Si el ESP32 pierde señal y se reconecta, el Hub debe enviarle inmediatamente su estado actual sin esperar a que el POS mande un nuevo Webhook.

**Acción para Desarrollo:**
1. Crear un `sync.Map` para almacenar `map[string]map[string]Order` (Mesa -> OrderID -> Order).
2. Cuando se detecte telemetría entrante de un dispositivo (`tablehub/device/{mac}/status`), buscar a qué mesa pertenece ese dispositivo, leer el cache y re-publicar todas las órdenes activas hacia el tópico MQTT de ese dispositivo.

---

## 3. Modificaciones en el Cloud (SaaS - Go)

*(⚠️ Corrección: El flujo de órdenes del Cloud al Hub ya funciona nativamente vía NATS a través de la suscripción `tenant.*.pos.order`. Enrutar por WebSocket `DispatchWebhookUseCase` duplicaría los eventos).*

**Acción para Desarrollo:**
Validar que el webhook de TastyIgniter impacte correctamente el topic NATS existente. **No** se requieren modificaciones de ruteo en el Cloud.

---

## 4. Modificaciones en el Firmware ESP32 (C++)

El ESP32 debe ser capaz de discriminar la nueva estructura de tópicos y almacenar el estado de múltiples órdenes simultáneas.

*(⚠️ Límite MQTT: Aumentar la constante `MQTT_MAX_PACKET_SIZE` de la librería PubSubClient a 1024 o 2048 bytes en `platformio.ini` usando `-DMQTT_MAX_PACKET_SIZE=2048`. Esto es estrictamente un cuello de botella de la librería y NO de la RAM del ESP32. NO es necesario comprimir, minificar ni hacer "denso" el payload JSON; el ESP32-S3 puede procesar arrays de órdenes largas en JSON sin problemas, siempre que la librería deje pasar los bytes).*

### 4.1 Suscripciones MQTT (`firmware/src/Network/MQTTService.cpp`)
El dispositivo debe suscribirse a los nuevos canales explícitos durante su inicialización.

**Acción para Desarrollo:**
En el método `MQTTService::setup()` o de reconexión, modificar las subscripciones:
```cpp
// Obtener MAC y construir base string
String baseTopic = "tablehub/device/" + getMacAddress();

// Suscribirse a los tópicos relevantes:
client.subscribe((baseTopic + "/config").c_str());
client.subscribe((baseTopic + "/branding").c_str());
client.subscribe((baseTopic + "/order/+/state").c_str()); // Wildcard para órdenes
```

### 4.2 Enrutamiento en el Callback MQTT (`firmware/src/Network/MQTTService.cpp`)
El método `mqttCallback(char* topic, byte* payload, unsigned int length)` debe usar funciones de parsing de C++ (`strtok` o partición por `/`) para determinar qué canal recibió un mensaje.

**Acción para Desarrollo:**
Implementar un parser de tópicos:
```cpp
void mqttCallback(char* topic, byte* payload, unsigned int length) {
    String topicStr = String(topic);
    String payloadStr = "";
    for (int i = 0; i < length; i++) { payloadStr += (char)payload[i]; }

    // Split topic: tablehub / device / {mac} / {action} / ...
    // ... lógica de partición de string ...
    
    if (topicStr.indexOf("/config") > 0) {
        processConfig(payloadStr);
    } else if (topicStr.indexOf("/branding") > 0) {
        processBranding(payloadStr);
    } else if (topicStr.indexOf("/order/") > 0) {
        // Extraer Order ID: /order/{id}/state
        String orderId = extractOrderId(topicStr);
        processOrderUpdate(orderId, payloadStr);
    }
    return &d, err
}
```

### 3.2 Ingesta y Mapeo del Webhook (`hub/internal/web/server.go`)
1.  **Interceptar evento en `handleCloudSession`:** Reenviar los payloads de tipo "orden" a `ProcessCloudOrder`.
2.  **Lógica en `normalizeOrderPayload`:** Extraer ítems, mantener estados textuales y calcular `order_progress_pct` basado en el ratio de ítems finalizados.

### 3.3 Publicación Directa a MQTT (`hub/internal/events/...`)
*(⚠️ Prevención de Loop: El sistema utilizará un flag en el mensaje o una distinción clara en el handler. El puente NATS-to-MQTT detectará si el origen del mensaje es el Webhook externo y evitará replicar esos eventos en el bus NATS nuevamente).*

1. Buscar MAC mediante `db.GetActiveDeviceByTable`.
2. Publicar directo a MQTT usando `mqtt.Client.Publish`.
3. Eliminar cualquier `nc.Publish` redundante hacia tópicos de órdenes para evitar loops.

### 3.4 Resiliencia: Cache de Órdenes en Memoria (`hub/internal/state/orders.go`)
Utilizar un `sync.Map` para almacenar `map[string]Order`. Al recibir telemetría (`/status`), verificar si hay órdenes pendientes en el cache para dicha mesa y republish automáticos para recuperación ante desconexiones.

---

## 4. Modificaciones en el Cloud (SaaS - Go)

*(⚠️ Corrección: El flujo de órdenes del Cloud al Hub ya utiliza NATS (`tenant.*.pos.order`). No se requiere modificar el Cloud, ya que el Hub actuará como el único responsable de transformar estos eventos NATS hacia el protocolo MQTT final del dispositivo).*

---

## 5. Modificaciones en el Firmware ESP32 (C++)

*(⚠️ Límite MQTT: Aumentar `MQTT_MAX_PACKET_SIZE` a 2048 bytes).*

### 5.1 Suscripciones MQTT (`firmware/src/Network/MQTTService.cpp`)
Suscribirse al wildcard: `client.subscribe((baseTopic + "/order/+/state").c_str());`.

### 5.2 Enrutamiento en el Callback MQTT (`firmware/src/Network/MQTTService.cpp`)
Implementar parser de strings para filtrar `/config`, `/branding` y `/order/`. Usar `extractOrderId(topicStr)` para aislar el ID dinámico de la orden.

### 5.3 Actualización de la Interfaz (`firmware/src/UI/Views/MyOrdersView.cpp`)
Actualmente la UI está diseñada para un array estático (`std::vector<OrderItem>`). Debe evolucionar para soportar un mapa de múltiples órdenes, cada una con su ID, y utilizar el ID del platillo para cargar las imágenes desde la caché local.

*(⚠️ Riesgo de RAM: LVGL solo cuenta con ~96KB de heap libre. Evitar destruir y recrear iterativamente objetos visuales. Se debe implementar un pool de memoria o reciclar los widgets existentes para evitar la fragmentación).*

**Acción para Desarrollo:**
1.  **Refactor del Modelo:** 
    * Agregar el campo `String id;` al struct `OrderItem` en `MQTTService.h` (actualmente no lo tiene).
    * Cambiar la estructura interna a `std::map<String, std::vector<OrderItem>> activeOrders;`.
2.  **Imágenes y Datos en Caché (MicroSD):** Respetando el plan de almacenamiento local existente, el ESP32 no descargará los datos gráficos ni del menú constantemente. Al renderizar en `MyOrdersView.cpp`, usará el `item.id` para buscar y cargar la imagen y los detalles desde la memoria **MicroSD**. Solo en caso de que el `id` no exista en la caché de la MicroSD o se requiera una actualización, el ESP32 solicitará y descargará el asset nuevo.
3.  **Callback `processOrderUpdate`:** Al recibir el JSON, reemplazar o insertar el array de ítems bajo el key `orderId`. Si una orden viene con estado "Finalizado", eliminarla del mapa.

---

## 5. Protocolo de Sincronización de Menú y Assets (Caché MicroSD)

Para mantener la MicroSD actualizada (cuando cambia un precio, un platillo nuevo o una imagen en TastyIgniter) sin saturar la red, se implementará un protocolo de sincronización bajo demanda.

### 5.1 Tópico Dedicado para Sincronización
El Hub notificará al dispositivo que existe una nueva versión del menú a través de un tópico exclusivo para la actualización de assets, evitando mezclarlo con configuraciones operativas:
`tablehub/device/{mac}/sync`

**Payload esperado (Catálogo Completo vía MQTT):**
*(Nota Técnica: El valor de `catalog_version` es calculado y estandarizado internamente por el Hub (ej. un hash de los datos o un correlativo interno). Sirve para que el ESP32 no dependa de la lógica ni de la versión de software del POS (TastyIgniter) utilizado).*
```json
{
  "action": "sync_catalog",
  "catalog_version": "rev_45_abc123",
  "items": [
    {"id": "item_1", "name": "Hamburguesa", "image_hash": "a1b2c3d4"},
    {"id": "item_2", "name": "Papas Fritas", "image_hash": "b5c6d7e8"}
  ]
}
```

### 5.2 Flujo de Descarga Diferencial en el ESP32 (Firmware)
1. **Validación de Versión:** Al recibir el payload en el tópico `/sync`, el ESP32 compara el `catalog_version` recibido contra la versión guardada localmente en la MicroSD (ej. `/sd/catalog_version.txt`). Si es la misma, ignora el mensaje.
2. **Reconciliación de Caché (MicroSD):** Si la versión es nueva, el ESP32 itera sobre el arreglo `"items"` recibido en el mismo mensaje MQTT y busca archivo por archivo en su MicroSD (ej. `/sd/assets/item_1_a1b2c3d4.png`). 
   - **Descarga de Binarios (HTTP):** Si el archivo de imagen no existe o el hash es distinto, el ESP32 construye la URL usando la IP del Hub que ya conoce (ej: `http://{hub_ip}:{port}/api/assets/{id}`) y hace un HTTP GET para guardar el binario en la tarjeta SD. Enviar solo el hash ahorra ancho de banda crítico en MQTT.
   - **Limpieza:** Si un platillo del menú antiguo ya no está en el nuevo arreglo de `"items"`, el ESP32 elimina su imagen de la SD para evitar llenar el disco.
3. **Reinicio de Interfaz:** Una vez descargados los assets faltantes, actualiza su `catalog_version.txt` y redibuja la UI de LVGL.

### 5.3 Distribución de Imágenes y Conversión de Formatos (Hub ➔ Dispositivos)
Para que el ESP32 pueda descargar las imágenes sin depender de internet y sin colapsar su memoria RAM, el Hub actúa como un CDN y motor de transcodificación local:
1. **Descarga y Archivo Maestro:** Al procesar un webhook de nuevo catálogo, el Hub descarga proactivamente las imágenes originales (PNG/JPG) desde TastyIgniter y las guarda como archivos maestros en disco (ej. `/data/assets/master/`).
2. **Pre-Conversión y Caché en Disco:** El Hub no realiza conversiones "al vuelo". Inmediatamente después de descargar el master, el backend en Go transcodifica y guarda las versiones en disco para cada perfil de hardware:
   - **ESP32-S3 / P4 (Microcontroladores):** Convierte y guarda en disco un formato nativo y ligero para LVGL (ej. binario crudo RGB565). Así, cuando las 30 mesas pidan la foto al mismo tiempo, el Hub solo sirve el archivo estático ya convertido, sin tocar el CPU.
   - **ARM64 / Android (Tablets/Displays):** Guarda una versión estándar u optimizada (ej. WebP).
3. **Entrega Ultrarrápida:** El endpoint inteligente (ej. `GET /api/assets/{id}?device_type=esp32_s3`) simplemente lee del disco la versión correspondiente a esa arquitectura y la escupe a la red local.

---

## 6. Fases de Implementación Sugeridas
Para evitar un colapso en el desarrollo debido al tamaño de las modificaciones, la implementación se dividirá estrictamente en 3 fases:

### Fase 1: Arquitectura de Ruteo y Caché Base
* **Hub Go:** Eliminar publicación duplicada en NATS y rutar `ProcessCloudOrder` directo a MQTT. Implementar caché en RAM (`sync.Map`) para recuperar órdenes tras desconexión.
* **Firmware C++:** Aumentar `MQTT_MAX_PACKET_SIZE`, suscribirse a `/order/+/state` y adaptar `MyOrdersView` con mitigaciones contra fragmentación de RAM.

### Fase 2: Sincronización Lógica (Caché Diferencial)
* **Hub Go:** Emitir avisos MQTT al tópico `/sync` cuando se actualice TastyIgniter.
* **Firmware C++:** Lógica para validar `catalog_version.txt` contra el payload de MQTT y purgar el disco local, sin manejar aún decodificación de binarios.

### Fase 3: Motor CDN y Transcodificación (Assets Pesados)
* **Hub Go:** Implementar el motor CDN: descarga de originales, guardado a disco, y transcodificación de imágenes a RGB565/WebP en segundo plano, sirviéndolas vía un router HTTP estático.
* **Firmware C++:** Integrar el cliente HTTP para consumir los binarios ligeros (RGB565) directamente del Hub local y guardarlos en la SD.
