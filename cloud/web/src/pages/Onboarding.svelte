<script>
    import { onMount } from "svelte";
    import { apiFetch } from "../lib/api.js";
    import { logout } from "../lib/auth.js";

    export let onComplete; // Callback function passed from App.svelte to reload profile

    let mode = "create"; // create
    let name = "";
    let slug = "";

    let error = "";
    let loading = false;

    // Fetch the current registration status
    async function checkStatus() {
        loading = true;
        error = "";
        try {
            const res = await apiFetch("/v1/me");
            if (res.ok) {
                const data = await res.json();
                if (data.registered) {
                    onComplete(data);
                } else {
                    mode = "create";
                }
            } else {
                error = "Error al conectar con el servidor.";
            }
        } catch (e) {
            error = "Error de red al verificar estado.";
        } finally {
            loading = false;
        }
    }

    async function handleCreateOrg() {
        if (!name || !slug) {
            error = "Todos los campos son requeridos.";
            return;
        }
        loading = true;
        error = "";
        try {
            const res = await apiFetch("/v1/onboarding", {
                method: "POST",
                body: JSON.stringify({ name, slug }),
            });
            const data = await res.json();
            if (res.ok) {
                onComplete(data);
            } else {
                error = data.error || "Ocurrió un error en el registro.";
            }
        } catch (e) {
            error = "Error al registrar la organización.";
        } finally {
            loading = false;
        }
    }

    onMount(() => {
        checkStatus();
    });
</script>

<div class="onboarding-page">
    <div class="glass-panel">
        <header class="header">
            <div style="display: flex; justify-content: space-between; align-items: center; width: 100%; margin-bottom: 0.5rem;">
                <h1 class="logo" style="margin: 0;">Table<span>hub</span></h1>
                <button type="button" on:click={logout} style="background: rgba(255,255,255,0.1); border: 1px solid rgba(255,255,255,0.2); color: #fff; padding: 0.5rem 1rem; border-radius: 6px; cursor: pointer; font-size: 0.85rem;">Cerrar Sesión</button>
            </div>
            <p class="subtitle" style="text-align: left; margin: 0;">Bienvenido a la plataforma de gestión y control local</p>
        </header>

        {#if loading}
            <div class="status-box">
                <div class="spinner"></div>
                <p>Cargando información...</p>
            </div>
        {:else}
            {#if error}
                <div class="error-alert">
                    <p>{error}</p>
                </div>
            {/if}

            {#if mode === "create"}
                <form class="form-container" on:submit|preventDefault={handleCreateOrg}>
                    <h2>Crear Organización (Plan Owner)</h2>
                    <p class="description">Registra tu empresa para comenzar a provisionar hubs y sucursales.</p>

                    <div class="input-group">
                        <label for="name">Nombre Comercial</label>
                        <input type="text" id="name" placeholder="Ej: Hamburguesas del Centro" bind:value={name} required />
                    </div>

                    <div class="input-group">
                        <label for="slug">Slug único de URL</label>
                        <input type="text" id="slug" placeholder="Ej: hamburguesas-centro" bind:value={slug} required />
                        <span class="hint">Este slug identificará tus sucursales y webhooks de POS.</span>
                    </div>

                    <button type="submit" class="btn-primary" disabled={loading}>
                        Registrar Empresa y Continuar
                    </button>
                </form>
            {/if}
        {/if}
    </div>
</div>

<style>
    .onboarding-page {
        display: flex;
        align-items: center;
        justify-content: center;
        min-height: 100vh;
        padding: 2rem;
        background: radial-gradient(circle at 15% 50%, rgba(59, 130, 246, 0.06), transparent 30%),
                    radial-gradient(circle at 85% 30%, rgba(16, 185, 129, 0.06), transparent 30%);
    }

    .glass-panel {
        background: var(--panel-bg);
        border: var(--panel-border);
        border-radius: 16px;
        box-shadow: var(--panel-shadow);
        backdrop-filter: var(--panel-blur);
        width: 100%;
        max-width: 650px;
        padding: 3rem;
        transition: all 0.3s ease;
    }

    .header {
        text-align: center;
        margin-bottom: 2.5rem;
    }

    .logo {
        font-size: 2.5rem;
        font-weight: 700;
        letter-spacing: -1px;
    }

    .logo span {
        color: var(--accent-color);
    }

    .subtitle {
        color: var(--text-muted);
        margin-top: 0.5rem;
    }

    .form-container h2 {
        font-size: 1.5rem;
        margin-bottom: 0.5rem;
    }

    .description {
        color: var(--text-muted);
        font-size: 0.9rem;
        margin-bottom: 2rem;
    }

    .input-group {
        display: flex;
        flex-direction: column;
        gap: 0.5rem;
        margin-bottom: 1.5rem;
    }

    .input-group label {
        font-size: 0.85rem;
        font-weight: 500;
        color: var(--text-muted);
    }

    .input-group input, .input-group select {
        background: var(--input-bg);
        border: var(--input-border);
        border-radius: 8px;
        padding: 0.8rem;
        color: var(--input-color);
        font-family: inherit;
        font-size: 1rem;
        outline: none;
        transition: border 0.2s;
    }

    .input-group input:focus, .input-group select:focus {
        border-color: var(--accent-color);
    }

    .hint {
        font-size: 0.75rem;
        color: var(--text-muted);
    }

    .btn-primary {
        background: var(--primary-btn-bg);
        border: none;
        border-radius: 8px;
        color: var(--primary-btn-color);
        cursor: pointer;
        font-size: 1rem;
        font-weight: 600;
        padding: 1rem;
        width: 100%;
        margin-top: 1rem;
        transition: all 0.2s ease;
    }

    .btn-primary:hover {
        background: var(--primary-btn-hover-bg);
        transform: var(--primary-btn-hover-transform);
        box-shadow: var(--primary-btn-hover-shadow);
    }

    .btn-primary:disabled {
        opacity: 0.5;
        cursor: not-allowed;
    }

    .error-alert {
        background: rgba(239, 68, 68, 0.1);
        border: 1px solid rgba(239, 68, 68, 0.2);
        color: #f87171;
        padding: 1rem;
        border-radius: 8px;
        margin-bottom: 1.5rem;
        font-size: 0.9rem;
    }

    .status-box {
        text-align: center;
        padding: 2rem 0;
        color: var(--text-muted);
    }

    .spinner {
        border: 3px solid rgba(255, 255, 255, 0.05);
        border-left-color: var(--accent-color);
        border-radius: 50%;
        width: 32px;
        height: 32px;
        animation: spin 1s linear infinite;
        margin: 0 auto 1rem auto;
    }

    @keyframes spin {
        0% { transform: rotate(0deg); }
        100% { transform: rotate(360deg); }
    }
</style>
