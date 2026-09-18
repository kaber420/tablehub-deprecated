<script>
  import { Activity, Smartphone, Bell, Users } from '@lucide/svelte';
  import { devicesState } from '../devices.svelte.js';
</script>

<div class="kpi-grid">
  <!-- Card 1: Hub Status -->
  <div class="panel kpi-card">
    <div class="kpi-icon-wrapper icon-green">
      <Activity size={24} />
    </div>
    <div class="kpi-info">
      <div class="kpi-value">
        <span class="led-indicator led-online" style="margin-right: 6px;"></span>
        Online
      </div>
      <div class="kpi-label">Estado de Maitre Hub</div>
    </div>
  </div>

  <!-- Card 2: Active Devices -->
  <div class="panel kpi-card">
    <div class="kpi-icon-wrapper icon-blue">
      <Smartphone size={24} />
    </div>
    <div class="kpi-info">
      <div class="kpi-value">{devicesState.activeDevicesCount}</div>
      <div class="kpi-label">Dispositivos Activos</div>
    </div>
  </div>

  <!-- Card 3: System Alerts -->
  <div class="panel kpi-card">
    <div class="kpi-icon-wrapper {devicesState.deviceAlerts.length > 0 ? 'icon-red' : 'icon-green'}">
      <Bell size={24} />
    </div>
    <div class="kpi-info">
      <div class="kpi-value">{devicesState.deviceAlerts.length}</div>
      <div class="kpi-label">Alertas Activas</div>
    </div>
  </div>

  <!-- Card 4: Shift Staff -->
  <div class="panel kpi-card">
    <div class="kpi-icon-wrapper icon-orange">
      <Users size={24} />
    </div>
    <div class="kpi-info">
      <div class="kpi-value">{devicesState.activeUsersCount}</div>
      <div class="kpi-label">Personal de Turno</div>
    </div>
  </div>
</div>
