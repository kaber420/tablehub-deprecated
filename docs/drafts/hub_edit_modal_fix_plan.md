# Plan de Rediseño: Modal de Configuración de Hub (Edit Modal)

## 1. Diagnóstico del Problema

### Ubicación
`cloud/web/src/pages/Devices.svelte:346-391`

### Problemas de Diseño
1. **Sin animación/transición** — el modal aparece/desaparece instantáneamente, sin `fade` ni `scale`
2. **Sin ícono de cabecera** — el provision modal tiene `<DownloadCloud size={48}>`, el edit modal no tiene nada visual
3. **Backdrop inconsistente** — el provision modal usa `.modal-panel` (centrado, padding 2.5rem, max-width 440px), el edit modal usa `.modal-content` (padding 2rem, sin sombra consistente)
4. **Sin información del hub actual** — no muestra ID, estado, ni datos actuales del hub que se está editando
5. **Formulario sin secciones ni agrupación visual** — los campos están simplemente apilados sin separación

### Problemas de Funcionalidad
1. **Sin manejo de tecla Escape** — el provision modal lo maneja via `<svelte:window on:keydown>`, el edit modal no
2. **Sin confirmación al cerrar con datos sucios** — si el usuario llena campos y hace clic en el backdrop, se pierde todo sin advertencia
3. **Sin estado de error inline** — si `updateHub()` falla, se muestra un `alert()` del navegador en vez de un error dentro del modal
4. **Sin estado de éxito** — al guardar exitosamente, el modal simplemente se cierra sin confirmación visible
5. **Campos editables durante el envío** — `updatingHub` no deshabilita los campos del formulario, solo los botones
6. **Sin validación visual** — no hay feedback visual sobre qué campos son requeridos

---

## 2. Propuesta de Diseño

### Layout del Modal (`.modal-panel`)

```
┌─────────────────────────────────────┐
│  ◉  Ícono decorativo (Settings)     │
│                                     │
│  Configurar Hub                     │
│  Asigna sucursal y configuración    │
│                                     │
│  ┌─ Info del Hub ──────────────┐   │
│  │  ID: hub_abc123...          │   │
│  │  Estado: ONLINE             │   │
│  │  Sucursal actual: Sucursal X│   │
│  └─────────────────────────────┘   │
│                                     │
│  ┌─ Asignación ───────────────┐   │
│  │  Sucursal Asignada [select] │   │
│  │  Modelo Hardware  [input]   │   │
│  └─────────────────────────────┘   │
│                                     │
│  ┌─ Configuración POS ─────────┐   │
│  │  Motor POS         [select] │   │
│  └─────────────────────────────┘   │
│                                     │
│  [◀] Error message                 │
│                                     │
│  [Cancelar]   [Guardar Cambios ◀]   │
└─────────────────────────────────────┘
```

### Principios de Diseño
- **Coherencia con el provision modal:** mismo `modal-backdrop`, `modal-panel`, animaciones de entrada/salida
- **Glassmorphism consistente:** fondo semitransparente, blur, borde sutil
- **Información contextual:** mostrar datos actuales del hub para dar referencia
- **Agrupación lógica:** secciones "Asignación" y "Configuración POS" con subtítulos y bordes
- **Feedback visual:** estados de loading, error inline, éxito con auto-cierre

---

## 3. Funcionalidad Detallada

### 3.1 Apertura del Modal
- `openEditModal(hub)` — setea `editingHub`, rellena `editFormData`, muestra el modal
- El modal debe mostrar información de solo lectura del hub actual:
  - ID (truncado con tooltip)
  - Estado de conexión (online/offline)
  - Sucursal actual asignada
  - Modelo de hardware actual
  - Motor POS actual

### 3.2 Manejo de Teclado
- `<svelte:window on:keydown>` debe manejar Escape para cerrar el modal (con chequeo de dirty form)
- Si hay datos modificados y se presiona Escape, mostrar confirmación

### 3.3 Protección de Datos Sucios (Dirty Form)
- `let isDirty = false`
- Actualizar vía reactive statement: `$: isDirty = editFormData.branch_id !== (editingHub?.branch_id || "") || ...`
- Al cerrar (backdrop click o Escape) si `isDirty`, mostrar confirmación con `confirm()` o un sub-modal
- Alternativa más elegante: pequeño diálogo de confirmación inline "¿Descartar cambios?"

### 3.4 Validación de Campos
- `branch_id`: requerido (no puede quedar "Sin asignar") — si está vacío, botón de guardar deshabilitado y tooltip/mensaje
- `hardware_model`: opcional, sin validación
- `pos_provider`: opcional, sin validación

