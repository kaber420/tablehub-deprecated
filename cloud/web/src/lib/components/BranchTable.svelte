<script>
    import { MapPin } from '@lucide/svelte';
    import { branches, hubs } from '../mock/clients.js';

    export let org;

    function getBranchesForOrg(orgId) {
        return branches.filter(b => b.org_id === orgId);
    }

    function getHubsForBranch(branchId) {
        return hubs.filter(h => h.branch_id === branchId);
    }

    $: orgBranches = getBranchesForOrg(org.id);
</script>

<div class="nested-container">
    <div style="display: flex; justify-content: space-between; align-items: flex-start; gap: 2rem; flex-wrap: wrap;">
        <div style="flex: 1; min-width: 300px;">
            <h4>Consumo de Recursos Cloud por Sucursal:</h4>
            {#if orgBranches.length === 0}
                <p class="empty-text">No hay sucursales registradas.</p>
            {:else}
                <div class="branches-grid">
                    {#each orgBranches as branch}
                        {@const branchHubs = getHubsForBranch(branch.id)}
                        <div class="panel-inset branch-item">
                            <div class="branch-meta">
                                <div class="branch-name">
                                    <MapPin size={16} style="color: var(--accent-color);" />
                                    <strong>{branch.name}</strong>
                                </div>
                                <span class="branch-address">{branch.address}</span>
                            </div>
                            
                            <div class="nested-hubs">
                                <h5>Gateways Conectados y Nodos IoT (ESP32):</h5>
                                {#if branchHubs.length === 0}
                                    <p class="empty-text">No hay Gateways Cloud registrados en esta sucursal.</p>
                                {:else}
                                    <ul class="nested-hubs-list">
                                        {#each branchHubs as hub}
                                            <li class="nested-hub-row">
                                                <div style="display: flex; flex-direction: column; gap: 2px;">
                                                    <span class="mono" style="font-weight: 600;">
                                                        {#if hub.type === 'android_screen_gateway'}
                                                            📱 Pantalla Cloud (AP)
                                                        {:else}
                                                            📟 Hub Físico
                                                        {/if}
                                                    </span>
                                                    <span class="mono" style="font-size: 0.75rem; color: var(--text-muted);">
                                                        ID: {hub.id.substring(0, 8)}...
                                                    </span>
                                                </div>
                                                <span class="pos-badge">{hub.settings.pos_provider}</span>
                                                <div style="display: flex; gap: 0.75rem; font-size: 0.8rem; color: var(--text-muted);">
                                                    <span style="color: var(--accent-color); font-weight: 600;">
                                                        🔌 {hub.esp32_iot_count} Nodos ESP32
                                                    </span>
                                                    <span>📱 {hub.cloud_waiters} Sesiones Cloud</span>
                                                </div>
                                            </li>
                                        {/each}
                                    </ul>
                                {/if}
                            </div>
                        </div>
                    {/each}
                </div>
            {/if}
        </div>
        
        <div style="min-width: 250px; background: rgba(255, 255, 255, 0.03); padding: 1.25rem; border-radius: 8px; border: 1px solid rgba(255, 255, 255, 0.05); align-self: flex-start;">
            <h4 style="margin-top: 0; color: var(--accent-color);">Módulos SaaS Activos</h4>
            {#if org.active_modules.length === 0}
                <p class="empty-text">Ninguno</p>
            {:else}
                <ul style="list-style: none; padding: 0; margin: 0; display: flex; flex-direction: column; gap: 0.5rem; margin-bottom: 1rem;">
                    {#each org.active_modules as mod}
                        <li style="display: flex; align-items: center; gap: 0.5rem; font-size: 0.85rem; color: #f1f5f9;">
                            <span style="display: inline-block; width: 6px; height: 6px; border-radius: 50%; background: var(--accent-color);"></span>
                            {mod}
                        </li>
                    {/each}
                </ul>
            {/if}
            
            <h4 style="margin-top: 0; color: var(--accent-color);">Estado de Cuenta</h4>
            <p style="margin: 0.25rem 0; font-size: 0.85rem;">
                <strong>Fecha Registro:</strong> {new Date(org.created_at).toLocaleDateString()}
            </p>
            <p style="margin: 0.25rem 0; font-size: 0.85rem;">
                <strong>Facturación:</strong> {org.billing_plan !== 'free' ? 'Tarjeta Registrada' : 'Sin Tarjeta'}
            </p>
            <p style="margin: 0.25rem 0; font-size: 0.85rem;">
                <strong>Licencias Adicionales:</strong> {org.plus_licenses} Licencias Plus+
            </p>
        </div>
    </div>
</div>

<style>
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
</style>
