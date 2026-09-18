<script>
    import { onMount } from "svelte";
    import { user, logout } from "../lib/auth.js";
    import { apiFetch } from "../lib/api.js";
    import PrimaryButton from "../lib/components/PrimaryButton.svelte";
    import { 
        Users, 
        Tablet, 
        Server, 
        CreditCard, 
        Search, 
        RefreshCw, 
        CheckCircle,
        XCircle,
        Building,
        Layers,
        ChevronDown,
        ChevronUp,
        MapPin
    } from '@lucide/svelte';

    // Active sub-tab in the dashboard
    let activeTab = 'organizations'; // 'organizations' | 'plans'
    let searchQuery = '';

    // Expanded state maps
    let expandedOrgs = {};

    // Mock data reflecting real multi-tenant SaaS schema with quota limits and activated modules
    let organizations = [
        { 
            id: "3b2eb519-724e-4f32-bb91-4cf15d2925b6", 
            name: "La Pizza Nostra", 
            slug: "pizza-nostra", 
            billing_plan: "pro", 
            stripe_customer_id: "cus_Q8k2m91a0b", 
            created_at: "2026-02-14T10:00:00Z", 
            owner: "Luciano Rossi", 
            email: "luciano@pizza.it",
            hubs_limit: 10,
            devices_limit: 100,
            waiter_app_limit: 400,
            active_modules: ["Notificaciones Cloud", "Integración POS Toast", "App Meseros Cloud"]
        },
        { 
            id: "e10a20cb-c31b-4b14-8f0a-ef31b818a4a5", 
            name: "Tasty Burgers", 
            slug: "tasty-burgers", 
            billing_plan: "starter", 
            stripe_customer_id: "cus_Q8k5m72x8a", 
            created_at: "2026-03-22T14:30:00Z", 
            owner: "John Doe", 
            email: "john@tastyburgers.com",
            hubs_limit: 3,
            devices_limit: 15,
            waiter_app_limit: 25,
            active_modules: ["Notificaciones Cloud"]
        },
        { 
            id: "98a12bc2-10f8-4e12-8812-7bb8c1a1fa99", 
            name: "Sushi Palace", 
            slug: "sushi-palace", 
            billing_plan: "pro", 
            stripe_customer_id: "cus_Q9a1l38z5c", 
            created_at: "2026-05-01T09:15:00Z", 
            owner: "Yuki Tanaka", 
            email: "yuki@sushipalace.co.jp",
            hubs_limit: 10,
            devices_limit: 100,
            waiter_app_limit: 400,
            active_modules: ["Notificaciones Cloud", "Integración POS TastyIgniter"]
        },
        { 
            id: "4f738a19-b00a-4712-9c12-32a76fa8bde2", 
            name: "Café París", 
            slug: "cafe-paris", 
            billing_plan: "free", 
            stripe_customer_id: "", 
            created_at: "2026-06-18T16:45:00Z", 
            owner: "Marie Dupont", 
            email: "marie@cafeparis.fr",
            hubs_limit: 1,
            devices_limit: 5,
            waiter_app_limit: 5,
            active_modules: []
        }
    ];

    let branches = [
        { id: "b1", org_id: "3b2eb519-724e-4f32-bb91-4cf15d2925b6", name: "Sucursal Centro", address: "Av. Principal 123" },
        { id: "b2", org_id: "3b2eb519-724e-4f32-bb91-4cf15d2925b6", name: "Sucursal Norte", address: "Calle Bosque 456" },
        { id: "b3", org_id: "e10a20cb-c31b-4b14-8f0a-ef31b818a4a5", name: "Downtown Mall", address: "Av. de los Malls 900" },
        { id: "b4", org_id: "98a12bc2-10f8-4e12-8812-7bb8c1a1fa99", name: "Ginza Main Store", address: "4-chōme Ginza" }
    ];

    let hubs = [
        { id: "84d5df68-96bb-49e0-8208-f40445d4ea81", organization_id: "3b2eb519-724e-4f32-bb91-4cf15d2925b6", branch_id: "b1", settings: { pos_provider: "TastyIgniter" }, cloud_screens: 8, cloud_waiters: 25 },
        { id: "a52ff61d-3b7c-482a-9e6e-c802871f3a21", organization_id: "3b2eb519-724e-4f32-bb91-4cf15d2925b6", branch_id: "b2", settings: { pos_provider: "Toast" }, cloud_screens: 4, cloud_waiters: 20 },
        { id: "287cf8bc-a10c-4b92-ba2e-8c381f72c448", organization_id: "e10a20cb-c31b-4b14-8f0a-ef31b818a4a5", branch_id: "b3", settings: { pos_provider: "TastyIgniter" }, cloud_screens: 5, cloud_waiters: 8 },
        { id: "9efc11a0-d128-4fb5-9c88-e219ba381c81", organization_id: "98a12bc2-10f8-4e12-8812-7bb8c1a1fa99", branch_id: "b4", settings: { pos_provider: "CustomAPI" }, cloud_screens: 2, cloud_waiters: 15 }
    ];

    let loadingStats = false;

    // Helper functions to get stats per organization
    function getHubsCountForOrg(orgId) {
        return hubs.filter(h => h.organization_id === orgId).length;
    }

    function getDevicesCountForOrg(orgId) {
        return hubs.filter(h => h.organization_id === orgId).reduce((acc, curr) => acc + curr.cloud_screens, 0);
    }

    function getWaiterConnectionsForOrg(orgId) {
        return hubs.filter(h => h.organization_id === orgId).reduce((acc, curr) => acc + curr.cloud_waiters, 0);
    }

    // Helper functions for global sums
    function getGlobalHubsCount() {
        return hubs.length;
    }

    function getGlobalDevicesCount() {
        return hubs.reduce((acc, curr) => acc + curr.cloud_screens, 0);
    }

    function getGlobalWaiterAppConnections() {
        return hubs.reduce((acc, curr) => acc + curr.cloud_waiters, 0);
    }

    function getBranchesForOrg(orgId) {
        return branches.filter(b => b.org_id === orgId);
    }

    function getHubsForBranch(branchId) {
        return hubs.filter(h => h.branch_id === branchId);
    }

    function toggleOrgExpand(orgId) {
        expandedOrgs[orgId] = !expandedOrgs[orgId];
    }

    async function loadStats() {
        loadingStats = true;
        setTimeout(() => {
            loadingStats = false;
        }, 300);
    }

    onMount(() => {
        loadStats();
    });

    // Filtering logic
    $: filteredOrganizations = organizations.filter(o => {
        const matchesSearch = o.name.toLowerCase().includes(searchQuery.toLowerCase()) || 
                              o.owner.toLowerCase().includes(searchQuery.toLowerCase()) ||
                              o.billing_plan.toLowerCase().includes(searchQuery.toLowerCase());
        return matchesSearch;
    });