### 3.5 Envío del Formulario
- `handleUpdateHub()` ya existe pero debe:
  1. Deshabilitar TODO el formulario (inputs, selects, botones) durante `updatingHub`
  2. Mostrar spinner en el botón de guardar (ya existe)
  3. En caso de error, mostrar mensaje inline en `.modal-alert-error` en vez de `alert()`
  4. En caso de éxito, mostrar estado de éxito (ícono check + mensaje + auto-cierre en 3s, similar al provision modal)
  5. Recargar datos (`loadData()`)

### 3.6 Estados del Modal

| Estado | Descripción |
|---|---|
| `idle` | Formulario activo, botones habilitados |
| `submitting` | Formulario deshabilitado, spinner en botón |
| `error` | Mensaje de error inline, formulario rehabilitado |
| `success` | Ícono check + mensaje + auto-cierre 3s |

### 3.7 Transiciones Svelte
- Modal backdrop: `transition:fade`
- Modal panel: `transition:scale={{ start: 0.95, opacity: 0 }}` + `transition:fade`
- El provision modal no tiene transiciones — agregarlas también mejora la UX general

---

## 4. Archivos a Editar

### 4.1 `cloud/web/src/pages/Devices.svelte` (principal)

**Cambios en `<script>`:**

```diff
+ import { fade, scale } from 'svelte/transition';
+ import { Settings } from '@lucide/svelte';

  let showEditModal = false;
  let editingHub = null;
+ let editModalState = 'idle'; // idle | submitting | error | success
+ let editError = '';
+ let isDirty = false;
+ let editSuccessTimer = null;

  let editFormData = { branch_id: "", hardware_model: "", pos_provider: "" };
  let updatingHub = false;
```

**Nuevas funciones:**
- `closeEditModal()` — cierra el modal, limpia timers, resetea estado
- `handleEditKeydown(e)` — si `showEditModal && e.key === 'Escape'`, cerrar con dirty check
- `confirmDiscard()` — si `isDirty`, preguntar antes de descartar
- Bloque reactivo `$: isDirty = ...` comparando `editFormData` contra `editingHub`

**Template editado (reemplazar bloque `{#if showEditModal}` actual):**

```svelte
{#if showEditModal}
<div class="modal-backdrop" transition:fade on:click|self={confirmDiscard}>
    <div class="modal-panel" transition:scale={{ start: 0.95, opacity: 0 }}>
        {#if editModalState === 'success'}
            <div class="modal-icon-success">
                <CheckCircle size={48} />
            </div>
            <h2>Hub Actualizado</h2>
            <p class="modal-description">
                Los cambios se guardaron correctamente.
                {#if editingHub}
                    <strong>{editingHub.id.substring(0, 12)}...</strong>
                {/if}
            </p>
            <button class="btn-close-modal" on:click={closeEditModal}>Cerrar</button>
        {:else}
            <div class="modal-icon-info">
                <Settings size={48} />
            </div>
            <h2>Configurar Hub</h2>
            <p class="modal-description">
                Asigna una sucursal y define el hardware y software de este equipo.
            </p>

            <!-- Hub Info Card -->
            <div class="edit-hub-info">
                <div class="edit-info-row">
                    <span class="edit-info-label">ID</span>
                    <span class="edit-info-value mono" title={editingHub?.id}>
                        {editingHub?.id?.substring(0, 16)}...
                    </span>
                </div>
                <div class="edit-info-row">
                    <span class="edit-info-label">Estado</span>
                    <span class="edit-info-value">
                        <span class="led-indicator led-{editingHub?.connection_status}"></span>
                        {(editingHub?.connection_status || '').toUpperCase()}
                    </span>
                </div>
                <div class="edit-info-row">
                    <span class="edit-info-label">Sucursal</span>
                    <span class="edit-info-value">{getBranchName(editingHub?.branch_id)}</span>
                </div>
                <div class="edit-info-row">
                    <span class="edit-info-label">Hardware</span>
                    <span class="edit-info-value">{editingHub?.metadata?.hardware_model || 'Genérico'}</span>
                </div>
                <div class="edit-info-row">
                    <span class="edit-info-label">Motor POS</span>
                    <span class="edit-info-value">{editingHub?.metadata?.pos_provider || 'No definido'}</span>
                </div>
            </div>

            <!-- Form Section: Asignación -->
            <div class="edit-section">
                <h4 class="edit-section-title">Asignación</h4>
                <div class="form-group">
                    <label for="hubBranch">Sucursal Asignada <span class="required">*</span></label>
                    <select id="hubBranch" bind:value={editFormData.branch_id} disabled={updatingHub}>
                        <option value="">Sin asignar</option>
                        {#each branches as branch}
                            <option value={branch.id}>{branch.name} ({branch.address})</option>
                        {/each}
                    </select>
                </div>
                <div class="form-group">
                    <label for="hubModel">Modelo de Hardware</label>
                    <input id="hubModel" type="text" bind:value={editFormData.hardware_model}
                           placeholder="Ej. Tablehub-Official-v1" disabled={updatingHub} />
                </div>
            </div>

            <!-- Form Section: Configuración POS -->
            <div class="edit-section">
                <h4 class="edit-section-title">Configuración POS</h4>
                <div class="form-group">
                    <label for="hubMotor">Motor POS</label>
                    <select id="hubMotor" bind:value={editFormData.pos_provider} disabled={updatingHub}>
                        <option value="">Seleccionar motor...</option>
                        <option value="Blackshot">Blackshot</option>
                        <option value="Toast">Toast</option>
                        <option value="TastyIgniter">TastyIgniter</option>
                        <option value="Clover">Clover</option>
                        <option value="Genérico">Genérico</option>
                    </select>
                </div>
            </div>

            {#if editError}
                <div class="modal-alert-error">
                    <AlertTriangle size={16} />
                    <span>{editError}</span>
                </div>
            {/if}

            <div class="modal-actions">
                <button class="btn-cancel" on:click={confirmDiscard} disabled={updatingHub}>
                    Cancelar
                </button>
                <button class="btn-generate" on:click={handleUpdateHub}
                        disabled={updatingHub || !editFormData.branch_id}
                        title={!editFormData.branch_id ? 'Debes asignar una sucursal' : ''}>
                    {#if updatingHub}
                        <span class="spinner"></span>
                        Guardando...
                    {:else}
                        Guardar Cambios
                    {/if}
                </button>
            </div>
        {/if}
    </div>
</div>
{/if}
```

