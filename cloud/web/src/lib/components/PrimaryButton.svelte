<script>
  export let type = 'button';
  export let disabled = false;
  export let id = '';
</script>

<button {type} {id} class="save-btn" {disabled} on:click>
  <slot></slot>
</button>

<style>
  .save-btn {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 0.5rem;
    padding: 0.75rem 1.5rem;
    background: var(--primary-btn-bg);
    color: var(--primary-btn-color);
    box-shadow: var(--primary-btn-shadow);
    border: none;
    border-radius: 8px;
    font-weight: 600;
    cursor: pointer;
    transition: all 0.2s;
    min-width: fit-content;
  }
  
  .save-btn:hover:not(:disabled) {
    background: var(--primary-btn-hover-bg);
    transform: var(--primary-btn-hover-transform);
    box-shadow: var(--primary-btn-hover-shadow);
  }

  .save-btn:active:not(:disabled) {
    box-shadow: var(--primary-btn-active-shadow);
    transform: translateY(1px);
  }
  
  .save-btn:disabled {
    opacity: 0.7;
    cursor: not-allowed;
  }
</style>