</script>

<div class="dashboard-container">
    <header class="top-nav panel">
        <div class="logo-area">
            <Layers size={28} class="accent-icon" />
            <h2 class="title">Tablehub <span class="badge SaaS">SaaS Cloud Console</span></h2>
        </div>
        <div class="header-actions">
            <PrimaryButton on:click={logout}>
                Cerrar Sesión
            </PrimaryButton>
        </div>
    </header>

    <!-- Top KPI metrics focused on SaaS Inventory & Usage -->
    <section class="kpi-grid">
        <div class="panel kpi-card">
            <div class="kpi-icon-wrapper icon-blue">
                <Building size={24} />
            </div>
            <div class="kpi-info">
                <span class="kpi-value">{organizations.length}</span>
                <span class="kpi-label">Organizaciones</span>
            </div>
        </div>

        <div class="panel kpi-card">
            <div class="kpi-icon-wrapper icon-green">
                <Server size={24} />
            </div>
            <div class="kpi-info">
                <span class="kpi-value">{getGlobalHubsCount()}</span>
                <span class="kpi-label">Hubs Cloud Activos</span>
            </div>
        </div>

        <div class="panel kpi-card">
            <div class="kpi-icon-wrapper icon-orange">
                <Tablet size={24} />
            </div>
            <div class="kpi-info">
                <span class="kpi-value">{getGlobalDevicesCount()}</span>
                <span class="kpi-label">Pantallas Cloud Registradas</span>
            </div>
        </div>

        <div class="panel kpi-card">
            <div class="kpi-icon-wrapper icon-red">
                <Users size={24} />
            </div>
            <div class="kpi-info">
                <span class="kpi-value">{getGlobalWaiterAppConnections()}</span>
                <span class="kpi-label">Sesiones de Meseros (Cloud)</span>
            </div>
        </div>
    </section>

    <!-- Active view container -->
    <main class="dashboard-content">
        <div class="panel info-panel">
            <h3>Consola de Administración SaaS</h3>
            <p>Bienvenido al centro de control global de Tablehub Cloud. Desde aquí puedes monitorear el estado general de la plataforma y navegar a las secciones correspondientes en el dock inferior.</p>
            <div class="features-grid">
                <div class="panel-inset feature-item">
                    <h4>🏢 Clientes y Organizaciones</h4>
                    <p>Monitorea consumos, cuotas asignadas, limites de dispositivos y estado de suscripciones activas.</p>
                </div>
                <div class="panel-inset feature-item">
                    <h4>📟 Dispositivos / IoT</h4>
                    <p>Controla el aprovisionamiento de hubs locales y pantallas de mesa registradas.</p>
                </div>
                <div class="panel-inset feature-item">
                    <h4>👥 Personal de Soporte</h4>
                    <p>Administra los roles del staff técnico con accesos a la infraestructura cloud.</p>
                </div>
            </div>
        </div>
    </main>
