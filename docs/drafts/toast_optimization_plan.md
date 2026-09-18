# Plan de Optimización y Unificación de Notificaciones (Toasts)

Este documento detalla la estrategia para simplificar la arquitectura de interfaz de usuario de **TableHub**, eliminando las 4 implementaciones redundantes de mensajes temporales (*toasts*) y consolidando el flujo táctil del sistema de mesa.

---

## 1. El Problema: Redundancia y Riesgo de Memoria

Actualmente coexisten cuatro lógicas idénticas para dibujar toasts sobre la pantalla, cada una con su propia gestión de memoria para objetos LVGL y temporizadores (`lv_timer_t`):
- `SmartCallModal::showToast()`
- `MenuView::showToast()`
- `CartView::showToast()`
- `CallWaiterView::showToast()`

### Impactos Negativos:
- **Uso de Memoria**: La duplicación de funciones estáticas estresa la memoria Flash del ESP32-S3.
- **Riesgo de Crashes**: Si un toast es interrumpido por otro, la eliminación síncrona con `lv_obj_del` y la manipulación de handles de timers destruidos (*dangling timers*) provocan fallos de tipo `StoreProhibited`.

---

## 2. Propuesta: Centralización en `UIManager`

Se propone unificar toda la lógica en una única función estática y segura administrada por el ciclo de vida de `UIManager`:

```cpp
// En UIManager.h
public:
    static void showToast(const char* message);

private:
    static lv_obj_t* toastObj;
    static lv_timer_t* toastTimer;
    static void toast_timer_cb(lv_timer_t* timer);
```

### Flujo Seguro de Destrucción:
- La creación del toast se realiza exclusivamente sobre `lv_layer_top()`.
- Antes de instanciar un nuevo toast, se elimina de forma asíncrona y segura (`lv_obj_delete`) cualquier toast anterior y se cancela su temporizador activo.

---

## 3. Plan de Eliminación de Código Duplicado

Se removerán por completo las siguientes estructuras redundantes:

1. **`SmartCallModal::showToast`** y **`SmartCallModal::toast_timer_cb`**.
2. **`CallWaiterView::showToast`** y **`CallWaiterView::toast_timer_cb`**.
3. **`MenuView::showToast`** y **`MenuView::toast_timer_cb`**.
4. **`CartView::showToast`** y **`CartView::toast_timer_cb`**.

Cada una de estas llamadas será redirigida a la función centralizada:
```cpp
UIManager::showToast("Tu mensaje aquí");
```

---

## 4. Evolución y Actualizaciones

### [2026-07-24] — Implementación Completa

**Estado:** ✅ Implementado

Se ejecutó el plan de centralización completo sobre el firmware del ESP32-S3:

#### Archivos modificados

| Archivo | Acción |
|---|---|
| `UIManager.h` | Añadida declaración `static void showToast(const char*)`, statics `toastObj`, `toastTimer` y `toast_timer_cb` |
| `UIManager.cpp` | Implementación de `showToast` y `toast_timer_cb`. Inicialización de statics. Include de `NeumorphicStyles.h` |
| `SmartCallModal.h` | Eliminados `showToast`, `toastObj`, `toastTimer`, `toast_timer_cb` |
| `SmartCallModal.cpp` | Eliminados statics e implementación local. Llamada redirigida a `UIManager::showToast` |
| `CallWaiterView.h` | Eliminados `showToast`, `toastObj`, `toastTimer`, `toast_timer_cb` |
| `CallWaiterView.cpp` | Eliminados statics e implementación local. Llamada redirigida a `UIManager::showToast` |
| `CartView.h` | Eliminada declaración de `showToast` |
| `CartView.cpp` | Eliminada implementación local. Llamada redirigida a `UIManager::showToast` |
| `MenuView.h` | Eliminada declaración de `showToast` |
| `MenuView.cpp` | Eliminada implementación local. Llamada redirigida a `UIManager::showToast` |

#### Mejoras de la implementación centralizada vs. las locales
- **Timer único global**: Un solo par `toastObj / toastTimer`. Si llega un nuevo toast mientras hay uno activo, el anterior se cancela y destruye limpiamente antes de crear el nuevo — sin *dangling timers*.
- **`lv_layer_top()` garantizado**: El toast siempre se pinta sobre cualquier pantalla o modal activo.
- **Estilo unificado**: Todos los toasts tienen el mismo aspecto (`applySunkenCard`, borde accent, `lv_font_montserrat_12`), eliminando variaciones visuales entre vistas.
- **Reducción de código**: ~120 líneas de código duplicado eliminadas.
