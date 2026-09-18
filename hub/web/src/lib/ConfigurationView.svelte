<script>
  import { onMount } from 'svelte';
  import { 
    Cloud, 
    Copy, 
    Check, 
    Info, 
    Save, 
    Upload, 
    Unlink, 
    Server, 
    ShieldAlert, 
    CheckCircle2, 
    AlertCircle, 
    Globe,
    Palette,
    Image
  } from '@lucide/svelte';
  import { 
    getCloudSettings, 
    saveCloudSettings, 
    uploadProvisionFile, 
    disconnectCloud,
    getScreensaverSettings,
    saveScreensaverSettings,
    uploadScreensaverImage
  } from './api.js';
  import PrimaryButton from './components/PrimaryButton.svelte';
  import ToggleSwitch from './components/ToggleSwitch.svelte';
  
  let activeSubTab = 'cloud'; // 'cloud' | 'link' | 'system' | 'general' | 'branding'

  let settings = {
    cloud_url: '',
    restaurant_id: '',
    language: 'es',
    public_key: '',
    cloud_enabled: false,
    is_connected: false
  };

  let screensaver = {
    venue_name: '',
    venue_subtitle: '',
    screensaver_mode: 'text',
    screensaver_image_path: ''
  };
  let savingScreensaver = false;
  let uploadingImage = false;
  
  let loading = true;
  let saving = false;
  let uploading = false;
  let disconnecting = false;
  let showDisconnectConfirm = false;
  let pollInterval;
  
  let error = '';
  let success = '';
  let copied = false;
  let showAdvanced = false;

  $: isConfigured = Boolean(settings.cloud_url && settings.restaurant_id);
  $: isRealConnected = isConfigured && settings.cloud_enabled && settings.is_connected;
  
  onMount(() => {
    loadSettings();
    loadScreensaver();
    pollInterval = setInterval(loadSettings, 5000);
    return () => clearInterval(pollInterval);
  });

  async function loadScreensaver() {
    try {
      screensaver = await getScreensaverSettings();
    } catch (err) {
      console.error('Error al cargar ajustes de screensaver:', err);
    }
  }

  async function handleSaveScreensaver() {
    error = '';
    savingScreensaver = true;
    try {
      await saveScreensaverSettings(screensaver);
      success = 'Personalización del protector de pantalla guardada correctamente.';
      setTimeout(() => success = '', 3000);
    } catch (err) {
      error = 'Error al guardar personalización: ' + err.message;
    } finally {
      savingScreensaver = false;
    }
  }

  async function handleImageSelect(event) {
    const file = event.target.files[0];
    if (!file) return;

    error = '';
    uploadingImage = true;
    try {
      const res = await uploadScreensaverImage(file);
      screensaver.screensaver_image_path = res.image_url;
      screensaver.screensaver_mode = 'image';
      success = 'Imagen del logotipo cargada exitosamente.';
      setTimeout(() => success = '', 3000);
    } catch (err) {
      error = 'Error al subir la imagen: ' + err.message;
    } finally {
      uploadingImage = false;
    }
  }

  async function loadSettings() {
    try {
      settings = await getCloudSettings();
    } catch (err) {
      error = 'No se pudo cargar la configuración de la nube.';
      console.error(err);
    } finally {
      loading = false;
    }
  }

  
  async function saveSettings() {
    error = '';
    saving = true;
    
    if (settings.cloud_url && !settings.cloud_url.startsWith('ws://') && !settings.cloud_url.startsWith('wss://')) {
      error = 'La URL debe comenzar con ws:// o wss://';
      saving = false;
      return;
    }
    
    try {
      await saveCloudSettings({
        cloud_url: settings.cloud_url,
        restaurant_id: settings.restaurant_id,
        language: settings.language,
        cloud_enabled: settings.cloud_enabled
      });
      
      success = 'Configuración guardada correctamente.';
      setTimeout(() => success = '', 3000);
    } catch (err) {
      error = err.message;
    } finally {
      saving = false;
    }
  }

  async function handleDisconnect() {
    disconnecting = true;
    error = '';
    try {
      await disconnectCloud();
      showDisconnectConfirm = false;
      success = 'Hub desvinculado y configuración eliminada exitosamente.';
      await loadSettings();
      activeSubTab = 'link';
    } catch (err) {
      error = 'Error al desconectar el Hub: ' + err.message;
    } finally {
      disconnecting = false;
    }
  }
  
  function copyKey() {
    if (navigator.clipboard && settings.public_key) {
      navigator.clipboard.writeText(settings.public_key);
      copied = true;
      setTimeout(() => copied = false, 2000);
    }
  }
  
  async function handleFileUpload(file) {
    if (!file) return;
    
    error = '';
    success = '';
    uploading = true;
    
    try {
      const result = await uploadProvisionFile(file);
      success = `Aprovisionamiento exitoso. Hub ID: ${result.hub_id}`;
      await loadSettings();
      activeSubTab = 'cloud';
    } catch (err) {
      error = 'Error al procesar el archivo de aprovisionamiento: ' + err.message;
    } finally {
      uploading = false;
    }
  }
  
  function handleFileInput(event) {
    const file = event.target.files[0];
    if (file) handleFileUpload(file);
  }
  
  function handleDrop(event) {
    event.preventDefault();
    dragOver = false;
    const file = event.dataTransfer.files[0];
    if (file) handleFileUpload(file);
  }
  
  function handleDragOver(event) {
    event.preventDefault();
    dragOver = true;
  }
  
  function handleDragLeave() {
    dragOver = false;
  }
