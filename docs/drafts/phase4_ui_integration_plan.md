# Fase 4: Integración de UI con Datos Reales (Eliminación de Mocks)

Este documento detalla los pasos necesarios para conectar las vistas del ESP32 (`MenuView` y `RequestBillView`) con los datos reales provenientes del Hub y de la MicroSD, eliminando por completo la data estática "mockeada".

## 1. Integración de la Vista de Menú (`MenuView.cpp`)

Actualmente, `MenuView.cpp` utiliza la función `initMockData()` para llenar arreglos estáticos de categorías y productos.

### 1.1 Evolución del Payload de Sincronización
El esquema JSON actual de `/sync` fue simplificado para la descarga de imágenes. Para que la UI pueda renderizar el menú, el Hub debe incluir metadatos adicionales en el `catalog_manifest.json`:

```json
{
  "catalog_version": "rev_45",
  "categories": [
    {"id": 1, "name": "Hamburguesas", "icon": "LV_SYMBOL_IMAGE"}
  ],
  "items": [
    {
      "id": "item_1",
      "category_id": 1,
      "name": "Burger Doble Queso",
      "description": "Carne 100% res...",
      "price": 145.00,
      "image_hash": "a1b2c3d4"
    }
  ]
}
```
*(Nota: El Hub deberá modificar la función que emite el evento `/sync` para incluir estos campos).*

### 1.2 Modificaciones en Firmware (`MenuView.cpp`)
1. **Eliminar `initMockData()`**: Remover los datos hardcodeados.
2. **Crear `loadDataFromSD()`**: 
   - Abrir el archivo `/catalog_manifest.json` desde la SD usando `FS.h`.
   - Utilizar `ArduinoJson` para parsear las categorías y los ítems.
   - Llenar los vectores estáticos `categories` y `menuItems` en tiempo de ejecución.
3. **Manejo de Imágenes Dinámicas**:
   - Al renderizar cada producto, buscar el archivo binario RGB565 correspondiente en la SD (`/assets/item_1_a1b2c3d4.bin`).
   - Si existe, cargarlo en el widget de imagen de LVGL; si no, usar un ícono por defecto.

---

## 2. Integración de la Vista de Cuenta (`RequestBillView.cpp`)

Actualmente, al pedir la cuenta, se muestra un subtotal fijo (`$24.50`). Se debe implementar un flujo de petición/respuesta vía MQTT.

### 2.1 Flujo MQTT Propuesto
1. El usuario presiona "Pedir Cuenta" en el ESP32.
2. El ESP32 publica un mensaje a `tablehub/device/{mac}/bill/request`.
3. El Hub (Go) recibe la petición, consulta el estado de la cuenta en TastyIgniter (vía API o base de datos/estado en memoria), y responde publicando en `tablehub/device/{mac}/bill/state`.

**Payload esperado del Hub al ESP32 (`/bill/state`):**
```json
{
  "subtotal": 450.00,
  "tax": 72.00,
  "total": 522.00,
  "items": [
    {"name": "Burger Doble Queso", "qty": 2, "price": 145.00},
    {"name": "Cerveza Artesanal", "qty": 2, "price": 80.00}
  ]
}
```

### 2.2 Modificaciones en Backend (Hub - Go)
1. **Suscripción MQTT**: Suscribirse al tópico `tablehub/device/+/bill/request`.
2. **Lógica de Negocio**: Al recibir la solicitud, obtener el total de la orden de la mesa, formatearlo en JSON y publicarlo de vuelta.

### 2.3 Modificaciones en Firmware (`RequestBillView.cpp`)
1. **Quitar Mocks**: Eliminar variables hardcodeadas de totales.
2. **Estado de Carga (Loading)**: Al abrir la vista, mostrar un spinner o texto "Calculando cuenta..." mientras se espera la respuesta del Hub.
3. **Recepción MQTT**: Modificar `MQTTService.cpp` para que intercepte `/bill/state` y ejecute un callback que actualice los Labels de subtotal, impuestos y total en `RequestBillView`.

---

## 3. Plan de Pruebas

1. **Menú**: Asegurar que al cambiar un precio o producto en TastyIgniter, el Hub mande un `/sync`, el ESP32 guarde el JSON, y al abrir la vista de menú se vean los nuevos datos reales.
2. **Cuenta**: Ingresar órdenes reales. Al abrir "Pedir Cuenta", verificar que el spinner aparezca temporalmente y los montos cuadren exactamente con lo ordenado.
