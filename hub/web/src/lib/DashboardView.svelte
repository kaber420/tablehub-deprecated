<script>
  import { onMount, onDestroy } from 'svelte';
  import { apiFetch } from './api.js';
  import { devicesState } from './devices.svelte.js';
  import { wsState } from './websocket.svelte.js';
  
  import KpiCards from './dashboard/KpiCards.svelte';
  import ServerStatus from './dashboard/ServerStatus.svelte';
  import SystemAlerts from './dashboard/SystemAlerts.svelte';
  import LiveEventsTunnel from './dashboard/LiveEventsTunnel.svelte';
  import ApprovalModal from './dashboard/ApprovalModal.svelte';

  let statusData = $state({
    status: 'offline',
    db_initialized: false,
    ed25519_publickey: '',
    nats_running: false,
    nats_client: 'not_initialized',
    mqtt_port: 0,
    nats_port: 0
  });

  let showModal = $state(false);
  let pendingDevice = $state(null);

  onMount(async () => {
    await fetchServerStatus();
    await devicesState.loadDevices();
    await devicesState.loadUsers();
    
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const host = window.location.host;
    const wsUrl = import.meta.env.DEV ? `ws://localhost:8080/ws/local` : `${protocol}//${host}/ws/local`;
    
    wsState.init(wsUrl);
  });

  onDestroy(() => {
    wsState.close();
  });

  async function fetchServerStatus() {
    try {
      const res = await apiFetch('/status');
      if (res.ok) statusData = await res.json();
    } catch (e) {
      statusData.status = 'error';
    }
  }

  async function handleLogout() {
    try {
      await apiFetch('/api/logout', { method: 'POST' });
      window.dispatchEvent(new CustomEvent('auth-expired'));
    } catch (e) {
      console.error('Error logging out', e);
    }
  }

  function promptApprove(device) {
    pendingDevice = device;
    showModal = true;
  }

  async function handleApprove(device, locationOrTable) {
    let tableNum = null;
    let locName = null;
    
    if (device.device_type === 'table_pad') {
      tableNum = locationOrTable;
    } else {
      locName = locationOrTable;
    }

    try {
      const res = await apiFetch('/api/devices/approve', {
        method: 'POST',
        body: JSON.stringify({
          mac: device.mac_address,
          device_type: device.device_type || 'table_pad',
          table_number: tableNum,
          location_name: locName,
          public_key: device.public_key
        })
      });
      if (res.ok) {
        await devicesState.loadDevices();
      } else {
        alert('Error al aprobar el dispositivo');
      }
    } catch (e) {
      alert('Error de conexión');
    }
  }
</script>

<div class="view-header">
  <div class="header-content">
    <h1>Maitre Hub</h1>
    <button class="neo-btn neo-btn-danger" on:click={handleLogout}>Cerrar Sesión</button>
  </div>
</div>

<KpiCards />

<div class="grid">
  <ServerStatus {statusData} />
  
  <SystemAlerts onApprove={promptApprove} />
  
  <LiveEventsTunnel />
</div>

<ApprovalModal bind:show={showModal} device={pendingDevice} onApprove={handleApprove} />
