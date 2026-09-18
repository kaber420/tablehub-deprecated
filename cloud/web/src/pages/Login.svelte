<script>
    import { login, authProvider } from "../lib/auth.js";
    import PrimaryButton from "../lib/components/PrimaryButton.svelte";
</script>

<div class="auth-container">
    <div class="auth-card">
        <h2 style="color: var(--accent-color);">Tablehub Cloud</h2>
        <p>Acceso seguro para administradores de franquicias.</p>
        
        <div style="margin-top: 2rem;">
            <PrimaryButton style="width: 100%;" on:click={login}>
                Iniciar Sesión
            </PrimaryButton>
        </div>
    </div>
</div>

<style>
  .auth-container {
    display: flex;
    justify-content: center;
    align-items: center;
    min-height: 100vh;
    padding: 20px;
    background: var(--bg-color);
  }
  
  .auth-card {
    background: rgba(255, 255, 255, 0.05);
    backdrop-filter: blur(10px);
    padding: 2.5rem;
    border-radius: 16px;
    border: 1px solid rgba(255, 255, 255, 0.1);
    box-shadow: 0 8px 32px rgba(0, 0, 0, 0.2);
    width: 100%;
    max-width: 400px;
    text-align: center;
  }
  
  h2 {
    margin-top: 0;
    margin-bottom: 0.5rem;
  }
  
  p {
    font-size: 0.9rem;
    color: var(--text-muted);
    margin-bottom: 2rem;
  }
</style>