</script>

<div class="view-header">
  <h1>Configuración</h1>
  <p>Gestiona tu perfil, parámetros de red local y sincronización con el sistema cloud.</p>
</div>

{#if loading}
  <div class="loading-state">
    <div class="upload-spinner"></div>
    <p>Cargando configuración...</p>
  </div>
{:else}
  <div class="settings-container">
    <!-- Sub-tab Navigation matching Cloud UI Settings style -->
    <div class="panel subtabs-panel">
      <div class="subtabs-navigation">
        <button 
          class="subtab-btn {activeSubTab === 'cloud' ? 'active' : ''}" 
          on:click={() => activeSubTab = 'cloud'}
        >
          <Cloud size={16} /> Estado Nube
          {#if isRealConnected}
            <span class="status-dot online"></span>
          {:else if isConfigured && settings.cloud_enabled}
            <span class="status-dot paused"></span>
          {:else}
            <span class="status-dot offline"></span>
          {/if}
        </button>

        <button 
          class="subtab-btn {activeSubTab === 'link' ? 'active' : ''}" 
          on:click={() => activeSubTab = 'link'}
        >
          <Unlink size={16} /> Vincular Cuenta
        </button>

        <button 
          class="subtab-btn {activeSubTab === 'system' ? 'active' : ''}" 
          on:click={() => activeSubTab = 'system'}
        >
          <Server size={16} /> Sistema y Red
        </button>

        <button 
          class="subtab-btn {activeSubTab === 'general' ? 'active' : ''}" 
          on:click={() => activeSubTab = 'general'}
        >
          <Globe size={16} /> General
        </button>

        <button 
          class="subtab-btn {activeSubTab === 'branding' ? 'active' : ''}" 
          on:click={() => activeSubTab = 'branding'}
        >
          <Palette size={16} /> Personalización
        </button>
      </div>
    </div>

    <!-- Alert Messages -->
    {#if error}
      <div class="alert-box error">
        <AlertCircle size={20} />
        <span>{error}</span>
      </div>
    {/if}

    {#if success}
      <div class="alert-box success">
        <CheckCircle2 size={20} />
        <span>{success}</span>
      </div>
    {/if}

    <!-- TAB 1: Estado Nube -->
    {#if activeSubTab === 'cloud'}
      <section class="panel">
        <div class="setting-header-row">
          <div class="title-with-icon">
            <Cloud size={20} class="icon" />
            <h3>Conexión Cloud (SaaS)</h3>
          </div>
          <div class="badge-status {isRealConnected ? 'online' : (isConfigured && settings.cloud_enabled ? 'paused' : 'offline')}">
            {#if isRealConnected}
              <span class="pulse-dot"></span> Conectado (Túnel Activo)
            {:else if isConfigured && settings.cloud_enabled}
              <span>Reconectando / Sin respuesta SaaS</span>
            {:else if isConfigured}
              <span>Pausado</span>
            {:else}
              <span>Desvinculado</span>
            {/if}
          </div>
        </div>


        {#if isConfigured}
          <div class="card-grid">
            <div class="info-card">
              <span class="card-label">ID del Restaurante</span>
              <span class="card-value font-mono">{settings.restaurant_id}</span>
            </div>
            <div class="info-card">
              <span class="card-label">URL del Túnel Cloud</span>
              <span class="card-value font-mono">{settings.cloud_url}</span>
            </div>
          </div>

          <div class="input-group toggle-group mt-4">
            <div>
              <label for="cloudEnabled">Sincronización activa con la nube</label>
              <p class="help-text">Activa o pausa el túnel de eventos NATS</p>
            </div>
            <ToggleSwitch id="cloudEnabled" bind:checked={settings.cloud_enabled} on:change={saveSettings} />
          </div>

          <div class="input-group mt-4">
            <label>Llave Pública (Ed25519) - Hub ID</label>
            <div class="key-display-group">
              <input type="text" readonly value={settings.public_key} class="key-input" />
              <button class="copy-btn {copied ? 'copied' : ''}" on:click={copyKey} title="Copiar al portapapeles">
                {#if copied}
                  <Check size={18} />
                {:else}
                  <Copy size={18} />
                {/if}
              </button>
            </div>
          </div>

          <div class="danger-zone mt-6">
            <div class="danger-info">
              <h4>Desvincular Hub de la Nube</h4>
              <p>Elimina las credenciales guardadas para vincular a otra cuenta o restaurante.</p>
            </div>
            <button class="btn-danger" on:click={() => showDisconnectConfirm = true}>
              <Unlink size={16} /> Desconectar Cuenta
            </button>
          </div>
        {:else}
          <div class="empty-state">
            <ShieldAlert size={44} class="muted-icon" />
            <h3>Este Hub no está vinculado a ninguna cuenta</h3>
            <p>Selecciona la pestaña <strong>"Vincular Cuenta"</strong> para cargar tu archivo <code>.thub</code> de aprovisionamiento.</p>
            <button class="btn-primary mt-4" on:click={() => activeSubTab = 'link'}>
              Ir a Vincular Cuenta
            </button>
          </div>
        {/if}
      </section>
    {/if}

    <!-- TAB 2: Vincular Cuenta -->
    {#if activeSubTab === 'link'}
      <section class="panel">
        <div class="setting-header">
          <Upload size={20} class="icon" />
          <h3>Aprovisionar desde archivo .thub</h3>
        </div>

        <div class="upload-section mt-4">
          <div 
            class="drop-zone {dragOver ? 'drag-over' : ''} {uploading ? 'uploading' : ''}"
            on:drop={handleDrop}
            on:dragover={handleDragOver}
            on:dragleave={handleDragLeave}
            role="button"
            tabindex="0"
          >
            {#if uploading}
              <div class="upload-spinner"></div>
              <p>Conectando y activando túnel con la nube...</p>
            {:else}
              <Upload size={36} class="upload-icon" />
              <h3>Arrastra y suelta tu archivo <code>.thub</code> aquí</h3>
              <p>O presiona el botón para seleccionar el archivo de aprovisionamiento</p>
              <label for="fileInput" class="file-label-btn">
                Seleccionar Archivo .thub
              </label>
              <input 
                type="file" 
                id="fileInput" 
                accept=".thub" 
                on:change={handleFileInput}
                class="file-input"
              />
            {/if}
          </div>
        </div>

        <div class="advanced-toggle mt-4">
          <button class="advanced-btn" on:click={() => showAdvanced = !showAdvanced}>
            {showAdvanced ? 'Ocultar configuración manual' : 'Configuración manual (Avanzado)'}
          </button>
        </div>
        
        {#if showAdvanced}
          <div class="advanced-settings mt-4">
            <div class="input-group">
              <label for="cloudUrl">URL del WebSocket (ws:// o wss://)</label>
              <input 
                type="text" 
                id="cloudUrl" 
                bind:value={settings.cloud_url} 
                placeholder="ej. wss://api.tablehub.com/ws" 
              />
            </div>
            
            <div class="input-group">
              <label for="restaurantId">ID del Restaurante</label>
              <input 
                type="text" 
                id="restaurantId" 
                bind:value={settings.restaurant_id} 
                placeholder="ej. mi-restaurante-001" 
              />
            </div>
            
            <div class="actions">
              <PrimaryButton disabled={saving} on:click={saveSettings}>
                <Save size={18} />
                {saving ? 'Guardando...' : 'Guardar Manualmente'}
              </PrimaryButton>
            </div>
          </div>
        {/if}
      </section>
    {/if}

    <!-- TAB 3: Sistema y Red -->
    {#if activeSubTab === 'system'}
      <section class="panel">
        <div class="setting-header">
          <Server size={20} class="icon" />
          <h3>Parámetros de Red y Servidor Local</h3>
        </div>

        <div class="card-grid mt-4">
          <div class="info-card">
            <span class="card-label">Puerto Web HTTP Local</span>
            <span class="card-value font-mono">8080</span>
          </div>
          <div class="info-card">
            <span class="card-label">Puerto NATS Embebido</span>
            <span class="card-value font-mono">4222</span>
          </div>
          <div class="info-card">
            <span class="card-label">Broker MQTT Local</span>
            <span class="card-value font-mono">1883</span>
          </div>
          <div class="info-card">
            <span class="card-label">Almacenamiento Local</span>
            <span class="card-value font-mono">SQLite (tablehub.db)</span>
          </div>
        </div>
      </section>
    {/if}

    <!-- TAB 4: General -->
    {#if activeSubTab === 'general'}
      <section class="panel">
        <div class="setting-header">
          <Info size={20} class="icon" />
          <h3>General</h3>
        </div>
        
        <div class="input-group mt-4">
          <label for="language">Idioma</label>
          <select id="language" bind:value={settings.language} on:change={saveSettings}>
            <option value="es">Español</option>
            <option value="en">English</option>
          </select>
        </div>
      </section>
    {/if}

    <!-- TAB 5: Personalización / Screensaver -->
    {#if activeSubTab === 'branding'}
      <section class="panel">
        <div class="setting-header">
          <Palette size={20} class="icon" />
          <h3>Personalización del Protector de Pantalla</h3>
        </div>
        <p class="help-text mt-1">Configura la identidad visual que se mostrará en los terminales cuando entren en modo de reposo.</p>

        <div class="input-group mt-4">
          <label for="venueName">Nombre del Establecimiento</label>
          <input 
            type="text" 
            id="venueName" 
            bind:value={screensaver.venue_name} 
            placeholder="ej. La Trattoria del Mar" 
          />
          <p class="help-text">Reemplaza el nombre de demostración en las pantallas.</p>
        </div>

        <div class="input-group mt-4">
          <label for="venueSubtitle">Subtítulo / Lema de Marca (Opcional)</label>
          <input 
            type="text" 
            id="venueSubtitle" 
            bind:value={screensaver.venue_subtitle} 
            placeholder="ej. Cocina Internacional & Bar" 
          />
        </div>

        <div class="input-group mt-4">
          <label>Modo del Protector de Pantalla</label>
          <div class="mode-selector">
            <label class="mode-option {screensaver.screensaver_mode === 'text' ? 'selected' : ''}">
              <input type="radio" bind:group={screensaver.screensaver_mode} value="text" />
              <span>Texto Elegante (Tipografía Dorado/Gris)</span>
            </label>
            <label class="mode-option {screensaver.screensaver_mode === 'image' ? 'selected' : ''}">
              <input type="radio" bind:group={screensaver.screensaver_mode} value="image" />
              <span>Imagen / Logotipo Personalizado</span>
            </label>
          </div>
        </div>

        <div class="input-group mt-4">
          <label>Imagen de Logotipo</label>
          <div class="image-upload-box">
            {#if screensaver.screensaver_image_path}
              <div class="preview-container">
                <img src={screensaver.screensaver_image_path} alt="Vista previa del logo" class="logo-preview" />
                <span class="image-path">{screensaver.screensaver_image_path}</span>
              </div>
            {/if}

            <div class="upload-controls mt-2">
              <label for="screensaverImageInput" class="file-label-btn">
                <Upload size={16} /> {uploadingImage ? 'Subiendo...' : 'Seleccionar Imagen...'}
              </label>
              <input 
                type="file" 
                id="screensaverImageInput" 
                accept="image/png, image/jpeg, image/bmp" 
                on:change={handleImageSelect}
                class="file-input"
                disabled={uploadingImage}
              />
            </div>
          </div>
        </div>

        <div class="actions mt-6">
          <PrimaryButton disabled={savingScreensaver} on:click={handleSaveScreensaver}>
            <Save size={18} />
            {savingScreensaver ? 'Guardando...' : 'Guardar Personalización'}
          </PrimaryButton>
        </div>
      </section>
    {/if}
  </div>
{/if}

<!-- Modal de Confirmación de Desconexión -->
{#if showDisconnectConfirm}
  <div class="modal-backdrop">
    <div class="modal-card">
      <div class="modal-header">
        <ShieldAlert size={28} class="danger-icon" />
        <h3>¿Desconectar Hub de la Nube?</h3>
      </div>
      <p class="modal-body">
        Esta acción eliminará la vinculación del restaurante <strong>{settings.restaurant_id}</strong>. 
        El Hub dejará de comunicarse con la nube hasta que cargues un nuevo archivo <code>.thub</code>.
      </p>
      <div class="modal-actions">
        <button class="btn-cancel" disabled={disconnecting} on:click={() => showDisconnectConfirm = false}>
          Cancelar
        </button>
        <button class="btn-danger" disabled={disconnecting} on:click={handleDisconnect}>
          {disconnecting ? 'Desconectando...' : 'Sí, Desconectar'}
        </button>
      </div>
    </div>
  </div>
{/if}

<style>
  .view-header {
    text-align: center;
    margin-bottom: 2rem;
  }
  
  .view-header h1 {
    font-size: 2.5rem;
    letter-spacing: -0.02em;
    background: linear-gradient(135deg, #fff 0%, #a5b4fc 100%);
    -webkit-background-clip: text;
    -webkit-text-fill-color: transparent;
    margin-bottom: 0.5rem;
  }

  .view-header p {
    color: var(--text-muted);
    font-size: 1.1rem;
    margin: 0;
  }
  
  .settings-container {
    max-width: 800px;
    margin: 0 auto;
    display: flex;
    flex-direction: column;
    gap: 2rem;
  }

  /* Subtabs Panel & Navigation matching Cloud UI */
  .subtabs-panel {
    padding: 0.75rem;
  }

  .subtabs-navigation {
    display: flex;
    gap: 0.5rem;
    flex-wrap: wrap;
  }

  .subtab-btn {
    background: transparent;
    border: 1px solid transparent;
    color: var(--text-muted);
    padding: 0.6rem 1.2rem;
    border-radius: 10px;
    font-weight: 500;
    font-size: 0.9rem;
    cursor: pointer;
    display: flex;
    align-items: center;
    gap: 0.5rem;
    transition: all 0.2s ease-in-out;
  }

  .subtab-btn:hover {
    color: var(--text-color);
    background: rgba(255, 255, 255, 0.03);
  }

  .subtab-btn.active {
    color: var(--accent-color);
    background: var(--button-active-bg, rgba(59, 130, 246, 0.15));
    border-color: rgba(59, 130, 246, 0.15);
    box-shadow: var(--button-hover-shadow, inset 2px 2px 5px rgba(0,0,0,0.5));
  }

  .status-dot {
    width: 6px;
    height: 6px;
    border-radius: 50%;
    margin-left: 0.25rem;
  }
  .status-dot.online { background: #10b981; }
  .status-dot.paused { background: #f59e0b; }
  .status-dot.offline { background: #ef4444; }

  /* Setting Header */
  .setting-header {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    margin-bottom: 0.5rem;
  }

  .setting-header :global(.icon), .title-with-icon :global(.icon) {
    color: var(--accent-color);
  }

  .setting-header-row {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 1rem;
  }

  .title-with-icon {
    display: flex;
    align-items: center;
    gap: 0.75rem;
  }

  .title-with-icon h3, .setting-header h3 {
    margin: 0;
    font-size: 1.2rem;
  }

  /* Status Badge */
  .badge-status {
    padding: 0.25rem 0.75rem;
    border-radius: 20px;
    font-size: 0.8rem;
    font-weight: 600;
    display: flex;
    align-items: center;
    gap: 0.4rem;
  }

  .badge-status.online {
    background: rgba(16, 185, 129, 0.15);
    color: #10b981;
    border: 1px solid rgba(16, 185, 129, 0.3);
  }

  .badge-status.paused {
    background: rgba(245, 158, 11, 0.15);
    color: #f59e0b;
    border: 1px solid rgba(245, 158, 11, 0.3);
  }

  .badge-status.offline {
    background: rgba(239, 68, 68, 0.15);
    color: #ef4444;
    border: 1px solid rgba(239, 68, 68, 0.3);
  }

  .pulse-dot {
    width: 6px;
    height: 6px;
    background: #10b981;
    border-radius: 50%;
    box-shadow: 0 0 0 0 rgba(16, 185, 129, 0.7);
    animation: pulse 1.6s infinite;
  }

  @keyframes pulse {
    0% { transform: scale(0.95); box-shadow: 0 0 0 0 rgba(16, 185, 129, 0.7); }
    70% { transform: scale(1); box-shadow: 0 0 0 6px rgba(16, 185, 129, 0); }
    100% { transform: scale(0.95); box-shadow: 0 0 0 0 rgba(16, 185, 129, 0); }
  }

  .card-grid {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 1rem;
  }

  .info-card {
    background: rgba(0,0,0,0.15);
    border: 1px solid rgba(255, 255, 255, 0.05);
    border-radius: 10px;
    padding: 1rem;
    display: flex;
    flex-direction: column;
    gap: 0.3rem;
  }

  .card-label {
    font-size: 0.75rem;
    color: var(--text-muted);
    text-transform: uppercase;
    letter-spacing: 0.04em;
  }

  .card-value {
    font-size: 0.95rem;
    color: var(--text-color);
  }

  .font-mono {
    font-family: monospace;
  }

  /* Form Controls */
  .input-group {
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
  }

  .toggle-group {
    flex-direction: row;
    align-items: center;
    justify-content: space-between;
    background: rgba(0,0,0,0.1);
    padding: 1rem;
    border-radius: 10px;
  }

  .help-text {
    margin: 0;
    font-size: 0.8rem;
    color: var(--text-muted);
  }

  label {
    font-size: 0.85rem;
    font-weight: 600;
  }

  input, select {
    padding: 0.9rem;
    border-radius: 8px;
    border: 1px solid rgba(255,255,255,0.1);
    background: rgba(0,0,0,0.2);
    color: var(--text-color);
    font-family: inherit;
    font-size: 1rem;
    transition: border-color 0.2s;
  }

  input:focus, select:focus {
    outline: none;
    border-color: var(--accent-color);
  }

  .key-display-group {
    display: flex;
    gap: 0.5rem;
  }

  .key-input {
    flex: 1;
    font-family: monospace;
    font-size: 0.9rem;
    color: var(--accent-color);
    background: rgba(0,0,0,0.3);
  }

  .copy-btn {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 48px;
    background: rgba(255,255,255,0.1);
    border: 1px solid rgba(255,255,255,0.1);
    border-radius: 8px;
    color: var(--text-color);
    cursor: pointer;
  }

  .copy-btn.copied {
    background: #10b981;
    border-color: #10b981;
    color: white;
  }

  /* Danger Zone */
  .danger-zone {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 1.25rem;
    background: rgba(239, 68, 68, 0.08);
    border: 1px solid rgba(239, 68, 68, 0.2);
    border-radius: 10px;
  }

  .danger-info h4 {
    margin: 0 0 0.2rem 0;
    color: #ef4444;
    font-size: 0.95rem;
  }

  .danger-info p {
    margin: 0;
    font-size: 0.8rem;
    color: var(--text-muted);
  }

  .btn-danger {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    padding: 0.65rem 1.2rem;
    background: #ef4444;
    color: white;
    border: none;
    border-radius: 8px;
    font-size: 0.875rem;
    font-weight: 600;
    cursor: pointer;
  }

  .btn-danger:hover {
    background: #dc2626;
  }

  /* Dropzone */
  .drop-zone {
    border: 2px dashed rgba(255,255,255,0.2);
    border-radius: 12px;
    padding: 2.5rem;
    text-align: center;
    cursor: pointer;
    background: rgba(0,0,0,0.1);
    transition: all 0.2s ease;
  }

  .drop-zone:hover, .drop-zone.drag-over {
    border-color: var(--accent-color);
    background: rgba(167, 139, 250, 0.05);
  }

  .drop-zone h3 {
    margin: 0.75rem 0 0.3rem 0;
    font-size: 1.1rem;
  }

  .drop-zone p {
    margin: 0 0 1.25rem 0;
    font-size: 0.875rem;
    color: var(--text-muted);
  }

  .file-label-btn {
    display: inline-block;
    padding: 0.75rem 1.5rem;
    background: var(--accent-color);
    color: white;
    font-weight: 600;
    font-size: 0.9rem;
    border-radius: 8px;
    cursor: pointer;
  }

  .file-input { display: none; }

  /* Modal */
  .modal-backdrop {
    position: fixed;
    top: 0; left: 0; right: 0; bottom: 0;
    background: rgba(0, 0, 0, 0.75);
    backdrop-filter: blur(4px);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 999;
  }

  .modal-card {
    background: var(--panel-bg, #1a1c23);
    border: 1px solid rgba(255, 255, 255, 0.1);
    border-radius: 16px;
    width: 90%;
    max-width: 440px;
    padding: 1.75rem;
  }

  .modal-header {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    margin-bottom: 1rem;
  }

  .modal-header h3 {
    margin: 0;
    font-size: 1.2rem;
  }

  :global(.danger-icon) {
    color: #ef4444;
  }

  .modal-body {
    color: var(--text-muted);
    font-size: 0.9rem;
    line-height: 1.5;
    margin-bottom: 1.5rem;
  }

  .modal-actions {
    display: flex;
    justify-content: flex-end;
    gap: 0.75rem;
  }

  .btn-cancel {
    padding: 0.65rem 1.2rem;
    background: rgba(255, 255, 255, 0.08);
    border: 1px solid rgba(255, 255, 255, 0.1);
    color: var(--text-color);
    border-radius: 8px;
    font-weight: 500;
    cursor: pointer;
  }

  /* Alerts */
  .alert-box {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    padding: 1rem 1.25rem;
    border-radius: 10px;
    font-size: 0.9rem;
  }

  .alert-box.error {
    background: rgba(239, 68, 68, 0.1);
    border: 1px solid rgba(239, 68, 68, 0.2);
    color: #ef4444;
  }

  .alert-box.success {
    background: rgba(16, 185, 129, 0.1);
    border: 1px solid rgba(16, 185, 129, 0.2);
    color: #10b981;
  }

  .actions {
    display: flex;
    justify-content: flex-end;
    margin-top: 1rem;
  }

  .advanced-toggle {
    margin-top: 1rem;
  }

  .advanced-btn {
    background: none;
    border: none;
    color: var(--text-muted);
    font-size: 0.85rem;
    cursor: pointer;
    text-decoration: underline;
    padding: 0;
  }

  .advanced-settings {
    padding-top: 1rem;
    border-top: 1px solid rgba(255,255,255,0.1);
  }

  .mt-4 { margin-top: 1rem; }
  .mt-6 { margin-top: 1.5rem; }

  .empty-state {
    text-align: center;
    padding: 2.5rem 1rem;
  }

  .empty-state h3 {
    margin: 1rem 0 0.5rem 0;
  }

  .empty-state p {
    color: var(--text-muted);
    margin-bottom: 1rem;
  }

  .btn-primary {
    padding: 0.75rem 1.5rem;
    background: var(--accent-color);
    color: white;
    border: none;
    border-radius: 8px;
    font-weight: 600;
    cursor: pointer;
  }

  .upload-spinner {
    width: 32px;
    height: 32px;
    border: 3px solid rgba(255,255,255,0.1);
    border-top-color: var(--accent-color);
    border-radius: 50%;
    animation: spin 1s linear infinite;
    margin: 0 auto 0.75rem;
  }

  @keyframes spin {
    to { transform: rotate(360deg); }
  }

  .mode-selector {
    display: flex;
    gap: 1rem;
    flex-wrap: wrap;
  }

  .mode-option {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    padding: 0.75rem 1rem;
    background: rgba(0,0,0,0.15);
    border: 1px solid rgba(255,255,255,0.08);
    border-radius: 8px;
    cursor: pointer;
    font-size: 0.9rem;
    transition: all 0.2s ease;
  }

  .mode-option:hover {
    background: rgba(255,255,255,0.05);
  }

  .mode-option.selected {
    border-color: var(--accent-color);
    background: var(--button-active-bg, rgba(59, 130, 246, 0.15));
  }

  .image-upload-box {
    background: rgba(0,0,0,0.15);
    border: 1px solid rgba(255,255,255,0.08);
    border-radius: 10px;
    padding: 1rem;
  }

  .preview-container {
    display: flex;
    align-items: center;
    gap: 1rem;
    margin-bottom: 0.75rem;
  }

  .logo-preview {
    max-height: 80px;
    max-width: 160px;
    object-fit: contain;
    border-radius: 6px;
    background: #000;
    padding: 4px;
    border: 1px solid rgba(255,255,255,0.1);
  }

  .image-path {
    font-family: monospace;
    font-size: 0.85rem;
    color: var(--text-muted);
  }

  .upload-controls {
    display: flex;
    align-items: center;
    gap: 0.5rem;
  }
</style>
