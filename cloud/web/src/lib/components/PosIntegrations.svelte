<script>
  import { onMount } from 'svelte';
  import { Check, Copy, Link, Activity } from '@lucide/svelte';
  import PrimaryButton from './PrimaryButton.svelte';

  export let profile;

  // Simulate supported POS integrations
  const supportedPOS = [
    {
      id: 'tastyigniter',
      name: 'TastyIgniter',
      logo: 'https://tastyigniter.com/assets/img/logo.svg',
      description: 'Conecta tu restaurante con TastyIgniter para sincronizar pedidos automáticamente.'
    }
  ];

  let selectedPOS = null;
  let webhookUrl = '';
  let secretToken = '';
  let showCopiedUrl = false;
  let showCopiedToken = false;
  
  // Connection status states: 'disconnected', 'waiting', 'connected'
  let connectionStatus = 'disconnected'; 

  function selectPOS(pos) {
    selectedPOS = pos;
    const apiBase = window.location.origin.includes('localhost') 
      ? 'http://localhost:8080/v1' 
      : 'https://api.tablehub.io/v1';
    
    webhookUrl = `${apiBase}/webhooks/${pos.id}/${profile.organization_id}`;
    secretToken = `th_sk_${profile.organization_id}_${Math.random().toString(36).substring(2, 15)}`;
    connectionStatus = 'waiting';
  }

  function goBack() {
    selectedPOS = null;
    connectionStatus = 'disconnected';
  }

  function copyToClipboard(text, isUrl) {
    navigator.clipboard.writeText(text).then(() => {
      if (isUrl) {
        showCopiedUrl = true;
        setTimeout(() => showCopiedUrl = false, 2000);
      } else {
        showCopiedToken = true;
        setTimeout(() => showCopiedToken = false, 2000);
      }
    });
  }

</script>

