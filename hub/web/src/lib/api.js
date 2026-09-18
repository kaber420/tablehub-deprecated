// lib/api.js

/**
 * Wrapper for the native fetch API that automatically includes credentials (cookies)
 * and handles 401 Unauthorized responses.
 */
export async function apiFetch(url, options = {}) {
  // Siempre incluir credenciales para enviar la cookie HttpOnly
  const fetchOptions = {
    ...options,
    credentials: 'include', 
    headers: {
      'Content-Type': 'application/json',
      ...options.headers
    }
  };

  const response = await fetch(url, fetchOptions);

  // Si el servidor rechaza el token (expirado o inválido)
  if (response.status === 401) {
    // Disparar un evento global para que App.svelte cambie a LoginView
    window.dispatchEvent(new CustomEvent('auth-expired'));
  }

  return response;
}

export async function getCloudSettings() {
  const res = await apiFetch('/api/settings/cloud');
  if (!res.ok) throw new Error('Failed to fetch cloud settings');
  return await res.json();
}

export async function saveCloudSettings(settings) {
  const res = await apiFetch('/api/settings/cloud', {
    method: 'POST',
    body: JSON.stringify(settings)
  });
  if (!res.ok) throw new Error('Failed to save cloud settings');
  return await res.json();
}

export async function uploadProvisionFile(file) {
  const formData = new FormData();
  formData.append('file', file);

  const res = await fetch('/api/provision', {
    method: 'POST',
    credentials: 'include',
    body: formData
  });

  if (res.status === 401) {
    window.dispatchEvent(new CustomEvent('auth-expired'));
  }

  if (!res.ok) throw new Error('Failed to upload provision file');
  return await res.json();
}

export async function disconnectCloud() {
  const res = await apiFetch('/api/settings/cloud/disconnect', {
    method: 'POST'
  });
  if (!res.ok) throw new Error('Failed to disconnect cloud');
  return await res.json();
}

export async function getScreensaverSettings() {
  const res = await apiFetch('/api/settings/screensaver');
  if (!res.ok) throw new Error('Failed to fetch screensaver settings');
  return await res.json();
}

export async function saveScreensaverSettings(settings) {
  const res = await apiFetch('/api/settings/screensaver', {
    method: 'POST',
    body: JSON.stringify(settings)
  });
  if (!res.ok) throw new Error('Failed to save screensaver settings');
  return await res.json();
}

export async function getWifiNetworks() {
  const res = await apiFetch('/api/settings/wifi');
  if (!res.ok) throw new Error('Failed to fetch wifi networks');
  return await res.json();
}

export async function saveWifiNetwork(ssid, password) {
  const res = await apiFetch('/api/settings/wifi', {
    method: 'POST',
    body: JSON.stringify({ ssid, password })
  });
  if (!res.ok) throw new Error('Failed to save wifi network');
  return await res.json();
}

export async function deleteWifiNetwork(id) {
  const res = await apiFetch(`/api/settings/wifi?id=${id}`, {
    method: 'DELETE'
  });
  if (!res.ok) throw new Error('Failed to delete wifi network');
  return await res.json();
}

export async function uploadScreensaverImage(file) {
  const formData = new FormData();
  formData.append('image', file);

  const res = await fetch('/api/settings/screensaver/image', {
    method: 'POST',
    credentials: 'include',
    body: formData
  });

  if (res.status === 401) {
    window.dispatchEvent(new CustomEvent('auth-expired'));
  }

  if (!res.ok) throw new Error('Failed to upload screensaver image');
  return await res.json();
}


