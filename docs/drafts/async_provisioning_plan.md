# Plan de Refactorización: Aprovisionamiento Asíncrono (FreeRTOS)

## El Problema Diagnosticado por Opencode/Mimo
El lag extremo de la interfaz ("casi congelada") después de meter el PIN tiene 3 causas críticas que saturan el hilo principal de la pantalla:
1. **Bloqueo de SD.open():** Cuando la memoria SD falla o tarda en responder por inestabilidad, la función `SD.open()` bloquea la ejecución entre 2 y 10 segundos continuos. Durante ese tiempo, la pantalla deja de actualizarse por completo (0 FPS).
2. **Criptografía Pesada Síncrona:** La validación del PIN ejecuta 10,000 iteraciones del algoritmo criptográfico PBKDF2-HMAC-SHA256 (`mbedtls`). En el ESP32, esto tarda hasta 5 segundos más, congelando la UI.
3. **Apilamiento Infinito de Modales:** Si tocas el botón repetidamente mientras está "congelado", cada pulsación crea un nuevo proceso pesado en la cola y dibuja un modal de error nuevo encima del anterior, colapsando la memoria gráfica RAM.

## Solución C++ Propuesta (FreeRTOS + Protección UI)

Para que la pantalla del PIN jamás se vuelva a congelar, moveremos todo el trabajo criptográfico y de disco a un hilo de fondo del procesador, dejando el hilo de la pantalla 100% libre.

### 1. Deshabilitar el Teclado al Procesar (Protección UI)
En `SetupPinView.cpp`:
- Antes de procesar: `lv_obj_add_state(btnm, LV_STATE_DISABLED);`
- Si falla: Se rehabilita el teclado al cerrar el modal de error.
- Esto evita el "apilamiento infinito" de pulsaciones.

### 2. Procesamiento Asíncrono con FreeRTOS
En lugar de llamar a `ConfigManager::attemptProvisioning` de forma directa, se envolverá en una Tarea FreeRTOS (`xTaskCreatePinnedToCore`):

```cpp
// Pseudocódigo de la solución a implementar en SetupPinView.cpp
void SetupPinView::attempt_provisioning(const std::string& pin) {
    lv_obj_add_state(btnm, LV_STATE_DISABLED); // Bloquear botones
    
    // Crear copia del PIN en el heap para el hilo secundario
    std::string* pin_copy = new std::string(pin);

    // Mandar el trabajo pesado al núcleo secundario del ESP32
    xTaskCreatePinnedToCore([](void* param) {
        std::string* p = (std::string*)param;
        String err;
        bool ok = ConfigManager::getInstance().attemptProvisioning(*p, err);
        
        // Devolver el resultado a la pantalla de forma segura usando LVGL Async
        // ... (dibujar modal de éxito o error)
        
        delete p;
        vTaskDelete(NULL);
    }, "ProvisionTask", 8192, pin_copy, 1, NULL, 1);
}
```

Esta refactorización garantizará que la interfaz responda fluidamente sin importar cuánto tarde la memoria SD o la desencriptación.