<div class="pos-integrations">
  {#if !selectedPOS}
    <div class="grid-header">
      <h2>Sistemas POS Soportados</h2>
      <p>Selecciona tu sistema de punto de venta para conectarlo con Tablehub y sincronizar pedidos a tu panel SaaS.</p>
    </div>
    <div class="pos-grid">
      {#each supportedPOS as pos}
        <div class="pos-card" on:click={() => selectPOS(pos)}>
          <div class="pos-logo-placeholder">
            {pos.name.substring(0, 2).toUpperCase()}
          </div>
          <h3>{pos.name}</h3>
          <p>{pos.description}</p>
          <PrimaryButton>Conectar</PrimaryButton>
        </div>
      {/each}
    </div>
  {:else}
    <div class="integration-details">
      <div class="details-header">
        <button class="back-btn" on:click={goBack}>← Volver</button>
        <h2>Configurar Integración con {selectedPOS.name}</h2>
      </div>

      <div class="instructions-card">
        <div class="step">
          <h3>Paso 1: Copia tu información de conexión</h3>
          <p>Usa estas credenciales para autenticar tu POS con Tablehub.</p>
          
          <div class="credential-field">
            <label for="webhookUrl">Webhook URL:</label>
            <div class="input-group">
              <input type="text" id="webhookUrl" value={webhookUrl} readonly />
              <button class="icon-btn" on:click={() => copyToClipboard(webhookUrl, true)} title="Copiar URL">
                {#if showCopiedUrl}
                  <Check size={18} class="success-icon" />
                {:else}
                  <Copy size={18} />
                {/if}
              </button>
            </div>
            <small>Esta es la dirección a la que {selectedPOS.name} enviará los datos.</small>
          </div>

          <div class="credential-field">
            <label for="secretToken">Secret Token (API Key):</label>
            <div class="input-group">
              <input type="text" id="secretToken" value={secretToken} readonly />
              <button class="icon-btn" on:click={() => copyToClipboard(secretToken, false)} title="Copiar Token">
                {#if showCopiedToken}
                  <Check size={18} class="success-icon" />
                {:else}
                  <Copy size={18} />
                {/if}
              </button>
            </div>
            <div class="alert warning-alert">
              <strong>Importante:</strong> Este Secret Token solo se muestra una vez por motivos de seguridad. 
            </div>
          </div>
        </div>

        <div class="divider"></div>

        <div class="step">
          <h3>Paso 2: Configura tu POS</h3>
          <div class="guide-box">
            <ol>
              <li>Inicia sesión en el panel de administrador de tu <strong>{selectedPOS.name}</strong>.</li>
              <li>Ve a <strong>System</strong> > <strong>Webhooks</strong>.</li>
              <li>Haz clic en <strong>New Webhook</strong>.</li>
              <li>En el campo <em>Payload URL</em>, pega la <strong>URL de Webhook</strong> que copiaste arriba.</li>
              <li>En el campo <em>Secret</em>, pega el <strong>Secret Token</strong> que copiaste arriba.</li>
              <li>En <em>Events</em>, selecciona <code>order.created</code> o <code>order.updated</code>.</li>
              <li>Guarda los cambios.</li>
            </ol>
          </div>
        </div>
      </div>

      <div class="status-card">
        <div class="status-indicator">
          {#if connectionStatus === 'waiting'}
            <div class="pulse-dot orange"></div>
            <span><strong>Estado:</strong> Esperando primer evento...</span>
          {:else if connectionStatus === 'connected'}
            <div class="pulse-dot green"></div>
            <span><strong>Estado:</strong> 🟢 Activo y Conectado</span>
          {/if}
        </div>
        <div class="status-actions">
          <button class="test-btn">Probar Conexión</button>
        </div>
      </div>
    </div>
  {/if}
</div>

<style>
  .pos-integrations {
    display: flex;
    flex-direction: column;
    gap: 1.5rem;
  }

  .grid-header h2 {
    font-size: 1.5rem;
    margin-bottom: 0.5rem;
  }
  
  .grid-header p {
    color: var(--text-muted);
  }

  .pos-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(250px, 1fr));
    gap: 1.5rem;
  }

  .pos-card {
    background: rgba(255, 255, 255, 0.03);
    border: 1px solid rgba(255, 255, 255, 0.1);
    border-radius: 12px;
    padding: 1.5rem;
    display: flex;
    flex-direction: column;
    align-items: center;
    text-align: center;
    gap: 1rem;
    cursor: pointer;
    transition: all 0.2s;
  }

  .pos-card:hover {
    background: rgba(255, 255, 255, 0.05);
    border-color: var(--accent-color);
    transform: translateY(-2px);
  }

  .pos-logo-placeholder {
    width: 80px;
    height: 80px;
    background: rgba(0, 0, 0, 0.2);
    border-radius: 16px;
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 2rem;
    font-weight: bold;
    color: var(--accent-color);
  }

  .pos-card p {
    font-size: 0.9rem;
    color: var(--text-muted);
    flex-grow: 1;
  }

  .details-header {
    display: flex;
    align-items: center;
    gap: 1rem;
    margin-bottom: 1.5rem;
  }

  .back-btn {
    background: transparent;
    border: none;
    color: var(--text-muted);
    cursor: pointer;
    font-weight: 500;
    padding: 0.5rem;
  }

  .back-btn:hover {
    color: var(--text-color);
  }

  .instructions-card {
    background: rgba(0, 0, 0, 0.2);
    border: 1px solid rgba(255, 255, 255, 0.1);
    border-radius: 12px;
    padding: 2rem;
    display: flex;
    flex-direction: column;
    gap: 2rem;
  }

  .step h3 {
    margin-bottom: 0.5rem;
    color: var(--accent-color);
  }

  .step p {
    color: var(--text-muted);
    margin-bottom: 1.5rem;
  }

  .credential-field {
    margin-bottom: 1.5rem;
  }

  .credential-field label {
    display: block;
    font-weight: 500;
    margin-bottom: 0.5rem;
  }

  .input-group {
    display: flex;
    gap: 0.5rem;
    margin-bottom: 0.5rem;
  }

  .input-group input {
    flex: 1;
    background: var(--input-bg);
    border: var(--input-border);
    color: var(--text-color);
    padding: 0.75rem;
    border-radius: 8px;
    font-family: monospace;
    font-size: 0.95rem;
  }

  .icon-btn {
    background: var(--button-bg, rgba(255,255,255,0.1));
    border: 1px solid rgba(255,255,255,0.2);
    color: var(--text-color);
    padding: 0 1rem;
    border-radius: 8px;
    cursor: pointer;
    display: flex;
    align-items: center;
    justify-content: center;
    transition: all 0.2s;
  }

  .icon-btn:hover {
    background: rgba(255, 255, 255, 0.15);
  }

  .success-icon {
    color: #10b981;
  }

  .credential-field small {
    color: var(--text-muted);
  }

  .alert {
    padding: 0.75rem;
    border-radius: 8px;
    font-size: 0.9rem;
    margin-top: 0.5rem;
  }

  .warning-alert {
    background: rgba(245, 158, 11, 0.1);
    border: 1px solid rgba(245, 158, 11, 0.2);
    color: #fbbf24;
  }

  .divider {
    height: 1px;
    background: rgba(255, 255, 255, 0.1);
    width: 100%;
  }

  .guide-box {
    background: rgba(0, 0, 0, 0.3);
    border-radius: 8px;
    padding: 1.5rem;
  }

  .guide-box ol {
    margin: 0;
    padding-left: 1.5rem;
    color: var(--text-color);
  }

  .guide-box li {
    margin-bottom: 0.5rem;
    line-height: 1.5;
  }
  
  .guide-box code {
    background: rgba(255, 255, 255, 0.1);
    padding: 0.2rem 0.4rem;
    border-radius: 4px;
    font-family: monospace;
    font-size: 0.9em;
  }

  .status-card {
    display: flex;
    justify-content: space-between;
    align-items: center;
    background: rgba(255, 255, 255, 0.03);
    border: 1px solid rgba(255, 255, 255, 0.1);
    border-radius: 12px;
    padding: 1.5rem;
    margin-top: 0.5rem;
  }

  .status-indicator {
    display: flex;
    align-items: center;
    gap: 0.75rem;
  }

  .pulse-dot {
    width: 12px;
    height: 12px;
    border-radius: 50%;
  }
  
  .pulse-dot.orange {
    background-color: #fbbf24;
    box-shadow: 0 0 0 0 rgba(251, 191, 36, 0.7);
    animation: pulse-orange 2s infinite;
  }

  .pulse-dot.green {
    background-color: #10b981;
    box-shadow: 0 0 0 0 rgba(16, 185, 129, 0.7);
    animation: pulse-green 2s infinite;
  }

  @keyframes pulse-orange {
    0% { transform: scale(0.95); box-shadow: 0 0 0 0 rgba(251, 191, 36, 0.7); }
    70% { transform: scale(1); box-shadow: 0 0 0 10px rgba(251, 191, 36, 0); }
    100% { transform: scale(0.95); box-shadow: 0 0 0 0 rgba(251, 191, 36, 0); }
  }

  @keyframes pulse-green {
    0% { transform: scale(0.95); box-shadow: 0 0 0 0 rgba(16, 185, 129, 0.7); }
    70% { transform: scale(1); box-shadow: 0 0 0 10px rgba(16, 185, 129, 0); }
    100% { transform: scale(0.95); box-shadow: 0 0 0 0 rgba(16, 185, 129, 0); }
  }

  .test-btn {
    background: transparent;
    border: 1px solid rgba(255, 255, 255, 0.2);
    color: var(--text-color);
    padding: 0.5rem 1rem;
    border-radius: 6px;
    cursor: pointer;
    transition: all 0.2s;
  }

  .test-btn:hover {
    background: rgba(255, 255, 255, 0.1);
  }
</style>
