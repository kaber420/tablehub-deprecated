<script>
    import { onMount } from "svelte";
    import { Search, RefreshCw, ChevronDown, ChevronUp } from '@lucide/svelte';
    import BranchTable from "../lib/components/BranchTable.svelte";
    import { organizations as _organizations, hubs } from "../lib/mock/clients.js";

    let organizations = [..._organizations];

    let searchQuery = '';
    let currentPage = 1;
    let pageSize = 3;

    let expandedOrgs = {};
    let loadingStats = false;

    // Helper functions to get stats per organization
    function getGatewaysCountForOrg(orgId) {
        return hubs.filter(h => h.organization_id === orgId).length;
    }

    function getEsp32CountForOrg(orgId) {
        return hubs.filter(h => h.organization_id === orgId).reduce((acc, curr) => acc + curr.esp32_iot_count, 0);
    }

    function getWaiterConnectionsForOrg(orgId) {
        return hubs.filter(h => h.organization_id === orgId).reduce((acc, curr) => acc + curr.cloud_waiters, 0);
    }

    function toggleOrgExpand(orgId) {
        expandedOrgs[orgId] = !expandedOrgs[orgId];
    }

    // Dynamic quota calculators based on Plus+ licenses
    // Each Plus+ license bought adds:
    // +1 Gateway (Hub or Android Screen acting as Hub)
    // +5 ESP32 IoT Devices
    // +10 Waiter/clients session limit
    function getGatewaysLimit(org) {
        return org.gateways_limit + (org.plus_licenses * 1);
    }

    function getEsp32Limit(org) {
        return org.esp32_iot_limit + (org.plus_licenses * 5);
    }

    function getWaiterLimit(org) {
        return org.waiter_app_limit + (org.plus_licenses * 10);
    }

    function modifyPlusLicenses(orgId, delta) {
        organizations = organizations.map(org => {
            if (org.id === orgId) {
                const current = org.plus_licenses || 0;
                return { ...org, plus_licenses: Math.max(0, current + delta) };
            }
            return org;
        });
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
        return o.name.toLowerCase().includes(searchQuery.toLowerCase()) || 
               o.owner.toLowerCase().includes(searchQuery.toLowerCase()) ||
               o.billing_plan.toLowerCase().includes(searchQuery.toLowerCase());
    });

    $: totalPages = Math.max(1, Math.ceil(filteredOrganizations.length / pageSize));
    $: paginatedOrganizations = filteredOrganizations.slice((currentPage - 1) * pageSize, currentPage * pageSize);

    function goToPage(page) {
        if (page >= 1 && page <= totalPages) {
            currentPage = page;
        }
    }

    function handleSearchInput(e) {
        searchQuery = e.target.value;
        currentPage = 1;
    }
</script>

