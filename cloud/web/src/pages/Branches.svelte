<script>
    import { onMount } from "svelte";
    import { fetchBranches, createBranch } from "../lib/api.js";
    import { showToast } from "../lib/toast.js";
    import { Building, Plus, MapPin, RefreshCw } from '@lucide/svelte';

    export let orgID = "";

    let branches = [];
    let loading = true;
    let error = "";
    
    // Modal state
    let showCreateModal = false;
    let newBranchName = "";
    let newBranchAddress = "";
    let creating = false;

    async function loadData() {
        loading = true;
        error = "";
        try {
            const data = await fetchBranches(orgID);
            branches = data || [];
        } catch (e) {
            error = e.message || "Error al cargar sucursales";
            console.error(e);
        } finally {
            loading = false;
        }
    }

    async function handleCreateBranch() {
        if (!newBranchName) {
            showToast("El nombre es obligatorio", "error");
            return;
        }
        if (newBranchName.length > 100) {
            showToast("El nombre no puede exceder los 100 caracteres", "error");
            return;
        }
        creating = true;
        try {
            await createBranch(orgID, {
                name: newBranchName,
                address: newBranchAddress
            });
            showCreateModal = false;
            newBranchName = "";
            newBranchAddress = "";
            showToast("Sucursal creada exitosamente");
            await loadData();
        } catch (e) {
            showToast(e.message || "Error al crear sucursal", "error");
        } finally {
            creating = false;
        }
    }

    onMount(() => {
        loadData();
    });
</script>

