import { devicesState } from './devices.svelte.js';

class WSState {
  events = $state([]);
  connectionStatus = $state('connecting...');
  ws = null;

  init(wsUrl) {
    const saved = localStorage.getItem('tablehub_live_events');
    if (saved) {
      try { 
        this.events = JSON.parse(saved).filter(ev => ev.event !== 'ping'); 
        localStorage.setItem('tablehub_live_events', JSON.stringify(this.events));
      } catch (e) {}
    }
    this.connect(wsUrl);
  }

  connect(url) {
    this.ws = new WebSocket(url);
    this.ws.onopen = () => this.connectionStatus = 'online';
    this.ws.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data);
        if (data.event === 'ping') return;
        data.payload = data.payload ? this.parsePayload(data.payload) : {};
        
        // Update events list
        this.events = [data, ...this.events].slice(0, 30);
        localStorage.setItem('tablehub_live_events', JSON.stringify(this.events));
        
        // Handle telemetry status updates in real-time
        if (data.event === 'device.status_update') {
          const payload = typeof data.payload === 'string' ? JSON.parse(data.payload) : data.payload;
          devicesState.updateDeviceInList(payload.mac, {
            battery_level: payload.battery,
            wifi_signal: payload.wifi,
            status: payload.status,
            ip_address: payload.ip,
            last_seen: new Date().toISOString()
          });
        } else if (data.event === 'device.provision_request') {
          const payload = typeof data.payload === 'string' ? JSON.parse(data.payload) : data.payload;
          const exists = devicesState.devices.some(d => d.mac_address === payload.mac);
          if (!exists) {
            devicesState.devices = [{
              mac_address: payload.mac,
              device_type: payload.type || 'table_pad',
              ip_address: payload.ip || '0.0.0.0',
              public_key: payload.public_key,
              status: 'unprovisioned',
              battery_level: 100,
              wifi_signal: 0,
              last_seen: new Date().toISOString(),
              created_at: new Date().toISOString()
            }, ...devicesState.devices];
          } else {
            devicesState.updateDeviceInList(payload.mac, {
              public_key: payload.public_key,
              ip_address: payload.ip || '0.0.0.0',
              last_seen: new Date().toISOString()
            });
          }
        }
      } catch (e) {}
    };
    
    this.ws.onclose = () => {
      this.connectionStatus = 'offline';
      setTimeout(() => this.connect(url), 5000);
    };
    
    this.ws.onerror = () => {
      this.connectionStatus = 'error';
    };
  }

  close() {
    if (this.ws) {
      this.ws.close();
    }
  }

  clearTunnelLogs() {
    this.events = [];
    localStorage.removeItem('tablehub_live_events');
  }

  parsePayload(payload) {
    if (!payload) return {};
    return typeof payload === 'string' ? JSON.parse(payload) : payload;
  }
}

export const wsState = new WSState();
