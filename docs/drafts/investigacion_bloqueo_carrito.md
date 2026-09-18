# Investigación de Bloqueos de UI: Modal de Producto y Llamar Mesero

## Resumen Ejecutivo

Se identificaron **3 bugs críticos** que causan freezes/colgamientos al abrir el modal de producto en el menú y al tocar "Llamar Mesero", más **2 problemas graves de rendimiento** y **varios defectos adicionales**. Todos los bugs fueron introducidos o agravados por la implementación del carrito.

---

## Bug CRÍTICO 1: Use-After-Free / Doble Liberación en `MenuView::showProductModal`

**Archivo:** `firmware/src/UI/Views/MenuView.cpp:315-337`
**Severidad:** CRÍTICO — Causa directa de Freeze/Reset

### El Problema

```cpp
struct ModalData { lv_obj_t* mask; MenuItem item; };
ModalData* data = new ModalData{mask, item};  // ← única asignación

// Botón "Cerrar" recibe el mismo puntero data
lv_obj_add_event_cb(closeBtn, [](lv_event_t* e) {
    ModalData* d = (ModalData*)lv_event_get_user_data(e);
    ...
    delete d;  // ← LIBERA ModalData
}, LV_EVENT_CLICKED, data);

// Botón "Agregar" recibe el MISMO puntero data
lv_obj_add_event_cb(addBtn, [](lv_event_t* e) {
    ModalData* d = (ModalData*)lv_event_get_user_data(e);
    ...
    delete d;  // ← LIBERA EL MISMO ModalData (use-after-free)
}, LV_EVENT_CLICKED, data);
```

**`data` se pasa a DOS botones.** Cuando cualquier callback se ejecuta primero, hace `delete d`. Cuando el segundo callback se ejecuta (incluso si es milisegundos después por el sistema de eventos de LVGL), **accede a memoria ya liberada** al leer `d->mask` → comportamiento indefinido → crash o freeze del hilo de LVGL.

### Por qué ocurre en la práctica

1. `lv_obj_delete_async(d->mask)` no elimina el mask de forma síncrona. La eliminación se difiere al próximo ciclo de `lv_timer_handler()`.
2. Entre el `delete d` del primer callback y la eliminación real del mask, el segundo botón sigue existiendo en el árbol de LVGL.
3. Si LVGL despacha el evento `LV_EVENT_CLICKED` para el segundo botón (por ráfagas táctiles o cola de eventos), se ejecuta el segundo callback → **uso de puntero inválido**.

### Solución Inmediata

Cada botón debe tener su **propia copia** de `ModalData`, o usar un contador de referencias. Alternativamente, asociar el mask al contenedor del modal como user data y no compartir el struct entre botones.

---

## Bug CRÍTICO 2: Overlay del Modal en `lv_screen_active()` en vez de `lv_layer_top()`

**Archivos:** `MenuView.cpp:237`, `SmartCallModal.cpp:27`
**Severidad:** CRÍTICO — Causa directa de pantalla congelada

### El Problema

```cpp
// MenuView.cpp:237
lv_obj_t* mask = lv_obj_create(lv_screen_active());  // ← MAL

// SmartCallModal.cpp:27
lv_obj_t* mask = lv_obj_create(lv_screen_active());  // ← MAL
```

Ambos modales crean su máscara de fondo sobre **la pantalla activa actual**. Si la pantalla cambia (por el temporizador de inactividad o por navegación), el modal queda **huérfano** en la pantalla anterior (oculta) y el usuario ve la interfaz sin poder interactuar con el modal.

### Escenario de bloqueo

1. Usuario abre modal de producto.
2. `inactivityTimerCb` (15s) se dispara y llama a `loadScreensaver()`.
3. `lv_scr_load_anim(screensaverScreen, ...)` cambia la pantalla activa.
4. El mask del modal sigue en la `dashboardScreen` o `menuScreen` (ahora oculta).
5. La screensaver es la pantalla visible, pero el modal está en la pantalla oculta.
6. El usuario no puede cerrar el modal → **interfaz congelada**.

### Solución

Usar `lv_layer_top()` en lugar de `lv_screen_active()` para overlays modales:
```cpp
lv_obj_t* mask = lv_obj_create(lv_layer_top());
```

`lv_layer_top()` es una capa independiente que siempre está visible sobre cualquier pantalla.

---

## Bug CRÍTICO 3: Falta de Reseteo del Timer de Inactividad Dentro del Modal

**Archivo:** `MenuView.cpp:235-338`
**Severidad:** ALTO — Causa de pantalla congelada

### El Problema

`MenuView::showProductModal()` **nunca llama a** `UIManager::getInstance().resetInactivityTimer()`.

```cpp
void MenuView::showProductModal(const MenuItem& item) {
    // ← NO hay resetInactivityTimer aquí
    lv_obj_t* mask = lv_obj_create(lv_screen_active());
    // ... crear modal ...
    // Los botones tampoco resetan el timer
}
```

### Escenario de bloqueo