<div class="organizations-page">
    <div class="view-header">
        <h1>Clientes y Organizaciones</h1>
        <p>Monitorea y administra cuentas de clientes, cuotas SaaS y licencias Plus+ acumulativas.</p>
    </div>

    <!-- Controls Panel -->
    <div class="panel controls-panel">
        <div class="search-bar-wrapper">
            <div class="search-input-container">
                <Search size={18} class="search-icon" />
                <input 
                    type="text" 
                    placeholder="Buscar por nombre, propietario o plan..." 
                    value={searchQuery}
                    on:input={handleSearchInput}
                />
            </div>
            <button class="btn-icon" on:click={loadStats} title="Recargar">
                <RefreshCw size={18} class={loadingStats ? "spin" : ""} />
            </button>
        </div>
    </div>

    <!-- Active view container -->
    <main class="dashboard-content">
        <div class="panel table-panel">
            <div class="table-header">
                <h3>Clientes, Sucursales y Consumo de Licencia</h3>
            </div>
            <div class="table-responsive">
                <table class="neomorphic-table">
                    <thead>
                        <tr>
                            <th style="width: 40px;"></th>
                            <th>Nombre Cliente</th>
                            <th>Contacto Principal</th>
                            <th>Plan SaaS</th>
                            <th>Gateways (Hubs/Pantallas)</th>
                            <th>Dispositivos IoT (ESP32)</th>
                            <th>Sesiones Cloud (Staff)</th>
                            <th>Licencias Plus+</th>
                        </tr>
                    </thead>
                    <tbody>
                        {#each paginatedOrganizations as org}
                            <tr>
                                <td>
                                    <button class="btn-expand" on:click={() => toggleOrgExpand(org.id)}>
                                        {#if expandedOrgs[org.id]}
                                            <ChevronUp size={16} />
                                        {:else}
                                            <ChevronDown size={16} />
                                        {/if}
                                    </button>
                                </td>
                                <td class="font-bold">
                                    {org.name} 
                                    <span class="badge slug">/{org.slug}</span>
                                </td>
                                <td>
                                    <div class="contact-cell">
                                        <span class="owner-name">{org.owner}</span>
                                        <span class="owner-email">{org.email}</span>
                                    </div>
                                </td>
                                <td>
                                        <div style="display: flex; flex-direction: column; gap: 4px;">
                                            <span class="badge plan-{org.billing_plan}">
                                                {org.billing_plan.toUpperCase()}
                                            </span>
                                            {#if org.plus_licenses > 0}
                                                <span class="badge-plus-multiplier">
                                                    Plus+ <span class="mult-num">x{org.plus_licenses}</span>
                                                </span>
                                            {/if}
                                        </div>
                                    </td>
                                    <td>
                                        <span class="font-bold">{getGatewaysCountForOrg(org.id)}</span> / 
                                        <span class="quota-total">{getGatewaysLimit(org)}</span>
                                        {#if org.plus_licenses > 0}
                                            <span class="plus-addition"> (+{org.plus_licenses * 1})</span>
                                        {/if}
                                    </td>
                                    <td>
                                        <span class="font-bold">{getEsp32CountForOrg(org.id)}</span> / 
                                        <span class="quota-total">{getEsp32Limit(org)}</span>
                                        {#if org.plus_licenses > 0}
                                            <span class="plus-addition"> (+{org.plus_licenses * 5})</span>
                                        {/if}
                                    </td>
                                    <td>
                                        <span class="font-bold">{getWaiterConnectionsForOrg(org.id)}</span> / 
                                        <span class="quota-total">{getWaiterLimit(org)}</span>
                                        {#if org.plus_licenses > 0}
                                            <span class="plus-addition"> (+{org.plus_licenses * 10})</span>
                                        {/if}
                                    </td>
                                    <td>
                                        <div class="license-controls">
                                            <button class="btn-license-adjust minus" on:click={() => modifyPlusLicenses(org.id, -1)}>-</button>
                                            <span class="license-count font-bold">{org.plus_licenses}</span>
                                            <button class="btn-license-adjust plus" on:click={() => modifyPlusLicenses(org.id, 1)}>+</button>
                                        </div>
                                    </td>
                                </tr>
                                {#if expandedOrgs[org.id]}
                                    <tr class="expanded-row">
                                        <td colspan="8">
                                            <BranchTable {org} />
                                        </td>
                                    </tr>
                                  {/if}
                              {/each}
                         </tbody>
                    </table>
                </div>
            </div>

            <div class="pagination-controls">
                <button class="pagination-btn" disabled={currentPage <= 1} on:click={() => goToPage(currentPage - 1)}>
                    Anterior
                </button>
                <span class="pagination-info">
                    Página {currentPage} de {totalPages} ({filteredOrganizations.length} resultados)
                </span>
                <button class="pagination-btn" disabled={currentPage >= totalPages} on:click={() => goToPage(currentPage + 1)}>
                    Siguiente
                </button>
            </div>
    </main>
</div>

<style>
    .organizations-page {
        display: flex;
        flex-direction: column;
        gap: 1.5rem;
        width: 100%;
        margin-top: 1rem;
    }

    .view-header {
        text-align: center;
        margin-bottom: 2rem;
    }
    
    .view-header h1 {
        font-size: 2.5rem;
        background: linear-gradient(135deg, #fff 0%, #a5b4fc 100%);
        -webkit-background-clip: text;
        -webkit-text-fill-color: transparent;
    }

    .view-header p {
        color: var(--text-muted);
        font-size: 1.1rem;
    }

    .badge {
        padding: 0.2rem 0.5rem;
        border-radius: 6px;
        font-size: 0.75rem;
        font-weight: 600;
        letter-spacing: 0.05em;
        display: inline-block;
        text-align: center;
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

    /* Plus+ License Multiplier Badge */
    .badge-plus-multiplier {
        background: rgba(245, 158, 11, 0.15);
        color: #fbbf24;
        border: 1px solid rgba(245, 158, 11, 0.3);
        padding: 0.1rem 0.4rem;
        border-radius: 5px;
        font-size: 0.7rem;
        font-weight: 700;
        display: inline-flex;
        align-items: center;
        gap: 2px;
        text-transform: uppercase;
        width: fit-content;
    }

    .mult-num {
        font-size: 0.8rem;
        font-weight: 900;
    }

    .quota-total {
        font-weight: 700;
    }

    .plus-addition {
        color: #fbbf24;
        font-weight: 600;
        font-size: 0.8rem;
    }

    /* License adjustment control panel */
    .license-controls {
        display: flex;
        align-items: center;
        gap: 0.5rem;
        background: var(--input-bg, rgba(0,0,0,0.1));
        padding: 0.2rem;
        border-radius: 8px;
        width: fit-content;
        border: var(--input-border, 1px solid rgba(255,255,255,0.05));
    }

    .btn-license-adjust {
        border: none;
        width: 24px;
        height: 24px;
        border-radius: 6px;
        background: rgba(255,255,255,0.05);
        color: var(--text-color);
        cursor: pointer;
        display: flex;
        align-items: center;
        justify-content: center;
        font-size: 1rem;
        font-weight: bold;
        transition: background 0.15s;
    }

    .btn-license-adjust:hover {
        background: var(--accent-color);
        color: white;
    }

    .license-count {
        min-width: 20px;
        text-align: center;
        font-size: 0.9rem;
    }

    .controls-panel {
        display: flex;
        flex-direction: column;
        gap: 1rem;
        padding: 1.25rem;
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

    .pagination-controls {
        display: flex;
        justify-content: center;
        align-items: center;
        gap: 1rem;
        padding: 1.25rem 0;
    }

    .pagination-btn {
        background: var(--input-bg);
        border: var(--input-border);
        color: var(--text-color);
        padding: 0.5rem 1.25rem;
        border-radius: 8px;
        cursor: pointer;
        font-size: 0.9rem;
        font-weight: 500;
        transition: all 0.15s;
    }

    .pagination-btn:hover:not(:disabled) {
        background: var(--accent-color);
        color: white;
    }

    .pagination-btn:disabled {
        opacity: 0.4;
        cursor: not-allowed;
    }

    .pagination-info {
        font-size: 0.85rem;
        color: var(--text-muted);
    }
</style>
