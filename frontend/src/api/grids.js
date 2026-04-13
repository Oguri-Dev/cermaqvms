const API_BASE = import.meta.env.VITE_API_URL || 'http://localhost:8080/api';

async function request(path, options = {}) {
  const res = await fetch(`${API_BASE}${path}`, {
    headers: { 'Content-Type': 'application/json' },
    ...options,
  });
  if (!res.ok) {
    const text = await res.text();
    throw new Error(text || res.statusText);
  }
  if (res.status === 204) return null;
  return res.json();
}

// Devices (NVRs and cameras)
export const listDevices = () => request('/devices');
export const getDevice = (id) => request(`/devices/${id}`);
export const createDevice = (dev) => request('/devices', { method: 'POST', body: JSON.stringify(dev) });
export const updateDevice = (id, dev) => request(`/devices/${id}`, { method: 'PUT', body: JSON.stringify(dev) });
export const deleteDevice = (id) => request(`/devices/${id}`, { method: 'DELETE' });

// Grids (layout only)
export const listGrids = () => request('/grids');
export const getGrid = (id) => request(`/grids/${id}`);
export const createGrid = (grid) => request('/grids', { method: 'POST', body: JSON.stringify(grid) });
export const updateGrid = (id, grid) => request(`/grids/${id}`, { method: 'PUT', body: JSON.stringify(grid) });
export const deleteGrid = (id) => request(`/grids/${id}`, { method: 'DELETE' });

// Streams (grid + cameras + stream IP)
export const listStreams = () => request('/streams');
export const getStream = (id) => request(`/streams/${id}`);
export const getStreamFull = (id) => request(`/streams/${id}/full`);
export const createStream = (stream) => request('/streams', { method: 'POST', body: JSON.stringify(stream) });
export const updateStream = (id, stream) => request(`/streams/${id}`, { method: 'PUT', body: JSON.stringify(stream) });
export const deleteStream = (id) => request(`/streams/${id}`, { method: 'DELETE' });
export const syncStream = (id) => request(`/streams/${id}/sync`, { method: 'POST' });
export const previewScreenConfig = (id) => request(`/streams/${id}/screen-config`);

// PTZ
export const ptzMove = (streamId, row, col, command) =>
  request(`/streams/${streamId}/ptz/${row}/${col}/move`, { method: 'POST', body: JSON.stringify(command) });
export const ptzPreset = (streamId, row, col, preset) =>
  request(`/streams/${streamId}/ptz/${row}/${col}/preset`, { method: 'POST', body: JSON.stringify(preset) });

// Screen config
export const getScreenConfig = () => request('/screen-config');
export const updateScreenConfig = (config) => request('/screen-config', { method: 'PUT', body: JSON.stringify(config) });