1. Usuario abre modal de producto y se toma >15 segundos leyendo la descripción.
2. El timer de inactividad se dispara: `inactivityTimerCb` → `loadScreensaver()`.
3. La screensaver se carga y oculta el modal (combinado con Bug #2).
4. Usuario no puede hacer nada → interfaz congelada.

### Solución

Llamar `UIManager::getInstance().resetInactivityTimer()` al inicio de `showProductModal()` y en los callbacks de los botones del modal.

---

## Bug 4: SmartCallModal — Variable estática `overlay` nunca asignada

**Archivo:** `SmartCallModal.cpp:6, 25-120, 122-127`
**Severidad:** MEDIO

### El Problema

```cpp
lv_obj_t* SmartCallModal::overlay = nullptr;  // Se declara pero...

void SmartCallModal::show(SmartCallCallback cb) {
    lv_obj_t* mask = lv_obj_create(lv_screen_active());  // ← variable LOCAL
    // mask nunca se asigna a overlay
}

void SmartCallModal::close() {
    if (overlay && lv_obj_is_valid(overlay)) {  // overlay SIEMPRE es nullptr
        lv_obj_delete_async(overlay);
        overlay = nullptr;
    }
}
```

`close()` es código muerto — nunca limpia nada porque `overlay` nunca se actualiza. Si se llama a `close()` después de `show()`, el modal permanece visible.

Además, ReqiestBillView.cpp:222 llama `SmartCallModal::showToast(msg)` que usa `lv_layer_top()()` correctamente, pero no puede cerrar el modal porque `close()` no funciona.

---

## Bug 5: `CartView::renderItems()` — Recreación Masiva y Frecuente de Objetos LVGL

**Archivo:** `CartView.cpp:68-191`
**Severidad:** ALTO — Degradación severa de rendimiento

### El Problema

```cpp
void CartView::renderItems() {
    lv_obj_clean(itemsContainer);  // Destruye TODO
    // Recrea TODO desde cero
    for (const auto& item : cartItems) {
        lv_obj_t* card = lv_obj_create(itemsContainer);
        // ... 10+ objetos por item ...
    }
}
```

`renderItems()` se llama desde:
- `increase_btn_cb` (línea 220)
- `decrease_btn_cb` (línea 232)

**Cada click en +/- destruye y recrea TODOS los objetos del carrito.** Con ~8 items de menú y ~12 objetos LVGL por item, cada click procesa ~96 operaciones de creación/eliminación. Esto causa:

- Fragmentación de memoria en el heap de LVGL
- Bloqueo del hilo de UI durante cientos de milisegundos
- Potencial desbordamiento del buffer de memoria dinámica en ESP32

### Solución

Implementar actualización diferencial (diff):
- Preservar las tarjetas de items existentes
- Solo agregar/remover las que cambiaron
- Actualizar labels in-place para cambios de cantidad

---

## Bug 6: Navegación al Carrito Solo Desde Dashboard

Todos los archivos analizados

### El Problema

No hay botón flotante de carrito. Para acceder al carrito:
1. El usuario debe estar en Dashboard
2. Tocar "Ver Carrito (N)"

Si el usuario está en MenuView y agrega un producto, debe navegar de vuelta al Dashboard (tocando "Volver" dos veces: Menú → Dashboard) para ver su carrito. Esto es una **fricción UX severa** y hace que los usuarios toquen repetidamente sin respuesta.

No hay ruta de navegación directa Menu → Cart en el código actual.

---

## Bug 7: `lv_font_montserrat_14` No Habilitado en `lv_conf.h`

**Archivo:** `firmware/include/lv_conf.h:35-39`
**Severidad:** MEDIO

### El Problema

```c
#define LV_FONT_MONTSERRAT_12 1
#define LV_FONT_MONTSERRAT_16 1
#define LV_FONT_MONTSERRAT_24 1
#define LV_FONT_MONTSERRAT_32 1
// LV_FONT_MONTSERRAT_14 NO está definido
```

El código usa `&lv_font_montserrat_14` en **11+ lugares** (CartView, MenuView, HeaderBar, etc.). Si la fuente 14 no está compilada en LVGL, el linker puede fallar o LVGL usa la fuente default, causando renderizado incorrecto.

---

## Bug 8: RequestBillView QR Modal — Uso Correcto de `lv_layer_top()` pero Sin Protección de Estado Estático

**Archivo:** `RequestBillView.cpp:228-285`
**Severidad:** BAJO

La variable estática `qrOverlay` puede causar fugas si `showQrModal()` se llama mientras un QR ya está abierto (la primera línea lo protege con `lv_obj_delete`, pero esto no es thread-safe bajo ciertas condiciones).

---

# Propuesta: Pantalla "Llamar Mesero" y Botón Flotante de Carrito

Se evaluaron ambas propuestas y se recomienda su implementación.

## Propuesta 1: Convertir "Llamar Mesero" a Pantalla Completa

### Beneficios
- Elimina todos los bugs relacionados con overlays (Bug #2, #4)
- UX consistente con el resto de la navegación (Menú, Carrito, Mis Pedidos, Pedir Cuenta)
- El timer de inactividad funciona correctamente (cada View puede resetearlo)
- No hay riesgo de orphan overlay por cambio de pantalla
- No compite con el botón flotante del carrito por la capa superior

### Implementación
1. Crear `CallWaiterView` (`.h`/`.cpp`) en `Views/`
2. Registrar en `UIManager` (`loadCallWaiter()`, nuevo `lv_obj_t* callWaiterScreen`)
3. DashboardView: botón "Llamar Mesero" → `UIManager::getInstance().loadCallWaiter()`
4. CallWaiterView renderiza las mismas 4 opciones como botones en una pantalla con HeaderBar
5. Al seleccionar una opción: publica el comando MQTT y navega de vuelta al Dashboard

## Propuesta 2: Botón Flotante del Carrito Siempre Visible

### Beneficios
- Acceso inmediato al carrito desde cualquier pantalla
- Muestra contador de items (badge)
- Elimina la fricción de navegación (Bug #6)

### Implementación
1. Crear componente `CartFloatingButton` que se crea en `lv_layer_top()` para estar sobre todas las pantallas
2. El botón:
   - Se posiciona en esquina inferior derecha
   - Muestra icono de carrito + badge con cantidad de items
   - Es `LV_OBJ_FLAG_CLICKABLE` con `LV_OBJ_FLAG_SCROLL_CHAIN` para no bloquear scroll
   - Al hacer clic → `UIManager::getInstance().loadCart()`
3. El badge se actualiza mediante un timer o callback del CartManager
4. Se oculta automáticamente si el carrito está vacío (o se muestra atenuado)
5. NO debe interferir con el HeaderBar, modales, ni scroll de contenedores

---

# Plan de Solución Detallado

## Fase 0: Hotfix Inmediato (Prioridad CRÍTICA)

| # | Acción | Archivo | Líneas |
|---|--------|---------|--------|
| 0.1 | Fix use-after-free en showProductModal → cada botón recibe su propia copia de ModalData | `MenuView.cpp` | 315-337 |
| 0.2 | Cambiar `lv_screen_active()` a `lv_layer_top()` en showProductModal | `MenuView.cpp` | 237 |
| 0.3 | Cambiar `lv_screen_active()` a `lv_layer_top()` en SmartCallModal::show | `SmartCallModal.cpp` | 27 |
| 0.4 | Agregar `resetInactivityTimer()` al inicio de showProductModal | `MenuView.cpp` | 235 |
| 0.5 | Agregar `resetInactivityTimer()` en los callbacks de botones del modal | `MenuView.cpp` | 321-337 |
| 0.6 | Arreglar SmartCallModal::close() → asignar overlay en show() | `SmartCallModal.cpp` | 25, 122 |
| 0.7 | Agregar `LV_FONT_MONTSERRAT_14 1` en lv_conf.h | `lv_conf.h` | 35-39 |

## Fase 1: Arquitectura de "Llamar Mesero" como Pantalla

| # | Acción | Archivos |
|---|--------|----------|
| 1.1 | Crear `Views/CallWaiterView.h` y `Views/CallWaiterView.cpp` | Nuevos |
| 1.2 | Registrar `loadCallWaiter()` en UIManager.h/cpp | `UIManager.h:19`, `UIManager.cpp` |
| 1.3 | Actualizar DashboardView para navegar a CallWaiterView | `DashboardView.cpp:26-29` |
| 1.4 | Eliminar SmartCallModal (o dejarlo como helper de toast) | `SmartCallModal.cpp` |
| 1.5 | Integrar comando MQTT en CallWaiterView | `CallWaiterView.cpp` |

## Fase 2: Botón Flotante de Carrito

| # | Acción | Archivos |
|---|--------|----------|
| 2.1 | Crear `Components/CartFloatingButton.h` y `.cpp` | Nuevos |
| 2.2 | Inicializar en UIManager::init() sobre `lv_layer_top()` | `UIManager.cpp:17-27` |
| 2.3 | Actualizar badge en cada cambio de carrito (callback o timer) | `CartManager.h`, `CartFloatingButton.cpp` |
| 2.4 | Manejar navegación: loadCart() al hacer clic | `CartFloatingButton.cpp` |
| 2.5 | Asegurar que no intercepte eventos de scroll/touch en pantallas subyacentes | Flags LVGL |

## Fase 3: Optimización de Rendimiento

| # | Acción | Archivo |
|---|--------|---------|
| 3.1 | Optimizar CartView::renderItems() con actualización diferencial | `CartView.cpp` |
| 3.2 | Agregar batch operations en CartManager (add/remove múltiples) | `CartManager.cpp` |
| 3.3 | Limitar frecuencia de llamadas a renderItems (debounce 50ms) | `CartView.cpp` |

---

## Checklist de Verificación Post-Fix

- [ ] Abrir modal de producto desde Menú → no se congela la UI
- [ ] Agregar producto al carrito desde modal → toast visible, modal se cierra
- [ ] Cerrar modal sin agregar → no hay fugas de memoria
- [ ] Llamar Mesero desde Dashboard → se abre pantalla completa (no modal)
- [ ] Botón flotante de carrito visible en todas las pantallas
- [ ] Botón flotante badge refleja cantidad actual de items
- [ ] Inactividad >15s con modal abierto → modal se cierra con la screensaver (o no se abre si hay timeout defensivo)
- [ ] No hay advertencias de `lv_obj_is_valid` ni use-after-free en logs LVGL
- [ ] Renderizado y scroll suave en CartView con 8+ items
- [ ] Compilación sin errores de linker por fuentes faltantes

---

## Referencias

- `MenuView.cpp` — modal, eventos, callbacks compartidos
- `SmartCallModal.cpp` — overlay huérfano, close() muerto
- `CartView.cpp` — recreación masiva de objetos
- `UIManager.cpp` — manejo de pantallas y timer de inactividad
- `lv_conf.h` — configuración de fuentes
- `RequestBillView.cpp` — ejemplo de uso correcto de `lv_layer_top()`

---

# Auditoria Integral de Arquitectura UI y Ciclo de Vida

## 1. Resumen Ejecutivo

Esta auditoría analiza la arquitectura completa del sistema UI en `firmware/src/UI/` (UIManager, 6 componentes compartidos, 7 vistas) identificando **problemas sistémicos de ciclo de vida, gestión de estado global, y acoplamiento entre componentes** que causan que las reparaciones en una vista propaguen fallos a otras.

### Hallazgos Clave

| # | Problema | Impacto | Archivos |
|---|----------|---------|----------|
| 1 | Estado global `activeHeader` es un puntero raw sin protección | Dangling pointer → crash al cambiar de pantalla | `HeaderBar.h:40`, `UIManager.cpp` |
| 2 | `LV_EVENT_DELETE` en `addBtn` libera `itemCopy` pero `closeBtn` no tiene cleanup simétrico | Fuga de memoria y doble-free potencial | `MenuView.cpp:340-345` |
| 3 | Punteros estáticos en vistas no se resetean al recrear la screen | Dangling pointers tras `lv_scr_load_anim` | Todas las vistas |
| 4 | `CartFloatingButton::update()` guarda `static int lastCount` | Estado fantasma entre reinicios de init | `CartFloatingButton.cpp:67` |
| 5 | SmartCallModal nunca asigna `overlay` pero `close()` lo verifica | Código muerto que da falsa sensación de seguridad | `SmartCallModal.cpp:6,122-127` |
| 6 | Sin cleanup de `HeaderBar*` al destruir vista → `activeHeader` apunta a memoria liberada | Crash en `updateActiveTime/Battery/Signal` | `HeaderBar.cpp:125-129` |
| 7 | Overlays en `lv_screen_active()` en vez de `lv_layer_top()` | Modal huérfano tras cambio de pantalla | `MenuView.cpp` (antes del fix), `SmartCallModal.cpp:27` |
| 8 | `renderItems()` recreación masiva desde cero | Congelamiento UI en ESP32 | `CartView.cpp:72-195` |
| 9 | Vista previa de screen cachéadas sin invalidación | Datos obsoletos al recargar | `UIManager.cpp` |

---

## 2. Problema Central: Estado Global `activeHeader` — El Talón de Aquiles del Sistema

### 2.1 Definición

`HeaderBar::activeHeader` es un puntero **estático, global, raw** que apunta a un `HeaderBar*` de instancia:

```cpp
// HeaderBar.h:40
static HeaderBar* activeHeader;   // Inicializado a nullptr en HeaderBar.cpp:7
```

Cada vista crea su propio `HeaderBar` en `create()` y lo registra vía `UIManager::loadXxx()` → `HeaderBar::setActiveHeader(getHeaderBar())`.

### 2.2 Por qué Arreglar una Vista Daña Otra

**Escenario concreto:**

1. DashboardView carga → `activeHeader = dashboardView->headerBar` (instancia A).
2. Usuario navega a MenuView → `setActiveHeader(menuView->headerBar)` → `activeHeader` ahora apunta a instancia B.
3. El HeaderBar de DashboardView (instancia A) **nunca se destruye** porque `dashboardScreen` se mantiene en `UIManager::dashboardScreen` (caché).
4. Usuario fuerza regreso rápido o el timer de inactividad llama a `loadScreensaver()` → `setActiveHeader(nullptr)`.
5. `ScreensaverView` **no tiene HeaderBar**. Pero `ScreensaverView::create()` no limpia `activeHeader` — solo `HeaderBar::~HeaderBar()` lo hace, y el destructor solo se ejecuta si el `HeaderBar*` se elimina.

**El problema real:** Si una vista se recrea (por ejemplo, porque `dashboardScreen` se destruyó por falta de memoria o por un `lv_obj_del`), su `HeaderBar` anterior sigue registrado como `activeHeader` y al llamar `setActiveHeader(nuevoHeaderBar)`, el viejo `HeaderBar` tiene sus widgets LVGL (`container`, `timeLabel`, etc.) en memoria ya liberada.

**¿Cuándo se recrea una vista?** El código tiene caché:

```cpp
// UIManager.cpp:48-51
void UIManager::loadDashboard() {
    if (!dashboardScreen) {
        dashboardScreen = DashboardView::create();
    }
    ...
}
```

Si por alguna razón `dashboardScreen` se invalida (ej: `lv_obj_del` externo, o el LVGL garbage collector la elimina por contar con 0 referencias), `DashboardView::create()` se llama de nuevo, creando un **nuevo** `headerBar`. Pero el `HeaderBar` **viejo** sigue siendo `activeHeader` si su destructor no se ejecutó (¡y no se ejecutará porque LVGL elimina el `lv_obj_t* container`, no el `HeaderBar*` wrapper de C++!).

**NO HAY CORRESPONDENCIA entre el ciclo de vida del `lv_obj_t* container` y el `HeaderBar*` wrapper.** Cuando LVGL elimina el `lv_obj_t` (por `lv_obj_del` o `lv_obj_clean`), el `HeaderBar` de C++ no se libera. Si `activeHeader` apuntaba a ese `HeaderBar`, ahora es un **dangling pointer**.

### 2.3 La Cadena de Fallo

```
loadDashboard() → setActiveHeader(dashboardHeaderBar) → activeHeader = ptr_A
    ↓
loadMenu() → setActiveHeader(menuHeaderBar) → activeHeader = ptr_B
    ↓
(Algo destruye menuScreen o su container de LVGL)
    ↓
ptr_B->container es inválido, pero ptr_B sigue vivo
    ↓
updateActiveTime() llama a ptr_B->updateTime() → timeLabel es inválido → LVGL crash
```

### 2.4 El destructor `~HeaderBar()` es Insuficiente

```cpp
HeaderBar::~HeaderBar() {
    if (activeHeader == this) {
        activeHeader = nullptr;  // Solo limpia si this es el activo
    }
}
```

Pero `~HeaderBar()` **nunca se llama** automáticamente cuando LVGL destruye `container`. Solo se llamaría si alguien hace `delete headerBarPtr`. El código actual **nunca hace `delete` de un `HeaderBar*`**. Las vistas son estáticas y nunca liberan su `headerBar`. Ergo: `~HeaderBar()` es código muerto.

---

## 3. Ciclo de Vida y Gestión de Memoria por Vista

### 3.1 UIManager — Caché de Screens sin Invalidación

```cpp
lv_obj_t* dashboardScreen = nullptr;
lv_obj_t* screensaverScreen = nullptr;
lv_obj_t* myOrdersScreen = nullptr;
// ... etc
```

**Problema:** las screens se crean una vez y se cachean para siempre. LVGL 9.5 no ofrece garantías de que un `lv_obj_t*` siga siendo válido si la screen se ocultó y hubo presión de memoria. Además, las vistas guardan **punteros estáticos a widgets hijos** que dependen de que la screen exista:

| Vista | Punteros estáticos | Riesgo |
|-------|-------------------|--------|
| DashboardView | `headerBar`, `ordersBtnLabel`, `ordersBtnIcon` | Dangling si screen se recrea |
| MenuView | `headerBar`, `categoriesContainer`, `productsContainer` | Dangling si screen se recrea |
| CartView | `headerBar`, `itemsContainer`, `totalLabel`, `emptyLabel`, `sendOrderBtn` | Dangling |
| RequestBillView | `headerBar`, `tipBtn10/15/20/0`, `qrOverlay`, 6 labels | Dangling |
| MyOrdersView | `headerBar`, `list` | Dangling |
| CallWaiterView | `headerBar`, `toastObj`, `toastTimer` | Dangling + timer leak |
| ScreensaverView | `timeLabel`, `weatherContainer`, `logoLabel`, `subtitleLabel`, `mainContainer` | Dangling |

### 3.2 Análisis Individual

#### DashboardView
- **create()**: Crea screen, HeaderBar, grid de 4 botones. `headerBar` se asigna al static.
- **Sin destructor**: No hay manera de liberar `headerBar`. Si screen se elimina y recrea, el viejo `headerBar` pierde sus hijos LVGL.
- **refreshState()**: usa `ordersBtnLabel` static — si la screen se recreó, este puntero es del viejo árbol LVGL.

#### MenuView
- **create()**: Crea screen, HeaderBar, categorías y productos. Llama a `initMockData()` que usa `static` para cargar datos mock una sola vez.
- **showProductModal()**: Crea mask en `lv_layer_top()` (fijo post-fix). Usa `lv_obj_delete_async` en lugar de `lv_obj_del` para cerrar.
- **LV_EVENT_DELETE en addBtn**: Libera `itemCopy` solo cuando el botón `addBtn` es eliminado. Pero si el modal se cierra por `closeBtn` (que llama a `lv_obj_delete_async(maskObj)`), el `addBtn` también se elimina con su `LV_EVENT_DELETE`. **Esto es correcto pero frágil**: si el modal se cierra por otro camino (timer de screensaver, etc.), `itemCopy` nunca se libera.
- **showToast()**: Crea toast en `lv_layer_top()` con timer de auto-destrucción. No hay protección contra toasts múltiples (el toast anterior no se limpia antes de crear uno nuevo).

#### CartView
- **create()**: Crea screen, HeaderBar, contenedor de items, bottomCard con total y botón.
- **renderItems()**: Destruye todo con `lv_obj_clean(itemsContainer)` y recrea. **Problema de rendimiento severo** (~100 objetos LVGL destruidos/recreados por cada click en +/-).
- **refresh()**: Solo llama a `renderItems()` — no hay invalidación selectiva.

#### RequestBillView
- **create()**: Crea screen compleja con resumen, selector de propina (4 botones con punteros estáticos), métodos de pago (3 botones).
- **showQrModal()**: Crea overlay en `lv_layer_top()` con QR. `qrOverlay` estático → fugas si se abre múltiples veces. La protección `if (qrOverlay && lv_obj_is_valid(qrOverlay)) lv_obj_delete(qrOverlay)` no es segura porque no hay sincronización.
- **closeQrModal()**: Llama a `lv_obj_delete` (síncrono) y luego `loadDashboard()`. Pero `closeQrModal` es un callback de LV_EVENT_CLICKED, que se ejecuta dentro del ciclo de eventos de LVGL. `lv_obj_delete` dentro de un callback puede causar problemas si hay otros eventos pendientes.

#### MyOrdersView
- **create()**: Crea screen con HeaderBar y contenedor de lista vacío.
- **updateOrders()**: `lv_obj_clean(list)` + recreación completa. Sin caché de items previos.
- **Mínimo**: Es la vista más simple y menos problemática.

#### CallWaiterView
- **create()**: Crea screen completa con HeaderBar y 4 tarjetas de opciones.
- **option_btn_cb()**: Llama a `callback`, luego `showToast()`, luego `loadDashboard()`. **Problema**: `loadDashboard()` carga la screen del dashboard, pero `CallWaiterView` no elimina su screen ni limpia punteros estáticos. El toast sigue visible porque se crea en `lv_layer_top()` — correcto.
- **showToast()**: Usa estáticos `toastObj` y `toastTimer`. Si se llama dos veces seguidas, el timer anterior se elimina y el toast anterior se borra. Correcto, pero `toastTimer = nullptr` solo se asigna en `toast_timer_cb`, no en la función que elimina el toast anterior.

#### ScreensaverView
- **create()**: Crea screen con fondo oscuro, logo, subtítulo. Todos los widgets son `static` pointers.
- **Problema**: `HeaderBar::setActiveHeader(nullptr)` se llama en `loadScreensaver()`, pero si `create()` de ScreensaverView se llama más de una vez (por invalidación de caché), los punteros estáticos viejos se pierden.
- **input_event_cb()**: Clic en cualquier parte → `resetInactivityTimer()` → `loadDashboard()`.

### 3.3 El Problema del `lv_obj_is_valid`

LVGL 9.5 introdujo `lv_obj_is_valid()` como protección contra punteros inválidos. El código lo usa extensivamente:
- `CartFloatingButton.cpp:14,64,73,85`
- `SmartCallModal.cpp:93,115,123,133,162`
- `MenuView.cpp:335,376`
- `RequestBillView.cpp:236,295`

**Pero `lv_obj_is_valid()` solo verifica que el puntero no sea NULL y que el objeto esté en la lista interna de LVGL. No puede detectar use-after-free si la memoria fue reasignada a otro objeto LVGL.** Es una mitigación, no una solución.

---

## 4. Interacciones Entre Vistas y Componentes Compartidos

### 4.1 HeaderBar — Singleton Falso

HeaderBar se diseñó como un componente reutilizable con una referencia activa global:

```
UIManager::loadXxx()
  → XxxView::create()          // Crea HeaderBar* local
  → HeaderBar::setActiveHeader(XxxView::getHeaderBar())  // Registra como activo
```

**Problemas:**

1. **No hay unbind**: Cuando una vista se descarga, `activeHeader` no se limpia a menos que otra vista llame a `setActiveHeader()`. Si la nueva vista crashea durante `create()`, `activeHeader` apunta a la vista anterior con widgets posiblemente inválidos.

2. **Carrera de actualizaciones**: `updateActiveTime/Battery/Signal` se llaman desde `main.cpp` (timer global). Si `activeHeader` es un dangling pointer, la siguiente actualización de hora/batería causa crash. No hay mutex ni protección atómica.

3. **Consistencia de estado**: `lastTimeStr`, `lastBatteryPercentage`, `lastSignalStrength` son estáticos que persisten entre vistas. Cuando se cambia de vista, `setActiveHeader()` refresca el nuevo header con los últimos valores conocidos. Esto es correcto pero frágil: si una vista tiene un header sin `showStatus=true`, `timeLabel`, `batteryShell`, `signalIcon` son `nullptr` y las actualizaciones son no-op. Esto es correcto porque hay checks de null.

### 4.2 CartFloatingButton — Capa Superior con Estado Fantasma

```
lv_obj_t* CartFloatingButton::container = nullptr;  // estático global
```

**init()**: Crea el botón flotante en `lv_layer_top()`.
- Guarda `static int lastCount = -1` dentro de `update()`. Este static **local de función** persiste entre llamadas. Si `init()` se llama dos veces (por reinicio parcial del sistema), `lastCount` mantiene el valor anterior y evita la actualización del badge. Es un bug sutil: tras reinicialización, el badge no se actualiza hasta que el contador cambie dos veces.

- **setVisible()**: Oculta/muestra el contenedor. Pero `loadCart()` llama a `setVisible(false)` y `loadDashboard()` a `setVisible(true)`. Si `loadCart()` se llama desde dentro de un callback que también resetea el timer, y el timer se dispara durante la transición, hay un breve período donde el botón flotante y el carrito están visibles simultáneamente.

### 4.3 SmartCallModal — Obsoleto pero Peligroso

SmartCallModal es código legacy que **debería haberse eliminado** cuando se creó `CallWaiterView`, pero sigue presente.

| Función | Estado | Problema |
|---------|--------|----------|
| `show()` | Obsoleto | Usa `lv_screen_active()` para mask → overlay huérfano si cambia pantalla |
| `close()` | Muerto | `overlay` nunca se asigna (`mask` es variable local) |
| `showToast()` | Usado | Correcto, usa `lv_layer_top()` |
| `option_btn_cb()` | Muerto | Nunca se asigna como callback (se usan lambdas inline) |
| `close_btn_cb()` | Muerto | Idem |

**RequestBillView.cpp:222** llama a `SmartCallModal::showToast(msg)` — la única función que realmente funciona de esta clase. Pero `showToast()` usa estáticos (`toastObj`, `toastTimer`) que **comparten espacio global con otras clases**. Si `CallWaiterView::showToast()` o `MenuView::showToast()` se llaman cerca, no hay conflicto porque cada clase tiene sus propios estáticos. Pero conceptualmente es mantenimiento confuso tener 4 implementaciones diferentes de toast (SmartCallModal, MenuView, CartView, CallWaiterView).

### 4.4 CartFloatingButton ↔ CartManager

La comunicación es unidireccional:

```
CartManager::addItem() → (sin notificación)
CartFloatingButton::update() (llamado desde UIManager::update())
  → CartManager::getItemCount() → actualiza badge
```

**Problema de latencia**: `UIManager::update()` se llama desde el loop principal. Si `CartManager::addItem()` se ejecuta y luego inmediatamente se consulta `getItemCount()`, el badge se actualiza en el próximo ciclo de `update()`. Con LVGL corriendo a 30-60fps, hay un retraso visible de 16-33ms. Para un badge numérico esto no es crítico, pero es un diseño frágil.

---

## 5. Árbol de Dependencias y Flujo de Navegación

```
UIManager (orquestador, dueño de screens)
  ├── HeaderBar (estado global: activeHeader, lastTimeStr, lastBattery, lastSignal)
  ├── CartFloatingButton (estado global: container, badgeLbl, cartBtn)
  ├── SmartCallModal (legacy: overlay, toastObj, toastTimer)
  │
  ├── DashboardView → dependencias: HeaderBar, SmartCallModal, CartManager, UIManager
  ├── MenuView     → dependencias: HeaderBar, CartManager, UIManager
  ├── CartView     → dependencias: HeaderBar, CartManager, UIManager
  ├── RequestBillView → dependencias: HeaderBar, SmartCallModal, UIManager
  ├── MyOrdersView → dependencias: HeaderBar, MQTTService, UIManager
  ├── CallWaiterView → dependencias: HeaderBar, UIManager
  └── ScreensaverView → dependencias: UIManager
```

**Todas las vistas dependen de UIManager, y UIManager depende de todas las vistas.** Esto es un acoplamiento circular a nivel de includes. `UIManager.h` incluye todas las vistas.

**HeaderBar** tiene una dependencia bidireccional con UIManager:
- `HeaderBar::back_event_cb()` → `UIManager::getInstance().loadDashboard()`
- `UIManager::loadXxx()` → `HeaderBar::setActiveHeader()`

Esto no es un problema en compilación (son includes de .cpp), pero crea una dependencia de runtime donde UIManager debe estar completamente inicializado antes de que cualquier HeaderBar se cree.

---

## 6. Análisis de LV_EVENT_DELETE y Limpieza de Memoria

### 6.1 El Único Uso de LV_EVENT_DELETE en el Sistema

```cpp
// MenuView.cpp:340-345
lv_obj_add_event_cb(addBtn, [](lv_event_t* e) {
    MenuItem* itemPtr = (MenuItem*)lv_event_get_user_data(e);
    if (itemPtr) {
        delete itemPtr;
    }
}, LV_EVENT_DELETE, itemCopy);
```

Este callback se registra en `addBtn` con `itemCopy` como user data. Cuando LVGL elimina `addBtn` (porque su padre `modalCard` se eliminó con `lv_obj_delete_async(maskObj)`), se dispara `LV_EVENT_DELETE` y se libera `itemCopy`.

**¿Por qué es peligroso?**
1. Si el modal se cierra por `closeBtn`, se ejecuta `lv_obj_delete_async(maskObj)`. Esto elimina `mask` → `modal` → todos los hijos, incluyendo `addBtn`. El evento `LV_EVENT_DELETE` se dispara correctamente.
2. **Pero**: si el modal se cierra por otro mecanismo (ej: el usuario toca fuera del modal, o la screensaver se activa), y no hay un `lv_obj_delete_async` del mask, `itemCopy` **nunca se libera**.
3. **Y**: si `addBtn` se clickea, el callback de `LV_EVENT_CLICKED` elimina `maskObj` (con `lv_obj_delete_async`). Luego LV_EVENT_DELETE se dispara → `delete itemCopy`. **Correcto.**
4. **Pero**: si el callback de `LV_EVENT_DELETE` se ejecuta **antes** de que el callback de `LV_EVENT_CLICKED` termine (en teoría no debería pasar porque la eliminación es async, pero LVGL ejecuta eventos durante `lv_timer_handler()`), podría haber una condición de carrera.

**Problema fundamental**: `itemCopy` se pasa como user data a dos botones (`addBtn` para LV_EVENT_CLICKED y LV_EVENT_DELETE). Aunque esto no es un double-free (porque DELETE es un evento de cleanup, no de click), el diseño es confuso y frágil. Un mantenedor podría agregar otro callback en otro botón con el mismo `itemCopy` sin darse cuenta.

### 6.2 Ausencia de LV_EVENT_DELETE en Otras Vistas

Ninguna otra vista usa `LV_EVENT_DELETE`. Los `HeaderBar*`, los punteros a contenedores LVGL, y las referencias estáticas **nunca se limpian**. Esto significa que si una vista se recrea, todos los punteros estáticos de la vista anterior quedan colgando.

---

## 7. Recomendaciones Arquitectónicas

### 7.1 Solución Inmediata: Gestión de activeHeader

Reemplazar el puntero raw con un patrón observer o un weak pointer:

```cpp
// Opción A: Validación antes de usar
static HeaderBar* activeHeader;
static bool isHeaderValid(HeaderBar* hb) {
    return hb && hb->container && lv_obj_is_valid(hb->container);
}
// Usar isHeaderValid() en todas las funciones estáticas
```

```cpp
// Opción B: Usar un ID único en vez de puntero
// (menos intrusivo)
static uint32_t activeHeaderId; // Cada HeaderBar tiene un ID único
// setActiveHeader asigna el ID; updateActiveXxx verifica por ID
```

**Recomendación**: Opción A + eliminar el caché de screens en UIManager (recrear siempre, es más seguro en un ESP32 con memoria limitada).

### 7.2 Ciclo de Vida de Vistas

1. **Destructores virtuales o métodos `destroy()`**: Cada vista debe tener un método `static void destroy(lv_obj_t* screen)` que limpie todos los punteros estáticos y libere la memoria C++ (`HeaderBar*`, etc.) antes de eliminar la screen LVGL.

2. **Eliminar el caché de screens**: En un sistema embebido con 512KB-8MB de RAM, cachear screens es contraproducente. Es más seguro recrear siempre:

   ```cpp
   void UIManager::loadDashboard() {
       if (dashboardScreen) {
           DashboardView::destroy(dashboardScreen);
       }
       dashboardScreen = DashboardView::create();
       ...
   }
   ```

3. **Consistencia en el cleanup de toasts**: Unificar todas las implementaciones de toast en un solo componente (por ejemplo, `ToastManager`) en `lv_layer_top()`.

### 7.3 CartFloatingButton

1. Eliminar `static int lastCount` local de función; usar una variable miembro estática con reseteo en `init()`.
2. Agregar un callback desde `CartManager` para notificar cambios al botón sin polling.

### 7.4 SmartCallModal

Eliminar por completo. Migrar `showToast()` a un `ToastManager` compartido. Eliminar `show()` y `close()` (redundantes desde que existe `CallWaiterView` como pantalla completa).

### 7.5 LV_EVENT_DELETE y Gestión de user_data

Regla: **cada `lv_obj_t` debe tener exactamente un owner del `user_data`**. Si se pasa un puntero `new`-allocated como user data, registrar un solo `LV_EVENT_DELETE` callback que haga `delete`. No compartir user data entre múltiples objetos.

### 7.6 Desacoplamiento

Romper el acoplamiento circular:
- `UIManager` no debe incluir todas las vistas. Usar registro o forward declarations.
- `HeaderBar` no debe incluir `UIManager`. El `back_event_cb` debería ser configurable (callback externo).

---

## 8. Conclusión

El sistema UI tiene una **deuda arquitectónica severa**: punteros globales sin protección de ciclo de vida, caché de screens sin invalidación, 4 implementaciones de toast, y un componente legacy (SmartCallModal) que coexiste con su reemplazo (CallWaiterView). La combinación de `HeaderBar::activeHeader` como puntero raw + vistas estáticas sin destructor + screens cacheadas crea una red de dependencias donde modificar una vista correctamente (ej: fix de use-after-free en MenuView) no evita que otra vista (ej: DashboardView al recargar) crashee por un dangling pointer.

**La causa raíz de "arreglar una vista rompe otra" no son los bugs individuales, sino la ausencia de un contrato de ciclo de vida claro entre LVGL (dueño de los objetos widget) y C++ (dueño de los wrappers HeaderBar).** Hasta que no se sincronicen ambos ciclos de vida (o se eliminen los wrappers C++ a favor de datos directamente en LVGL), cualquier reparación será frágil.

---

## 9. Mapa de Bugs Activos Post-Fix

| # | Bug | Archivo | Líneas | Severidad | Estado |
|---|-----|---------|--------|-----------|--------|
| A | `lastCount` estático en `CartFloatingButton::update()` | CartFloatingButton.cpp | 67 | Bajo | **Activo** |
| B | `toastTimer` no se resetea a nullptr tras eliminar toast en `CallWaiterView::showToast()` | CallWaiterView.cpp | 131-135 | Medio | **Activo** |
| C | `SmartCallModal::close()` no funciona (overlay nunca asignado) | SmartCallModal.cpp | 122-127 | Medio | **Activo** |
| D | `SmartCallModal::show()` usa `lv_screen_active()` | SmartCallModal.cpp | 27 | Alto | **Activo** |
| E | HeaderBar* colgante si screen se recrea | Todas las vistas | - | Alto | **Activo** |
| F | Fuga de `MenuItem*` si modal se cierra externamente | MenuView.cpp | 340-345 | Medio | **Activo** |
| G | Recreación masiva en renderItems (rendimiento) | CartView.cpp | 72-195 | Alto | **Activo** |
| H | 4 implementaciones duplicadas de toast | SmartCallModal, MenuView, CartView, CallWaiterView | - | Bajo | **Activo** |

Los bugs #0.1-0.7 del plan de solución original corrigen problemas sintomáticos, pero no resuelven los problemas arquitectónicos fundamentales (E, F, H) que son la verdadera causa de propagación de fallos entre vistas.
