<script>
    import { onMount, onDestroy } from "svelte";
    import { fade, scale } from 'svelte/transition';
    import { apiFetch, downloadHubProvision, createHub, fetchBranches, updateHub } from "../lib/api.js";
    import { Server, Search, RefreshCw, MapPin, Plus, DownloadCloud, CheckCircle, AlertTriangle, Edit, Settings, X } from '@lucide/svelte';

    export let orgID = "";

    let hubs = [];
    let branches = [];
    let loading = true;
    let error = "";
    let searchQuery = "";
    let statusFilter = "all";

    // Provisioning state
    let showProvisionModal = false;
    let provisioning = false;
    let provisionSuccess = false;
    let provisionError = "";
    let provAutoCloseTimer = null;
    let hubAutoCloseTimer = null;
    let downloadedFilename = "";
    let targetHubID = "";

    // Hub Modal state (Create/Edit)
    const POS_PROVIDERS = ["Blackshot", "Toast", "TastyIgniter", "Clover", "Genérico"];
    let showHubModal = false;
    let hubModalMode = 'create'; // 'create' or 'edit'
    let editingHub = null;
    let hubFormData = {
        name: "",
        branch_id: "",
        hardware_model: "",
        pos_provider: ""
    };
    let initialHubFormData = "";
    let hubSubmitting = false;
    let hubModalError = "";
    let hubModalSuccess = false;
    let newHubId = null;

    // Hubs list will be populated dynamically from the API

    async function loadData() {
        loading = true;
        error = "";
        try {
            // Load branches
            try {
                branches = await fetchBranches(orgID);
            } catch (e) {
                console.warn("Could not load branches", e);
                branches = [];
            }

            // Load hubs
            const res = await apiFetch(`/v1/orgs/${orgID}/hubs`);
            if (res.ok) {
                const data = await res.json();
                hubs = data.hubs || [];
            } else {
                throw new Error("Error en la respuesta de la API");
            }
        } catch (e) {
            error = e.message || "Error al cargar datos";
            hubs = [];
        } finally {
            loading = false;
        }
    }

    function getBranchName(branchId) {
        const b = branches.find(x => x.id === branchId);
        return b ? b.name : "Sin sucursal";
    }

    function openProvisionModal(hubID) {
        targetHubID = hubID;
        provisionSuccess = false;
        provisionError = "";
        provisioning = false;
        showProvisionModal = true;
    }

    function closeProvisionModal() {
        if (provAutoCloseTimer) {
            clearTimeout(provAutoCloseTimer);
            provAutoCloseTimer = null;
        }
        showProvisionModal = false;
        provisionSuccess = false;
        provisionError = "";
        provisioning = false;
        downloadedFilename = "";
    }

    function handleKeydown(e) {
        if (showProvisionModal && e.key === 'Escape') {
            closeProvisionModal();
        }
        if (showHubModal && e.key === 'Escape') {
            closeHubModal();
        }
    }

    async function handleProvision() {
        if (provisioning) return;
        provisioning = true;
        provisionError = "";
        provisionSuccess = false;
        try {
            downloadedFilename = await downloadHubProvision(orgID, targetHubID);
            provisionSuccess = true;
            loadData();
            provAutoCloseTimer = setTimeout(() => {
                closeProvisionModal();
            }, 5000);
        } catch (e) {
            provisionError = e.message || "Error desconocido al generar credenciales.";
        } finally {
            provisioning = false;
        }
    }

    function openCreateHubModal() {
        hubModalMode = 'create';
        editingHub = null;
        hubFormData = { name: "", branch_id: "", hardware_model: "", pos_provider: "" };
        initialHubFormData = JSON.stringify(hubFormData);
        hubModalError = "";
        hubModalSuccess = false;
        showHubModal = true;
    }

    function openEditHubModal(hub) {
        hubModalMode = 'edit';
        editingHub = hub;
        hubFormData = {
            name: hub.metadata?.name || "",
            branch_id: hub.branch_id || "",
            hardware_model: hub.metadata?.hardware_model || "",
            pos_provider: hub.metadata?.pos_provider || ""
        };
        initialHubFormData = JSON.stringify(hubFormData);
        hubModalError = "";
        hubModalSuccess = false;
        showHubModal = true;
    }

    function closeHubModal() {
        if (!hubModalSuccess && JSON.stringify(hubFormData) !== initialHubFormData) {
            if (!confirm("Tienes cambios sin guardar. ¿Seguro que quieres cerrar?")) return;
        }
        showHubModal = false;
        if (provAutoCloseTimer) {
            clearTimeout(provAutoCloseTimer);
            provAutoCloseTimer = null;
        }
    }

    async function submitHubModal() {
        if (!hubFormData.branch_id) {
            hubModalError = "Debes seleccionar una sucursal.";
            return;
        }
        hubSubmitting = true;
        hubModalError = "";
        
        try {
            const payload = {
                name: hubFormData.name,
                branch_id: hubFormData.branch_id || null,
                hardware_model: hubFormData.hardware_model,
                pos_provider: hubFormData.pos_provider
            };
            
            if (hubModalMode === 'create') {
                const res = await createHub(orgID, payload);
                newHubId = res.id || res.hub_id;
                hubModalSuccess = true;
                await loadData();
            } else {
                await updateHub(orgID, editingHub.id, payload);
                hubModalSuccess = true;
                await loadData();
                hubAutoCloseTimer = setTimeout(() => {
                    closeHubModal();
                }, 2000);
            }
        } catch (e) {
            hubModalError = e.message || "Error al procesar la solicitud.";
        } finally {
            hubSubmitting = false;
        }
    }

    async function downloadNewHubProvision() {
        if (!newHubId || provisioning) return;
        provisioning = true;
        try {
            downloadedFilename = await downloadHubProvision(orgID, newHubId);
            closeHubModal();
        } catch (e) {
            hubModalError = e.message || "Error al descargar credenciales.";
        } finally {
            provisioning = false;
        }
    }

    onMount(() => {
        loadData();
    });

    onDestroy(() => {
        if (autoCloseTimer) {
            clearTimeout(autoCloseTimer);
        }
    });

    $: filteredHubs = hubs.filter(h => {
        const branchName = getBranchName(h.branch_id);
        const matchesSearch = h.id.toLowerCase().includes(searchQuery.toLowerCase()) ||
                              branchName.toLowerCase().includes(searchQuery.toLowerCase()) ||
                              (h.organization_name && h.organization_name.toLowerCase().includes(searchQuery.toLowerCase()));
        const matchesStatus = statusFilter === "all" || h.connection_status === statusFilter;
        return matchesSearch && matchesStatus;
    });
