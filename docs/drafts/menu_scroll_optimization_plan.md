# Optimización del Scroll de la Carta (MenuView)

El objetivo de este plan es resolver el problema del scroll lento y pesado en la vista de la carta (`MenuView.cpp`), mejorando drásticamente el rendimiento de renderizado en LVGL para el ESP32-S3 basándonos en el análisis arquitectónico exhaustivo del código actual.

## Cambios Propuestos

### UI Views

#### 1. Modificaciones en MenuView.h
- Añadir el índice estático `poolStartIndex` y el array `cardPool` para almacenar las instancias reutilizables (solo las 4 o 5 que caben en el ancho de 480px).
- Declarar nuevos métodos: `initCardPool()`, `bindVisibleItems()`, y `onScroll(lv_event_t* e)`.
- Definir variables estáticas para caché de estilos (`style_raised_card`, `style_sunken_area`).

#### 2. Modificaciones en MenuView.cpp
- **Implementar Reciclaje de Objetos en el eje X**: Modificar `renderProducts()` para no inicializar los 50+ ítems de golpe. Llamará a `initCardPool()`, creando únicamente de 4 a 5 tarjetas.
- **Lógica de Scroll (Horizontal)**: Asignar un manejador de eventos de scroll al `productsContainer` que detecte el desplazamiento horizontal (`lv_obj_get_scroll_x()`). Al dividir el desplazamiento por el ancho de la tarjeta (220px + padding), decidirá qué objetos deben mostrarse y llamará a `bindVisibleItems()`.
- **Integración Inteligente con PSRAM**: En `bindVisibleItems()`, cuando una tarjeta sea reciclada y se le asigne un nuevo producto, consultaremos al `AssetManager` para precargar su imagen en PSRAM. Esto evita lecturas masivas y síncronas a la MicroSD.
- **Estilos Estáticos**: Inicializar `style_raised_card` y `style_sunken_area` una sola vez y asignarlos mediante `lv_obj_add_style` a las tarjetas del Pool, reemplazando el uso de estilos en línea.
- **Aplanar Jerarquía y Banderas**: Remover el contenedor intermedio `imgBtn` y la bandera `LV_OBJ_FLAG_SCROLL_CHAIN` para evitar sobrecarga inútil en el enrutamiento de eventos táctiles.
- **Optimizar Textos**: Utilizar `lv_label_set_text_fmt` al formatear el texto de precios.

## Plan de Verificación

1. Compilar y subir el firmware al microcontrolador usando el comando aprobado:
   `sudo chmod 666 /dev/ttyACM0 && pio run -d firmware -e esp32 -t upload`
2. Navegar a la pantalla del menú en el hardware físico (pantalla JC3248W535).
3. Deslizar la lista de productos de izquierda a derecha y confirmar que el scroll es consistente, fluido y sin caídas de frames.
4. Corroborar que las imágenes se carguen eficientemente usando el `AssetManager` al entrar en el campo visual.