**CSS nuevo (agregar al bloque `<style>` existente):**

```css
/* Edit section grouping */
.edit-section {
    margin-bottom: 1.25rem;
    padding: 1rem;
    background: rgba(255,255,255,0.03);
    border-radius: 10px;
    border: 1px solid rgba(255,255,255,0.05);
}

.edit-section-title {
    font-size: 0.75rem;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    color: var(--accent-hover);
    margin: 0 0 0.75rem;
}

/* Hub info card */
.edit-hub-info {
    background: rgba(0,0,0,0.2);
    border-radius: 10px;
    padding: 0.75rem 1rem;
    margin-bottom: 1.25rem;
    border: 1px solid rgba(255,255,255,0.05);
    display: flex;
    flex-direction: column;
    gap: 0.4rem;
}

.edit-info-row {
    display: flex;
    justify-content: space-between;
    align-items: center;
    font-size: 0.82rem;
}

.edit-info-label {
    color: var(--text-muted);
}

.edit-info-value {
    color: var(--text-color);
    font-weight: 500;
    display: inline-flex;
    align-items: center;
    gap: 0.4rem;
}

/* Required field indicator */
.required {
    color: var(--danger);
    margin-left: 2px;
}
```

### 4.2 `cloud/web/src/lib/api.js`

**No requiere cambios** — la función `updateHub` ya funciona correctamente. Se puede considerar agregar manejo de errores más granular si la API devuelve mensajes específicos, pero no es necesario para este fix.

---

## 5. Mejoras Adicionales (Opcionales)

1. **Extraer lista de POS providers a constante compartida** — tanto el edit modal como el futuro creation modal (`hub_creation_modal_plan.md`) usan los mismos providers. Crear `POS_PROVIDERS` en una lib compartida.
2. **Extraer modal a componente reutilizable** — si hay más modales en el futuro, considerar `Modal.svelte` con slots para header, body, footer, y manejo de teclas/backdrop integrado.
3. **Agregar transiciones también al provision modal** — por consistencia de UX.
4. **Usar `Toast.svelte` para feedback** — en vez de estado `success` inline, mostrar un toast de éxito (`showToast("Hub actualizado correctamente", "success")`).

---

## 6. Resumen de Cambios

| Archivo | Tipo de Cambio | Descripción |
|---|---|---|
| `Devices.svelte` | Script | Agregar imports (`fade`, `scale`, `Settings`), nuevas variables de estado (`editModalState`, `editError`, `isDirty`), funciones de cierre/confirmación |
| `Devices.svelte` | Template | Reemplazar `{#if showEditModal}` por versión con estados, info del hub, secciones agrupadas, errores inline, estado success |
| `Devices.svelte` | Style | Agregar CSS para `.edit-section`, `.edit-hub-info`, `.edit-section-title`, `.edit-info-row`, `.required` |
| `Devices.svelte` | Style | Agregar transiciones fade/scale al modal |
