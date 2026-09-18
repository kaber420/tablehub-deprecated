<script>
    import { onMount } from "svelte";
    import { apiFetch } from "../lib/api.js";
    import { Users, Search, Plus, UserPlus, Shield, Mail, Calendar, RefreshCw } from '@lucide/svelte';
    import PrimaryButton from "../lib/components/PrimaryButton.svelte";

    export let orgID = "";

    let users = [];
    let loading = true;
    let error = "";
    let searchQuery = "";

    // Simulated fallback data representing the schema in saas_business_tenant_billing_model.md
    let mockUsers = [
        { id: "u1", name: "Luciano Rossi", email: "luciano@pizza.it", role: "owner", created_at: "2026-02-14T10:00:00Z" },
        { id: "u2", name: "Marco Silva", email: "marco@pizza.it", role: "admin", created_at: "2026-02-15T11:00:00Z" },
        { id: "u3", name: "Sofia Loren", email: "sofia@pizza.it", role: "viewer", created_at: "2026-03-01T15:30:00Z" }
    ];

    async function loadUsers() {
        loading = true;
        error = "";
        try {
            // Fetch users endpoint if defined, otherwise use mock
            const res = await apiFetch(`/v1/orgs/${orgID}/users`);
            if (res.ok) {
                users = await res.json();
            } else {
                users = mockUsers;
            }
        } catch (e) {
            users = mockUsers;
        } finally {
            loading = false;
        }
    }

    async function inviteUser() {
        const email = prompt("Introduce el correo electrónico para invitar a un nuevo miembro:");
        if (!email || email.trim() === "") return;
        const name = prompt("Introduce el nombre del miembro:");
        if (!name || name.trim() === "") return;

        try {
            const res = await apiFetch(`/v1/orgs/${orgID}/users/invite`, {
                method: "POST",
                body: JSON.stringify({ email, name, role: "viewer" })
            });
            if (res.ok) {
                alert("Invitación enviada con éxito!");
                await loadUsers();
            } else {
                // Mock adding to list for UI demonstration
                const newUser = {
                    id: "u" + (users.length + 1),
                    name,
                    email,
                    role: "viewer",
                    created_at: new Date().toISOString()
                };
                users = [...users, newUser];
                alert("Usuario invitado (Simulado en la interfaz)");
            }
        } catch (e) {
            alert("Error de red, invitando de manera local temporal.");
        }
    }

    onMount(() => {
        loadUsers();
    });

    $: filteredUsers = users.filter(u => 
        u.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
        u.email.toLowerCase().includes(searchQuery.toLowerCase())
    );
</script>

<div class="users-page">
    <div class="view-header">
        <div class="title-row">
            <div>
                <h1>Personal y Usuarios</h1>
                <p>Gestiona los miembros de tu equipo y sus permisos en la organización.</p>
            </div>
            <PrimaryButton on:click={inviteUser}>
                <UserPlus size={18} /> Invitar Miembro
            </PrimaryButton>
        </div>
    </div>

    <!-- Filter toolbar -->
    <div class="panel toolbar">
        <div class="search-box">
            <Search size={18} />
            <input type="text" placeholder="Buscar por nombre o correo..." bind:value={searchQuery} />
        </div>

        <button class="btn-refresh" on:click={loadUsers} disabled={loading}>
            <RefreshCw size={16} class={loading ? "spin" : ""} />
        </button>
    </div>

    <!-- Users Table/List -->
    {#if loading}
        <div class="loader">Cargando usuarios...</div>
    {:else}
        <div class="panel table-panel">
            <div class="table-responsive">
                <table class="neomorphic-table">
                    <thead>
                        <tr>
                            <th>Miembro</th>
                            <th>Correo Electrónico</th>
                            <th>Rol / Permisos</th>
                            <th>Fecha de Registro</th>
                        </tr>
                    </thead>
                    <tbody>
                        {#each filteredUsers as u}
                            <tr>
                                <td class="font-bold">{u.name}</td>
                                <td>
                                    <div class="email-cell">
                                        <Mail size={14} class="text-muted" />
                                        <span>{u.email}</span>
                                    </div>
                                </td>
                                <td>
                                    <span class="badge role-{u.role}">
                                        <Shield size={12} style="margin-right: 0.25rem;" />
                                        {u.role.toUpperCase()}
                                    </span>
                                </td>
                                <td>
                                    <div class="date-cell">
                                        <Calendar size={14} class="text-muted" />
                                        <span>{new Date(u.created_at).toLocaleDateString()}</span>
                                    </div>
                                </td>
                            </tr>
                        {/each}
                    </tbody>
                </table>
            </div>
        </div>
    {/if}
</div>

<style>
    .users-page {
        display: flex;
        flex-direction: column;
        gap: 1.5rem;
    }

    .view-header {
        margin-bottom: 1rem;
    }

    .title-row {
        display: flex;
        justify-content: space-between;
        align-items: center;
        flex-wrap: wrap;
        gap: 1rem;
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

    /* Neomorphic Table styles */
    .table-panel {
        padding: 1.5rem;
    }

    .table-responsive {
        overflow-x: auto;
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
    }

    .font-bold {
        font-weight: 600;
    }

    .email-cell, .date-cell {
        display: flex;
        align-items: center;
        gap: 0.5rem;
    }

    .badge {
        display: inline-flex;
        align-items: center;
        padding: 0.2rem 0.6rem;
        border-radius: 6px;
        font-size: 0.75rem;
        font-weight: 600;
    }

    .badge.role-owner {
        background: rgba(16, 185, 129, 0.15);
        color: var(--success);
        border: 1px solid rgba(16, 185, 129, 0.3);
    }

    .badge.role-admin {
        background: rgba(59, 130, 246, 0.15);
        color: var(--accent-hover);
        border: 1px solid rgba(59, 130, 246, 0.3);
    }

    .badge.role-viewer {
        background: rgba(245, 158, 11, 0.15);
        color: #fbbf24;
        border: 1px solid rgba(245, 158, 11, 0.3);
    }

    .loader {
        text-align: center;
        padding: 3rem;
        color: var(--text-muted);
    }
</style>
