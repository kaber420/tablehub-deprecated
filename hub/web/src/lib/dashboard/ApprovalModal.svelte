<script>
  import { fade, scale } from 'svelte/transition';
  
  let { show = $bindable(false), device = null, onApprove } = $props();
  
  let inputValue = $state('');
  
  function handleApprove() {
    if (!inputValue.trim()) return;
    onApprove(device, inputValue);
    show = false;
    inputValue = '';
  }
  
  function handleClose() {
    show = false;
    inputValue = '';
  }
</script>

{#if show && device}
  <div class="modal-backdrop" transition:fade={{ duration: 150 }} on:click={handleClose}>
    <div class="modal-content panel" transition:scale={{ duration: 200, start: 0.95 }} on:click|stopPropagation>
      <h2>Aprobar Dispositivo</h2>
      <p class="monospace">{device.mac_address}</p>
      
      <div class="input-group" style="margin-top: 1rem;">
        <label>
          {#if device.device_type === 'table_pad'}
            Asignar Número de Mesa:
          {:else}
            Asignar Ubicación (ej: Cocina):
          {/if}
        </label>
        <input type="text" class="neo-input" bind:value={inputValue} autofocus on:keydown={(e) => e.key === 'Enter' && handleApprove()} />
      </div>
      
      <div class="modal-actions" style="margin-top: 1.5rem; display: flex; gap: 1rem; justify-content: flex-end;">
        <button class="neo-btn" on:click={handleClose}>Cancelar</button>
        <button class="neo-btn neo-btn-primary" on:click={handleApprove}>Aprobar</button>
      </div>
    </div>
  </div>
{/if}

<style>
  .modal-backdrop {
    position: fixed;
    top: 0;
    left: 0;
    width: 100vw;
    height: 100vh;
    background: rgba(0, 0, 0, 0.5);
    backdrop-filter: blur(4px);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 1000;
  }
  .modal-content {
    min-width: 350px;
    max-width: 90vw;
  }
  .modal-actions {
    display: flex;
    gap: 1rem;
    justify-content: flex-end;
  }
</style>
