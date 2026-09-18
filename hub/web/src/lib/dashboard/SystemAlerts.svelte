<script>
  import { 
    CheckCircle, 
    AlertTriangle, 
    WifiOff, 
    Battery, 
    BatteryWarning, 
    ShieldAlert, 
    Check 
  } from '@lucide/svelte';
  import { devicesState } from '../devices.svelte.js';

  let { onApprove } = $props();
</script>

<section class="panel">
  <h2>Alertas del Sistema</h2>
  
  <div class="alerts-container">
    {#if devicesState.deviceAlerts.length === 0}
      <div class="panel-inset empty-alerts">
        <CheckCircle size={32} class="text-success" />
        <h3>Sistemas Operativos</h3>
        <p>Todos los dispositivos funcionan normalmente. Sin alertas pendientes.</p>
      </div>
    {:else}
      <div class="alerts-list">
        {#each devicesState.deviceAlerts as alert (alert.mac + alert.type)}
          <div class="panel-inset alert-card border-{alert.severity}">
            <div class="alert-header-row">
              <div class="alert-title-info">
                {#if alert.type === 'unprovisioned'}
                  <AlertTriangle size={18} class="text-info" />
                {:else if alert.type === 'offline'}
                  <WifiOff size={18} class="text-danger" />
                {:else if alert.type === 'battery_critical'}
                  <Battery size={18} class="text-danger" />
                {:else if alert.type === 'battery_warning'}
                  <BatteryWarning size={18} class="text-warning" />
                {:else}
                  <ShieldAlert size={18} class="text-warning" />
                {/if}
                <div>
                  <h4 class="alert-name">{alert.name}</h4>
                  <p class="alert-msg">{alert.message}</p>
                </div>
              </div>
              
              {#if alert.type === 'unprovisioned'}
                <button class="neo-btn" on:click={() => onApprove(alert.device)}>
                  <Check size={14} style="margin-right: 4px;" /> Aprobar
                </button>
              {/if}
            </div>
            <div class="alert-footer">
              <span class="alert-desc">{alert.desc}</span>
              <span class="alert-mac monospace">{alert.mac}</span>
            </div>
          </div>
        {/each}
      </div>
    {/if}
  </div>
</section>
