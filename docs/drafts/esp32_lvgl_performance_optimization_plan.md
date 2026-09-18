# Plan de Optimización de Rendimiento LVGL (Reemplazo de Sombras Neomórficas)

## 1. Descripción del Problema
El diseño Neomórfico actual implementado en `NeumorphicStyles.cpp` hace uso intensivo de sombras amplias (`lv_obj_set_style_shadow_width` y `shadow_spread`). 
LVGL renderiza estas sombras por software (CPU). En un entorno sin GPU 2D dedicada (como el ESP32 o el emulador nativo de SDL sin aceleración), calcular el desenfoque (blur) de tantos píxeles por cada cuadro satura el procesador, causando bloqueos de la interfaz y caídas drásticas en los cuadros por segundo (FPS), haciendo que el dispositivo parezca que "deja de responder".

## 2. Pregunta Abierta (Requiere Decisión del Usuario)
> [!IMPORTANT]
> **Generación de las Imágenes (Assets):**
> Necesitaremos PNGs transparentes con las sombras ya horneadas para los botones y las tarjetas. 
> - **Opción A:** El usuario genera estas imágenes manualmente en Figma, Photoshop u otra herramienta de diseño, y me pasa los PNGs.
> - **Opción B:** Yo (el agente) creo un pequeño script en Python que utilice una librería gráfica para generar automáticamente los archivos PNG con las sombras exactas necesarias para el proyecto.
> 
> *Por favor, indica qué opción prefieres.*

## 3. Cambios Propuestos

### Fase A: Preparación de Recursos Gráficos
1. Obtener o generar los assets PNG:
   - `neumorphic_raised.png` (Sombra exterior para botones y tarjetas).
   - `neumorphic_sunken.png` (Sombra interior/cavada para botones presionados).
2. Convertir los archivos PNG a formato de array en C compatible con LVGL (usando el conversor oficial de LVGL).
3. Guardar los archivos generados en:
   - `firmware/src/UI/Assets/img_neumorphic_raised.c`
   - `firmware/src/UI/Assets/img_neumorphic_sunken.c`

### Fase B: Refactorización de Estilos (`firmware/src/UI/Themes/NeumorphicStyles.cpp`)
1. **Eliminar cálculos pesados:** Quitar todas las llamadas a propiedades `shadow` (como `lv_obj_set_style_shadow_width`, `lv_obj_set_style_shadow_ofs_x`, etc.).
2. **Aplicar Fondos de Imagen:** En lugar de sombras dinámicas, establecer el fondo de los objetos para que utilicen los nuevos arrays de imágenes generados mediante `lv_obj_set_style_bg_img_src()`.
3. **Manejo de Estados:** Configurar la transición para que al entrar en `LV_STATE_PRESSED`, el botón cambie su imagen de fondo de `raised` a `sunken`.

### Fase C: Ajustes en los Componentes
Debido a que ahora la sombra estará incrustada en la imagen de fondo, las dimensiones reales visuales del botón cambiarán (la imagen incluye la sombra transparente).
1. Ajustar el `padding` interno de los botones en `DashboardView.cpp` para evitar que el texto y los iconos se superpongan en la zona de la sombra de la imagen.
2. Modificar el `HeaderBar.cpp` para asegurar que el contenedor principal escale la imagen de fondo de manera correcta (posiblemente usando el modo de imagen ajustada o 9-patch de LVGL si fuera necesario).

## 4. Plan de Verificación
1. **Emulador (Local):** Ejecutar `pio run -e emulator -t exec` y comprobar que la interfaz responde instantáneamente sin la latencia anterior al hacer clics rápidos o cambiar de pantallas.
2. **Hardware Real (ESP32):** Subir el firmware al ESP32 (`pio run -e esp32 -t upload`) y asegurar que el rendimiento sea óptimo y fluido bajo renderizado de CPU.