</div>

<style>
    .dashboard-container {
        display: flex;
        flex-direction: column;
        gap: 1.5rem;
        width: 100%;
        margin-top: 1rem;
    }

    .top-nav {
        display: flex;
        justify-content: space-between;
        align-items: center;
        padding: 1.25rem 2rem;
        background: var(--panel-bg);
        border: var(--panel-border);
    }

    .logo-area {
        display: flex;
        align-items: center;
        gap: 0.75rem;
    }

    .logo-area :global(.accent-icon) {
        color: var(--accent-color);
        filter: drop-shadow(0 0 6px rgba(59, 130, 246, 0.4));
    }

    .title {
        margin: 0;
        font-size: 1.4rem;
        font-weight: 700;
        color: var(--text-color);
        display: flex;
        align-items: center;
        gap: 0.5rem;
    }

    .badge {
        padding: 0.2rem 0.5rem;
        border-radius: 6px;
        font-size: 0.75rem;
        font-weight: 600;
        letter-spacing: 0.05em;
    }

    .badge.SaaS {
        background: rgba(59, 130, 246, 0.15);
        color: var(--accent-hover);
        border: 1px solid rgba(59, 130, 246, 0.3);
    }

    .badge.slug {
        background: rgba(255, 255, 255, 0.05);
        color: var(--text-muted);
        font-family: monospace;
    }

    .badge.plan-pro {
        background: rgba(139, 92, 246, 0.15);
        color: #c084fc;
        border: 1px solid rgba(139, 92, 246, 0.3);
    }

    .badge.plan-starter {
        background: rgba(59, 130, 246, 0.15);
        color: #60a5fa;
        border: 1px solid rgba(59, 130, 246, 0.3);
    }

    .badge.plan-free {
        background: rgba(148, 163, 184, 0.15);
        color: #94a3b8;
        border: 1px solid rgba(148, 163, 184, 0.3);
    }

    .badge.pos-tag {
        background: rgba(16, 185, 129, 0.1);
        color: var(--success);
        border: 1px solid rgba(16, 185, 129, 0.2);
    }

    /* Controls Panel Neomorphic Design */
    .controls-panel {
        display: flex;
        flex-direction: column;
        gap: 1rem;
        padding: 1.25rem;
    }

    .tabs-navigation {
        display: flex;
        flex-wrap: wrap;
        gap: 0.75rem;
        border-bottom: 1px solid rgba(255, 255, 255, 0.05);
        padding-bottom: 0.75rem;
    }

    .tab-btn {
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

    .tab-btn:hover {
        color: var(--text-color);
        background: rgba(255, 255, 255, 0.03);
    }

    .tab-btn.active {
        color: var(--accent-color);
        background: var(--button-active-bg);
        border-color: rgba(59, 130, 246, 0.15);
        box-shadow: var(--button-hover-shadow, inset 2px 2px 5px rgba(0,0,0,0.5));
    }

    .search-bar-wrapper {
        display: flex;
        gap: 0.75rem;
        align-items: center;
    }

    .search-input-container {
        display: flex;
        align-items: center;
        gap: 0.5rem;
        flex: 1;
        background: var(--input-bg);
        border: var(--input-border);
        box-shadow: var(--input-shadow);
        border-radius: 8px;
        padding: 0.25rem 0.75rem;
    }

    .search-input-container input {
        border: none;
        background: transparent;
        box-shadow: none;
        width: 100%;
        color: var(--text-color);
        padding: 0.5rem 0;
    }

    .search-icon {
        color: var(--text-muted);
    }

    .btn-icon {
        background: var(--input-bg);
        border: var(--input-border);
        box-shadow: var(--input-shadow);
        color: var(--text-color);
        border-radius: 8px;
        width: 42px;
        height: 42px;
        display: flex;
        align-items: center;
        justify-content: center;
        cursor: pointer;
        transition: transform 0.2s;
    }

    .btn-icon:hover {
        transform: scale(1.05);
    }

    .spin {
        animation: loading-spin 1s linear infinite;
    }

    @keyframes loading-spin {
        0% { transform: rotate(0deg); }
        100% { transform: rotate(360deg); }
    }

    /* Neomorphic Tables */
    .table-panel {
        padding: 1.5rem;
        overflow: hidden;
    }

    .table-header {
        margin-bottom: 1.25rem;
    }

    .table-responsive {
        overflow-x: auto;
        width: 100%;
    }

    .neomorphic-table {
        width: 100%;
        border-collapse: collapse;
        text-align: left;
    }

    .neomorphic-table th {
        color: var(--text-muted);
        font-weight: 600;
        font-size: 0.85rem;
        text-transform: uppercase;
        letter-spacing: 0.05em;
        padding: 1rem;
        border-bottom: 1px solid rgba(255, 255, 255, 0.05);
    }

    .neomorphic-table td {
        padding: 1rem;
        border-bottom: 1px solid rgba(255, 255, 255, 0.02);
        font-size: 0.95rem;
        color: var(--text-color);
    }

    .neomorphic-table tr:hover td {
        background: rgba(255, 255, 255, 0.01);
    }

    .btn-expand {
        background: transparent;
        border: none;
        color: var(--text-muted);
        cursor: pointer;
        padding: 0.25rem;
        display: flex;
        align-items: center;
        justify-content: center;
        border-radius: 4px;
    }

    .btn-expand:hover {
        color: var(--text-color);
        background: rgba(255, 255, 255, 0.05);
    }

    .expanded-row td {
        padding: 1.5rem 2rem;
        background: rgba(0, 0, 0, 0.15) !important;
        border-bottom: 1px solid rgba(255, 255, 255, 0.05);
    }

    .nested-container h4 {
        margin-bottom: 1rem;
        font-size: 1.1rem;
        color: var(--accent-hover);
    }

    .nested-container h5 {
        margin-bottom: 0.5rem;
        font-size: 0.9rem;
        color: var(--text-muted);
    }

    .empty-text {
        font-size: 0.85rem;
        color: var(--text-muted);
        font-style: italic;
    }

    .branches-grid {
        display: grid;
        grid-template-columns: repeat(auto-fit, minmax(320px, 1fr));
        gap: 1.25rem;
        margin-top: 0.5rem;
    }

    .branch-item {
        padding: 1.25rem;
        display: flex;
        flex-direction: column;
        gap: 1rem;
    }

    .branch-meta {
        display: flex;
        flex-direction: column;
        gap: 0.25rem;
        border-bottom: 1px solid rgba(255, 255, 255, 0.05);
        padding-bottom: 0.75rem;
    }

    .branch-name {
        display: flex;
        align-items: center;
        gap: 0.5rem;
        font-size: 1.05rem;
    }

    .branch-address {
        font-size: 0.85rem;
        color: var(--text-muted);
        padding-left: 1.5rem;
    }

    .nested-hubs-list {
        list-style: none;
        display: flex;
        flex-direction: column;
        gap: 0.5rem;
    }

    .nested-hub-row {
        display: flex;
        justify-content: space-between;
        align-items: center;
        font-size: 0.85rem;
        background: rgba(255, 255, 255, 0.02);
        padding: 0.4rem 0.75rem;
        border-radius: 6px;
    }

    .pos-badge {
        font-size: 0.75rem;
        background: rgba(255, 255, 255, 0.05);
        padding: 0.1rem 0.4rem;
        border-radius: 4px;
        color: var(--text-muted);
    }

    .font-bold {
        font-weight: 600;
    }

    .mono {
        font-family: monospace;
    }

    .contact-cell {
        display: flex;
        flex-direction: column;
    }

    .owner-name {
        font-weight: 500;
    }

    .owner-email {
        font-size: 0.8rem;
        color: var(--text-muted);
    }

    /* Pricing Card Grid */
    .pricing-plans-grid {
        display: grid;
        grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
        gap: 1.5rem;
    }

    .plan-card {
        padding: 2rem;
        display: flex;
        flex-direction: column;
        gap: 1.25rem;
        position: relative;
    }

    .plan-card.featured {
        border-color: var(--accent-color);
        box-shadow: 0 0 15px rgba(59, 130, 246, 0.15);
    }

    .plan-card.pro-card {
        border-color: #8b5cf6;
        box-shadow: 0 0 15px rgba(139, 92, 246, 0.15);
    }

    .plan-header {
        display: flex;
        justify-content: space-between;
        align-items: center;
    }

    .plan-header h4 {
        margin: 0;
        font-size: 1.2rem;
        font-weight: 600;
    }

    .price {
        font-size: 2rem;
        font-weight: 700;
        color: #f8fafc;
    }

    .price .period {
        font-size: 0.9rem;
        color: var(--text-muted);
        font-weight: normal;
    }

    .plan-description {
        font-size: 0.9rem;
        line-height: 1.4;
    }

    .plan-features {
        list-style: none;
        display: flex;
        flex-direction: column;
        gap: 0.75rem;
        margin-top: 0.5rem;
    }

    .plan-features li {
        display: flex;
        align-items: center;
        gap: 0.6rem;
        font-size: 0.9rem;
    }

    .plan-features li.muted {
        color: var(--text-muted);
        opacity: 0.6;
    }
</style>
