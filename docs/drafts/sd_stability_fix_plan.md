# Plan de Estabilidad para la Memoria SD (SPI)

## El Problema
Al presionar "Aceptar" en la pantalla del PIN, la tarjeta SD se desconectaba repentinamente (arrojando el error de que el archivo no existe), a pesar de haberlo leído bien al encender la mesa.

## Análisis de Hardware
La pantalla JC3248W535 se comunica con la SD a través del bus SPI. Al usar `SD.begin(10)` sin especificar la velocidad, el ESP32 intenta negociar automáticamente la frecuencia del reloj (SCK). Esto provoca que la tarjeta responda al primer chequeo rápido durante el arranque, pero colapse bajo estrés cuando intentamos abrir el archivo de forma sostenida (`SD.open`). 

Además, **no revertiremos los cambios anteriores**, la UI neomórfica y la eliminación de las llamadas redundantes en el ConfigManager se mantienen intactas.

## Solución C++ Propuesta
Aplicar un blindaje en la negociación de frecuencia del bus SPI en `main.cpp`, bajándola a un nivel ultra-estable (4 MHz), que es el estándar de máxima compatibilidad en hardware integrado para lecturas de tarjetas micro-SD genéricas por SPI.

En `main.cpp`:
```cpp
// Antes:
bool sdMounted = SD.begin(10);

// Nuevo (4 MHz blindados):
bool sdMounted = SD.begin(10, SPI, 4000000);
```

Este simple ajuste estabilizará el bus permanentemente y evitará que la tarjeta se desconecte mientras el usuario teclea el PIN.
