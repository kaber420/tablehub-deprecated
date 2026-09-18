# Optimización del Scroll de la Carta (MenuView)

El objetivo de este plan es resolver el problema del scroll lento y pesado en la vista de la carta (`MenuView.cpp`), mejorando drásticamente el rendimiento de renderizado en LVGL para el ESP32-S3 basándonos en el análisis y recomendaciones de DeepSeek.

## User Review Required

> [!WARNING]
> La implementación del "Virtual Scrolling" (Pool de Tarjetas) requiere un cambio estructural importante en cómo `MenuView` instancia y vincula los datos de los productos (`menuItems`). Cambiaremos de un renderizado 100% estático a un flujo reactivo que asigna la información al vuelo durante el scroll.

## Proposed Changes

### UI Views

#### [MODIFY] [MenuView.h](file:///home/kaber420/Documentos/proyectos/tablehub2/firmware/src/UI/Views/MenuView.h)
- Añadir el índice estático `poolStartIndex` y el array `cardPool` para almacenar las instancias reutilizables (solo las que caben en pantalla).
- Declarar nuevos métodos: `initCardPool()`, `bindVisibleItems()`, y `onScroll(lv_event_t* e)`.
- Definir variables estáticas para caché de estilos (`style_raised_card`, `style_sunken_area`).

#### [MODIFY] [MenuView.cpp](file:///home/kaber420/Documentos/proyectos/tablehub2/firmware/src/UI/Views/MenuView.cpp)
- **Implementar Reciclaje de Objetos (Pool)**: Modificar `renderProducts()` para no inicializar todos los ítems. Llamará a `initCardPool()`, creando únicamente de 5 a 6 tarjetas genéricas.
- **Lógica de Scroll**: Asignar un manejador de eventos de scroll al `productsContainer` que detecte el desplazamiento vertical (`lv_obj_get_scroll_y()`), decida qué objetos deben mostrarse y mande llamar a `bindVisibleItems()`.
- **Establecer Estilos Estáticos**: Inicializar `style_raised_card` y `style_sunken_area` una única vez y asignarlos mediante `lv_obj_add_style` a todas las tarjetas, reemplazando el uso de estilos en línea.
- **Aplanar Jerarquía**: Simplificar el contenedor interno de las tarjetas, removiendo contenedores anidados (como `imgBtn`) y convirtiendo el contenedor base en el área interactiva.
- **Eliminar Banderas Innecesarias**: Quitar `LV_OBJ_FLAG_SCROLL_CHAIN` de los contenedores de tarjetas.
- **Optimizar Textos**: Utilizar `lv_label_set_text_fmt` al formatear el texto de precios en vez de múltiples llamadas a funciones string.

### Documentación del Proyecto

#### [MODIFY] [ui_performance_and_animations_plan.md](file:///home/kaber420/Documentos/proyectos/tablehub2/docs/drafts/ui_performance_and_animations_plan.md)
- Añadir al final del archivo una nueva sección "Actualización: Scroll Virtual en MenuView" detallando estas optimizaciones arquitectónicas. Siguiendo las reglas del proyecto, esta actualización se anexará (append) respetando íntegramente todo el contenido previo.

## Verification Plan

### Manual Verification
1. Compilar y subir el firmware al microcontrolador usando el comando aprobado:
   `sudo chmod 666 /dev/ttyACM0 && /home/kaber420/.platformio/penv/bin/pio run -d /home/kaber420/Documentos/proyectos/tablehub2/firmware -e esp32 -t upload`
2. Navegar a la pantalla del menú en el hardware físico de la pantalla JC3248W535.
3. Deslizar la lista de productos y confirmar que el scroll es consistente, fluido y sin caídas de frames (butter smooth).
4. Corroborar que al desplazar rápidamente hacia arriba y abajo, la información visual (nombres, precios y botones) se actualiza de manera correcta en el carrusel reciclado.
