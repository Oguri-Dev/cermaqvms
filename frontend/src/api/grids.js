// El backend corre en el mismo equipo que sirve el front: derivar el host de
// la URL del navegador permite acceder desde otros equipos de la red
const API_BASE = import.meta.env.VITE_API_URL || `http://${window.location.hostname}:8080/api`;

const TOKEN_KEY = 'omnifish_token';

export function getToken() {
  return localStorage.getItem(TOKEN_KEY) || sessionStorage.getItem(TOKEN_KEY);
}
export function setToken(token, remember) {
  clearToken();
  (remember ? localStorage : sessionStorage).setItem(TOKEN_KEY, token);
}
export function clearToken() {
  localStorage.removeItem(TOKEN_KEY);
  sessionStorage.removeItem(TOKEN_KEY);
}

// Callback que el AuthContext registra para reaccionar a sesiones expiradas
let onUnauthorized = null;
export function setUnauthorizedHandler(fn) { onUnauthorized = fn; }

async function request(path, options = {}) {
  const token = getToken();
  const res = await fetch(`${API_BASE}${path}`, {
    headers: {
      'Content-Type': 'application/json',
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
    },
    ...options,
  });
  if (res.status === 401) {
    clearToken();
    if (onUnauthorized) onUnauthorized();
    throw new Error('Sesión expirada o no autenticado');
  }
  if (!res.ok) {
    const text = await res.text();
    let msg = text || res.statusText;
    try { msg = JSON.parse(text).error || msg; } catch { /* texto plano */ }
    throw new Error(msg);
  }
  if (res.status === 204) return null;
  return res.json();
}

// ── Auth ──
export const login = (username, password, remember) =>
  request('/auth/login', { method: 'POST', body: JSON.stringify({ username, password, remember }) });
export const getMe = () => request('/auth/me');

// ── Usuarios (solo admin) ──
export const listUsers = () => request('/users');
export const createUser = (u) => request('/users', { method: 'POST', body: JSON.stringify(u) });
export const updateUser = (id, u) => request(`/users/${id}`, { method: 'PUT', body: JSON.stringify(u) });
export const deleteUser = (id) => request(`/users/${id}`, { method: 'DELETE' });

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

// Conexión al servidor del centro (singleton)
export const getCenterConfig = () => request('/center-config');
export const updateCenterConfig = (config) => request('/center-config', { method: 'PUT', body: JSON.stringify(config) });
export const getCenterStatus = () => request('/center-config/status');
export const importFromCenter = () => request('/center/import', { method: 'POST' });

// Centro: lectura de screen_configuration real + zoom del compresor GST-Grid
export const listCenterScreens = () => request('/center/screens');
export const getCenterScreen = (fileName) => request(`/center/screens/${encodeURIComponent(fileName)}`);
export const zoomCenterCell = (fileName, cellName) =>
  request(`/center/screens/${encodeURIComponent(fileName)}/zoom/${encodeURIComponent(cellName)}`, { method: 'POST' });
export const unzoomCenter = (fileName) =>
  request(`/center/screens/${encodeURIComponent(fileName)}/unzoom`, { method: 'POST' });
export const setCenterScreenActive = (fileName, active) =>
  request(`/center/screens/${encodeURIComponent(fileName)}/active`, { method: 'PUT', body: JSON.stringify({ active }) });
export const getCenterZoomStatus = (fileName) => request(`/center/screens/${encodeURIComponent(fileName)}/zoom-status`);
export const getCenterHealth = (fileName) => request(`/center/screens/${encodeURIComponent(fileName)}/health`);

// PTZ vía el servicio del centro (el front solo envía el id/nombre de cámara)
export const centerPtzMove = (camera, move) =>
  request(`/center/ptz/${encodeURIComponent(camera)}/move`, { method: 'POST', body: JSON.stringify(move) });
export const centerPtzStop = (camera) =>
  request(`/center/ptz/${encodeURIComponent(camera)}/stop`, { method: 'POST' });
export const centerPtzGoto = (camera, preset) =>
  request(`/center/ptz/${encodeURIComponent(camera)}/goto`, { method: 'POST', body: JSON.stringify({ preset }) });
export const centerPtzPresets = (camera) =>
  request(`/center/ptz/${encodeURIComponent(camera)}/presets`);

// Reiniciar el servicio de compresión del centro (corta el video unos segundos)
export const restartCompression = () =>
  request('/center/compression/restart', { method: 'POST' });

// Secuencia de arranque del wall: reinicia compresión + MediaMTX (si está
// activo) + servicio PTZ. El backend la deduplica (una vez por ventana de
// tiempo), así que es seguro llamarla al abrir la app aunque haya varias
// ventanas o un F5.
export const centerBootstrap = () =>
  request('/center/bootstrap', { method: 'POST' });

// ── Historial de uso (horas operativas + datos por stream para vigilar Starlink) ──
// Heartbeat: cada ventana reporta su consumo (delta de bytes) y el slot del día.
export const usageHeartbeat = (payload) =>
  request('/usage/heartbeat', { method: 'POST', body: JSON.stringify(payload) });

// Reporte por rango de fechas (admin): días con horas abiertas y bytes por stream.
export const getUsage = (from, to) =>
  request(`/usage?from=${encodeURIComponent(from)}&to=${encodeURIComponent(to)}`);

// Descarga el CSV del rango. Usa fetch con el Bearer y genera un blob, porque
// un <a href> directo no llevaría el token de autenticación.
export async function downloadUsageCSV(from, to) {
  const token = getToken();
  const res = await fetch(
    `${API_BASE}/usage/export?from=${encodeURIComponent(from)}&to=${encodeURIComponent(to)}`,
    { headers: token ? { Authorization: `Bearer ${token}` } : {} },
  );
  if (!res.ok) throw new Error(`Error al exportar (${res.status})`);
  const blob = await res.blob();
  const url = URL.createObjectURL(blob);
  const a = document.createElement('a');
  a.href = url;
  a.download = `uso_${from}_a_${to}.csv`;
  document.body.appendChild(a);
  a.click();
  a.remove();
  URL.revokeObjectURL(url);
}
