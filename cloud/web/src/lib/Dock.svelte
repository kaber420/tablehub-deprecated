<script>
  import { LayoutDashboard, Users, Settings, Palette, Tablet, Cloud, LogOut, ChevronUp, ChevronDown, Building } from '@lucide/svelte';
  import { logout } from './auth.js';
  
  export let position = 'bottom'; // 'bottom', 'top', 'left', 'right'
  export let activeView = 'dashboard';
  export let profile = null;

  let menuOpen = false;
  
  const STAFF_ROLES = ['platform_admin', 'support', 'billing'];
  $: isStaff = profile && STAFF_ROLES.includes(profile.role);
  
  $: navItems = [
    { id: 'dashboard', label: 'Dashboard', icon: LayoutDashboard },
    ...(isStaff ? [
      { id: 'organizations', label: 'Clientes y Orgs', icon: Building }
    ] : [
      { id: 'branches', label: 'Sucursales', icon: Building }
    ]),
    { id: 'devices', label: isStaff ? 'Devices / IoT' : 'Mis Hubs', icon: Tablet },
    { id: 'users', label: isStaff ? 'Staff' : 'Mi Equipo', icon: Users },
    { id: 'settings', label: 'Settings', icon: Settings },
    { id: 'theme', label: 'Appearance', icon: Palette }
  ];
  
  function handleSelect(id) {
    activeView = id;
    menuOpen = false;
  }

  async function handleLogoutClick() {
    menuOpen = false;
    try {
      await logout();
    } catch (e) {
      console.error('Logout error:', e);
    }
  }
</script>

<svelte:window 
  on:click={(e) => {
    if (menuOpen && !e.target.closest('.profile-menu-container')) {
      menuOpen = false;
    }
  }}
  on:keydown={(e) => {
    if (e.key === 'Escape') {
      menuOpen = false;
    }
  }}
/>

