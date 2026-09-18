<script>
  import { Database, Cpu, Server, Activity, Wifi, Key } from '@lucide/svelte';
  import { wsState } from '../websocket.svelte.js';
  
  let { statusData } = $props();
</script>

<section class="panel">
  <div class="panel-header-row">
    <h2>Server Status</h2>
    <div class="status-indicator">
      <span class="tag {statusData.status === 'online' ? 'online' : 'offline'}">
        {statusData.status.toUpperCase()}
      </span>
    </div>
  </div>
  
  <div class="console-metrics">
    <!-- Database Inset Card -->
    <div class="panel-inset metric-item">
      <div class="metric-icon-title">
        <Database size={18} class="text-success" />
        <span>Base de Datos:</span>
      </div>
      <div class="metric-value-led">
        <span class="value {statusData.db_initialized ? 'text-success' : 'text-danger'}">
          {statusData.db_initialized ? 'Initialized' : 'Error'}
        </span>
        <span class="led-indicator {statusData.db_initialized ? 'led-online' : 'led-offline'}"></span>
      </div>
    </div>
    
    <!-- NATS Broker Inset Card -->
    <div class="panel-inset metric-item">
      <div class="metric-icon-title">
        <Cpu size={18} class="text-success" />
        <span>Broker NATS:</span>
      </div>
      <div class="metric-value-led">
        <span class="value">{statusData.nats_running ? 'RUNNING' : 'OFFLINE'}</span>
        <span class="led-indicator {statusData.nats_running ? 'led-online' : 'led-offline'}"></span>
      </div>
    </div>

    <!-- JetStream Inset Card -->
    <div class="panel-inset metric-item">
      <div class="metric-icon-title">
        <Server size={18} class="text-success" />
        <span>JetStream Client:</span>
      </div>
      <div class="metric-value-led">
        <span class="value">{statusData.nats_client ? statusData.nats_client.toUpperCase() : 'UNKNOWN'}</span>
        <span class="led-indicator {statusData.nats_client === 'connected' ? 'led-online' : 'led-offline'}"></span>
      </div>
    </div>

    <!-- MQTT Port Inset Card -->
    {#if statusData.mqtt_port > 0}
      <div class="panel-inset metric-item">
        <div class="metric-icon-title">
          <Activity size={18} class="text-success" />
          <span>Puerto MQTT:</span>
        </div>
        <div class="metric-value-led">
          <span class="value monospace">{statusData.mqtt_port}</span>
          <span class="led-indicator led-online"></span>
        </div>
      </div>
    {/if}
    
    <!-- WebSocket Inset Card -->
    <div class="panel-inset metric-item">
      <div class="metric-icon-title">
        <Wifi size={18} class="text-success" />
        <span>Live Panel WS:</span>
      </div>
      <div class="metric-value-led">
        <span class="value">{wsState.connectionStatus.toUpperCase()}</span>
        <span class="led-indicator {wsState.connectionStatus === 'online' ? 'led-online' : 'led-offline'}"></span>
      </div>
    </div>

    <!-- Public Key Inset Card -->
    <div class="panel-inset metric-item key-metric">
      <div class="metric-icon-title">
        <Key size={18} class="text-success" />
        <span>Clave Pública (Ed25519):</span>
      </div>
      <span class="value monospace key-val">
        {statusData.ed25519_publickey ? statusData.ed25519_publickey : 'Not Generated'}
      </span>
    </div>
  </div>
</section>