</script>

<svelte:window on:keydown={handleKeydown} />

<div class="hubs-devices-page">
    <div class="view-header">
        <h1>Operaciones de Sucursal y Hubs</h1>
        <p class="text-muted">Monitorea el estado y el volumen de dispositivos conectados a tus Hubs locales.</p>
    </div>

    <!-- Filter toolbar -->
    <div class="panel toolbar">
        <div class="search-box">
            <Search size={18} />
            <input type="text" placeholder="Buscar por ID, sucursal u organización..." bind:value={searchQuery} />
        </div>

        <select bind:value={statusFilter} class="filter-select">
            <option value="all">Todos los estados de conexión</option>
            <option value="online">Hubs Online (Túnel activo)</option>
            <option value="offline">Hubs Offline (Túnel cerrado)</option>
        </select>

        <button class="btn-refresh" on:click={loadData} disabled={loading}>
            <RefreshCw size={16} class={loading ? "spin" : ""} />
        </button>

        <button class="btn-provision" on:click={openCreateHubModal}>
            <Plus size={16} />
            <span>Nuevo Hub Físico</span>
        </button>
    </div>

    <!-- Hubs list/grid representing devices -->
    {#if loading}
        <div class="loader">Cargando Hubs...</div>
    {:else}
        <div class="hubs-grid">
            {#each filteredHubs as hub}
                <div class="panel hub-card {hub.connection_status}">
                    <div class="card-header">
                        <div class="logo-area">
                            <Server size={24} style="color: var(--accent-color);" />
                            <span class="mono hub-title" title={hub.id}>{hub.metadata?.name || 'Hub ' + hub.id.substring(0, 6)}</span>
                        </div>
                        <div class="status-indicator">
                            <span class="led-indicator led-{hub.connection_status}"></span>
                            <span class="status-label">{hub.connection_status.toUpperCase()}</span>
                        </div>
                    </div>

                    <div class="hub-info">
                        <div class="meta-row">
                            <span class="meta-label">Organización:</span>
                            <span class="meta-value">
                                <strong>{hub.organization_name || "Desconocida"}</strong> 
                                <span class="badge plan-tag">{hub.billing_plan ? hub.billing_plan.toUpperCase() : "FREE"}</span>
                            </span>
                        </div>
                        <div class="meta-row">
                            <span class="meta-label">Sucursal:</span>
                            <span class="meta-value"><strong>{getBranchName(hub.branch_id)}</strong></span>
                        </div>
                        <div class="meta-row">
                            <span class="meta-label">Modelo Hardware:</span>
                            <span class="meta-value" style="font-weight: 500; color: #e2e8f0;">{hub.metadata?.hardware_model || "Genérico"}</span>
                        </div>
                        <div class="meta-row">
                            <span class="meta-label">Estado Administrativo:</span>
                            <span class="meta-value">
                                <span class="badge admin-status-{hub.administrative_status || 'activo'}">
                                    {(hub.administrative_status || 'activo').toUpperCase()}
                                </span>
                            </span>
                        </div>
                        <div class="meta-row">
                            <span class="meta-label">Motor POS local:</span>
                            <span class="meta-value badge pos-tag">{hub.metadata?.pos_provider || "No definido"}</span>
                        </div>
                    </div>

                    <!-- Simplified Counts Section -->
                    <div class="counts-section">
                        <h4 class="section-title">Dispositivos Conectados</h4>
                        <div class="counts-grid">
                            <div class="count-item">
                                <span class="count-value">{hub.iot_local_count || 0}</span>
                                <span class="count-label">📟 IoT Locales</span>
                            </div>

                            <div class="count-item">
                                <span class="count-value">{hub.local_waiters_count || 0}</span>
                                <span class="count-label">📱 Meseros WiFi</span>
                            </div>
                        </div>
                    </div>

                    <!-- Hub Actions Section -->
                    <div class="hub-actions">
                        {#if hub.connection_status === 'offline'}
                        <button class="btn-provision-card" on:click={() => openProvisionModal(hub.id)}>
                            <DownloadCloud size={16} />
                            Generar .thub
                        </button>
                        {/if}
                        <button class="btn-edit-card" on:click={() => openEditHubModal(hub)}>
                            <Edit size={16} />
                            Configurar
                        </button>
                    </div>
                </div>
            {/each}
        </div>
    {/if}

    {#if showProvisionModal}
        <div class="modal-backdrop" transition:fade={{ duration: 150 }} on:click|self={closeProvisionModal} role="dialog" aria-modal="true" aria-labelledby="prov-modal-title">
            <div class="modal-panel" transition:scale={{ duration: 200, start: 0.95 }}>
                {#if provisionSuccess}
                    <div class="modal-icon-success">
                        <CheckCircle size={48} />
                    </div>
                    <h2 id="prov-modal-title">Credenciales Generadas</h2>
                    <p class="modal-description">
                        Archivo <strong>{downloadedFilename}</strong> descargado. Carga este archivo en la interfaz de configuración de tu servidor físico.
                        Este modal se cerrará automáticamente en 5 segundos.
                    </p>
                    <button class="btn-close-modal" on:click={closeProvisionModal}>Cerrar</button>
                {:else}
                    <div class="modal-icon-info">
                        <DownloadCloud size={48} />
                    </div>
                    <h2 id="prov-modal-title">Nuevo Hub Físico</h2>
                    <p class="modal-description">
                        Estás a punto de registrar un nuevo Servidor Hub para tu restaurante.
                        Se descargará un archivo de credenciales de un solo uso.
                    </p>

                    {#if provisionError}
                        <div class="modal-alert-error">
                            <AlertTriangle size={16} />
                            <span>{provisionError}</span>
                        </div>
                    {/if}

                    <div class="modal-actions">
                        <button class="btn-cancel" on:click={closeProvisionModal} disabled={provisioning}>
                            Cancelar
                        </button>
                        <button class="btn-generate" on:click={handleProvision} disabled={provisioning}>
                            {#if provisioning}
                                <span class="spinner"></span>
                                Generando...
                            {:else}
                                <DownloadCloud size={16} />
                                Generar y Descargar Credenciales
                            {/if}
                        </button>
                    </div>
                {/if}
            </div>
        </div>
    {/if}
</div>

{#if showHubModal}
<div class="modal-backdrop" transition:fade={{ duration: 150 }} on:click|self={closeHubModal} role="dialog" aria-modal="true" aria-labelledby="hub-modal-title">
    <div class="modal-panel hub-modal-panel" transition:scale={{ duration: 200, start: 0.95 }}>
        <button class="modal-close-btn" on:click={closeHubModal} aria-label="Cerrar modal">
            <X size={20} />
        </button>
        
        {#if hubModalSuccess}
            <div class="modal-icon-success">
                <CheckCircle size={48} />
            </div>
            <h2 id="hub-modal-title">{hubModalMode === "create" ? "Hub Creado Exitosamente" : "Cambios Guardados"}</h2>
            
            {#if hubModalMode === 'create'}
                <p class="modal-description">
                    El Hub físico ha sido registrado. ¿Deseas descargar el archivo de credenciales (.thub) ahora?
                </p>
                {#if hubModalError}
                    <div class="modal-alert-error">
                        <AlertTriangle size={16} />
                        <span>{hubModalError}</span>
                    </div>
                {/if}
                <div class="modal-actions">
                    <button class="btn-cancel" on:click={closeHubModal} disabled={provisioning}>Cerrar</button>
                    <button class="btn-generate" on:click={downloadNewHubProvision} disabled={provisioning}>
                        {#if provisioning}
                            <span class="spinner"></span> Generando...
                        {:else}
                            <DownloadCloud size={16} /> Descargar .thub
                        {/if}
                    </button>
                </div>
            {:else}
                <p class="modal-description">La configuración se ha actualizado correctamente.</p>
            {/if}
        {:else}
            <div class="modal-header-icon">
                <Settings size={32} />
            </div>
            <h2 id="hub-modal-title">{hubModalMode === "create" ? "Nuevo Hub Físico" : "Configurar Hub"}</h2>
            <p class="modal-desc">
                {hubModalMode === 'create' ? 'Asigna una sucursal y define el hardware/software de este nuevo equipo.' : `Editando configuración del Hub ${editingHub.id.substring(0,8)}...`}
            </p>
            
            {#if hubModalError}
                <div class="modal-alert-error">
                    <AlertTriangle size={16} />
                    <span>{hubModalError}</span>
                </div>
            {/if}

            <div class="form-section">
                <h4 class="section-title">Datos Principales</h4>
                <div class="form-group">
                    <label for="hubName">Nombre del Hub</label>
                    <input id="hubName" type="text" bind:value={hubFormData.name} placeholder="Ej. Hub Principal Caja 1" />
                </div>
                <div class="form-group">
                    <label for="hubBranch">Sucursal <span class="required">*</span></label>
                    <select id="hubBranch" bind:value={hubFormData.branch_id} class:error={hubModalError && !hubFormData.branch_id}>
                        <option value="">Seleccionar sucursal...</option>
                        {#each branches as branch}
                            <option value={branch.id}>{branch.name} ({branch.address})</option>
                        {/each}
                    </select>
                </div>
            </div>

            <div class="form-section">
                <h4 class="section-title">Configuración Técnica</h4>
                <div class="form-group">
                    <label for="hubMotor">Motor POS</label>
                    <select id="hubMotor" bind:value={hubFormData.pos_provider}>
                        <option value="">Seleccionar motor...</option>
                        {#each POS_PROVIDERS as provider}
                            <option value={provider}>{provider}</option>
                        {/each}
                    </select>
                </div>

                <div class="form-group">
                    <label for="hubModel">Modelo de Hardware</label>
                    <select id="hubModel" bind:value={hubFormData.hardware_model}>
                        <option value="">Seleccionar modelo...</option>
                        <option value="Genérico">Genérico</option>
                        <option value="Tablehub-Official-v1">Tablehub Official v1</option>
                    </select>
                </div>
            </div>

            <div class="modal-actions">
                <button class="btn-cancel" on:click={closeHubModal} disabled={hubSubmitting}>Cancelar</button>
                <button class="btn-generate" on:click={submitHubModal} disabled={hubSubmitting || !hubFormData.branch_id}>
                    {#if hubSubmitting}
                        <span class="spinner"></span> {hubModalMode === 'create' ? 'Creando...' : 'Guardando...'}
                    {:else}
                        {hubModalMode === 'create' ? 'Crear Hub' : 'Guardar Cambios'}
                    {/if}
                </button>
            </div>
        {/if}
    </div>
</div>
{/if}

<style>
    .hubs-devices-page {
        display: flex;
        flex-direction: column;
        gap: 1.5rem;
    }

    .view-header {
        margin-bottom: 1rem;
    }

    .view-header h1 {
        font-size: 2.2rem;
        background: linear-gradient(135deg, #fff 0%, #a5b4fc 100%);
        -webkit-background-clip: text;
        -webkit-text-fill-color: transparent;
    }

    .toolbar {
        display: flex;
        gap: 1rem;
        align-items: center;
        padding: 1rem;
    }

    .search-box {
        display: flex;
        align-items: center;
        gap: 0.5rem;
        flex: 1;
        background: var(--input-bg);
        border: var(--input-border);
        box-shadow: var(--input-shadow);
        border-radius: 8px;
        padding: 0.25rem 0.75rem;
        color: var(--text-muted);
    }

    .search-box input {
        border: none;
        background: transparent;
        box-shadow: none;
        width: 100%;
        color: var(--text-color);
        padding: 0.5rem 0;
    }

    .filter-select {
        min-width: 220px;
    }

    .btn-refresh {
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
    }

    .spin {
        animation: spin 1s linear infinite;
    }

    @keyframes spin {
        0% { transform: rotate(0deg); }
        100% { transform: rotate(360deg); }
    }

    .hubs-grid {
        display: grid;
        grid-template-columns: repeat(auto-fill, minmax(360px, 1fr));
        gap: 1.5rem;
    }

    .hub-card {
        display: flex;
        flex-direction: column;
        gap: 1.25rem;
        padding: 1.5rem;
        transition: transform 0.2s, box-shadow 0.2s;
    }

    .hub-card:hover {
        transform: translateY(-2px);
    }

    .card-header {
        display: flex;
        justify-content: space-between;
        align-items: center;
        border-bottom: 1px solid rgba(255, 255, 255, 0.05);
        padding-bottom: 0.75rem;
    }

    .hub-title {
        font-weight: 700;
        font-size: 1.05rem;
    }

    .status-indicator {
        display: flex;
        align-items: center;
        gap: 0.5rem;
    }

    .status-label {
        font-size: 0.75rem;
        font-weight: 600;
        letter-spacing: 0.05em;
    }

    .hub-info {
        display: flex;
        flex-direction: column;
        gap: 0.5rem;
        font-size: 0.9rem;
    }

    .meta-row {
        display: flex;
        justify-content: space-between;
        align-items: center;
    }

    .meta-label {
        color: var(--text-muted);
    }

    .meta-value {
        color: var(--text-color);
        display: inline-flex;
        align-items: center;
        gap: 0.5rem;
    }

    .counts-section {
        margin-top: 0.5rem;
        border-top: 1px solid rgba(255, 255, 255, 0.05);
        padding-top: 1rem;
    }

    .section-title {
        font-size: 0.85rem;
        color: var(--accent-hover);
        text-transform: uppercase;
        letter-spacing: 0.05em;
        margin-bottom: 0.75rem;
        margin-top: 0;
    }

    .counts-grid {
        display: grid;
        grid-template-columns: repeat(3, 1fr);
        gap: 0.75rem;
        text-align: center;
    }

    .count-item {
        background: rgba(255, 255, 255, 0.02);
        border: 1px solid rgba(255, 255, 255, 0.05);
        padding: 0.75rem 0.5rem;
        border-radius: 8px;
        display: flex;
        flex-direction: column;
        gap: 0.25rem;
    }

    .count-value {
        font-size: 1.25rem;
        font-weight: 700;
        color: #f1f5f9;
    }

    .count-label {
        font-size: 0.75rem;
        color: var(--text-muted);
    }

    /* Badges */
    .badge {
        padding: 0.15rem 0.4rem;
        border-radius: 4px;
        font-size: 0.7rem;
        font-weight: 600;
    }

    .plan-tag {
        background: rgba(139, 92, 246, 0.15);
        color: #c084fc;
        border: 1px solid rgba(139, 92, 246, 0.3);
    }

    .admin-status-activo {
        background: rgba(16, 185, 129, 0.15);
        color: #34d399;
        border: 1px solid rgba(16, 185, 129, 0.3);
    }

    .admin-status-suspendido {
        background: rgba(245, 158, 11, 0.15);
        color: #fbbf24;
        border: 1px solid rgba(245, 158, 11, 0.3);
    }

    .admin-status-inhabilitado {
        background: rgba(239, 68, 68, 0.15);
        color: #fca5a5;
        border: 1px solid rgba(239, 68, 68, 0.3);
    }

    .pos-tag {
        background: rgba(59, 130, 246, 0.1);
        color: var(--accent-hover);
        border: 1px solid rgba(59, 130, 246, 0.2);
    }

    .loader {
        text-align: center;
        padding: 2rem;
        color: var(--text-muted);
    }

    /* Provision Button */
    .btn-provision {
        display: flex;
        align-items: center;
        gap: 0.5rem;
        background: linear-gradient(135deg, #6366f1, #8b5cf6);
        color: #fff;
        border: none;
        border-radius: 8px;
        padding: 0.6rem 1.1rem;
        font-weight: 600;
        font-size: 0.85rem;
        cursor: pointer;
        transition: opacity 0.2s, transform 0.15s;
        white-space: nowrap;
    }

    .btn-provision:hover {
        opacity: 0.9;
        transform: translateY(-1px);
    }

    /* Modal */
    .modal-backdrop {
        position: fixed;
        inset: 0;
        background: rgba(0, 0, 0, 0.7);
        display: flex;
        align-items: center;
        justify-content: center;
        z-index: 1000;
        backdrop-filter: blur(4px);
    }

    .modal-panel {
        background: #1e1b2e;
        border: 1px solid rgba(255, 255, 255, 0.08);
        border-radius: 16px;
        padding: 2.5rem;
        max-width: 440px;
        width: 90%;
        text-align: center;
        box-shadow: 0 25px 50px -12px rgba(0, 0, 0, 0.6);
    }

    .modal-icon-info {
        color: #818cf8;
        margin-bottom: 1rem;
    }

    .modal-icon-success {
        color: #34d399;
        margin-bottom: 1rem;
    }

    .modal-panel h2 {
        font-size: 1.4rem;
        margin: 0 0 0.75rem;
        color: #f1f5f9;
    }

    .modal-description {
        color: var(--text-muted);
        font-size: 0.9rem;
        line-height: 1.6;
        margin: 0 0 1.5rem;
    }

    .modal-alert-error {
        display: flex;
        align-items: center;
        gap: 0.5rem;
        background: rgba(239, 68, 68, 0.12);
        border: 1px solid rgba(239, 68, 68, 0.3);
        color: #fca5a5;
        padding: 0.75rem 1rem;
        border-radius: 8px;
        font-size: 0.85rem;
        margin-bottom: 1.25rem;
    }

    .modal-actions {
        display: flex;
        gap: 0.75rem;
        justify-content: center;
    }

    .btn-cancel {
        background: rgba(255, 255, 255, 0.06);
        color: var(--text-muted);
        border: 1px solid rgba(255, 255, 255, 0.1);
        border-radius: 8px;
        padding: 0.6rem 1.2rem;
        font-size: 0.85rem;
        cursor: pointer;
        transition: background 0.2s;
    }

    .btn-cancel:hover {
        background: rgba(255, 255, 255, 0.1);
    }

    .btn-cancel:disabled {
        opacity: 0.4;
        cursor: not-allowed;
    }

    .btn-generate {
        display: flex;
        align-items: center;
        gap: 0.5rem;
        background: linear-gradient(135deg, #6366f1, #8b5cf6);
        color: #fff;
        border: none;
        border-radius: 8px;
        padding: 0.6rem 1.2rem;
        font-weight: 600;
        font-size: 0.85rem;
        cursor: pointer;
        transition: opacity 0.2s;
    }

    .btn-generate:hover {
        opacity: 0.9;
    }

    .btn-generate:disabled {
        opacity: 0.6;
        cursor: not-allowed;
    }

    .btn-close-modal {
        background: linear-gradient(135deg, #6366f1, #8b5cf6);
        color: #fff;
        border: none;
        border-radius: 8px;
        padding: 0.6rem 1.5rem;
        font-weight: 600;
        font-size: 0.85rem;
        cursor: pointer;
    }

    .btn-close-modal:hover {
        opacity: 0.9;
    }

    .spinner {
        display: inline-block;
        width: 14px;
        height: 14px;
        border: 2px solid rgba(255, 255, 255, 0.3);
        border-top-color: #fff;
        border-radius: 50%;
        animation: spin 0.6s linear infinite;
    }

    .hub-actions {
        margin-top: 0.5rem;
        padding-top: 1rem;
        border-top: 1px dashed rgba(255, 255, 255, 0.1);
        display: flex;
        justify-content: flex-end;
        gap: 0.75rem;
    }

    .btn-provision-card, .btn-edit-card {
        display: flex;
        align-items: center;
        gap: 0.5rem;
        border-radius: 6px;
        padding: 0.5rem 0.8rem;
        font-weight: 600;
        font-size: 0.8rem;
        cursor: pointer;
        transition: all 0.2s;
    }

    .btn-provision-card {
        background: rgba(99, 102, 241, 0.15);
        color: #818cf8;
        border: 1px solid rgba(99, 102, 241, 0.3);
    }

    .btn-provision-card:hover {
        background: rgba(99, 102, 241, 0.25);
        border-color: rgba(99, 102, 241, 0.5);
    }

    .btn-edit-card {
        background: rgba(255, 255, 255, 0.05);
        color: var(--text-color);
        border: 1px solid rgba(255, 255, 255, 0.1);
    }

    .btn-edit-card:hover {
        background: rgba(255, 255, 255, 0.1);
        border-color: rgba(255, 255, 255, 0.2);
    }

    .spinner-small {
        width: 16px;
        height: 16px;
        border: 2px solid rgba(255, 255, 255, 0.3);
        border-top-color: #fff;
        border-radius: 50%;
        animation: spin 0.6s linear infinite;
    }

    .modal-close-btn {
        position: absolute;
        top: 1rem;
        right: 1rem;
        background: transparent;
        border: none;
        color: var(--text-muted);
        cursor: pointer;
        transition: color 0.2s;
    }
    .modal-close-btn:hover {
        color: var(--text-color);
    }
    .hub-modal-panel {
        max-width: 500px;
        position: relative;
    }
    .modal-header-icon {
        color: var(--accent-color);
        margin-bottom: 1rem;
        display: flex;
        justify-content: center;
    }
    .modal-desc {
        color: var(--text-muted);
        font-size: 0.9rem;
        margin-bottom: 1.5rem;
    }
    .form-section {
        background: rgba(255, 255, 255, 0.02);
        border: 1px solid rgba(255, 255, 255, 0.05);
        border-radius: 8px;
        padding: 1.25rem;
        margin-bottom: 1rem;
        text-align: left;
    }
    .form-group {
        display: flex;
        flex-direction: column;
        gap: 0.5rem;
        margin-bottom: 1rem;
    }
    .form-group:last-child {
        margin-bottom: 0;
    }
    .form-group label {
        font-size: 0.85rem;
        font-weight: 600;
        color: var(--text-color);
    }
    .required {
        color: #ef4444;
    }
    .form-group select, .form-group input {
        background: var(--input-bg);
        border: var(--input-border);
        color: var(--text-color);
        padding: 0.65rem;
        border-radius: 6px;
        font-size: 0.9rem;
    }
    .form-group select.error {
        border-color: #ef4444;
    }
</style>
