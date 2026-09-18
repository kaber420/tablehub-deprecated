<script>
  import { X } from '@lucide/svelte';
  import { fade, fly } from 'svelte/transition';

  export let show = false;
  export let title = '';
  
  function closeDrawer() {
    show = false;
  }
</script>

{#if show}
  <div class="drawer-backdrop" transition:fade={{ duration: 200 }} on:click={closeDrawer}></div>
  <div class="drawer panel" transition:fly={{ x: 400, duration: 300, opacity: 1 }}>
    <div class="drawer-header">
      <h2>{title}</h2>
      <button class="close-btn" on:click={closeDrawer}>
        <X size={24} />
      </button>
    </div>
    <div class="drawer-content">
      <slot></slot>
    </div>
  </div>
{/if}

<style>
  .drawer-backdrop {
    position: fixed;
    top: 0; left: 0; right: 0; bottom: 0;
    background: transparent;
    z-index: 1000;
  }

  .drawer {
    position: fixed;
    top: 0;
    right: 0;
    bottom: 0;
    width: 100%;
    max-width: 450px;
    z-index: 1010;
    border-radius: 20px 0 0 20px;
    border-right: none;
    display: flex;
    flex-direction: column;
    padding: 0;
    overflow: hidden;
  }

  .drawer-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 1.5rem 2rem;
    border-bottom: 1px solid rgba(255,255,255,0.05);
  }

  .drawer-header h2 {
    margin: 0;
    font-size: 1.5rem;
  }

  .close-btn {
    background: transparent;
    border: none;
    color: var(--text-muted);
    cursor: pointer;
    padding: 0.5rem;
    border-radius: 50%;
    display: flex;
    align-items: center;
    justify-content: center;
    transition: all 0.2s;
  }

  .close-btn:hover {
    background: rgba(255,255,255,0.1);
    color: var(--text-color);
  }

  .drawer-content {
    padding: 2rem;
    overflow-y: auto;
    flex-grow: 1;
  }
</style>
