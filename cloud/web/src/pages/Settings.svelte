<script>
  import { Monitor, Layout, Save, User, Palette, Sliders, Link } from '@lucide/svelte';
  import PrimaryButton from '../lib/components/PrimaryButton.svelte';
  import PosIntegrations from '../lib/components/PosIntegrations.svelte';
  
  export let theme = 'glass'; // 'glass' or 'neumorph'
  export let dockPosition = 'bottom'; // 'bottom', 'top', 'left', 'right'
  export let profile = null;
  
  let activeSubTab = 'profile'; // 'profile' | 'appearance' | 'system' | 'integrations'

  function saveSettings() {
    localStorage.setItem('hub-theme', theme);
    localStorage.setItem('hub-dock-pos', dockPosition);
    // Simple visual feedback
    const btn = document.getElementById('save-btn');
    if (btn) {
      const originalText = btn.innerHTML;
      btn.innerHTML = 'Saved!';
      setTimeout(() => btn.innerHTML = originalText, 2000);
    }
  }

  // System settings simulated/editable state
  let systemSettings = {
    globalMaintenanceMode: false,
    sessionTimeoutMinutes: 30,
    apiLogsRetentionDays: 7
  };

  function saveSystemSettings() {
    alert("Configuración del sistema guardada (Simulado)");
  }
</script>

<div class="view-header">
  <h1>Configuración</h1>
  <p>Gestiona tu perfil, preferencias visuales y parámetros del sistema cloud.</p>
</div>

