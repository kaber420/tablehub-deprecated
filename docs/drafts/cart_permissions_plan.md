# Plan de Implementación: Permisos de Mesa y Flujo del Carrito

Este documento detalla la arquitectura propuesta para manejar los diferentes niveles de permisos que tendrá cada TableHub (ESP32) respecto a la interacción del cliente con el menú y los pedidos.

## 1. Modos de Operación (Permisos)

Proponemos añadir una variable de configuración (`tableMode` o `orderPermission`) en el `DeviceConfig` y que sea actualizable vía MQTT desde el administrador central. Los 3 modos serán:

1. **Modo "Solo Menú" (View Only):**
   - El cliente solo puede navegar por las categorías y ver los detalles de los productos.
   - El botón "Agregar al Pedido" en el modal del producto estará oculto o deshabilitado.
   - El botón flotante del Carrito estará oculto.
   - *Uso:* Restaurantes muy formales donde el mesero siempre toma la orden en persona.

2. **Modo "Llamar al Mesero con Pedido" (Pre-Order):**
   - El cliente puede agregar productos al carrito.
   - En la vista del Carrito, el botón final dirá **"Enviar al Mesero"**.
   - Al presionarlo, el pedido se envía vía MQTT a la tablet/reloj del mesero. El mesero se acerca a la mesa, confirma verbalmente y él mismo lo ingresa al POS.
   - *Uso:* Restaurantes casuales que quieren agilizar la orden pero mantener contacto humano y control.

3. **Modo "Kiosco Directo" (Direct Order):**
   - El cliente agrega productos y en el Carrito presiona **"Ordenar y Preparar"**.
   - La orden se envía directamente al POS y a la impresora de cocina por MQTT.
   - *Uso:* Bares, comida rápida, food trucks o mesas VIP.

## 2. Cambios en el Código

### A. Configuración y MQTT (`ConfigManager` & `MQTTService`)
- Añadir el enum `TablePermission { VIEW_ONLY, PRE_ORDER, DIRECT_ORDER }` al `DeviceConfig`.
- Configurar un tópico MQTT (ej. `tablehub/config/table_1`) para que el servidor pueda cambiar este permiso en tiempo real.

### B. Interfaz de Usuario (`MenuView` & `CartView`)
- Modificar el botón "Agregar" en el Modal del producto para que se oculto si el permiso es `VIEW_ONLY`.
- En `CartView`, leer el permiso actual para decidir qué texto mostrar en el botón de pagar ("Enviar al Mesero" vs "Confirmar Orden").

### C. Gestor del Carrito (`CartManager`)
- Añadir funciones `submitPreOrder()` y `submitDirectOrder()` que empaquetarán los ítems en un JSON y los enviarán usando `MQTTService::publish()`.

## 3. Preguntas Abiertas a Definir
1. ¿Estos 3 modos cubren las necesidades de tus clientes de restaurantes? si las cubre
2. ¿Quieres que el permiso se guarde localmente en la memoria del dispositivo o que siempre se asigne desde el servidor al conectarse? debe ser asignado por el server (hub) al momento de conectarse y si se cae el server debe mantener el ultimo permiso otorgado
3. ¿El cliente podrá pagar directamente desde la pantalla (ej. escaneando un QR de MercadoPago) en el modo Kiosco Directo? igual deberia ser opcional, si el server ofresce el servicio y puede generar la orden  por su api debe estar habilitado si no, no debe estar disponible.
puede generar el qr con el link para el pago o el si el codigo de la transacion lo puede tomar y representarlo en qr pues mejor