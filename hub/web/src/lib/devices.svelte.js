import { apiFetch } from './api.js';

class DevicesState {
  devices = $state([]);
  users = $state([]);
  deviceAlerts = $state([]);
  deviceSummary = $state({ all: 0, active: 0, unprovisioned: 0, blocked: 0 });

  async loadDevices(status = 'all') {
    try {
      const qs = status !== 'all' ? `?status=${status}` : '';
      const res = await apiFetch(`/api/devices${qs}`);
      if (res.ok) {
        this.devices = await res.json();
      }
      this.loadSummary();
      this.loadAlerts(); // Load alerts alongside devices
    } catch (e) {
      console.error("Error loading devices:", e);
    }
  }

  async loadSummary() {
    try {
      const res = await apiFetch('/api/devices/summary');
      if (res.ok) {
        this.deviceSummary = await res.json();
      }
    } catch (e) {
      console.error("Error loading device summary:", e);
    }
  }

  async loadAlerts() {
    try {
      const res = await apiFetch('/api/alerts');
      if (res.ok) {
        this.deviceAlerts = await res.json();
      }
    } catch (e) {
      console.error("Error loading alerts:", e);
    }
  }

  async loadUsers() {
    try {
      const res = await apiFetch('/api/users');
      if (res.ok) {
        this.users = await res.json();
      }
    } catch (e) {
      console.error("Error loading users:", e);
    }
  }

  updateDeviceInList(mac, updatedFields) {
    this.devices = this.devices.map(d => {
      if (d.mac_address === mac) {
        return { ...d, ...updatedFields };
      }
      return d;
    });
    this.loadAlerts(); // Update alerts when device status changes
  }



  get activeDevicesCount() {
    return this.devices.filter(d => d.status === 'active').length;
  }

  get activeUsersCount() {
    return this.users.filter(u => u.active).length;
  }
}

export const devicesState = new DevicesState();
