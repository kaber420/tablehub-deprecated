# Plan de Diseño: Rediseño Neumórfico de Dispositivos y Sincronización Hub-ESP32

**Fecha:** 2026-07-26  
**Proyecto:** TableHub2 - Módulo de Dispositivos & Frontend Hub (`hub/web/src/lib/DevicesView.svelte`)  
**Estado:** Borrador para Edición y Revisión del Usuario  

---

## 1. Problemas Identificados en la Interfaz Actual

1. **Inconsistencia de Estilo (Tarjetas Planas Tipo Tailwind):**
   * Las tarjetas de dispositivos usan actualmente un borde plano de color a la izquierda (`border-left: 4px solid #10b981 / #f59e0b / #ef4444`).
   * Esto rompe completamente con la estética Neumórfica del resto de TableHub (sombras suaves, volumen de extruido/hundido, acabado matte profesional).
2. **Botones fuera del Tema Neumórfico:**
   * Hay botones con estilos hardcodeados en línea (`style="background: #10b981..."`) en lugar de utilizar el sistema de diseño del proyecto (`.neo-btn`, `.neo-btn-primary`, `.neo-btn-danger`, etc.).
3. **Botón superior redundante:**
   * La cabecera superior tiene un botón global de `"Aprovisionamiento SD"` que confunde al usuario, ya que el archivo `tablehub.enc` se genera individualmente por mesa.
4. **Etiqueta engañosa en tarjeta:**
   * El botón se llama `"Cifrar SD"`, cuando su función real es **descargar el paquete de aprovisionamiento cifrado (`tablehub.enc`)**.

---

## 2. Propuesta de Rediseño Neumórfico Consistente

### 2.1. Rediseño de Tarjetas de Dispositivo (`.device-card`)
* **Eliminación Total de la Pestaña Lateral de Color:** Se remueve el `border-left: 4px solid ...`. No más tarjetas planas genéricas.
* **Cuerpo y Volumen Neumórfico (`.panel` / `.neo-card`):**
  * Superficie plana con sombras suaves proyectadas (`var(--panel-shadow)` y hover neumórfico `var(--panel-shadow-hover)`).
* **Indicador de Estado Neumórfico (Glowing Status Pill):**
  * El estado (`Activo`, `En Espera`, `Bloqueado`) se integrará en la esquina superior derecha con una **píldora neumórfica rehundida** (`inset shadow`) que incluye un **punto luminoso brillante con pulso (Glowing Status Dot)**:
    - 🟢 **Verde neón brillante con pulso:** Dispositivo Enlazado y Activo (`active`).
    - 🟡 **Amarillo neón:** Esperando Aprovisionamiento (`unprovisioned`).
    - 🔴 **Rojo neón:** Bloqueado (`blocked`).
* **Sección de Datos e Información (`.info-grid` y Clave Pública):**
  * Todos los recuadros de información técnica (IP, Batería, Señal Wi-Fi, Clave Ed25519) utilizarán **paneles neumórficos rehundidos (`--panel-inset-bg` / `--panel-inset-shadow`)**, logrando un aspecto de consola integrada.

---

### 2.2. Estandarización de Botones Neumórficos (`.neo-btn`)

Se eliminarán todos los estilos CSS planos e inline, adoptando el sistema de botones neumórficos global:

#### Cabecera Superior (`filters-row`)
* 🟩 **`+ Crear Mesa`**: `<button class="neo-btn neo-btn-success">`  
  *(Botón neumórfico extruido con relieve y texto/icono verde neón).*
* 🟪 **`Configurar Wi-Fi`**: `<button class="neo-btn neo-btn-primary">`  
  *(Abre directamente el gestor modal de redes Wi-Fi guardadas en el Hub).*
* ⬜ **`Refrescar`**: `<button class="neo-btn">`  
  *(Botón neumórfico neutro).*
* ❌ **Eliminación:** Se elimina el botón superior global `Aprovisionamiento SD`.

#### Pie de Tarjeta (`card-footer`)
* 📥 **`Descargar Aprovisionamiento`**: `<button class="neo-btn">`  
  *(Reemplaza a "Cifrar SD". Genera y descarga el binario `tablehub.enc` para la MicroSD).*
* ⚙️ **`Aprobar` / `Configurar`**: `<button class="neo-btn neo-btn-primary">`  
  *(Botón neumórfico acentuado).*
* 🔒 **`Bloquear` / `Activar`**: `<button class="neo-btn neo-btn-warning">`  
  *(Botón neumórfico de advertencia).*
* 🗑️ **Icono Eliminar**: `<button class="neo-btn neo-btn-danger">`  
  *(Botón neumórfico rojo de peligro).*

---

## 3. Recomendación de Arquitectura: Sincronización Directa Hub -> ESP32

### 3.1. Consulta del Usuario
* ¿Es posible sincronizar la placa ESP32 directamente desde el Hub sin tener que usar siempre el archivo en la MicroSD?
* ¿Es recomendable permitir la configuración directa desde el panel Hub?

### 3.2. Respuesta y Arquitectura Recomendada (Modelo Híbrido)

**Recomendación:** **100% Sí.** Se recomienda implementar un **Modelo Híbrido de 2 fases**:

#### Fase 1: Bootstrapping Inicial (Offline / MicroSD)
* La tarjeta MicroSD con el archivo `tablehub.enc` se utiliza únicamente en el **primer encendido/aprovisionamiento** de una placa nueva o tras un reset de fábrica.
* Su propósito es entregar de forma segura la red Wi-Fi inicial y el `bootstrap_token` o clave pública sin requerir portales Wi-Fi abiertos.

#### Fase 2: Sincronización en Vivo y Reconfiguración (Online / MQTT + NVS)
* **Una vez que el ESP32 ha completado la adopción y está conectado al Hub vía Wi-Fi/MQTT:**
  1. **Configuración desde el Hub:** Desde el panel web del Hub, el administrador puede modificar el número de mesa, ubicación, red Wi-Fi de destino o PIN.
  2. **Envío en Caliente:** El Hub publica la nueva configuración mediante mensajes MQTT cifrados en el tópico `tablehub/device/{mac}/config`.
  3. **Persistencia en Flash:** El firmware del ESP32 recibe el mensaje y guarda las nuevas credenciales/configuraciones en su memoria **NVS (Non-Volatile Storage)** interna.
  4. **Resultado:** El ESP32 aplica los cambios de inmediato (o se reconecta al nuevo Wi-Fi) **sin necesidad de tocar ni reescribir la MicroSD jamás**.

---

## 4. Archivos a Modificar
1. **[hub_device_buttons_and_sync_plan.md](file:///home/kaber420/Documentos/proyectos/tablehub2/docs/drafts/hub_device_buttons_and_sync_plan.md)** (Este borrador de especificación).
2. **`hub/web/src/lib/DevicesView.svelte`**: Rediseño visual neumórfico de la vista de dispositivos.
3. **`hub/web/src/app.css`**: Utilidades adicionales `.neo-btn-success`, `.neo-btn-warning` si se requieren.

---

## 5. Notas para Edición del Usuario
*(Este borrador está listo en `docs/drafts/hub_device_buttons_and_sync_plan.md` para que lo edites, modifiques o apruebes).*