<div class="settings-container">
  <!-- Sub-tab Navigation -->
  <div class="panel subtabs-panel">
    <div class="subtabs-navigation">
      <button 
        class="subtab-btn {activeSubTab === 'profile' ? 'active' : ''}" 
        on:click={() => activeSubTab = 'profile'}
      >
        <User size={16} /> Mi Perfil
      </button>
      <button 
        class="subtab-btn {activeSubTab === 'appearance' ? 'active' : ''}" 
        on:click={() => activeSubTab = 'appearance'}
      >
        <Palette size={16} /> Apariencia
      </button>
      <button 
        class="subtab-btn {activeSubTab === 'system' ? 'active' : ''}" 
        on:click={() => activeSubTab = 'system'}
      >
        <Sliders size={16} /> Sistema
      </button>
      <button 
        class="subtab-btn {activeSubTab === 'integrations' ? 'active' : ''}" 
        on:click={() => activeSubTab = 'integrations'}
      >
        <Link size={16} /> Integraciones
      </button>
    </div>
  </div>

  {#if activeSubTab === 'profile'}
    {#if profile}
    <section class="panel profile-panel">
      <div class="profile-header">
        <div class="avatar">{profile.name ? profile.name[0].toUpperCase() : 'U'}</div>
        <div class="user-meta">
          <h3>{profile.name || 'Usuario'}</h3>
          <p class="email">{profile.email}</p>
          <div class="badge-row">
            <span class="role-badge">{profile.role.toUpperCase()}</span>
            <span class="org-badge">Org ID: {profile.organization_id.substring(0, 8)}...</span>
          </div>
        </div>
      </div>
    </section>
    {/if}
  {/if}

  {#if activeSubTab === 'appearance'}
    <section class="panel">
      <div class="setting-group">
        <div class="setting-header">
          <Monitor size={20} class="icon" />
          <h3>Tema Visual</h3>
        </div>
        
        <div class="options">
          <label class="option-card {theme === 'glass' ? 'selected' : ''}">
            <input type="radio" bind:group={theme} value="glass" name="theme">
            <div class="preview glass-preview"></div>
            <span>Glassmorphism</span>
          </label>
          
          <label class="option-card {theme === 'neumorph' ? 'selected' : ''}">
            <input type="radio" bind:group={theme} value="neumorph" name="theme">
            <div class="preview neumorph-preview"></div>
            <span>Neumorphism</span>
          </label>
        </div>
      </div>
    </section>

    <section class="panel">
      <div class="setting-group">
        <div class="setting-header">
          <Layout size={20} class="icon" />
          <h3>Posición del Dock</h3>
        </div>
        
        <div class="options dock-options">
          {#each ['bottom', 'top', 'left', 'right'] as pos}
            <label class="option-card {dockPosition === pos ? 'selected' : ''}">
              <input type="radio" bind:group={dockPosition} value={pos} name="dock">
              <span style="text-transform: capitalize;">{pos}</span>
            </label>
          {/each}
        </div>
      </div>
    </section>
    
    <div class="actions">
      <PrimaryButton id="save-btn" on:click={saveSettings}>
        <Save size={18} /> Guardar Preferencias
      </PrimaryButton>
    </div>
  {/if}

  {#if activeSubTab === 'system'}
    <section class="panel">
      <div class="setting-group">
        <div class="setting-header">
          <Sliders size={20} class="icon" />
          <h3>Parámetros del Servidor Cloud</h3>
        </div>

        <div style="display: flex; flex-direction: column; gap: 1.5rem; margin-top: 1rem;">
          <label style="display: flex; align-items: center; justify-content: space-between; cursor: pointer;">
            <div>
              <strong style="display: block;">Modo Mantenimiento Global</strong>
              <span style="font-size: 0.85rem; color: var(--text-muted);">Bloquea el acceso a todas las terminales excepto administradores SaaS.</span>
            </div>
            <input type="checkbox" bind:checked={systemSettings.globalMaintenanceMode} style="width: 20px; height: 20px; accent-color: var(--accent-color);" />
          </label>

          <label style="display: flex; align-items: center; justify-content: space-between;">
            <div>
              <strong style="display: block;">Tiempo de Expiración de Sesión (Minutos)</strong>
              <span style="font-size: 0.85rem; color: var(--text-muted);">Duración antes de requerir re-autenticación en la consola.</span>
            </div>
            <input type="number" bind:value={systemSettings.sessionTimeoutMinutes} min="5" max="1440" style="width: 80px; padding: 0.4rem; background: var(--input-bg); border: var(--input-border); color: var(--text-color); border-radius: 6px; text-align: center;" />
          </label>

          <label style="display: flex; align-items: center; justify-content: space-between;">
            <div>
              <strong style="display: block;">Retención de Logs de API (Días)</strong>
              <span style="font-size: 0.85rem; color: var(--text-muted);">Periodo de almacenamiento de logs de peticiones de IoT.</span>
            </div>
            <input type="number" bind:value={systemSettings.apiLogsRetentionDays} min="1" max="90" style="width: 80px; padding: 0.4rem; background: var(--input-bg); border: var(--input-border); color: var(--text-color); border-radius: 6px; text-align: center;" />
          </label>
        </div>
      </div>
    </section>

    <div class="actions">
      <PrimaryButton on:click={saveSystemSettings}>
        <Save size={18} /> Guardar Configuración
      </PrimaryButton>
    </div>
  {/if}

  {#if activeSubTab === 'integrations'}
    <section class="panel">
      <PosIntegrations {profile} />
    </section>
  {/if}
</div>

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
  }

  .view-header p {
    color: var(--text-muted);
    font-size: 1.1rem;
  }
  
  .settings-container {
    max-width: 800px;
    margin: 0 auto;
    display: flex;
    flex-direction: column;
    gap: 2rem;
  }
  
  .setting-header {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    margin-bottom: 0.5rem;
  }
  
  .setting-header :global(.icon) {
    color: var(--accent-color);
  }

  /* Subtabs */
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
    background: var(--button-active-bg);
    border-color: rgba(59, 130, 246, 0.15);
    box-shadow: var(--button-hover-shadow, inset 2px 2px 5px rgba(0,0,0,0.5));
  }
  
  .options {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 1rem;
  }
  
  .dock-options {
    grid-template-columns: repeat(4, 1fr);
  }
  
  @media (max-width: 600px) {
    .dock-options {
      grid-template-columns: 1fr 1fr;
    }
  }
  
  .option-card {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 1rem;
    padding: 1.5rem 1rem;
    border-radius: 12px;
    background: rgba(0,0,0,0.1);
    border: 2px solid transparent;
    cursor: pointer;
    transition: all 0.2s;
  }
  
  .option-card:hover {
    background: rgba(255,255,255,0.05);
  }
  
  .option-card.selected {
    border-color: var(--accent-color);
    background: var(--button-active-bg);
  }
  
  .option-card input {
    display: none;
  }
  
  .preview {
    width: 60px;
    height: 40px;
    border-radius: 8px;
  }
  
  .glass-preview {
    background: rgba(255, 255, 255, 0.1);
    border: 1px solid rgba(255, 255, 255, 0.2);
    box-shadow: 0 4px 12px rgba(0,0,0,0.1);
  }
  
  .neumorph-preview {
    background: #1a1c23;
    box-shadow: 3px 3px 6px #121419, -3px -3px 6px #22242d;
  }
  
  .actions {
    display: flex;
    justify-content: flex-end;
    margin-top: 1rem;
  }

  .profile-panel {
    margin-bottom: 1.5rem;
  }

  .profile-header {
    display: flex;
    align-items: center;
    gap: 1.5rem;
  }

  .avatar {
    width: 60px;
    height: 60px;
    border-radius: 50%;
    background: var(--accent-color);
    color: white;
    font-size: 1.8rem;
    font-weight: 700;
    display: flex;
    align-items: center;
    justify-content: center;
  }

  .user-meta h3 {
    font-size: 1.3rem;
    font-weight: 600;
    margin-bottom: 0.2rem;
  }

  .user-meta .email {
    color: var(--text-muted);
    font-size: 0.9rem;
    margin-bottom: 0.5rem;
  }

  .badge-row {
    display: flex;
    gap: 0.5rem;
  }

  .role-badge {
    background: rgba(16, 185, 129, 0.1);
    color: var(--success);
    border: 1px solid rgba(16, 185, 129, 0.2);
    font-size: 0.75rem;
    font-weight: 600;
    padding: 0.15rem 0.5rem;
    border-radius: 4px;
  }

  .org-badge {
    background: rgba(255, 255, 255, 0.05);
    color: var(--text-muted);
    border: 1px solid rgba(255, 255, 255, 0.1);
    font-size: 0.75rem;
    padding: 0.15rem 0.5rem;
    border-radius: 4px;
  }
</style>