<div class="dock-container pos-{position}">
  <nav class="dock">
    {#each navItems as item}
      <button 
        class="dock-item {activeView === item.id ? 'active' : ''}" 
        on:click={() => handleSelect(item.id)}
        title={item.label}
      >
        <svelte:component this={item.icon} size={24} strokeWidth={1.5} />
        <span class="tooltip">{item.label}</span>
      </button>
    {/each}

    {#if profile}
      <div class="dock-separator"></div>

      <div class="profile-menu-container">
        <button 
          class="dock-profile-btn {menuOpen ? 'active' : ''}" 
          on:click={() => menuOpen = !menuOpen}
          aria-haspopup="menu"
          aria-expanded={menuOpen}
          title="Usuario"
        >
          <div class="profile-avatar">
            <svelte:component this={Cloud} size={18} strokeWidth={1.5} />
          </div>
          <div class="profile-info">
            <span class="username">{profile.name || profile.email || 'Usuario'}</span>
            <span class="role">{profile.role || 'Miembro'}</span>
          </div>
          <div class="chevron-wrapper">
            <svelte:component this={menuOpen ? ChevronDown : ChevronUp} size={14} strokeWidth={1.5} />
          </div>
        </button>

        {#if menuOpen}
          <div class="profile-dropdown pos-{position}">
            <div class="dropdown-header">
              <span class="dropdown-name">{profile.name || 'SaaS User'}</span>
              <span class="dropdown-role">{profile.email || ''}</span>
            </div>
            <div class="dropdown-divider"></div>
            <button class="dropdown-item" on:click={() => handleSelect('settings')}>
              <svelte:component this={Settings} size={16} strokeWidth={1.5} />
              <span>Configuración</span>
            </button>
            <button class="dropdown-item logout" on:click={handleLogoutClick}>
              <svelte:component this={LogOut} size={16} strokeWidth={1.5} />
              <span>Cerrar Sesión</span>
            </button>
          </div>
        {/if}
      </div>
    {/if}
  </nav>
</div>

<style>
  .dock-container {
    position: fixed;
    z-index: 1000;
    display: flex;
    justify-content: center;
    align-items: center;
    transition: all 0.4s cubic-bezier(0.2, 0.8, 0.2, 1);
  }
  
  .pos-bottom { bottom: 20px; left: 0; right: 0; }
  .pos-top { top: 20px; left: 0; right: 0; }
  .pos-left { left: 20px; top: 0; bottom: 0; flex-direction: column; }
  .pos-right { right: 20px; top: 0; bottom: 0; flex-direction: column; }
  
  .dock {
    display: flex;
    align-items: center;
    background: var(--dock-bg);
    backdrop-filter: var(--panel-blur);
    -webkit-backdrop-filter: var(--panel-blur);
    border: var(--dock-border);
    box-shadow: var(--dock-shadow);
    padding: 0.5rem;
    gap: 0.5rem;
    transition: all 0.3s ease;
  }
  
  /* Flex direction depending on position */
  .pos-bottom .dock, .pos-top .dock {
    flex-direction: row;
    border-radius: 20px;
  }
  
  .pos-left .dock, .pos-right .dock {
    flex-direction: column;
    border-radius: 20px;
  }
  
  .dock-item {
    position: relative;
    display: flex;
    align-items: center;
    justify-content: center;
    width: 48px;
    height: 48px;
    border-radius: 14px;
    border: none;
    background: transparent;
    color: var(--text-muted);
    cursor: pointer;
    transition: all 0.2s cubic-bezier(0.25, 1, 0.5, 1);
  }
  
  /* Mac-like hover effect */
  .dock-item:hover {
    background: var(--button-hover-bg);
    box-shadow: var(--button-hover-shadow, none);
    color: var(--text-color);
    transform: scale(1.15) translateY(-2px);
    z-index: 10;
  }
  
  .pos-left .dock-item:hover { transform: scale(1.15) translateX(2px); }
  .pos-right .dock-item:hover { transform: scale(1.15) translateX(-2px); }
  
  .dock-item.active {
    background: var(--button-active-bg);
    color: var(--accent-color);
    box-shadow: var(--button-hover-shadow, none);
  }
  
  .dock-item.active::after {
    content: '';
    position: absolute;
    width: 6px;
    height: 6px;
    border-radius: 50%;
    background: var(--accent-color);
  }
  
  .pos-bottom .dock-item.active::after { bottom: -10px; }
  .pos-top .dock-item.active::after { top: -10px; }
  .pos-left .dock-item.active::after { left: -10px; }
  .pos-right .dock-item.active::after { right: -10px; }

  /* Separator */
  .dock-separator {
    width: 1px;
    height: 24px;
    background: var(--border-color, rgba(255, 255, 255, 0.15));
    margin: 0 4px;
  }

  .pos-left .dock-separator, .pos-right .dock-separator {
    width: 24px;
    height: 1px;
    margin: 4px 0;
  }

  /* Profile Button and Menu */
  .profile-menu-container {
    position: relative;
    display: flex;
    align-items: center;
  }

  .dock-profile-btn {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 6px 12px;
    height: 48px;
    border-radius: 14px;
    border: none;
    background: transparent;
    color: var(--text-muted);
    cursor: pointer;
    transition: all 0.2s cubic-bezier(0.25, 1, 0.5, 1);
  }

  .pos-left .dock-profile-btn, .pos-right .dock-profile-btn {
    flex-direction: column;
    padding: 8px 6px;
    height: auto;
    width: 48px;
    gap: 4px;
  }

  .dock-profile-btn:hover {
    background: var(--button-hover-bg);
    box-shadow: var(--button-hover-shadow, none);
    color: var(--text-color);
    transform: scale(1.08) translateY(-2px);
  }

  .pos-left .dock-profile-btn:hover { transform: scale(1.08) translateX(2px); }
  .pos-right .dock-profile-btn:hover { transform: scale(1.08) translateX(-2px); }

  .dock-profile-btn.active {
    background: var(--button-active-bg);
    box-shadow: var(--button-hover-shadow, none);
    color: var(--accent-color);
  }

  .profile-avatar {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 32px;
    height: 32px;
    border-radius: 10px;
    background: var(--button-active-bg, rgba(255, 255, 255, 0.1));
    color: var(--accent-color);
  }

  .profile-info {
    display: flex;
    flex-direction: column;
    align-items: flex-start;
    font-size: 0.75rem;
    line-height: 1.2;
    max-width: 100px;
  }

  .pos-left .profile-info, .pos-right .profile-info {
    display: none; /* Hide text in vertical dock positions to save space */
  }

  .profile-info .username {
    font-weight: 600;
    color: var(--text-color);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    width: 100%;
  }

  .profile-info .role {
    color: var(--text-muted);
    font-size: 0.65rem;
    text-transform: capitalize;
  }

  .chevron-wrapper {
    display: flex;
    align-items: center;
    color: var(--text-muted);
    opacity: 0.7;
  }

  .pos-left .chevron-wrapper, .pos-right .chevron-wrapper {
    display: none;
  }

  /* Popover Dropdown */
  .profile-dropdown {
    position: absolute;
    background: var(--panel-bg, rgba(20, 20, 20, 0.85));
    backdrop-filter: var(--panel-blur, blur(20px));
    -webkit-backdrop-filter: var(--panel-blur, blur(20px));
    border: var(--panel-border, 1px solid rgba(255, 255, 255, 0.1));
    box-shadow: var(--shadow-lg, 0 10px 30px rgba(0, 0, 0, 0.3));
    border-radius: 14px;
    padding: 6px;
    width: 190px;
    display: flex;
    flex-direction: column;
    gap: 4px;
    z-index: 1001;
    animation: fadeIn 0.2s cubic-bezier(0.25, 1, 0.5, 1);
  }

  /* Directional placements */
  .profile-dropdown.pos-bottom {
    bottom: calc(100% + 12px);
    right: 0;
  }

  .profile-dropdown.pos-top {
    top: calc(100% + 12px);
    right: 0;
  }

  .profile-dropdown.pos-left {
    left: calc(100% + 12px);
    bottom: 0;
  }

  .profile-dropdown.pos-right {
    right: calc(100% + 12px);
    bottom: 0;
  }

  @keyframes fadeIn {
    from { opacity: 0; transform: scale(0.95) translateY(5px); }
    to { opacity: 1; transform: scale(1) translateY(0); }
  }

  .dropdown-header {
    padding: 8px 12px;
    display: flex;
    flex-direction: column;
    overflow: hidden;
  }

  .dropdown-name {
    font-size: 0.85rem;
    font-weight: 600;
    color: var(--text-color);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .dropdown-role {
    font-size: 0.7rem;
    color: var(--text-muted);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .dropdown-divider {
    height: 1px;
    background: rgba(255, 255, 255, 0.1);
    margin: 4px 0;
  }

  .dropdown-item {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 8px 12px;
    border-radius: 8px;
    border: none;
    background: transparent;
    color: var(--text-color);
    font-size: 0.8rem;
    cursor: pointer;
    text-align: left;
    transition: background 0.15s ease;
  }

  .dropdown-item:hover {
    background: var(--button-hover-bg, rgba(255, 255, 255, 0.08));
  }

  .dropdown-item.logout {
    color: #ff4a4a;
  }

  .dropdown-item.logout:hover {
    background: rgba(255, 74, 74, 0.1);
  }
  
  /* Tooltip */
  .tooltip {
    position: absolute;
    background: rgba(0,0,0,0.8);
    color: white;
    padding: 0.3rem 0.6rem;
    border-radius: 6px;
    font-size: 0.75rem;
    pointer-events: none;
    opacity: 0;
    transition: opacity 0.2s;
    white-space: nowrap;
  }
  
  .dock-item:hover .tooltip {
    opacity: 1;
  }
  
  .pos-bottom .tooltip { top: -35px; }
  .pos-top .tooltip { bottom: -35px; }
  .pos-left .tooltip { left: 60px; }
  .pos-right .tooltip { right: 60px; }
</style>
