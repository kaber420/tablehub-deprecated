<script>
  export let checked = false;
  export let id = '';
</script>

<label class="switch">
  <input type="checkbox" {id} bind:checked on:change>
  <span class="slider"></span>
</label>

<style>
  /* Toggle Switch Styles */
  .switch {
    position: relative;
    display: inline-block;
    width: 48px;
    height: 24px;
  }

  .switch input {
    opacity: 0;
    width: 0;
    height: 0;
  }

  .slider {
    position: absolute;
    cursor: pointer;
    top: 0;
    left: 0;
    right: 0;
    bottom: 0;
    background-color: var(--toggle-bg);
    box-shadow: var(--toggle-shadow);
    border: var(--toggle-border);
    transition: .3s;
    border-radius: 24px;
  }

  .slider:before {
    position: absolute;
    content: "";
    height: 18px;
    width: 18px;
    left: 2px;
    bottom: 2px;
    background-color: var(--toggle-thumb-bg);
    box-shadow: var(--toggle-thumb-shadow);
    transition: .3s;
    border-radius: 50%;
  }

  /* Adjust for border vs no border */
  :global(body.theme-glass) .slider:before {
    bottom: 2px;
    left: 2px;
  }
  
  :global(body.theme-neumorph) .slider:before {
    bottom: 3px;
    left: 3px;
  }

  input:checked + .slider {
    background-color: var(--button-active-bg);
    border-color: var(--accent-color);
  }
  
  :global(body.theme-neumorph) input:checked + .slider {
    box-shadow: inset 3px 3px 6px #121419, inset -3px -3px 6px #22242d, inset 0 0 8px rgba(59, 130, 246, 0.3);
  }

  input:focus + .slider {
    box-shadow: 0 0 1px var(--accent-color);
  }

  input:checked + .slider:before {
    transform: translateX(24px);
    background-color: var(--accent-color);
  }
  
  :global(body.theme-neumorph) input:checked + .slider:before {
    box-shadow: 2px 2px 5px #121419, -2px -2px 5px #22242d, 0 0 6px var(--accent-color);
  }
</style>
