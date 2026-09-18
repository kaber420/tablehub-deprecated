<script>
    import { onMount } from "svelte";
    import { userManager } from "../lib/auth.js";

    onMount(async () => {
        try {
            await userManager.signinRedirectCallback();
            window.location.replace("/");
        } catch (e) {
            console.error("Error validando el token de Zitadel", e);
            window.location.replace("/");
        }
    });
</script>

<div style="display: flex; justify-content: center; align-items: center; height: 100vh; flex-direction: column;">
    <h3 style="color: var(--accent-color); margin-bottom: 1rem;">Verificando Identidad...</h3>
    <p>Conectando con Zitadel Security</p>
</div>
