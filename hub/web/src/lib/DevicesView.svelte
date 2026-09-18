<script>
  import { onMount, onDestroy } from 'svelte';
  import { apiFetch, getWifiNetworks, saveWifiNetwork, deleteWifiNetwork } from './api.js';
  import { devicesState } from './devices.svelte.js';
  import { 
    Cpu, 
    Smartphone, 
    Check, 
    X, 
    Trash2, 
    ShieldAlert, 
    Wifi, 
    Battery, 
    Edit3,
    AlertCircle,
    Server,
    Settings,
    Info,
    CheckCircle,
    ChevronDown,
    Download,
    Plus,
    Save,
    Network
  } from '@lucide/svelte';

  let filter = 'all';
  let loading = true;
  let errorMsg = '';

  // Modal de Aprobación/Edición
  let showModal = false;
  let selectedDevice = null;
  let formType = 'table_pad';
  let formTableNumber = '';
  let formLocationName = '';
  let formWifiNetworkId = '';
  let formWifiSSID = '';
  let formWifiPass = '';

  // Modal Crear Mesa / Dispositivo
  let showCreateModal = false;
  let createTableNumber = '';
  let createLocationName = '';
  let createDeviceType = 'table_pad';
  let createError = '';
  let createLoading = false;

  async function handleCreateDevice() {
    if (!createTableNumber.trim()) {
      createError = 'El número/ID de mesa es obligatorio';
      return;
    }
    createError = '';
    createLoading = true;
    try {
      const res = await apiFetch('/api/devices/create', {
        method: 'POST',
        body: JSON.stringify({
          table_number: createTableNumber.trim(),
          location_name: createLocationName.trim(),
          device_type: createDeviceType
        })
      });
      if (res.ok) {
        showCreateModal = false;
        createTableNumber = '';
        createLocationName = '';
        devicesState.loadDevices(filter);
      } else {
        const text = await res.text();
        createError = text || 'Error al crear la mesa';
      }
    } catch (e) {
      createError = 'Error de conexión';
    } finally {
      createLoading = false;
    }
  }

  function openProvisionModalForDevice(device) {
    provTableId = device.table_number || device.mac_address || '';
    provFileUrl = null;
    provError = '';
    showProvisionModal = true;
  }

  // Modal Generador de Aprovisionamiento
  let showProvisionModal = false;
  let provTableId = '';
  let provWifiSSID = '';
  let provWifiPass = '';
  let provPIN = '';
  let provFileUrl = null;
  let provError = '';
  let provLoading = false;
  let provSaveNetwork = false;
  let provSelectedWifiId = '';

  // Redes Wi-Fi guardadas
  let wifiNetworks = [];
  let wifiLoading = false;
  let wifiError = '';

  // Modal de gestión de Wi-Fi
  let showWifiManager = false;
  let newWifiSSID = '';
  let newWifiPass = '';
  let wifiSaveLoading = false;
  let wifiManagerError = '';

  async function loadWifiNetworks() {
    try {
      wifiNetworks = await getWifiNetworks();
    } catch (e) {
      wifiError = 'Error al cargar redes Wi-Fi';
    }
  }

  async function generateProvisionFile() {
    provError = '';
    provFileUrl = null;
    provLoading = true;
    try {
      const body = {
        table_id: provTableId,
        pin: provPIN.toString()
      };

      if (provSelectedWifiId && provSelectedWifiId !== '__manual') {
        body.wifi_network_id = parseInt(provSelectedWifiId);
      } else {
        body.wifi_ssid = provWifiSSID;
        body.wifi_pass = provWifiPass;
      }

      const res = await apiFetch('/api/devices/provision_file', {
        method: 'POST',
        body: JSON.stringify(body)
      });
      
      if (res.ok) {
        if (provSaveNetwork && provSelectedWifiId === '__manual' && provWifiSSID && provWifiPass) {
          try {
            await saveWifiNetwork(provWifiSSID, provWifiPass);
            loadWifiNetworks();
          } catch (e) {
            // Silencioso — no bloquear la descarga
          }
        }
        const blob = await res.blob();
        provFileUrl = URL.createObjectURL(blob);
      } else {
        const text = await res.text();
        provError = text || 'Error al generar el archivo';
      }
    } catch (e) {
      provError = 'Error de conexión';
    } finally {
      provLoading = false;
    }
  }

  function onWifiSelect() {
    if (provSelectedWifiId && provSelectedWifiId !== '__manual') {
      const net = wifiNetworks.find(n => n.id === parseInt(provSelectedWifiId));
      if (net) {
        provWifiSSID = net.ssid;
      }
    } else if (provSelectedWifiId === '__manual') {
      provWifiSSID = '';
      provWifiPass = '';
    }
  }

  function openProvisionModal() {
    provTableId = '';
    provWifiSSID = '';
    provWifiPass = '';
    provPIN = '';
    provFileUrl = null;
    provError = '';
    provSaveNetwork = false;
    provSelectedWifiId = '';
    showProvisionModal = true;
    loadWifiNetworks();
  }

  function closeProvisionModal() {
    showProvisionModal = false;
    if (provFileUrl) {
      URL.revokeObjectURL(provFileUrl);
      provFileUrl = null;
    }
    provTableId = '';
    provWifiSSID = '';
    provWifiPass = '';
    provPIN = '';
    provError = '';
    provSaveNetwork = false;
    provSelectedWifiId = '';
  }

  // Gestión de Wi-Fi
  async function handleSaveWifi() {
    if (!newWifiSSID || !newWifiPass) {
      wifiManagerError = 'SSID y contraseña son obligatorios';
      return;
    }
    wifiSaveLoading = true;
    wifiManagerError = '';
    try {
      await saveWifiNetwork(newWifiSSID, newWifiPass);
      newWifiSSID = '';
      newWifiPass = '';
      await loadWifiNetworks();
    } catch (e) {
      wifiManagerError = 'Error al guardar la red Wi-Fi';
    } finally {
      wifiSaveLoading = false;
    }
  }

  async function handleDeleteWifi(id) {
    if (!confirm('¿Eliminar esta red Wi-Fi guardada?')) return;
    try {
      await deleteWifiNetwork(id);
      await loadWifiNetworks();
    } catch (e) {
      wifiManagerError = 'Error al eliminar la red Wi-Fi';
    }
  }

  function closeWifiManager() {
    showWifiManager = false;
    newWifiSSID = '';
    newWifiPass = '';
    wifiManagerError = '';
  }

  onMount(async () => {
    loading = true;
    errorMsg = '';
    try {
      await devicesState.loadDevices(filter);
    } catch (e) {
      errorMsg = 'Error al cargar dispositivos';
    } finally {
      loading = false;
    }
  });

  async function fetchDevices() {
    loading = true;
    errorMsg = '';
    try {
      await devicesState.loadDevices(filter);
    } catch (e) {
      errorMsg = 'Error de conexión con el servidor';
    } finally {
      loading = false;
    }
  }

  function openApproveModal(device) {
    selectedDevice = device;
    formType = device.device_type || 'table_pad';
    formTableNumber = device.table_number || '';
    formLocationName = device.location_name || '';
    formWifiNetworkId = '';
    formWifiSSID = '';
    formWifiPass = '';
    loadWifiNetworks();
    showModal = true;
  }

  async function submitApproval() {
    if (!selectedDevice) return;
    
    try {
      const body = {
        mac: selectedDevice.mac_address,
        device_type: formType,
        table_number: formType === 'table_pad' && formTableNumber ? formTableNumber : null,
        location_name: formLocationName ? formLocationName : null,
        public_key: selectedDevice.public_key
      };

      if (formWifiNetworkId && formWifiNetworkId !== '__manual') {
        body.wifi_network_id = parseInt(formWifiNetworkId);
      } else if (formWifiNetworkId === '__manual' && formWifiSSID) {
        body.wifi_ssid = formWifiSSID;
        body.wifi_pass = formWifiPass;
      }

      const res = await apiFetch('/api/devices/approve', {
        method: 'POST',
        body: JSON.stringify(body)
      });

      if (res.ok) {
        showModal = false;
        await fetchDevices();
      } else {
        alert('Error al guardar configuración del dispositivo');
      }
    } catch (e) {
      alert('Error de conexión al guardar configuración');
    }
  }

  async function toggleBlockDevice(device) {
    const isBlocking = device.status !== 'blocked';
    const endpoint = isBlocking ? '/api/devices/block' : '/api/devices/approve';
    
    try {
      const body = isBlocking 
        ? { mac: device.mac_address }
        : { 
            mac: device.mac_address, 
            device_type: device.device_type, 
            table_number: device.table_number, 
            location_name: device.location_name,
            public_key: device.public_key
          };

      const res = await apiFetch(endpoint, {
        method: 'POST',
        body: JSON.stringify(body)
      });

      if (res.ok) {
        await fetchDevices();
      } else {
        alert(`Error al ${isBlocking ? 'bloquear' : 'desbloquear'} dispositivo`);
      }
    } catch (e) {
      alert('Error de conexión');
    }
  }

  async function deleteDevice(mac) {
    if (!confirm('¿Estás seguro de que deseas eliminar este dispositivo del sistema? Tendrá que volver a aprovisionarse para conectarse.')) {
      return;
    }
    
    try {
      const res = await apiFetch(`/api/devices?mac=${encodeURIComponent(mac)}`, {
        method: 'DELETE'
      });

      if (res.ok) {
        devicesState.devices = devicesState.devices.filter(d => d.mac_address !== mac);
      } else {
        alert('Error al eliminar el dispositivo');
      }
    } catch (e) {
      alert('Error de conexión');
    }
  }

  $: filteredDevices = (devicesState.devices || []);

  function getBatteryClass(level) {
    if (level > 50) return 'text-success';
    if (level > 20) return 'text-warning';
    return 'text-danger';
  }

  function getWiFiLevel(dbm) {
    if (dbm === 0) return 'No Signal';
    if (dbm > -60) return 'Excellent';
    if (dbm > -80) return 'Good';
    return 'Weak';
  }
  
  function getWiFiClass(dbm) {
    if (dbm === 0) return 'text-danger';
    if (dbm > -65) return 'text-success';
    if (dbm > -85) return 'text-warning';
    return 'text-danger';
  }

  function getStatusLabel(status) {
    switch (status) {
      case 'unprovisioned': return 'Esperando SD / Adopción';
      case 'active': return 'Enlazado y Activo';
      case 'blocked': return 'Bloqueado';
      default: return status;
    }
  }

  let showFilterDropdown = false;

  function selectFilter(value) {
    if (filter === value) return;
    filter = value;
    showFilterDropdown = false;
    fetchDevices();
  }
  
  function toggleDropdown(e) {
    e.stopPropagation();
    showFilterDropdown = !showFilterDropdown;
  }

  function closeDropdown() {
    showFilterDropdown = false;
  }
