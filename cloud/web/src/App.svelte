<script>
    import { onMount } from "svelte";
    import { initAuth, isAuthenticated } from "./lib/auth.js";
    import { apiFetch } from "./lib/api.js";
    import Login from "./pages/Login.svelte";
    import Dashboard from "./pages/Dashboard.svelte";
    import Callback from "./pages/Callback.svelte";
    import Settings from "./pages/Settings.svelte";
    import Onboarding from "./pages/Onboarding.svelte";
    import Devices from "./pages/Devices.svelte";
    import UsersPage from "./pages/Users.svelte";
    import Organizations from "./pages/Organizations.svelte";
    import Branches from "./pages/Branches.svelte";
    import Dock from "./lib/Dock.svelte";
    import ClientDashboard from "./pages/ClientDashboard.svelte";
    import Toast from "./lib/Toast.svelte";

    let authInitialized = false;
    let loadingProfile = true;
    let currentPath = window.location.pathname;
    
    // User profile state
    let profile = null;

    // Staff role detection
    const STAFF_ROLES = ['platform_admin', 'support', 'billing'];
    $: isStaff = profile && STAFF_ROLES.includes(profile.role);
    
    // Hub theme logic
    let activeView = 'dashboard';
    let theme = 'glass';
    let dockPosition = 'bottom';
    
    // Reactively update the body class when theme changes
    $: {
      if (typeof document !== 'undefined') {
        document.body.className = `theme-${theme}`;
      }
    }

    async function checkUserProfile() {
        loadingProfile = true;
        try {
            const res = await apiFetch("/v1/me");
            if (res.ok) {
                const data = await res.json();
                if (data.registered) {
                    profile = data;
                } else {
                    profile = null;
                }
            } else {
                profile = null;
            }
        } catch (e) {
            console.error("Failed to load user profile:", e);
            profile = null;
        } finally {
            loadingProfile = false;
        }
    }

    function handleOnboardingComplete(newProfile) {
        profile = newProfile;
        activeView = 'dashboard';
    }

    onMount(async () => {
        // Load saved preferences
        const savedTheme = localStorage.getItem('hub-theme');
        const savedDockPos = localStorage.getItem('hub-dock-pos');
        
        if (savedTheme) theme = savedTheme;
        if (savedDockPos) dockPosition = savedDockPos;
        
        document.body.className = `theme-${theme}`;

        await initAuth();
        authInitialized = true;
        
        if (window.location.pathname !== "/auth/callback") {
            // Only check profile if we are not in OIDC login callback page
            const loggedIn = await new Promise(resolve => {
                const unsubscribe = isAuthenticated.subscribe(val => {
                    resolve(val);
                });
                unsubscribe();
            });
            if (loggedIn) {
                await checkUserProfile();
            } else {
                loadingProfile = false;
            }
        } else {
            loadingProfile = false;
        }
    });
</script>

{#if !authInitialized || loadingProfile}
    <div class="loading-screen">
        <div class="spinner"></div>
        <p>Cargando aplicación...</p>
    </div>
{:else}
    {#if currentPath === "/auth/callback"}
        <Callback />
    {:else}
        {#if $isAuthenticated}
            {#if profile && profile.registered}
                <div class="app-container has-dock-{dockPosition}">
                    <main class="content-area">
                       {#if activeView === 'dashboard'}
                        {#if isStaff}
                          <Dashboard />
                        {:else}
                          <ClientDashboard {profile} on:navigate={e => activeView = e.detail.view} />
                        {/if}
                      {:else if activeView === 'organizations'}
                        {#if isStaff}
                          <Organizations />
                        {:else}
                          <ClientDashboard {profile} on:navigate={e => activeView = e.detail.view} />
                        {/if}
                      {:else if activeView === 'devices'}
                        <Devices orgID={profile.organization_id} />
                      {:else if activeView === 'branches'}
                        <Branches orgID={profile.organization_id} />
                      {:else if activeView === 'users'}
                        <UsersPage orgID={profile.organization_id} />
                      {:else if activeView === 'theme' || activeView === 'settings'}
                        <Settings bind:theme bind:dockPosition {profile} />
                      {:else}
                        <div class="placeholder">
                          <h2>{activeView.toUpperCase()}</h2>
                          <p>Esta vista se encuentra en construcción.</p>
                        </div>
                      {/if}
                    </main>
                    <Dock bind:activeView {dockPosition} {profile} />
                </div>
            {:else}
                <Onboarding onComplete={handleOnboardingComplete} />
            {/if}
        {:else}
            <Login />
        {/if}
    {/if}
{/if}

<Toast />

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
