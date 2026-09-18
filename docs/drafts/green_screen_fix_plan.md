# Plan de Corrección: Pantalla Verde (Corrupción de Colores) y Modales Nativos

## El Problema Diagnosticado
1. **Corrupción de Colores (Todo Verde):** La inicialización de la tarjeta SD en `main.cpp` forzaba el uso del bus `SPI` global. Dado que el driver de la pantalla (JC3248W535) depende de ese mismo bus para transmitir los píxeles, la interrupción del bus causó una desincronización total en el LCD, invirtiendo los canales de color y tiñendo toda la interfaz de verde brillante.
2. **Modales Genéricos Desentonados:** Al retirar la tarjeta SD, el sistema lanzaba el mensaje de error ("Mesa No Configurada") utilizando la función nativa de LVGL `lv_msgbox_create`. Esta función utiliza el estilo base de LVGL, ignorando por completo la estética oscura (NeumorphicStyles) del resto del sistema.

## Solución C++ Propuesta

### 1. Aislamiento del Bus SPI (Hardware)
En lugar de secuestrar el bus global, instanciar un bus secundario de hardware (`FSPI`) que sea de uso exclusivo para la tarjeta SD. Esto garantiza cero interferencia con la pantalla.

**Cambios en `main.cpp`:**
```cpp
// Se creará un puntero a un bus SPI aislado
SPIClass* sdSPI = new SPIClass(FSPI);
sdSPI->begin(12, 13, 11, 10); // Pines exclusivos

// Se montará la SD sobre el bus aislado, a 4MHz
bool sdMounted = SD.begin(10, *sdSPI, 4000000);
```

### 2. Eliminación de los Modales Nativos de LVGL
Reemplazar absolutamente todos los llamados a `lv_msgbox_create` en el archivo `main.cpp` por una nueva función helper llamada `showSystemModal`.

**Características de `showSystemModal`:**
- Genera un overlay negro con `LV_OPA_80` (idéntico a `SmartCallModal`).
- Genera una tarjeta elevada mediante `NeumorphicStyles::applyRaisedCard`.
- Utiliza la paleta oficial de colores: `getTextColor()`, `getMutedTextColor()` y `getPrimaryAccent()`.
- Integra botones redondeados usando `NeumorphicStyles::applyButton`.

## Conclusión
Implementar estos dos pasos (aislamiento de hardware y modales personalizados) restaurará de inmediato los colores oscuros originales de la interfaz y evitará que la SD colisione con el driver de video.

## Evolución: Diagnóstico de Endianness / Byte Swap (Solución Definitiva)
Tras aislar el bus SPI de la tarjeta SD, se confirmó que la pantalla mantenía el tinte verde neón debido a una llamada manual a `lv_draw_sw_rgb565_swap(px_map, w * h)` en `my_disp_flush()`.

### Causa del Tinte Verde:
1. El driver `Arduino_AXS15231B` y la librería `Arduino_GFX` procesan nativamente los colores de 16 bits en formato RGB565 sin requerir intercambio de bytes.
2. Al ejecutar `lv_draw_sw_rgb565_swap`, el byte alto y bajo de cada píxel se invierten.
3. Para colores oscuros del tema neomórfico como `#1A1D24` (`0x1A1D`), la inversión a `0x1D1A` desplaza los 6 bits centrales correspondientes al canal **Verde** a `011101` (58/63, equivalente al 92% de intensidad verde), convirtiendo todos los fondos oscuros en verde neón.

### Corrección Aplicada:
Se eliminó la llamada a `lv_draw_sw_rgb565_swap` en `main.cpp`, dejando que `draw16bitRGBBitmap` envíe los colores nativos RGB565 directamente a la pantalla.

