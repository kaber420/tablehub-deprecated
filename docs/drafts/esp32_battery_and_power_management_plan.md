# Plan de Monitoreo de Batería y Alimentación USB (JC3248W535)

## 1. Contexto y Hardware
La pantalla / dispositivo **Guition JC3248W535 (ESP32-S3)** cuenta con:
- Circuito integrado de carga LiPo mediante el puerto USB-C.
- Medición analógica del voltaje de batería a través de un divisor de tensión interno conectado al **GPIO 5** (Canal ADC1_4 del ESP32-S3).

Actualmente, el header (`HeaderBar`) utiliza un valor mock estático (85%) y una lectura de WiFi mock/real.

---

## 2. Objetivos
1. **Medición Real de la Batería:** Leer el pin ADC GPIO 5 para calcular el porcentaje real de carga de la batería LiPo.
2. **Detección de Fuente de Energía (USB / AC vs Batería):** 
   - Cuando el dispositivo está conectado a USB-C / AC, el chip cargador mantiene un voltaje de flotación (~4.15V - 4.25V).
   - Detectar la presencia de energía AC/USB para alternar el indicador UI.
3. **Indicador dinámico en `HeaderBar`:**
   - **Modo AC/Cargando:** Mostrar icono de rayo / enchufe (`⚡` o símbolo de carga LVGL) cuando se detecte alimentación externa.
   - **Modo Batería:** Mostrar icono de batería proporcional (0% - 100%) indicando la autonomía restante cuando se opera de forma inalámbrica.

---

## 3. Arquitectura Propuesta

### A. Componente `BatteryDriver` (`firmware/src/Hardware/BatteryDriver.h` y `.cpp`)
Crear una clase singleton o driver estático en el firmware para encapsular el muestreo del ADC:

- **Configuración ADC:**
  - Pin: `GPIO 5`
  - Atenuación: `ADC_ATTEN_DB_12` (para rango 0V - 3.3V en el ESP32-S3).
- **Procesamiento de Lecturas:**
  - **Filtro de promedio móvil:** Realizar 10-16 muestras por segundo para eliminar el ruido inherente al ADC del ESP32-S3.
  - **Mapeo de Voltaje:**
    - `Voltage <= 3.2V` -> `0%`
    - `Voltage >= 4.1V` (sin USB) -> `100%`
    - `Voltage > 4.18V` -> **Modo Alimentación AC / Cargando** (USB conectado).

### B. Integración con `HeaderBar`
Actualizar `HeaderBar::updateActiveBattery(int percentage, bool isCharging)`:
- Si `isCharging == true`:
  - Aplicar estilo verde/activo con símbolo de rayo/enchufe.
- Si `isCharging == false`:
  - Renderizar el relleno (`batteryFill`) con el ancho proporcional al porcentaje medido.
  - Cambiar el color del relleno a rojo (<20%), amarillo (<40%) o verde (>40%).

### C. Ciclo de Ejecución en `main.cpp`
En el `loop()` principal de Arduino, ejecutar la actualización periódica (cada 2 a 5 segundos) para no saturar la CPU:
```cpp
#ifdef ARDUINO
static uint32_t lastBatteryCheck = 0;
if (millis() - lastBatteryCheck > 3000) {
    lastBatteryCheck = millis();
    BatteryState state = BatteryDriver::getInstance().read();
    HeaderBar::updateActiveBattery(state.percentage, state.isCharging);
}
#endif
```

---

## 4. Evolución y Notas Futuras
- **Calibración:** Dependiendo de las tolerancias de las resistencias del divisor en la placa JC3248W535, se puede agregar un factor de escala ajustable (`calibration_factor`) en NVS/ConfigManager.
- **Deep Sleep:** Si el voltaje de la batería baja de 3.1V, enviar el ESP32-S3 a Deep Sleep para proteger la batería de sobre-descarga.
