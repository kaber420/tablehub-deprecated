# Plan de Implementación: Interfaz de Registro y Descarga de `.thub` en la Nube

Este plan detalla los cambios para crear la interfaz visual (modal y botón) en el panel de la Nube (`Devices.svelte`) para que los clientes puedan registrar un Hub físicamente, asociarlo a una sucursal y descargar el archivo de credenciales `.thub` directamente desde el navegador, eliminando la necesidad de usar comandos `curl`.

---

## Cambios Propuestos

### 1. Nube: Frontend Web (Svelte)

#### [MODIFY] [Devices.svelte](file:///home/kaber420/Documentos/proyectos/tablehub2/cloud/web/src/pages/Devices.svelte)
- **Variables de Estado:**
  - Agregar `showProvisionModal = false` para controlar la visibilidad del modal de registro.
  - Agregar `selectedBranchId = ""` para guardar la sucursal seleccionada en el formulario.
  - Agregar `provisioning = false` para deshabilitar botones durante la llamada a la API.
- **Acciones y Lógica:**
  - Implementar la función `registerHub()`:
    - Realizar una petición POST HTTP con `apiFetch` al endpoint `/v1/orgs/${orgID}/hubs/provision` enviando `{ branch_id: selectedBranchId }` en formato JSON.
    - Leer la respuesta como un objeto binario (`blob`).
    - Crear un enlace temporal en el DOM, simular un clic para forzar la descarga del archivo con el nombre `tablehub.thub` y liberar el objeto URL de memoria.
    - Cerrar el modal, limpiar la selección y volver a invocar `loadData()` para refrescar el listado de hubs en pantalla (el cual mostrará el nuevo Hub recién registrado en estado `OFFLINE`).
- **Diseño e Interfaz (HTML/CSS):**
  - Agregar el botón **"Registrar Nuevo Hub"** en el panel superior (toolbar) al lado del buscador.
  - Diseñar el modal flotante con:
    - Selector dropdown (`<select>`) alimentado por la lista de sucursales del cliente (`branches`) para seleccionar la sucursal.
    - Botones de acción: "Generar y Descargar (.thub)" y "Cancelar".
  - Añadir estilos CSS en el bloque `<style>` para centrar el modal con un fondo traslúcido y estilo premium tipo cristal (glassmorphism).

---

## Plan de Verificación

### Pruebas de Flujo Visual (Manual)
1. Levantar el backend y frontend de desarrollo de la Nube.
2. Iniciar sesión en el portal del cliente y navegar a la sección de **Dispositivos**.
3. Hacer clic en **"Registrar Nuevo Hub"**.
4. Seleccionar una sucursal en el menú desplegable del modal y hacer clic en **"Generar y Descargar"**.
5. Verificar que:
   - El navegador descargue un archivo válido llamado `tablehub.thub`.
   - El listado de la pantalla se actualice y muestre la tarjeta del nuevo Hub con estado `OFFLINE` y asignado a la sucursal elegida.
6. Subir el archivo en el panel local del Hub de desarrollo y verificar que se complete la activación Ed25519 con el backend.
