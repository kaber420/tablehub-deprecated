<script>
  import { Monitor, Layout, Save } from '@lucide/svelte';
  import PrimaryButton from './components/PrimaryButton.svelte';
  
  export let theme = 'glass'; // 'glass' or 'neumorph'
  export let dockPosition = 'bottom'; // 'bottom', 'top', 'left', 'right'
  
  function saveSettings() {
    localStorage.setItem('hub-theme', theme);
    localStorage.setItem('hub-dock-pos', dockPosition);
    // Simple visual feedback
    const btn = document.getElementById('save-btn');
    const originalText = btn.innerHTML;
    btn.innerHTML = 'Saved!';
    setTimeout(() => btn.innerHTML = originalText, 2000);
  }
</script>

<div class="view-header">
  <h1>Appearance & Layout</h1>
</div>

<div class="settings-container">
  <section class="panel">
    <div class="setting-group">
      <div class="setting-header">
        <Monitor size={20} class="icon" />
        <h3>Visual Theme</h3>
      </div>
      
      <div class="options">
        <label class="option-card {theme === 'glass' ? 'selected' : ''}">
          <input type="radio" bind:group={theme} value="glass" name="theme">
          <div class="preview glass-preview"></div>
          <span>Glassmorphism</span>
        </label>
        
        <label class="option-card {theme === 'neumorph' ? 'selected' : ''}">
          <input type="radio" bind:group={theme} value="neumorph" name="theme">
          <div class="preview neumorph-preview"></div>
          <span>Neumorphism</span>
        </label>
      </div>
    </div>
  </section>

  <section class="panel">
    <div class="setting-group">
      <div class="setting-header">
        <Layout size={20} class="icon" />
        <h3>Dock Position</h3>
      </div>
      
      <div class="options dock-options">
        {#each ['bottom', 'top', 'left', 'right'] as pos}
          <label class="option-card {dockPosition === pos ? 'selected' : ''}">
            <input type="radio" bind:group={dockPosition} value={pos} name="dock">
            <span style="text-transform: capitalize;">{pos}</span>
          </label>
        {/each}
      </div>
    </div>
  </section>
  
  <div class="actions">
    <PrimaryButton id="save-btn" on:click={saveSettings}>
      <Save size={18} /> Save Preferences
    </PrimaryButton>
  </div>
</div>

<style>
  .view-header {
    text-align: center;
    margin-bottom: 3rem;
  }
  
  .view-header h1 {
    font-size: 2.5rem;
    letter-spacing: -0.02em;
    background: linear-gradient(135deg, #fff 0%, #a5b4fc 100%);
    -webkit-background-clip: text;
    -webkit-text-fill-color: transparent;
  }
  
  .settings-container {
    max-width: 800px;
    margin: 0 auto;
    display: flex;
    flex-direction: column;
    gap: 2rem;
  }
  
  .setting-header {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    margin-bottom: 0.5rem;
  }
  
  .setting-header :global(.icon) {
    color: var(--accent-color);
  }
  
  .description {
    margin-bottom: 1.5rem;
    font-size: 0.95rem;
  }
  
  .options {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 1rem;
  }
  
  .dock-options {
    grid-template-columns: repeat(4, 1fr);
  }
  
  @media (max-width: 600px) {
    .dock-options {
      grid-template-columns: 1fr 1fr;
    }
  }
  
  .option-card {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 1rem;
    padding: 1.5rem 1rem;
    border-radius: 12px;
    background: rgba(0,0,0,0.1);
    border: 2px solid transparent;
    cursor: pointer;
    transition: all 0.2s;
  }
  
  .option-card:hover {
    background: rgba(255,255,255,0.05);
  }
  
  .option-card.selected {
    border-color: var(--accent-color);
    background: var(--button-active-bg);
  }
  
  .option-card input {
    display: none;
  }
  
  .preview {
    width: 60px;
    height: 40px;
    border-radius: 8px;
  }
  
  .glass-preview {
    background: rgba(255, 255, 255, 0.1);
    border: 1px solid rgba(255, 255, 255, 0.2);
    box-shadow: 0 4px 12px rgba(0,0,0,0.1);
  }
  
  .neumorph-preview {
    background: #1a1c23;
    box-shadow: 3px 3px 6px #121419, -3px -3px 6px #22242d;
  }
  
  .actions {
    display: flex;
    justify-content: flex-end;
    margin-top: 1rem;
  }
</style>