</script>

<svelte:window on:click={closeDropdown} />

<div class="view-header">
  <h1>Dispositivos</h1>
</div>

<div class="filters-row">
  <div class="dropdown-wrapper">
    <button class="dropdown-trigger" on:click={toggleDropdown} class:open={showFilterDropdown}>
      <span>
        {#if filter === 'all'}
          Todos ({devicesState.deviceSummary?.all || 0})
        {:else if filter === 'unprovisioned'}
          En Espera ({devicesState.deviceSummary?.unprovisioned || 0})
        {:else if filter === 'active'}
          Activos ({devicesState.deviceSummary?.active || 0})
        {:else if filter === 'blocked'}
          Bloqueados ({devicesState.deviceSummary?.blocked || 0})
        {/if}
      </span>
      <ChevronDown size={16} />
    </button>
    
    {#if showFilterDropdown}
      <div class="dropdown-menu panel" on:click|stopPropagation>
        <button class="dropdown-item" class:selected={filter === 'all'} on:click={() => selectFilter('all')}>
          Todos <span class="badge">{devicesState.deviceSummary?.all || 0}</span>
        </button>
        <button class="dropdown-item" class:selected={filter === 'unprovisioned'} on:click={() => selectFilter('unprovisioned')}>
          En Espera <span class="badge badge-warn">{devicesState.deviceSummary?.unprovisioned || 0}</span>
        </button>
        <button class="dropdown-item" class:selected={filter === 'active'} on:click={() => selectFilter('active')}>
          Activos <span class="badge badge-success">{devicesState.deviceSummary?.active || 0}</span>
        </button>
        <button class="dropdown-item" class:selected={filter === 'blocked'} on:click={() => selectFilter('blocked')}>
          Bloqueados <span class="badge badge-danger">{devicesState.deviceSummary?.blocked || 0}</span>
        </button>
      </div>
    {/if}
  </div>
  
  <div style="display: flex; gap: 0.5rem;">
    <button class="neo-btn neo-btn-success" on:click={() => { showCreateModal = true; }}>
      <Plus size={16} /> Crear Mesa
    </button>
    <button class="neo-btn neo-btn-primary" on:click={() => { loadWifiNetworks(); showWifiManager = true; }}>
      <Wifi size={16} /> Configurar Wi-Fi
    </button>
    <button class="neo-btn" on:click={fetchDevices} disabled={loading}>
      Refrescar
    </button>
  </div>
</div>

{#if loading && (devicesState.devices || []).length === 0}
  <div class="empty-state">
    <div class="spinner"></div>
    <p>Cargando dispositivos registrados...</p>
  </div>
{:else if errorMsg}
  <div class="empty-state error">
    <AlertCircle size={40} />
    <p>{errorMsg}</p>
    <button class="refresh-btn" on:click={fetchDevices}>Reintentar</button>
  </div>
{:else if filteredDevices.length === 0}
  <div class="empty-state">
    <Info size={40} />
    <p>No se encontraron dispositivos en esta categoría.</p>
    {#if filter === 'unprovisioned'}
      <p class="hint">Enciende un nuevo ESP32 y conéctalo al WiFi local. Aparecerá aquí al instante para registro.</p>
    {/if}
  </div>
{:else}
  <div class="devices-grid">
    {#each filteredDevices as device (device.mac_address)}
      <div class="device-card panel">
        <div class="card-header">
          <div class="device-icon">
            <Cpu size={24} />
          </div>
          <div class="device-info-header">
            <h3>{device.table_number ? `Mesa ${device.table_number}` : (device.location_name || 'Sin Asignación')}</h3>
            <span class="mac">{device.mac_address.startsWith('PENDING-') ? 'Hardware no enlazado' : device.mac_address}</span>
          </div>
          <div class="status-pill">
            <span class="status-dot {device.status}"></span>
            <span>{getStatusLabel(device.status)}</span>
          </div>
        </div>

        <div class="card-body">
          <div class="info-grid panel-inset">
            <div class="info-item">
              <span class="info-label">Rol:</span>
              <span class="info-val capitalize">{device.device_type.replace('_', ' ')}</span>
            </div>
            
            {#if device.location_name}
              <div class="info-item">
                <span class="info-label">Ubicación:</span>
                <span class="info-val font-semibold">{device.location_name}</span>
              </div>
            {/if}

            <div class="info-item">
              <span class="info-label">IP Address:</span>
              <span class="info-val monospace">
                {device.status === 'unprovisioned' || device.mac_address.startsWith('PENDING-') || device.ip_address === '0.0.0.0' ? 'Pendiente' : device.ip_address}
              </span>
            </div>

            <div class="info-item">
              <span class="info-label">Batería:</span>
              <span class="info-val flex-center {device.status === 'unprovisioned' || device.battery_level < 0 ? '' : getBatteryClass(device.battery_level)}">
                <Battery size={16} class="mr-1" />
                {#if device.status === 'unprovisioned' || device.mac_address.startsWith('PENDING-') || device.battery_level < 0}
                  N/A (Sin conexión)
                {:else if device.battery_level === 100}
                  Alimentado (USB-C)
                {:else}
                  {device.battery_level}%
                {/if}
              </span>
            </div>

            <div class="info-item">
              <span class="info-label">Señal WiFi:</span>
              <span class="info-val flex-center {device.status === 'unprovisioned' || device.wifi_signal === 0 ? '' : getWiFiClass(device.wifi_signal)}">
                <Wifi size={16} class="mr-1" />
                {#if device.status === 'unprovisioned' || device.mac_address.startsWith('PENDING-') || device.wifi_signal === 0}
                  N/A (Sin conexión)
                {:else}
                  {device.wifi_signal} dBm ({getWiFiLevel(device.wifi_signal)})
                {/if}
              </span>
            </div>

            <div class="info-item">
              <span class="info-label">Último contacto:</span>
              <span class="info-val time">
                {#if device.status === 'unprovisioned' || device.mac_address.startsWith('PENDING-')}
                  Pendiente de primer inicio
                {:else}
                  {new Date(device.last_seen).toLocaleTimeString()}
                {/if}
              </span>
            </div>
          </div>
          
          {#if device.public_key}
            <div class="key-box panel-inset">
              <span class="key-title">Clave Pública (Ed25519)</span>
              <span class="key-val monospace">{device.public_key.substring(0, 24)}...</span>
            </div>
          {/if}
        </div>

        <div class="card-footer">
          <button class="neo-btn" on:click={() => openProvisionModalForDevice(device)} title="Descargar archivo tablehub.enc para MicroSD">
            <Download size={16} /> Descargar Aprovisionamiento
          </button>

          {#if device.status === 'unprovisioned'}
            <button class="neo-btn neo-btn-primary" on:click={() => openApproveModal(device)}>
              <CheckCircle size={16} /> Aprobar
            </button>
          {:else}
            <button class="neo-btn neo-btn-primary" on:click={() => openApproveModal(device)}>
              <Edit3 size={16} /> Configurar
            </button>
            
            <button class="neo-btn neo-btn-warning" on:click={() => toggleBlockDevice(device)}>
              {#if device.status === 'blocked'}
                Activar
              {:else}
                Bloquear
              {/if}
            </button>
          {/if}
          
          <button class="neo-btn neo-btn-danger" on:click={() => deleteDevice(device.mac_address)} title="Eliminar del Hub">
            <Trash2 size={16} />
          </button>
        </div>
      </div>
    {/each}
  </div>
{/if}

<!-- MODAL CREAR NUEVA MESA / DISPOSITIVO -->
{#if showCreateModal}
  <div class="modal-backdrop" on:click={() => showCreateModal = false}>
    <div class="modal panel" on:click|stopPropagation>
      <h2>Crear Nueva Mesa / Entidad</h2>
      <p class="modal-subtitle">Registra una mesa en el Hub antes de aprovisionarla físicamente</p>

      <div class="form">
        <div class="input-group">
          <label for="createTableNumber">Número o ID de Mesa *</label>
          <input type="text" id="createTableNumber" bind:value={createTableNumber} placeholder="Ej. 1, 5, VIP-2" />
        </div>

        <div class="input-group">
          <label for="createLocationName">Ubicación Física (Opcional)</label>
          <input type="text" id="createLocationName" bind:value={createLocationName} placeholder="Ej. Salón Principal, Terraza" />
        </div>

        <div class="input-group">
          <label for="createDeviceType">Tipo de Dispositivo</label>
          <select id="createDeviceType" bind:value={createDeviceType}>
            <option value="table_pad">TablePad (Servicio en Mesa)</option>
            <option value="kitchen_button">Avisador de Cocina</option>
            <option value="sensor">Sensor de Telemetría</option>
          </select>
        </div>

        {#if createError}
          <div class="error-msg" style="color: #ef4444; font-size: 0.85rem; margin-top: 0.5rem;">{createError}</div>
        {/if}

        <div class="modal-actions" style="margin-top: 1.5rem;">
          <button class="action-btn" on:click={() => showCreateModal = false}>Cancelar</button>
          <button class="action-btn btn-primary" on:click={handleCreateDevice} disabled={createLoading}>
            {#if createLoading}
              Creando...
            {:else}
              <Plus size={16} /> Crear Mesa
            {/if}
          </button>
        </div>
      </div>
    </div>
  </div>
{/if}

<!-- MODAL DE CONFIGURACIÓN / APROBACIÓN -->
{#if showModal}
  <div class="modal-backdrop" on:click={() => showModal = false}>
    <div class="modal panel" on:click|stopPropagation>
      <h2>Configurar Dispositivo</h2>
      <p class="modal-subtitle">MAC: {selectedDevice.mac_address}</p>

      <div class="form">
        <div class="input-group">
          <label for="device_type">Rol del Dispositivo</label>
          <select id="device_type" bind:value={formType}>
            <option value="table_pad">TablePad (Servicio en Mesa)</option>
            <option value="kitchen_button">Avisador de Cocina (Botonera)</option>
            <option value="sensor">Sensor de Telemetría (Monitoreo)</option>
            <option value="generic">Genérico / Custom</option>
          </select>
        </div>

        {#if formType === 'table_pad'}
          <div class="input-group">
            <label for="table_number">Número de Mesa</label>
            <input type="text" id="table_number" bind:value={formTableNumber} placeholder="Ej. 4, 12, VIP-1" />
          </div>
        {/if}

        <div class="input-group">
          <label for="location_name">Ubicación Física</label>
          <input type="text" id="location_name" bind:value={formLocationName} placeholder="Ej: Barra Central, Mesa 4, Parrilla" />
        </div>

        <div class="input-group">
          <label for="formWifiSelect">Reconfigurar Red Wi-Fi (Sincronización en Vivo)</label>
          <select id="formWifiSelect" bind:value={formWifiNetworkId}>
            <option value="">-- Mantiene Wi-Fi actual --</option>
            {#each wifiNetworks as net}
              <option value={net.id}>{net.ssid}</option>
            {/each}
            <option value="__manual">-- Introducir red manualmente --</option>
          </select>
        </div>

        {#if formWifiNetworkId === '__manual'}
          <div class="input-group">
            <label for="formWifiSSID">Wi-Fi SSID</label>
            <input type="text" id="formWifiSSID" bind:value={formWifiSSID} placeholder="Nombre de red Wi-Fi" />
          </div>
          <div class="input-group">
            <label for="formWifiPass">Contraseña Wi-Fi</label>
            <input type="password" id="formWifiPass" bind:value={formWifiPass} placeholder="••••••••" />
          </div>
        {/if}

        <p style="font-size: 0.78rem; color: var(--text-muted); margin-top: 0.5rem; display: flex; align-items: center; gap: 0.35rem;">
          ⚡ Al guardar, los cambios se enviarán en vivo por MQTT a la placa ESP32 si está conectada.
        </p>

        <div class="modal-actions" style="margin-top: 1.5rem;">
          <button class="neo-btn" on:click={() => showModal = false}>Cancelar</button>
          <button class="neo-btn neo-btn-primary" on:click={submitApproval}>Guardar y Sincronizar</button>
        </div>
      </div>
    </div>
  </div>
{/if}

<!-- MODAL DE GENERACIÓN DE ARCHIVO DE APROVISIONAMIENTO -->
{#if showProvisionModal}
  <div class="modal-backdrop" on:click={closeProvisionModal}>
    <div class="modal panel" on:click|stopPropagation>
      <h2>Aprovisionar Nueva Mesa</h2>
      <p class="modal-subtitle">Genera tablehub.enc para la memoria SD</p>

      <div class="form">
        <div class="input-group">
          <label for="provTableId">ID de la Mesa (Ej: Mesa-4)</label>
          <input type="text" id="provTableId" bind:value={provTableId} placeholder="mesa-4" />
        </div>

        <div class="input-group">
          <label for="provWifiSelect">Red Wi-Fi</label>
          <select id="provWifiSelect" bind:value={provSelectedWifiId} on:change={onWifiSelect}>
            <option value="">-- Seleccionar red guardada --</option>
            {#each wifiNetworks as net}
              <option value={net.id}>{net.ssid}</option>
            {/each}
            <option value="__manual">-- Introducir manualmente --</option>
          </select>
          <div class="wifi-manage-link">
            <button class="link-btn" on:click={() => { showWifiManager = true; }} on:mousedown|stopPropagation>
              <Network size={14} /> Gestionar redes Wi-Fi
            </button>
          </div>
        </div>

        {#if !provSelectedWifiId || provSelectedWifiId === '__manual'}
          <div class="input-group">
            <label for="provWifiSSID">Wi-Fi SSID</label>
            <input type="text" id="provWifiSSID" bind:value={provWifiSSID} placeholder="MiRedWiFi" />
          </div>

          <div class="input-group">
            <label for="provWifiPass">Contraseña Wi-Fi</label>
            <input type="password" id="provWifiPass" bind:value={provWifiPass} placeholder="••••••••" />
          </div>

          {#if provSelectedWifiId === '__manual' && provWifiSSID}
            <label class="checkbox-label">
              <input type="checkbox" bind:checked={provSaveNetwork} />
              <span>Guardar esta red Wi-Fi para futuros aprovisionamientos</span>
            </label>
          {/if}
        {/if}

        <div class="input-group">
          <label for="provPIN">PIN de Seguridad (para cifrado)</label>
          <input type="number" id="provPIN" bind:value={provPIN} placeholder="123456" />
        </div>

        {#if provError}
          <div class="error-msg" style="color: #ef4444; font-size: 0.85rem;">{provError}</div>
        {/if}

        <div class="modal-actions" style="margin-top: 1.5rem;">
          <button class="action-btn" on:click={closeProvisionModal}>Cancelar</button>
          
          {#if provFileUrl}
            <a href={provFileUrl} download="tablehub.enc" class="action-btn btn-primary" style="text-decoration: none;">
              <Download size={16} /> Descargar tablehub.enc
            </a>
            <button class="action-btn" on:click={closeProvisionModal}>Cerrar</button>
          {:else}
            <button class="action-btn btn-primary" on:click={generateProvisionFile} disabled={provLoading}>
              {#if provLoading}
                Generando...
              {:else}
                Generar Archivo
              {/if}
            </button>
          {/if}
        </div>
      </div>
    </div>
  </div>
{/if}

<!-- MODAL GESTOR DE REDES WI-FI -->
{#if showWifiManager}
  <div class="modal-backdrop" on:click={closeWifiManager}>
    <div class="modal panel" on:click|stopPropagation>
      <h2>Gestionar Redes Wi-Fi</h2>
      <p class="modal-subtitle">Redes guardadas para aprovisionamiento</p>

      <div class="form">
        {#if wifiNetworks.length > 0}
          <div class="wifi-list">
            {#each wifiNetworks as net}
              <div class="wifi-list-item">
                <div class="wifi-list-info">
                  <Wifi size={16} />
                  <span>{net.ssid}</span>
                </div>
                <button class="action-btn btn-danger btn-sm" on:click={() => handleDeleteWifi(net.id)}>
                  <Trash2 size={14} />
                </button>
              </div>
            {/each}
          </div>
        {:else}
          <p style="color: var(--text-muted); text-align: center; padding: 1rem 0;">
            No hay redes guardadas
          </p>
        {/if}

        <div class="wifi-divider"></div>

        <div class="input-group">
          <label for="newWifiSSID">Nueva Red SSID</label>
          <input type="text" id="newWifiSSID" bind:value={newWifiSSID} placeholder="Nombre de la red" />
        </div>

        <div class="input-group">
          <label for="newWifiPass">Contraseña</label>
          <input type="password" id="newWifiPass" bind:value={newWifiPass} placeholder="••••••••" />
        </div>

        {#if wifiManagerError}
          <div class="error-msg" style="color: #ef4444; font-size: 0.85rem;">{wifiManagerError}</div>
        {/if}

        <div class="modal-actions">
          <button class="action-btn" on:click={closeWifiManager}>Cerrar</button>
          <button class="action-btn btn-primary" on:click={handleSaveWifi} disabled={wifiSaveLoading}>
            {#if wifiSaveLoading}
              Guardando...
            {:else}
              <Save size={16} /> Guardar Red
            {/if}
          </button>
        </div>
      </div>
    </div>
  </div>
{/if}

<style>
  .view-header {
    margin-bottom: 2rem;
  }
  
  .view-header h1 {
    font-size: 2.2rem;
    background: linear-gradient(135deg, #fff 0%, #a5b4fc 100%);
    -webkit-background-clip: text;
    -webkit-text-fill-color: transparent;
    margin: 0 0 0.5rem 0;
  }
  
  .subtitle {
    color: var(--text-muted);
    margin: 0;
  }

  .filters-row {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 2rem;
    gap: 1rem;
    flex-wrap: wrap;
  }

  .dropdown-wrapper {
    position: relative;
    display: inline-block;
  }

  .dropdown-trigger {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 1rem;
    padding: 0.6rem 1.2rem;
    background: var(--panel-bg);
    border: var(--panel-border);
    box-shadow: var(--panel-shadow);
    color: var(--text-color);
    border-radius: 12px;
    cursor: pointer;
    font-size: 0.95rem;
    font-weight: 500;
    transition: all 0.3s cubic-bezier(0.25, 0.8, 0.25, 1);
    min-width: 180px;
  }

  .dropdown-trigger:hover {
    box-shadow: var(--panel-shadow-hover);
  }

  .dropdown-trigger.open {
    box-shadow: var(--button-hover-shadow, inset 2px 2px 5px rgba(0,0,0,0.2));
  }

  .dropdown-menu {
    position: absolute;
    top: calc(100% + 8px);
    left: 0;
    z-index: 1100;
    background: var(--panel-bg);
    border: var(--panel-border);
    box-shadow: var(--panel-shadow);
    border-radius: 12px;
    padding: 0.5rem;
    min-width: 200px;
    display: flex;
    flex-direction: column;
    gap: 0.25rem;
    animation: dropdown-enter 0.2s ease-out;
  }

  @keyframes dropdown-enter {
    from { opacity: 0; transform: translateY(-10px); }
    to { opacity: 1; transform: translateY(0); }
  }

  .dropdown-item {
    background: transparent;
    border: none;
    color: var(--text-muted);
    padding: 0.6rem 1rem;
    border-radius: 8px;
    cursor: pointer;
    font-size: 0.9rem;
    font-weight: 500;
    text-align: left;
    display: flex;
    align-items: center;
    justify-content: space-between;
    transition: all 0.2s;
    width: 100%;
  }

  .dropdown-item:hover {
    background: var(--button-hover-bg);
    color: var(--text-color);
  }

  .dropdown-item.selected {
    background: var(--button-active-bg);
    color: var(--accent-color);
    box-shadow: var(--button-hover-shadow, none);
  }

  .badge {
    background: rgba(255, 255, 255, 0.1);
    color: var(--text-color);
    font-size: 0.75rem;
    padding: 0.1rem 0.4rem;
    border-radius: 6px;
    font-weight: 600;
  }

  .badge-warn { background: rgba(245, 158, 11, 0.2); color: #f59e0b; }
  .badge-success { background: rgba(16, 185, 129, 0.2); color: #10b981; }
  .badge-danger { background: rgba(239, 68, 68, 0.2); color: #ef4444; }

  .refresh-btn {
    padding: 0.6rem 1.2rem;
    background: var(--panel-bg);
    border: var(--panel-border);
    box-shadow: var(--panel-shadow);
    color: var(--text-color);
    border-radius: 12px;
    cursor: pointer;
    font-weight: 500;
    transition: all 0.3s cubic-bezier(0.25, 0.8, 0.25, 1);
  }

  .refresh-btn:hover:not(:disabled) {
    box-shadow: var(--panel-shadow-hover);
  }

  .refresh-btn:active:not(:disabled) {
    box-shadow: var(--button-hover-shadow, inset 2px 2px 5px rgba(0,0,0,0.2));
  }
  
  .refresh-btn:disabled {
    opacity: 0.4;
    cursor: not-allowed;
    box-shadow: none;
  }

  .devices-grid {
    display: grid;
    grid-template-columns: 1fr;
    gap: 1.5rem;
  }

  @media (min-width: 768px) {
    .devices-grid {
      grid-template-columns: repeat(2, 1fr);
    }
  }

  @media (min-width: 1200px) {
    .devices-grid {
      grid-template-columns: repeat(3, 1fr);
    }
  }

  .device-card {
    display: flex;
    flex-direction: column;
    height: 100%;
    transition: all 0.3s cubic-bezier(0.25, 1, 0.5, 1);
  }

  .device-card:hover {
    transform: translateY(-2px);
    box-shadow: var(--panel-shadow-hover);
  }

  .card-header {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    padding-bottom: 1rem;
    border-bottom: 1px solid rgba(255, 255, 255, 0.05);
    margin-bottom: 1rem;
  }

  .device-icon {
    width: 40px;
    height: 40px;
    border-radius: 10px;
    background: var(--panel-inset-bg);
    border: var(--panel-inset-border);
    box-shadow: var(--panel-inset-shadow);
    display: flex;
    align-items: center;
    justify-content: center;
    color: var(--accent-color);
  }

  .device-info-header {
    flex: 1;
  }

  .device-info-header h3 {
    margin: 0;
    font-size: 1.1rem;
    font-weight: 600;
  }

  .mac {
    font-size: 0.8rem;
    color: var(--text-muted);
    font-family: monospace;
  }

  .card-body {
    flex: 1;
    margin-bottom: 1.5rem;
  }

  .info-grid {
    display: flex;
    flex-direction: column;
    gap: 0.75rem;
    background: var(--panel-inset-bg);
    border: var(--panel-inset-border);
    box-shadow: var(--panel-inset-shadow);
    border-radius: 12px;
    padding: 1rem;
  }

  .info-item {
    display: flex;
    justify-content: space-between;
    font-size: 0.85rem;
  }

  .info-label {
    color: var(--text-muted);
  }

  .info-val {
    color: var(--text-color);
    font-weight: 500;
  }

  .flex-center {
    display: flex;
    align-items: center;
  }

  .mr-1 { margin-right: 0.25rem; }
  .capitalize { text-transform: capitalize; }
  .font-semibold { font-weight: 600; }

  .key-box {
    background: var(--panel-inset-bg);
    border: var(--panel-inset-border);
    box-shadow: var(--panel-inset-shadow);
    border-radius: 12px;
    padding: 0.75rem;
    margin-top: 1rem;
  }

  .key-title {
    display: block;
    font-size: 0.75rem;
    color: var(--text-muted);
    margin-bottom: 0.25rem;
  }

  .key-val {
    font-size: 0.75rem;
    color: var(--text-muted);
    word-break: break-all;
  }

  .card-footer {
    display: flex;
    gap: 0.5rem;
    border-top: 1px solid rgba(255, 255, 255, 0.05);
    padding-top: 1rem;
  }

  .action-btn {
    padding: 0.4rem 0.8rem;
    background: rgba(255, 255, 255, 0.05);
    border: 1px solid rgba(255, 255, 255, 0.1);
    color: var(--text-color);
    border-radius: 8px;
    cursor: pointer;
    font-size: 0.8rem;
    font-weight: 500;
    transition: all 0.2s;
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 0.4rem;
  }

  .action-btn:hover {
    background: rgba(255, 255, 255, 0.1);
  }

  .btn-primary {
    background: var(--accent-color);
    border-color: var(--accent-color);
    color: white;
  }

  .btn-primary:hover {
    background: var(--accent-hover);
  }

  .btn-danger {
    color: #ef4444;
    background: rgba(239, 68, 68, 0.1);
    border-color: rgba(239, 68, 68, 0.2);
  }

  .btn-danger:hover {
    background: #ef4444;
    color: white;
  }

  .btn-sec-danger {
    color: #ef4444;
    border-color: rgba(239, 68, 68, 0.1);
  }

  .btn-sec-danger:hover {
    background: rgba(239, 68, 68, 0.1);
  }

  .btn-sm {
    padding: 0.3rem 0.5rem;
    font-size: 0.75rem;
  }

  .spinner {
    border: 3px solid rgba(255, 255, 255, 0.1);
    border-left-color: var(--accent-color);
    border-radius: 50%;
    width: 30px;
    height: 30px;
    animation: spin 1s linear infinite;
    margin: 0 auto 1rem auto;
  }

  @keyframes spin {
    0% { transform: rotate(0deg); }
    100% { transform: rotate(360deg); }
  }

  .empty-state {
    text-align: center;
    padding: 4rem 2rem;
    color: var(--text-muted);
    background: rgba(255, 255, 255, 0.02);
    border-radius: 16px;
    border: 1px dashed rgba(255, 255, 255, 0.05);
  }

  .empty-state.error {
    color: #ef4444;
    border-color: rgba(239, 68, 68, 0.2);
  }

  .hint {
    font-size: 0.85rem;
    color: var(--text-muted);
    max-width: 400px;
    margin: 0.5rem auto 0 auto;
  }

  .modal-backdrop {
    position: fixed;
    top: 0;
    left: 0;
    right: 0;
    bottom: 0;
    background: rgba(0, 0, 0, 0.6);
    backdrop-filter: blur(4px);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 2000;
  }

  .modal {
    width: 100%;
    max-width: 450px;
    padding: 2rem;
    animation: modal-enter 0.3s cubic-bezier(0.34, 1.56, 0.64, 1);
  }

  @keyframes modal-enter {
    from { transform: scale(0.9); opacity: 0; }
    to { transform: scale(1); opacity: 1; }
  }

  .modal h2 {
    margin: 0 0 0.25rem 0;
    font-size: 1.5rem;
  }

  .modal-subtitle {
    color: var(--text-muted);
    font-family: monospace;
    font-size: 0.85rem;
    margin-bottom: 1.5rem;
  }

  .form {
    display: flex;
    flex-direction: column;
    gap: 1.2rem;
  }

  .input-group {
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
  }

  .input-group label {
    font-size: 0.9rem;
    font-weight: 500;
    color: var(--text-muted);
  }

  .input-group input, .input-group select {
    padding: 0.6rem 0.75rem;
    background: rgba(0, 0, 0, 0.2);
    border: 1px solid rgba(255, 255, 255, 0.1);
    border-radius: 8px;
    color: white;
    font-size: 0.95rem;
    transition: all 0.2s;
  }

  .input-group input:focus, .input-group select:focus {
    outline: none;
    border-color: var(--accent-color);
    box-shadow: 0 0 0 2px rgba(99, 102, 241, 0.2);
  }

  .checkbox-label {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    font-size: 0.85rem;
    color: var(--text-muted);
    cursor: pointer;
  }

  .checkbox-label input[type="checkbox"] {
    width: 16px;
    height: 16px;
    accent-color: var(--accent-color);
  }

  .wifi-manage-link {
    margin-top: 0.25rem;
  }

  .link-btn {
    background: none;
    border: none;
    color: var(--accent-color);
    cursor: pointer;
    font-size: 0.8rem;
    display: flex;
    align-items: center;
    gap: 0.3rem;
    padding: 0.25rem 0;
    opacity: 0.8;
    transition: opacity 0.2s;
  }

  .link-btn:hover {
    opacity: 1;
    text-decoration: underline;
  }

  .wifi-list {
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
    max-height: 200px;
    overflow-y: auto;
  }

  .wifi-list-item {
    display: flex;
    align-items: center;
    justify-content: space-between;
    background: rgba(0, 0, 0, 0.15);
    border-radius: 8px;
    padding: 0.5rem 0.75rem;
  }

  .wifi-list-info {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    font-size: 0.9rem;
    color: var(--text-color);
  }

  .wifi-divider {
    height: 1px;
    background: rgba(255, 255, 255, 0.08);
    margin: 0.5rem 0;
  }

  .modal-actions {
    display: flex;
    justify-content: flex-end;
    gap: 0.75rem;
    margin-top: 1rem;
  }
</style>
