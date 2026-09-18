<script>
  import { createEventDispatcher, onMount } from 'svelte';
  import {
    LayoutDashboard,
    Tablet,
    Building,
    Smartphone,
    ArrowRight,
    TrendingUp,
    Wifi,
    Users,
    CreditCard
  } from '@lucide/svelte';
  import { fetchOrgStats } from '../lib/api.js';

  export let profile = null;

  const dispatch = createEventDispatcher();

  let kpis = [
    { label: 'Hubs Activos', value: 0, icon: Wifi, trend: 'Activos', iconClass: 'icon-blue' },
    { label: 'Dispositivos IoT', value: 0, icon: Tablet, trend: 'Próximamente', iconClass: 'icon-green' },
    { label: 'Sucursales', value: 0, icon: Building, trend: 'Registradas', iconClass: 'icon-orange' },
    { label: 'Sesiones de App', value: 0, icon: Smartphone, trend: 'Sin datos', iconClass: 'icon-red' }
  ];

  onMount(async () => {
    if (profile?.organization_id) {
      try {
        const stats = await fetchOrgStats(profile.organization_id);
        kpis[0].value = stats.hubs_count;
        kpis[2].value = stats.branches_count;
        kpis = kpis;
      } catch (err) {
        console.error("Failed to load stats", err);
      }
    }
  });

  const quickActions = [
    { label: 'Mis Hubs', desc: 'Gestiona tus dispositivos y gateways', icon: Tablet, view: 'devices', color: 'from-blue-500 to-blue-600' },
    { label: 'Sucursales', desc: 'Administra tus ubicaciones', icon: Building, view: 'branches', color: 'from-orange-500 to-orange-600' },
    { label: 'Mi Equipo', desc: 'Administra usuarios y roles', icon: Users, view: 'users', color: 'from-emerald-500 to-emerald-600' },
    { label: 'Facturación', desc: 'Revisa planes y pagos', icon: CreditCard, view: 'settings', color: 'from-violet-500 to-violet-600' }
  ];

  function navigate(view) {
    dispatch('navigate', { view });
  }
</script>