<div class="branches-view">
    <div class="header">
        <div class="title-group">
            <Building size={24} class="title-icon" />
            <h2>Sucursales</h2>
        </div>
        <div class="actions">
            <button class="btn-refresh" on:click={loadData} disabled={loading}>
                <RefreshCw size={16} class={loading ? "spin" : ""} />
            </button>
            <button class="btn-primary" on:click={() => showCreateModal = true}>
                <Plus size={16} />
                <span>Nueva Sucursal</span>
            </button>
        </div>
    </div>

    {#if loading}
        <div class="state-message">
            <div class="spinner"></div>
            <p>Cargando sucursales...</p>
        </div>
    {:else if error}
        <div class="state-message error">
            <p>{error}</p>
            <button on:click={loadData} class="btn-retry">Reintentar</button>
        </div>
    {:else if branches.length === 0}
        <div class="state-message empty">
            <Building size={48} strokeWidth={1} />
            <p>No tienes sucursales registradas.</p>
            <button class="btn-primary mt-4" on:click={() => showCreateModal = true}>
                Añadir tu primera sucursal
            </button>
        </div>
    {:else}
        <div class="branches-grid">
            {#each branches as branch}
                <div class="branch-card panel">
                    <div class="branch-header">
                        <Building size={20} class="branch-icon" />
                        <h3>{branch.name}</h3>
                    </div>
                    <div class="branch-details">
                        <div class="detail-row">
                            <MapPin size={14} />
                            <span>{branch.address || "Sin dirección"}</span>
                        </div>
                    </div>
                </div>
            {/each}
        </div>
    {/if}
</div>

{#if showCreateModal}
<div class="modal-backdrop" on:click|self={() => showCreateModal = false}>
    <div class="modal-content">
        <h3>Nueva Sucursal</h3>
        <div class="form-group">
            <label for="bName">Nombre de la Sucursal</label>
            <input id="bName" type="text" bind:value={newBranchName} placeholder="Ej. Sucursal Centro" />
        </div>
        <div class="form-group">
            <label for="bAddress">Dirección</label>
            <input id="bAddress" type="text" bind:value={newBranchAddress} placeholder="Ej. Av. Principal 123" />
        </div>
        <div class="modal-actions">
            <button class="btn-secondary" on:click={() => showCreateModal = false} disabled={creating}>Cancelar</button>
            <button class="btn-primary" on:click={handleCreateBranch} disabled={creating}>
                {#if creating}
                    <div class="spinner-small"></div>
                {:else}
                    Crear Sucursal
                {/if}
            </button>
        </div>
    </div>
</div>
{/if}

<style>
    .branches-view {
        display: flex;
        flex-direction: column;
        gap: 1.5rem;
        animation: dashboardIn 0.5s cubic-bezier(0.25, 1, 0.5, 1);
    }
    
    @keyframes dashboardIn {
        from { opacity: 0; transform: translateY(16px); }
        to { opacity: 1; transform: translateY(0); }
    }

    .header {
        display: flex;
        justify-content: space-between;
        align-items: center;
        background: var(--panel-bg);
        border: 1px solid var(--panel-border);
        padding: 1.25rem 1.5rem;
        border-radius: 12px;
        box-shadow: var(--panel-shadow);
    }

    .title-group {
        display: flex;
        align-items: center;
        gap: 0.75rem;
    }

    .title-icon {
        color: var(--accent-color);
    }

    h2 {
        font-size: 1.25rem;
        font-weight: 600;
        margin: 0;
        color: var(--text-color);
    }

    .actions {
        display: flex;
        gap: 0.75rem;
    }

    .btn-primary {
        display: flex;
        align-items: center;
        gap: 0.5rem;
        background: var(--accent-color);
        color: #fff;
        border: none;
        padding: 0.5rem 1rem;
        border-radius: 6px;
        font-weight: 600;
        cursor: pointer;
        transition: all 0.2s;
    }

    .btn-primary:hover:not(:disabled) {
        background: var(--accent-hover);
        transform: translateY(-1px);
    }

    .btn-primary:disabled {
        opacity: 0.7;
        cursor: not-allowed;
    }

    .btn-secondary {
        background: transparent;
        color: var(--text-color);
        border: 1px solid var(--border-color);
        padding: 0.5rem 1rem;
        border-radius: 6px;
        cursor: pointer;
        transition: all 0.2s;
    }

    .btn-secondary:hover {
        background: rgba(255,255,255,0.05);
    }

    .btn-refresh {
        background: transparent;
        border: 1px solid rgba(255, 255, 255, 0.1);
        color: var(--text-color);
        width: 36px;
        height: 36px;
        border-radius: 6px;
        display: flex;
        align-items: center;
        justify-content: center;
        cursor: pointer;
        transition: all 0.2s;
    }

    .btn-refresh:hover:not(:disabled) {
        background: rgba(255, 255, 255, 0.05);
    }

    .spin {
        animation: spin 1s linear infinite;
    }

    .branches-grid {
        display: grid;
        grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
        gap: 1.25rem;
    }

    .branch-card {
        padding: 1.5rem;
        display: flex;
        flex-direction: column;
        gap: 1rem;
        transition: transform 0.2s;
    }

    .branch-card:hover {
        transform: translateY(-2px);
    }

    .branch-header {
        display: flex;
        align-items: center;
        gap: 0.75rem;
    }

    .branch-icon {
        color: #f59e0b;
        background: rgba(245, 158, 11, 0.1);
        padding: 6px;
        border-radius: 8px;
        width: 32px;
        height: 32px;
    }

    .branch-header h3 {
        margin: 0;
        font-size: 1.1rem;
        color: var(--text-color);
    }

    .branch-details {
        display: flex;
        flex-direction: column;
        gap: 0.5rem;
    }

    .detail-row {
        display: flex;
        align-items: center;
        gap: 0.5rem;
        color: var(--text-muted);
        font-size: 0.9rem;
    }

    /* States */
    .state-message {
        display: flex;
        flex-direction: column;
        align-items: center;
        justify-content: center;
        padding: 4rem 2rem;
        text-align: center;
        background: var(--panel-bg);
        border: 1px solid var(--panel-border);
        border-radius: 12px;
        color: var(--text-muted);
    }

    .mt-4 { margin-top: 1rem; }

    .spinner {
        width: 30px;
        height: 30px;
        border: 3px solid rgba(255, 255, 255, 0.1);
        border-top-color: var(--accent-color);
        border-radius: 50%;
        animation: spin 1s linear infinite;
        margin-bottom: 1rem;
    }

    .spinner-small {
        width: 16px;
        height: 16px;
        border: 2px solid rgba(255, 255, 255, 0.3);
        border-top-color: #fff;
        border-radius: 50%;
        animation: spin 0.6s linear infinite;
    }

    /* Modal */
    .modal-backdrop {
        position: fixed;
        inset: 0;
        background: rgba(0, 0, 0, 0.6);
        backdrop-filter: blur(4px);
        display: flex;
        align-items: center;
        justify-content: center;
        z-index: 1000;
    }

    .modal-content {
        background: #1e1e24;
        border: 1px solid rgba(255, 255, 255, 0.1);
        padding: 2rem;
        border-radius: 12px;
        width: 100%;
        max-width: 400px;
        box-shadow: 0 20px 25px -5px rgba(0, 0, 0, 0.5);
    }

    .modal-content h3 {
        margin: 0 0 1.5rem 0;
        color: #fff;
    }

    .form-group {
        margin-bottom: 1.25rem;
        display: flex;
        flex-direction: column;
        gap: 0.5rem;
    }

    .form-group label {
        font-size: 0.9rem;
        color: var(--text-muted);
    }

    .form-group input {
        background: rgba(0, 0, 0, 0.2);
        border: 1px solid rgba(255, 255, 255, 0.1);
        padding: 0.75rem;
        border-radius: 6px;
        color: #fff;
        font-family: inherit;
    }

    .form-group input:focus {
        outline: none;
        border-color: var(--accent-color);
    }

    .modal-actions {
        display: flex;
        justify-content: flex-end;
        gap: 0.75rem;
        margin-top: 2rem;
    }
</style>
