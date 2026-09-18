<script>
  import { apiFetch } from './api.js';
  import { createEventDispatcher } from 'svelte';
  
  const dispatch = createEventDispatcher();
  
  let username = '';
  let password = '';
  let confirmPassword = '';
  let error = '';
  let loading = false;

  async function submitSetup() {
    error = '';
    if (!username.trim()) {
      error = 'El nombre de usuario es obligatorio.';
      return;
    }
    if (password.length < 8) {
      error = 'La contraseña debe tener al menos 8 caracteres.';
      return;
    }
    if (password !== confirmPassword) {
      error = 'Las contraseñas no coinciden.';
      return;
    }

    loading = true;
    try {
      const res = await apiFetch('/api/setup', {
        method: 'POST',
        body: JSON.stringify({ username, password })
      });
      
      if (!res.ok) {
        throw new Error('Error al configurar la contraseña');
      }
      
      // Notificar al componente padre que el setup terminó con éxito
      dispatch('success');
    } catch (err) {
      error = err.message;
    } finally {
      loading = false;
    }
  }
</script>

<div class="auth-container">
  <div class="auth-card">
    <h2>Bienvenido a Tablehub</h2>
    <p>Este es el primer inicio. Por favor, crea una contraseña de administrador segura. Esta acción solo puede realizarse una vez.</p>
    
    <form on:submit|preventDefault={submitSetup}>
      <div class="input-group">
        <label for="username">Nombre de Usuario</label>
        <input type="text" id="username" bind:value={username} required autofocus autocomplete="username" />
      </div>

      <div class="input-group">
        <label for="password">Nueva Contraseña</label>
        <input type="password" id="password" bind:value={password} required autocomplete="new-password" />
      </div>
      
      <div class="input-group">
        <label for="confirmPassword">Confirmar Contraseña</label>
        <input type="password" id="confirmPassword" bind:value={confirmPassword} required autocomplete="new-password" />
      </div>
      
      {#if error}
        <div class="error-msg">{error}</div>
      {/if}
      
      <button type="submit" disabled={loading}>
        {loading ? 'Guardando...' : 'Completar Setup'}
      </button>
    </form>
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
    max-width: 450px;
    text-align: center;
  }
  
  h2 {
    margin-top: 0;
    margin-bottom: 1rem;
  }
  
  p {
    font-size: 0.9rem;
    color: var(--text-muted);
    margin-bottom: 2rem;
  }
  
  .input-group {
    margin-bottom: 1.5rem;
    text-align: left;
  }
  
  label {
    display: block;
    margin-bottom: 0.5rem;
    font-size: 0.85rem;
    font-weight: 600;
  }
  
  input {
    width: 100%;
    padding: 0.8rem;
    border-radius: 8px;
    border: 1px solid rgba(255,255,255,0.2);
    background: rgba(0,0,0,0.1);
    color: inherit;
    font-family: inherit;
  }
  
  button {
    width: 100%;
    padding: 0.9rem;
    border-radius: 8px;
    border: none;
    background: var(--accent-color);
    color: white;
    font-weight: 600;
    cursor: pointer;
    transition: transform 0.1s, opacity 0.2s;
  }
  
  button:hover:not(:disabled) {
    transform: translateY(-2px);
  }
  
  button:disabled {
    opacity: 0.7;
    cursor: not-allowed;
  }
  
  .error-msg {
    color: #ef4444;
    font-size: 0.85rem;
    margin-bottom: 1rem;
    padding: 0.5rem;
    background: rgba(239, 68, 68, 0.1);
    border-radius: 4px;
  }
</style>
