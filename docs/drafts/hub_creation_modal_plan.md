# Plan de Implementación: Modal de Creación de Hubs Físicos

## Descripción del Problema
Actualmente, en la sección de Hubs (`cloud/web/src/pages/Devices.svelte`), la creación de un nuevo hub físico se realiza de forma automática al hacer clic en el botón "Nuevo Hub Físico". Esto genera un hub vacío, sin datos asociados (como sucursal, motor POS o modelo de hardware). El usuario debe editarlo posteriormente para añadir esta información antes de poder generar el archivo `.thub` de aprovisionamiento.

## Solución Propuesta
Añadir un modal de creación (similar al modal de edición existente) que se abra al hacer clic en "Nuevo Hub Físico". Este modal solicitará los datos iniciales del hub antes de crearlo en el backend. Una vez guardado, el hub se creará con todos sus metadatos y estará listo para generar el archivo de aprovisionamiento de inmediato.

## Cambios Propuestos (Incluyendo sugerencias de DeepSeek)

### Componente Frontend: `cloud/web/src/pages/Devices.svelte`
- **Estado del Modal:** Añadir `let showCreateModal = false;` y variables para manejo de errores/éxito inline (evitando el uso de `alert()`).
- **Formulario:** Añadir `let createFormData = { branch_id: "", hardware_model: "", pos_provider: "" };`
- **Validación:** Requerir obligatoriamente que se seleccione una sucursal antes de permitir el envío.
- **Flujo Post-Creación:** Al crearse el hub exitosamente, el modal mostrará un estado de éxito con dos opciones: "Descargar .thub" y "Cerrar".
- **UX del Modal:** Prevenir el cierre accidental al hacer clic en el fondo (backdrop) si hay datos rellenados, y manejar el cierre con la tecla `Escape`.
- **Refactorización:** Extraer los proveedores POS ("Blackshot", "Toast", etc.) a una constante para usarlos tanto en creación como edición.

### Cliente API: `cloud/web/src/lib/api.js`
- Modificar la función `createHub(orgID)` para aceptar un segundo parámetro: `createHub(orgID, payload = {})`.
- **Refactorización:** Cambiar la implementación interna de `createHub` para utilizar `apiFetch` (manteniendo consistencia con `updateHub` y el resto de la API).

## Verificación
- Comprobar que al pulsar "Nuevo Hub Físico" se abre el modal.
- Rellenar los campos, validar que no se pueda enviar sin sucursal.
- Enviar, verificar manejo de errores inline si falla.
- Comprobar la transición a la pantalla de éxito con la opción de descargar el archivo `.thub`.
- Descargar el archivo `.thub` y cerrar el modal.