<div class="client-dashboard">
  <header class="header">
    <div class="header-content">
      <div class="header-text">
        <h1>Mi Organización: {profile?.organization_name || profile?.name || 'Cargando...'}</h1>
        <p class="subtitle">Panel de control de tu restaurante — todo bajo control.</p>
      </div>
      <div class="header-badge">
        <LayoutDashboard size={18} />
        <span>Dashboard</span>
      </div>
    </div>
  </header>

  <div class="metrics-section">
    <div class="section-header">
      <h2>Métricas Clave</h2>
      <TrendingUp size={20} class="section-icon" />
    </div>
    <div class="kpi-grid">
      {#each kpis as kpi}
        <div class="kpi-card panel">
          <div class="kpi-icon-wrapper {kpi.iconClass}">
            <svelte:component this={kpi.icon} size={22} strokeWidth={1.5} />
          </div>
          <div class="kpi-body">
            <span class="kpi-label">{kpi.label}</span>
            <span class="kpi-value">{kpi.value}</span>
            <span class="kpi-trend">{kpi.trend}</span>
          </div>
        </div>
      {/each}
    </div>
  </div>

  <div class="actions-section">
    <div class="section-header">
      <h2>Acciones Rápidas</h2>
    </div>
    <div class="actions-grid">
      {#each quickActions as action}
        <button class="action-card panel" on:click={() => navigate(action.view)}>
          <div class="action-icon-wrapper">
            <svelte:component this={action.icon} size={28} strokeWidth={1.5} />
          </div>
          <div class="action-body">
            <span class="action-label">{action.label}</span>
            <span class="action-desc">{action.desc}</span>
          </div>
          <div class="action-arrow">
            <ArrowRight size={20} strokeWidth={1.5} />
          </div>
        </button>
      {/each}
    </div>
  </div>
</div>

<style>
  .client-dashboard {
    display: flex;
    flex-direction: column;
    gap: 2.5rem;
    animation: dashboardIn 0.5s cubic-bezier(0.25, 1, 0.5, 1);
  }

  @keyframes dashboardIn {
    from { opacity: 0; transform: translateY(16px); }
    to { opacity: 1; transform: translateY(0); }
  }

  .header-content {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    gap: 1rem;
    flex-wrap: wrap;
  }

  .header-text h1 {
    font-size: 1.85rem;
    margin-bottom: 0.35rem;
    color: var(--accent-color, #60a5fa);
  }

  .subtitle {
    color: var(--text-muted);
    font-size: 1rem;
  }

  .header-badge {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    padding: 0.5rem 1rem;
    background: rgba(59, 130, 246, 0.12);
    border: 1px solid rgba(59, 130, 246, 0.25);
    border-radius: 9999px;
    color: var(--text-color);
    font-size: 0.85rem;
    font-weight: 500;
    white-space: nowrap;
  }

  .section-header {
    display: flex;
    align-items: center;
    gap: 0.6rem;
    margin-bottom: 0.25rem;
  }

  .section-header h2 {
    font-size: 1.25rem;
    color: var(--text-color);
    margin: 0;
  }

  .section-icon {
    color: var(--accent-color);
    opacity: 0.7;
  }

  .kpi-grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(230px, 1fr));
    gap: 1.25rem;
  }

  .kpi-card {
    display: flex;
    align-items: center;
    gap: 1.25rem;
    padding: 1.25rem 1.5rem;
    cursor: default;
    position: relative;
    overflow: hidden;
  }

  .kpi-card::before {
    content: '';
    position: absolute;
    top: 0;
    left: 0;
    right: 0;
    height: 3px;
    background: linear-gradient(90deg, transparent, var(--accent-color), transparent);
    opacity: 0.4;
  }

  .kpi-card:hover::before {
    opacity: 0.8;
  }

  .kpi-body {
    display: flex;
    flex-direction: column;
    gap: 0.15rem;
  }

  .kpi-label {
    font-size: 0.8rem;
    color: var(--text-muted);
    text-transform: uppercase;
    letter-spacing: 0.06em;
    font-weight: 500;
  }

  .kpi-value {
    font-size: 2rem;
    font-weight: 700;
    color: #f8fafc;
    line-height: 1.1;
  }

  .kpi-trend {
    font-size: 0.78rem;
    color: var(--success);
    font-weight: 500;
  }

  .actions-grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(260px, 1fr));
    gap: 1.25rem;
  }

  .action-card {
    display: flex;
    align-items: center;
    gap: 1rem;
    padding: 1.5rem;
    cursor: pointer;
    border: none;
    color: var(--text-color);
    font-family: inherit;
    font-size: 1rem;
    text-align: left;
    width: 100%;
    position: relative;
    overflow: hidden;
    transition: all 0.3s cubic-bezier(0.25, 1, 0.5, 1);
  }

  .action-card::after {
    content: '';
    position: absolute;
    inset: 0;
    background: var(--button-hover-bg, rgba(255, 255, 255, 0.02));
    opacity: 0;
    transition: opacity 0.3s ease;
  }

  .action-card:hover {
    transform: translateY(-4px);
    box-shadow: var(--panel-shadow-hover);
  }

  .action-card:hover::after {
    opacity: 1;
  }

  .action-card:active {
    transform: translateY(-1px) scale(0.99);
  }

  .action-icon-wrapper {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 52px;
    height: 52px;
    border-radius: 14px;
    background: rgba(59, 130, 246, 0.12);
    color: #60a5fa;
    flex-shrink: 0;
    transition: all 0.3s ease;
  }

  .action-card:hover .action-icon-wrapper {
    background: rgba(59, 130, 246, 0.2);
    transform: scale(1.05);
  }

  .action-body {
    display: flex;
    flex-direction: column;
    gap: 0.2rem;
    flex: 1;
    min-width: 0;
  }

  .action-label {
    font-weight: 600;
    font-size: 1.05rem;
    color: #f8fafc;
  }

  .action-desc {
    font-size: 0.8rem;
    color: var(--text-muted);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .action-arrow {
    color: var(--text-muted);
    opacity: 0.4;
    transition: all 0.3s ease;
    flex-shrink: 0;
  }

  .action-card:hover .action-arrow {
    opacity: 1;
    color: var(--accent-color);
    transform: translateX(4px);
  }

  @media (max-width: 640px) {
    .header-content {
      flex-direction: column;
    }

    .kpi-grid {
      grid-template-columns: 1fr 1fr;
    }

    .actions-grid {
      grid-template-columns: 1fr;
    }

    .kpi-value {
      font-size: 1.5rem;
    }
  }

  @media (max-width: 420px) {
    .kpi-grid {
      grid-template-columns: 1fr;
    }
  }
</style>
