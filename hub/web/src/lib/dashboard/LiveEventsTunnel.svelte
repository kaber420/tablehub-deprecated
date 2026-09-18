<script>
  import { slide } from 'svelte/transition';
  import { 
    Trash2, 
    Utensils, 
    ShoppingCart, 
    Activity, 
    Smartphone, 
    Battery, 
    Wifi 
  } from '@lucide/svelte';
  import { wsState } from '../websocket.svelte.js';
</script>

<section class="panel full-width-column">
  <div class="panel-header-row">
    <h2>Live Events Tunnel</h2>
    <button class="neo-btn" on:click={wsState.clearTunnelLogs}>
      <Trash2 size={14} style="margin-right: 4px;" /> Limpiar Pantalla
    </button>
  </div>
  
  {#if wsState.events.length === 0}
    <div class="panel-inset empty-state">
      <Utensils size={24} />
      <p>Listening for cloud & local events...</p>
    </div>
  {:else}
    <div class="console-box">
      <div class="event-list">
        {#each wsState.events as ev (ev.timestamp || Math.random())}
          <div class="event-card neo-card" transition:slide|local>
            {#if ev.event === 'order.created'}
              <div class="event-card-header">
                <div class="event-icon-wrapper icon-order">
                  <ShoppingCart size={18} />
                </div>
                <div class="event-title">
                  <span class="event-type-badge badge-order">Nueva Orden</span>
                  <span class="event-time">{new Date(ev.timestamp ? ev.timestamp * 1000 : Date.now()).toLocaleTimeString()}</span>
                </div>
              </div>
              <div class="event-card-body">
                <div class="detail-row">
                  <span class="detail-label">Mesa</span>
                  <span class="detail-value">{ev.payload.order?.table_number || '—'}</span>
                </div>
                <div class="detail-row">
                  <span class="detail-label">Orden</span>
                  <span class="detail-value">#{ev.payload.order?.id || '—'}</span>
                </div>
                <div class="detail-row">
                  <span class="detail-label">Cliente</span>
                  <span class="detail-value">{ev.payload.order?.customer_name || '—'}</span>
                </div>
                <div class="detail-row">
                  <span class="detail-label">Total</span>
                  <span class="detail-value">${ev.payload.order?.total || '—'}</span>
                </div>
              </div>
              <details class="event-raw-details">
                <summary>Ver JSON original</summary>
                <pre class="raw-payload">{JSON.stringify(ev.payload, null, 2)}</pre>
              </details>
            {:else if ev.event === 'device.status_update'}
              <div class="event-card-header">
                <div class="event-icon-wrapper icon-telemetry">
                  <Activity size={18} />
                </div>
                <div class="event-title">
                  <span class="event-type-badge badge-telemetry">Telemetría</span>
                  <span class="event-time">{new Date(ev.timestamp ? ev.timestamp * 1000 : Date.now()).toLocaleTimeString()}</span>
                </div>
              </div>
              <div class="event-card-body">
                <div class="detail-row">
                  <span class="detail-label"><Smartphone size={14} /> MAC</span>
                  <span class="detail-value monospace">{ev.payload.mac || '—'}</span>
                </div>
                <div class="detail-row">
                  <span class="detail-label"><Battery size={14} /> Batería</span>
                  <span class="detail-value">{ev.payload.battery !== undefined ? ev.payload.battery + '%' : '—'}</span>
                </div>
                <div class="detail-row">
                  <span class="detail-label"><Wifi size={14} /> Señal</span>
                  <span class="detail-value">{ev.payload.wifi !== undefined ? ev.payload.wifi + ' dBm' : '—'}</span>
                </div>
              </div>
              <details class="event-raw-details">
                <summary>Ver JSON original</summary>
                <pre class="raw-payload">{JSON.stringify(ev.payload, null, 2)}</pre>
              </details>
            {:else if ev.event === 'device.provision_request'}
              <div class="event-card-header">
                <div class="event-icon-wrapper icon-provision">
                  <Smartphone size={18} />
                </div>
                <div class="event-title">
                  <span class="event-type-badge badge-provision">Provision</span>
                  <span class="event-time">{new Date(ev.timestamp ? ev.timestamp * 1000 : Date.now()).toLocaleTimeString()}</span>
                </div>
              </div>
              <div class="event-card-body">
                <div class="detail-row">
                  <span class="detail-label">MAC</span>
                  <span class="detail-value monospace">{ev.payload.mac || '—'}</span>
                </div>
                <div class="detail-row">
                  <span class="detail-label">Tipo</span>
                  <span class="detail-value">{ev.payload.type || '—'}</span>
                </div>
                <div class="detail-row">
                  <span class="detail-label">IP</span>
                  <span class="detail-value monospace">{ev.payload.ip || '—'}</span>
                </div>
              </div>
              <details class="event-raw-details">
                <summary>Ver JSON original</summary>
                <pre class="raw-payload">{JSON.stringify(ev.payload, null, 2)}</pre>
              </details>
            {:else}
              <div class="event-card-header">
                <div class="event-icon-wrapper icon-generic">
                  <Activity size={18} />
                </div>
                <div class="event-title">
                  <span class="event-type-badge badge-generic">{ev.event || 'EVENT'}</span>
                  <span class="event-time">{new Date(ev.timestamp ? ev.timestamp * 1000 : Date.now()).toLocaleTimeString()}</span>
                </div>
              </div>
              <details class="event-raw-details" open>
                <summary>Ver JSON original</summary>
                <pre class="raw-payload">{JSON.stringify(ev.payload, null, 2)}</pre>
              </details>
            {/if}
          </div>
        {/each}
      </div>
    </div>
  {/if}
</section>
