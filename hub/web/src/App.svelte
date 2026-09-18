<script>
  import { onMount } from 'svelte';
  import { apiFetch } from './lib/api.js';
  import Dock from './lib/Dock.svelte';
  import DashboardView from './lib/DashboardView.svelte';
  import SettingsView from './lib/SettingsView.svelte';
  import ConfigurationView from './lib/ConfigurationView.svelte';
  import SetupView from './lib/SetupView.svelte';
  import LoginView from './lib/LoginView.svelte';
  import UsersView from './lib/UsersView.svelte';
  import DevicesView from './lib/DevicesView.svelte';
  
  // Auth states: 'loading', 'setup', 'login', 'authenticated'
  let authState = 'loading';

  let activeView = 'dashboard';
  let theme = 'glass';
  let dockPosition = 'bottom';
  
  // Reactively update the body class when theme changes
  $: {
    if (typeof document !== 'undefined') {
      document.body.className = `theme-${theme}`;
    }
  }
  
  onMount(async () => {
    // Load saved preferences
    const savedTheme = localStorage.getItem('hub-theme');
    const savedDockPos = localStorage.getItem('hub-dock-pos');
    
    if (savedTheme) theme = savedTheme;
    if (savedDockPos) dockPosition = savedDockPos;
    
    document.body.className = `theme-${theme}`;

    // Escuchar expiración de token global
    window.addEventListener('auth-expired', () => {
      authState = 'login';
    });

    // Chequear estado de autenticación inicial
    await checkAuth();
  });

  async function checkAuth() {
    try {
      // Usamos el fetch nativo aquí para evitar un bucle de eventos si el token está expirado,
      // ya que solo queremos saber el estado actual
      const res = await fetch('/api/auth/check', { credentials: 'include' });
      if (res.ok) {
        const data = await res.json();
        if (data.status === 'setup_required') {
          authState = 'setup';
        } else {
          authState = 'authenticated';
        }
      } else {
        authState = 'login';
      }
    } catch (e) {
      console.error('Error checking auth', e);
      authState = 'login'; // Fallback a login en caso de error
    }
  }

  function handleSetupSuccess() {
    // Después del setup, se requiere login
    authState = 'login';
  }

  function handleLoginSuccess() {
    authState = 'authenticated';
  }
</script>

{#if authState === 'loading'}
  <div class="loading-screen">
    <div class="spinner"></div>
    <p>Cargando Tablehub...</p>
  </div>
{:else if authState === 'setup'}
  <SetupView on:success={handleSetupSuccess} />
{:else if authState === 'login'}
  <LoginView on:success={handleLoginSuccess} />
{:else if authState === 'authenticated'}
  <!-- The Main Application Container -->
  <div class="app-container has-dock-{dockPosition}">
    
    <main class="content-area">
      {#if activeView === 'dashboard'}
        <DashboardView />
      {:else if activeView === 'devices'}
        <DevicesView />
      {:else if activeView === 'settings'}
        <ConfigurationView />
      {:else if activeView === 'theme'}
        <SettingsView bind:theme bind:dockPosition />
      {:else if activeView === 'users'}
        <UsersView />
      {:else}
        <div class="placeholder">
          <h2>{activeView.toUpperCase()}</h2>
          <p>This view is under construction.</p>
        </div>
      {/if}
    </main>
    
    <Dock bind:activeView {dockPosition} />
    
  </div>
{/if}

<style>
  .loading-screen {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    height: 100vh;
    color: var(--text-color);
  }

  .spinner {
    border: 4px solid rgba(255, 255, 255, 0.1);
    border-left-color: var(--accent-color);
    border-radius: 50%;
    width: 40px;
    height: 40px;
    animation: spin 1s linear infinite;
    margin-bottom: 1rem;
  }

  @keyframes spin {
    0% { transform: rotate(0deg); }
    100% { transform: rotate(360deg); }
  }

  .app-container {
    display: flex;
    min-height: 100vh;
    width: 100%;
    /* We use padding to ensure content isn't hidden behind the dock */
    transition: padding 0.3s ease;
  }
  
  .has-dock-bottom { padding-bottom: 90px; }
  .has-dock-top { padding-top: 90px; }
  .has-dock-left { padding-left: 90px; }
  .has-dock-right { padding-right: 90px; }
  
  .content-area {
    flex: 1;
    max-width: 1200px;
    margin: 0 auto;
    padding: 2rem;
    width: 100%;
  }
  
  .placeholder {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    height: 50vh;
    text-align: center;
    color: var(--text-muted);
  }
</style>
