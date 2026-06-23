import { useState, useEffect } from 'react';
import {
  listDevices, createDevice, updateDevice, deleteDevice,
  listGrids, createGrid, updateGrid, deleteGrid,
  listStreams, createStream, updateStream, deleteStream, syncStream,
  getScreenConfig, updateScreenConfig,
  getCenterConfig, updateCenterConfig, getCenterStatus, listCenterScreens, importFromCenter,
  setCenterScreenActive,
  listUsers, createUser, updateUser, deleteUser,
  getUsage, downloadUsageCSV,
} from '../api/grids';
import ScreenMapper from '../components/ScreenMapper';
import './Config.css';

// Formato de datos consumidos: GB con 2 decimales si supera 1 GB, si no MB.
function fmtBytes(b) {
  if (!b) return '0 MB';
  const mb = b / 1e6;
  if (mb >= 1000) return `${(mb / 1000).toFixed(2)} GB`;
  return `${mb.toFixed(1)} MB`;
}
// Segundos a "Xh Ym".
function fmtHours(secs) {
  if (!secs) return '0h';
  const h = Math.floor(secs / 3600);
  const m = Math.round((secs % 3600) / 60);
  return m > 0 ? `${h}h ${m}m` : `${h}h`;
}

export default function Config() {
  const [configTab, setConfigTab] = useState('server');

  // Data
  const [devices, setDevices] = useState([]);
  const [grids, setGrids] = useState([]);
  const [streams, setStreams] = useState([]);
  const [screenConfig, setScreenConfig] = useState({ center_name: '', layout: 1, screens: [] });
  const [centerConfig, setCenterConfig] = useState({ mongo_uri: '', db_name: '', host: '' });
  const [centerStatus, setCenterStatus] = useState(null); // null | 'saving' | {mongo_connected, compressor_connected}
  const [centerScreens, setCenterScreens] = useState(null);
  const [users, setUsers] = useState([]);
  const [newUser, setNewUser] = useState({ username: '', password: '', role: 'operator' });
  const [userError, setUserError] = useState(null);

  // Monitoreo (historial de uso). Rango por defecto: últimos 7 días.
  const [usageRange, setUsageRange] = useState(() => {
    const fmt = (d) => `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`;
    const to = new Date();
    const from = new Date();
    from.setDate(from.getDate() - 6);
    return { from: fmt(from), to: fmt(to) };
  });
  const [usage, setUsage] = useState(null);     // {days, totals, ...} | null
  const [usageState, setUsageState] = useState(null); // null | 'loading' | 'exporting' | {error}

  // Selection
  const [selectedId, setSelectedId] = useState(null);
  const [form, setForm] = useState(null);

  const nvrs = devices.filter(d => d.type === 'nvr');
  const cameras = devices.filter(d => d.type === 'camera');

  const loadAll = () => {
    listDevices().then(setDevices).catch(console.error);
    listGrids().then(setGrids).catch(console.error);
    listStreams().then(setStreams).catch(console.error);
    getScreenConfig().then(setScreenConfig).catch(console.error);
    getCenterConfig().then(setCenterConfig).catch(console.error);
    listCenterScreens().then(setCenterScreens).catch(() => setCenterScreens(null));
    getCenterStatus().then(setCenterStatus).catch(() => {});
    listUsers().then(setUsers).catch(console.error);
  };

  // ─── USUARIOS ─────────────────────────────────────
  const addUser = () => {
    setUserError(null);
    createUser(newUser)
      .then(() => { setNewUser({ username: '', password: '', role: 'operator' }); listUsers().then(setUsers); })
      .catch(err => setUserError(err.message));
  };
  const changeUserRole = (id, role) => updateUser(id, { role }).then(() => listUsers().then(setUsers)).catch(console.error);
  const resetUserPassword = (id) => {
    const pwd = prompt('Nueva contraseña:');
    if (pwd) updateUser(id, { password: pwd }).then(() => alert('Contraseña actualizada')).catch(err => alert(err.message));
  };
  // ─── MONITOREO (historial de uso) ──────────────────
  const loadUsage = () => {
    setUsageState('loading');
    getUsage(usageRange.from, usageRange.to)
      .then(data => { setUsage(data); setUsageState(null); })
      .catch(err => { setUsage(null); setUsageState({ error: err.message }); });
  };
  const exportUsage = () => {
    setUsageState('exporting');
    downloadUsageCSV(usageRange.from, usageRange.to)
      .then(() => setUsageState(null))
      .catch(err => setUsageState({ error: err.message }));
  };
  // Carga automática al entrar a la pestaña o cambiar el rango
  useEffect(() => {
    if (configTab === 'monitor') loadUsage();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [configTab, usageRange.from, usageRange.to]);

  const removeUser = (id) => {
    if (confirm('¿Eliminar este usuario?')) deleteUser(id).then(() => listUsers().then(setUsers)).catch(err => alert(err.message));
  };

  useEffect(() => { loadAll(); }, []);

  const clearSelection = () => { setSelectedId(null); setForm(null); };

  // ─── DEVICES ──────────────────────────────────────
  const newDevice = (type) => {
    const name = type === 'nvr' ? `NVR ${nvrs.length + 1}` : `Cámara ${cameras.length + 1}`;
    createDevice({ name, type, ondemand_mode: type === 'camera' ? 'nvr' : '' })
      .then(d => { loadAll(); setSelectedId(d.id); setForm({ ...d }); })
      .catch(console.error);
  };

  const saveDevice = () => {
    if (!form || !selectedId) return;
    updateDevice(selectedId, form).then(loadAll).catch(console.error);
  };

  const removeDevice = () => {
    if (!selectedId) return;
    deleteDevice(selectedId).then(() => { loadAll(); clearSelection(); }).catch(console.error);
  };

  // ─── GRIDS ────────────────────────────────────────
  const newGrid = () => {
    createGrid({ name: `Grilla ${grids.length + 1}`, type: 'submarine', rows: 4, cols: 4 })
      .then(g => { loadAll(); setSelectedId(g.id); setForm({ ...g }); })
      .catch(console.error);
  };

  const saveGrid = () => {
    if (!form || !selectedId) return;
    updateGrid(selectedId, form).then(loadAll).catch(console.error);
  };

  const removeGrid = () => {
    if (!selectedId) return;
    deleteGrid(selectedId).then(() => { loadAll(); clearSelection(); }).catch(console.error);
  };

  // ─── STREAMS ──────────────────────────────────────
  const newStream = () => {
    const grid = grids[0];
    if (!grid) return alert('Crea una grilla primero');
    const cells = [];
    for (let r = 0; r < grid.rows; r++)
      for (let c = 0; c < grid.cols; c++)
        cells.push({ row: r, col: c });
    createStream({
      name: `Stream ${streams.length + 1}`, grid_id: grid.id, stream_ip: '',
      file_name: '', ip_server: 'localhost', is_active: false,
      width_resolution: 1920, height_resolution: 1080,
      select_flow: 1, bitrate: 3000, fps: 20, gop: 25,
      hardware_encoding: 1, pc_id: 0, cells,
    })
      .then(s => { loadAll(); setSelectedId(s.id); setForm({ ...s }); })
      .catch(console.error);
  };

  const [syncStatus, setSyncStatus] = useState(null);

  const saveStream = () => {
    if (!form || !selectedId) return;
    updateStream(selectedId, form)
      .then(() => { loadAll(); setSyncStatus('synced'); setTimeout(() => setSyncStatus(null), 3000); })
      .catch(err => { console.error(err); setSyncStatus('error'); });
  };

  const manualSync = () => {
    if (!selectedId) return;
    setSyncStatus('syncing');
    syncStream(selectedId)
      .then(() => { setSyncStatus('synced'); setTimeout(() => setSyncStatus(null), 3000); })
      .catch(err => { console.error(err); setSyncStatus('error'); });
  };

  const removeStream = () => {
    if (!selectedId) return;
    deleteStream(selectedId).then(() => { loadAll(); clearSelection(); }).catch(console.error);
  };

  const changeStreamGrid = (gridId) => {
    const grid = grids.find(g => g.id === gridId);
    if (!grid) return;
    const cells = [];
    for (let r = 0; r < grid.rows; r++)
      for (let c = 0; c < grid.cols; c++) {
        const existing = form.cells?.find(cl => cl.row === r && cl.col === c);
        cells.push(existing || { row: r, col: c });
      }
    setForm(prev => ({ ...prev, grid_id: gridId, cells }));
  };

  const assignCameraToCell = (row, col, cameraId) => {
    setForm(prev => {
      const cells = prev.cells.map(c =>
        c.row === row && c.col === col
          ? { ...c, camera_id: cameraId || undefined, active: !!cameraId }
          : c
      );
      return { ...prev, cells };
    });
  };

  // ─── SCREEN CONFIG ────────────────────────────────
  const saveScreenConfig = () => {
    updateScreenConfig(screenConfig).then(loadAll).catch(console.error);
  };

  // ─── CENTER CONFIG (servidor) ─────────────────────
  const saveCenterConfig = () => {
    setCenterStatus('saving');
    updateCenterConfig(centerConfig)
      .then(res => {
        setCenterStatus(res);
        listCenterScreens().then(setCenterScreens).catch(() => setCenterScreens(null));
      })
      .catch(err => {
        console.error(err);
        setCenterStatus({ error: err.message });
      });
  };

  const refreshCenter = () => {
    setCenterStatus('refreshing');
    Promise.all([
      getCenterStatus().catch(() => ({ error: 'sin respuesta del backend' })),
      listCenterScreens().catch(() => null),
    ]).then(([status, screens]) => {
      setCenterStatus(status);
      setCenterScreens(screens);
    });
  };

  const toggleCenterScreen = (fileName, active) => {
    setCenterScreenActive(fileName, active)
      .then(() => listCenterScreens().then(setCenterScreens))
      .catch(err => alert('No se pudo cambiar el estado: ' + err.message));
  };

  const [importStatus, setImportStatus] = useState(null); // null | 'importing' | {counts} | {error}
  const runImport = () => {
    setImportStatus('importing');
    importFromCenter()
      .then(res => {
        setImportStatus(res);
        loadAll(); // refrescar dispositivos, grillas y streams importados
      })
      .catch(err => {
        console.error(err);
        setImportStatus({ error: err.message });
      });
  };

  // ─── RENDER ───────────────────────────────────────
  const streamDefaults = {
    width_resolution: 1920, height_resolution: 1080,
    bitrate: 3000, fps: 20, gop: 25,
    hardware_encoding: 1, select_flow: 1, pc_id: 0,
  };

  const selectItem = (item) => {
    setSelectedId(item.id);
    // Apply defaults to stream fields that are 0/undefined
    if (configTab === 'streams') {
      const merged = { ...item };
      for (const [key, def] of Object.entries(streamDefaults)) {
        if (!merged[key]) merged[key] = def;
      }
      setForm(merged);
    } else {
      setForm({ ...item });
    }
  };

  const navItems = [
    { key: 'server', label: 'Servidor', icon: 'M2 2h20v8H2zM2 14h20v8H2zM6 6h.01M6 18h.01' },
    { key: 'devices', label: 'Dispositivos', icon: 'M15.6 11.6L22 7v10l-6.4-4.6M2 5h13a2 2 0 0 1 2 2v10a2 2 0 0 1-2 2H2z' },
    { key: 'grids', label: 'Grillas', icon: 'M3 3h7v7H3zM14 3h7v7h-7zM3 14h7v7H3zM14 14h7v7h-7z' },
    { key: 'streams', label: 'Streams', icon: 'M1 4v6h6 M23 20v-6h-6 M20.49 9A9 9 0 0 0 5.64 5.64L1 10 M3.51 15A9 9 0 0 0 18.36 18.36L23 14' },
    { key: 'screens', label: 'Pantallas', icon: 'M2 3h20v14H2zM8 21h8M12 17v4' },
    { key: 'monitor', label: 'Monitoreo', icon: 'M3 3v18h18 M7 14l4-4 3 3 5-6' },
    { key: 'users', label: 'Usuarios', icon: 'M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2M9 11a4 4 0 1 0 0-8 4 4 0 0 0 0 8M23 21v-2a4 4 0 0 0-3-3.87M16 3.13a4 4 0 0 1 0 7.75' },
  ];

  const currentList = configTab === 'devices' ? devices
    : configTab === 'grids' ? grids
    : configTab === 'streams' ? streams
    : [];

  // Dispositivos y grillas vienen de "Importar del Centro"; solo los streams
  // pueden crearse a mano si hiciera falta
  const currentNew = configTab === 'streams' ? newStream : null;

  return (
    <div className="config">
      <div className="config-nav">
        <div className="config-nav-header">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" width="18" height="18">
            <circle cx="12" cy="12" r="3" />
            <path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1-2.83 2.83l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-4 0v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83-2.83l.06-.06A1.65 1.65 0 0 0 4.68 15a1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1 0-4h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 2.83-2.83l.06.06A1.65 1.65 0 0 0 9 4.68a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 4 0v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 2.83l-.06.06A1.65 1.65 0 0 0 19.4 9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 0 4h-.09a1.65 1.65 0 0 0-1.51 1z" />
          </svg>
        </div>
        {navItems.map(item => (
          <button
            key={item.key}
            className={`config-nav-item ${configTab === item.key ? 'active' : ''}`}
            onClick={() => { setConfigTab(item.key); clearSelection(); }}
            title={item.label}
          >
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round">
              <path d={item.icon} />
            </svg>
            <span className="config-nav-label">{item.label}</span>
          </button>
        ))}
      </div>

      {configTab !== 'screens' && configTab !== 'server' && configTab !== 'users' && configTab !== 'monitor' && (
        <div className="config-sub-sidebar">
          <div className="config-sidebar-header">
            <span className="config-sidebar-title">{navItems.find(n => n.key === configTab)?.label}</span>
            {currentNew && <button className="config-add-btn" onClick={currentNew}>+ Nueva</button>}
          </div>
          <div className="config-grid-list">
            {configTab === 'devices' && nvrs.length > 0 && (
              <div className="config-list-group">NVRs</div>
            )}
            {configTab === 'devices' && nvrs.map(item => (
              <div key={item.id} className={`config-grid-item ${selectedId === item.id ? 'active' : ''}`} onClick={() => selectItem(item)}>
                <div className="config-grid-item-name">{item.name}</div>
                <div className="config-grid-item-meta">{item.ip || 'Sin IP'}</div>
              </div>
            ))}
            {configTab === 'devices' && cameras.length > 0 && (
              <div className="config-list-group">Cámaras</div>
            )}
            {configTab === 'devices' && cameras.map(item => {
              const nvr = nvrs.find(n => n.id === item.nvr_id);
              return (
                <div key={item.id} className={`config-grid-item ${selectedId === item.id ? 'active' : ''}`} onClick={() => selectItem(item)}>
                  <div className="config-grid-item-name">{item.name}</div>
                  <div className="config-grid-item-meta">{nvr ? `${nvr.name} ch${item.nvr_channel}` : item.ip || 'Sin config'} {item.has_ptz ? '· PTZ' : ''}</div>
                </div>
              );
            })}
            {configTab !== 'devices' && currentList.map(item => (
              <div key={item.id} className={`config-grid-item ${selectedId === item.id ? 'active' : ''}`} onClick={() => selectItem(item)}>
                <div className="config-grid-item-name">{item.name}</div>
                <div className="config-grid-item-meta">
                  {configTab === 'grids' && `${item.type === 'submarine' ? 'Submarinas' : 'Domos'} · ${item.rows}x${item.cols}`}
                  {configTab === 'streams' && `${item.stream_ip || 'Sin stream IP'}`}
                </div>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* ── SERVER (conexión al centro) ── */}
      {configTab === 'server' && (
        <div className="config-main">
          <div className="config-section">
            <div className="config-section-title">Conexión al Servidor del Centro</div>
            <div className="config-form-row">
              <div className="config-form-group"><label>Host del centro (zoom :8087 / WHEP :8889)</label>
                <input value={centerConfig.host || ''} onChange={e => setCenterConfig(c => ({ ...c, host: e.target.value }))} placeholder="10.1.1.229" />
              </div>
            </div>
            <div className="config-form-row">
              <div className="config-form-group"><label>URI de MongoDB</label>
                <input value={centerConfig.mongo_uri || ''} onChange={e => setCenterConfig(c => ({ ...c, mongo_uri: e.target.value }))} placeholder="mongodb://10.1.1.229:27017" />
              </div>
              <div className="config-form-group"><label>Base de datos</label>
                <input value={centerConfig.db_name || ''} onChange={e => setCenterConfig(c => ({ ...c, db_name: e.target.value }))} placeholder="camancha_vsmweb" />
              </div>
            </div>
            <div className="config-form-row config-server-actions">
              <button
                className="config-save-btn"
                onClick={saveCenterConfig}
                disabled={centerStatus === 'saving' || centerStatus === 'refreshing'}
              >
                {centerStatus === 'saving' ? 'Conectando...' : 'Guardar y Conectar'}
              </button>
              <button
                className="config-sync-btn"
                onClick={refreshCenter}
                disabled={centerStatus === 'saving' || centerStatus === 'refreshing'}
              >
                {centerStatus === 'refreshing' ? 'Actualizando...' : 'Actualizar'}
              </button>
              {centerStatus && typeof centerStatus === 'object' && (
                centerStatus.error ? (
                  <span className="config-sync-err">Error: {centerStatus.error}</span>
                ) : (
                  <>
                    <span className={centerStatus.mongo_connected ? 'config-sync-ok' : 'config-sync-err'}>
                      MongoDB: {centerStatus.mongo_connected ? 'conectado' : 'sin conexión'}
                    </span>
                    <span className={centerStatus.compressor_connected ? 'config-sync-ok' : 'config-sync-err'}>
                      Compresor: {centerStatus.compressor_connected ? 'conectado' : 'sin conexión'}
                    </span>
                  </>
                )
              )}
            </div>
          </div>

          <div className="config-section">
            <div className="config-section-title">Pantallas del Centro (screen_configuration)</div>
            {centerScreens === null && (
              <div className="config-center-empty">Sin conexión a la base de datos del centro</div>
            )}
            {centerScreens?.length === 0 && (
              <div className="config-center-empty">La colección screen_configuration está vacía</div>
            )}
            {centerScreens?.map(s => (
              <div key={s.file_name} className="config-center-screen">
                <span className={`config-center-dot ${s.active ? 'on' : 'off'}`} />
                <span className="config-center-name">{s.file_name}</span>
                <span className="config-center-meta">
                  {s.rows}x{s.cols} · {s.width_resolution}x{s.height_resolution} · {s.cells.filter(c => c.on).length} cámaras · {s.active ? 'transmitiendo' : 'inactiva'}
                </span>
                <button
                  className={`config-center-toggle ${s.active ? 'on' : ''}`}
                  onClick={() => toggleCenterScreen(s.file_name, !s.active)}
                  title={s.active ? 'Detener transmisión' : 'Activar transmisión'}
                >
                  {s.active ? 'Detener' : 'Activar'}
                </button>
              </div>
            ))}
          </div>

          <div className="config-section">
            <div className="config-section-title">Importar al VMS</div>
            <p className="config-import-hint">
              Crea/actualiza en la base local las cámaras, grillas y streams con sus coordenadas,
              leyendo la configuración que corre en el centro. Luego, en <b>Dispositivos</b>, marca
              qué cámaras son PTZ para darles controles. Las ediciones locales se preservan al reimportar.
            </p>
            <div className="config-form-row">
              <button className="config-save-btn" onClick={runImport} disabled={importStatus === 'importing'}>
                {importStatus === 'importing' ? 'Importando...' : 'Importar del Centro'}
              </button>
              {importStatus?.counts && (
                <span className="config-sync-ok">
                  Importado: {importStatus.counts.cameras} cámaras, {importStatus.counts.nvrs} NVRs,{' '}
                  {importStatus.counts.grids} grillas, {importStatus.counts.streams} streams
                </span>
              )}
              {importStatus?.error && (
                <span className="config-sync-err">Error: {importStatus.error}</span>
              )}
            </div>
          </div>

        </div>
      )}

      {/* ── DEVICES (importados; solo se edita PTZ + OSD global) ── */}
      {configTab === 'devices' && (
        <div className="config-main">
          <div className="config-section">
            <div className="config-section-title">OSD de las Celdas</div>
            <p className="config-import-hint">
              Tamaño y posición de las etiquetas (nombre de jaula) y color de las líneas
              de la grilla que se dibujan sobre el mosaico.
            </p>
            <div className="config-form-row">
              <div className="config-form-group"><label>Tamaño (px)</label>
                <input
                  type="number" min="8" max="48"
                  value={screenConfig.osd_size || 10}
                  onChange={e => setScreenConfig(p => ({ ...p, osd_size: parseInt(e.target.value) || 10 }))}
                />
              </div>
              <div className="config-form-group"><label>Posición</label>
                <select
                  value={screenConfig.osd_position || 'top-left'}
                  onChange={e => setScreenConfig(p => ({ ...p, osd_position: e.target.value }))}
                >
                  <option value="top-left">Superior izquierda</option>
                  <option value="top-right">Superior derecha</option>
                  <option value="bottom-left">Inferior izquierda</option>
                  <option value="bottom-right">Inferior derecha</option>
                </select>
              </div>
            </div>
            <div className="config-form-row">
              <div className="config-form-group"><label>Color de líneas de grilla</label>
                <div className="config-color-row">
                  <input
                    type="color"
                    value={screenConfig.grid_color || '#ffffff'}
                    onChange={e => setScreenConfig(p => ({ ...p, grid_color: e.target.value }))}
                  />
                  <div className="config-color-presets">
                    {[
                      ['#ffffff', 'Blanco'],
                      ['#ffaa3c', 'Ámbar'],
                      ['#ffd84d', 'Amarillo'],
                      ['#ff7a5c', 'Coral'],
                    ].map(([hex, name]) => (
                      <button
                        key={hex}
                        type="button"
                        className={`config-color-preset ${(screenConfig.grid_color || '#ffffff') === hex ? 'active' : ''}`}
                        style={{ background: hex }}
                        title={name}
                        onClick={() => setScreenConfig(p => ({ ...p, grid_color: hex }))}
                      />
                    ))}
                  </div>
                </div>
              </div>
            </div>
            <div className="config-form-row config-server-actions">
              <button className="config-save-btn" onClick={saveScreenConfig}>Guardar OSD</button>
              <span className="config-import-hint" style={{ margin: 0 }}>Se aplica al recargar las pantallas</span>
            </div>
          </div>

          {form ? (
            <div className="config-section">
              <div className="config-section-title">{form.type === 'nvr' ? `NVR · ${form.name}` : `Cámara · ${form.name}`}</div>

              <div className="config-info-list">
                <div className="config-info-row"><span>IP</span><b>{form.ip || '—'}</b></div>
                {form.type === 'camera' && (
                  <>
                    <div className="config-info-row"><span>NVR</span>
                      <b>{(() => { const n = nvrs.find(x => x.id === form.nvr_id); return n ? `${n.name} (${n.ip}) · canal ${form.nvr_channel || '—'}` : 'Directo'; })()}</b>
                    </div>
                    <div className="config-info-row"><span>MediaMTX</span><b>{form.mediamtx_camera1 || '—'} / {form.mediamtx_camera2 || '—'}</b></div>
                  </>
                )}
                {form.type === 'nvr' && (
                  <div className="config-info-row"><span>Usuario</span><b>{form.user || '—'}</b></div>
                )}
              </div>

              {form.type === 'camera' && (
                <>
                  <div className="config-form-row" style={{ marginTop: 12 }}>
                    <div className="config-form-group"><label>&nbsp;</label>
                      <div className="config-checkbox">
                        <input
                          type="checkbox"
                          checked={form.has_ptz || false}
                          onChange={e => setForm(f => ({
                            ...f,
                            has_ptz: e.target.checked,
                            camera_type: e.target.checked ? 'PTZ' : 'submarina',
                          }))}
                        />
                        <label>Es PTZ (muestra controles de movimiento en pantalla)</label>
                      </div>
                    </div>
                  </div>
                  <div className="config-actions">
                    <button className="config-save-btn" onClick={saveDevice}>Guardar</button>
                  </div>
                </>
              )}
            </div>
          ) : (
            <div className="config-section">
              <p className="config-import-hint" style={{ margin: 0 }}>
                Selecciona una cámara de la lista para ver su información y marcar si es PTZ.
                Las cámaras y NVRs se crean y actualizan con <b>Importar del Centro</b> (sección Servidor).
              </p>
            </div>
          )}
        </div>
      )}

      {/* ── GRIDS (importadas; solo visor) ── */}
      {configTab === 'grids' && form && (
        <div className="config-main">
          <div className="config-section">
            <div className="config-section-title">Grilla · {form.name}</div>
            <div className="config-info-list">
              <div className="config-info-row"><span>Dimensiones</span><b>{form.rows} filas × {form.cols} columnas</b></div>
            </div>
            <p className="config-import-hint" style={{ marginTop: 10 }}>
              Las grillas reflejan la configuración que corre en el centro y se actualizan
              con <b>Importar del Centro</b>.
            </p>
          </div>
          <div className="config-section">
            <div className="config-section-title">Vista previa</div>
            <div className="config-cells-grid" style={{
              gridTemplateColumns: `repeat(${form.cols}, 1fr)`,
              gridTemplateRows: `repeat(${form.rows}, 1fr)`,
            }}>
              {Array.from({ length: (form.rows || 0) * (form.cols || 0) }, (_, i) => (
                <div key={i} className="config-cell">
                  <div className="config-cell-pos">[{Math.floor(i / form.cols)},{i % form.cols}]</div>
                </div>
              ))}
            </div>
          </div>
        </div>
      )}

      {/* ── STREAMS EDITOR ── */}
      {configTab === 'streams' && form && (
        <div className="config-main">
          {/* Stream identification */}
          <div className="config-section">
            <div className="config-section-title">Identificación</div>
            <div className="config-form-row">
              <div className="config-form-group"><label>Nombre</label>
                <input value={form.name || ''} onChange={e => setForm(f => ({ ...f, name: e.target.value }))} />
              </div>
              <div className="config-form-group"><label>Nombre archivo (URL RTSP)</label>
                <input value={form.file_name || ''} onChange={e => setForm(f => ({ ...f, file_name: e.target.value }))} placeholder="pantalla1 (solo nombre, sin URL)" />
              </div>
            </div>
            <div className="config-form-row">
              <div className="config-form-group"><label>Grilla</label>
                <select value={form.grid_id || ''} onChange={e => changeStreamGrid(e.target.value)}>
                  <option value="">-- Seleccionar --</option>
                  {grids.map(g => <option key={g.id} value={g.id}>{g.name} ({g.rows}x{g.cols})</option>)}
                </select>
              </div>
              <div className="config-form-group"><label>Stream WHEP (visualización)</label>
                <input value={form.stream_ip || ''} onChange={e => setForm(f => ({ ...f, stream_ip: e.target.value }))} placeholder="http://10.1.1.220:8889/pantalla1/" />
              </div>
            </div>
          </div>

          {/* Compression settings (pontón) */}
          <div className="config-section">
            <div className="config-section-title">Compresión (Pontón)</div>
            <div className="config-form-row">
              <div className="config-form-group"><label>IP Servidor</label>
                <input value={form.ip_server || ''} onChange={e => setForm(f => ({ ...f, ip_server: e.target.value }))} placeholder="localhost" />
              </div>
              <div className="config-form-group"><label>Activo</label>
                <div className="config-checkbox">
                  <input type="checkbox" checked={form.is_active || false} onChange={e => setForm(f => ({ ...f, is_active: e.target.checked }))} />
                  <label>Transmitir</label>
                </div>
              </div>
            </div>
            <div className="config-form-row">
              <div className="config-form-group"><label>Resolución ancho</label>
                <input type="number" value={form.width_resolution || 1920} onChange={e => setForm(f => ({ ...f, width_resolution: parseInt(e.target.value) || 1920 }))} />
              </div>
              <div className="config-form-group"><label>Resolución alto</label>
                <input type="number" value={form.height_resolution || 1080} onChange={e => setForm(f => ({ ...f, height_resolution: parseInt(e.target.value) || 1080 }))} />
              </div>
            </div>
            <div className="config-form-row">
              <div className="config-form-group"><label>Fuente RTSP</label>
                <select value={form.select_flow || 1} onChange={e => setForm(f => ({ ...f, select_flow: parseInt(e.target.value) }))}>
                  <option value={1}>1 - NVR primario</option>
                  <option value={2}>2 - NVR secundario</option>
                  <option value={3}>3 - MediaMTX camera1</option>
                  <option value={4}>4 - MediaMTX camera2</option>
                  <option value={5}>5 - Cámara directa</option>
                  <option value={6}>6 - Cámara directa (sec)</option>
                </select>
              </div>
              <div className="config-form-group"><label>Bitrate (kbps)</label>
                <input type="number" value={form.bitrate || 3000} onChange={e => setForm(f => ({ ...f, bitrate: parseInt(e.target.value) || 3000 }))} />
              </div>
            </div>
            <div className="config-form-row">
              <div className="config-form-group"><label>FPS</label>
                <input type="number" value={form.fps || 20} onChange={e => setForm(f => ({ ...f, fps: parseInt(e.target.value) || 20 }))} />
              </div>
              <div className="config-form-group"><label>GOP</label>
                <input type="number" value={form.gop || 25} onChange={e => setForm(f => ({ ...f, gop: parseInt(e.target.value) || 25 }))} />
              </div>
            </div>
            <div className="config-form-row">
              <div className="config-form-group"><label>Codificación HW</label>
                <select value={form.hardware_encoding || 1} onChange={e => setForm(f => ({ ...f, hardware_encoding: parseInt(e.target.value) }))}>
                  <option value={1}>1 - CPU</option>
                  <option value={2}>2 - GPU completo</option>
                  <option value={3}>3 - GPU dec + CPU comp + GPU enc</option>
                  <option value={4}>4 - CPU dec + GPU enc</option>
                </select>
              </div>
              <div className="config-form-group"><label>PC ID</label>
                <input type="number" value={form.pc_id || 0} onChange={e => setForm(f => ({ ...f, pc_id: parseInt(e.target.value) || 0 }))} />
              </div>
            </div>
          </div>

          {/* Camera cell assignments */}
          {form.grid_id && (
            <div className="config-section">
              <div className="config-section-title">Asignar Cámaras a Celdas</div>
              {(() => {
                const grid = grids.find(g => g.id === form.grid_id);
                if (!grid) return null;
                return (
                  <div className="config-cells-grid" style={{
                    gridTemplateColumns: `repeat(${grid.cols}, 1fr)`,
                    gridTemplateRows: `repeat(${grid.rows}, 1fr)`,
                  }}>
                    {form.cells?.map(cell => {
                      const cam = cameras.find(c => c.id === cell.camera_id);
                      return (
                        <div key={`${cell.row}-${cell.col}`} className={`config-cell ${cam ? 'active' : ''}`}>
                          <div className="config-cell-pos">[{cell.row},{cell.col}]</div>
                          <select
                            className="config-cell-select"
                            value={cell.camera_id || ''}
                            onChange={e => assignCameraToCell(cell.row, cell.col, e.target.value)}
                          >
                            <option value="">Vacío</option>
                            {cameras.map(c => <option key={c.id} value={c.id}>{c.name}</option>)}
                          </select>
                          {cam && <div className="config-cell-detail">{cam.cage_name || cam.ip}</div>}
                        </div>
                      );
                    })}
                  </div>
                );
              })()}
            </div>
          )}

          <div className="config-actions">
            <button className="config-delete-btn" onClick={removeStream}>Eliminar</button>
            <button className="config-sync-btn" onClick={manualSync} disabled={syncStatus === 'syncing'}>
              {syncStatus === 'syncing' ? 'Sincronizando...' : 'Sync Pontón'}
            </button>
            <button className="config-save-btn" onClick={saveStream}>Guardar</button>
            {syncStatus === 'synced' && <span className="config-sync-ok">Sincronizado</span>}
            {syncStatus === 'error' && <span className="config-sync-err">Error de sync</span>}
          </div>
        </div>
      )}

      {/* ── SCREENS ── */}
      {configTab === 'screens' && (
        <div className="config-main">
          <div className="config-section">
            <div className="config-section-title">Centro</div>
            <div className="config-form-row">
              <div className="config-form-group"><label>Nombre del Centro</label>
                <input value={screenConfig.center_name || ''} onChange={e => setScreenConfig(p => ({ ...p, center_name: e.target.value }))} placeholder="Centro Oguri" />
              </div>
            </div>
          </div>
          <div className="config-section">
            <div className="config-section-title">Pantallas Físicas</div>
            <ScreenMapper
              grids={streams}
              screens={screenConfig.screens}
              onScreensChange={(newScreens, newLayout) => {
                setScreenConfig(p => ({ ...p, screens: newScreens, layout: newLayout !== undefined ? newLayout : p.layout }));
              }}
            />
          </div>
          <div className="config-actions">
            <button className="config-save-btn" onClick={saveScreenConfig}>Guardar Configuración</button>
          </div>
        </div>
      )}

      {/* ── MONITOREO (historial de uso / Starlink) ── */}
      {configTab === 'monitor' && (
        <div className="config-main">
          <div className="config-section">
            <div className="config-section-title">Historial de Uso</div>
            <p className="config-monitor-hint">
              Horas de operación del wall y datos consumidos por cada stream (medidos en el navegador,
              equivalen a lo que baja por el enlace Starlink).
            </p>
            <div className="config-form-row config-monitor-controls">
              <div className="config-form-group">
                <label>Desde</label>
                <input type="date" value={usageRange.from}
                  max={usageRange.to}
                  onChange={e => setUsageRange(r => ({ ...r, from: e.target.value }))} />
              </div>
              <div className="config-form-group">
                <label>Hasta</label>
                <input type="date" value={usageRange.to}
                  min={usageRange.from}
                  onChange={e => setUsageRange(r => ({ ...r, to: e.target.value }))} />
              </div>
              <button className="config-sync-btn" onClick={loadUsage} disabled={usageState === 'loading'}>
                {usageState === 'loading' ? 'Cargando...' : 'Actualizar'}
              </button>
              <button className="config-save-btn" onClick={exportUsage}
                disabled={usageState === 'exporting' || !usage?.days?.length}>
                {usageState === 'exporting' ? 'Exportando...' : 'Exportar CSV'}
              </button>
            </div>

            {usageState?.error && (
              <div className="config-sync-err">Error: {usageState.error}</div>
            )}

            {usage && (
              <>
                <div className="config-monitor-totals">
                  <div className="config-monitor-card">
                    <span className="config-monitor-card-label">Horas operativas</span>
                    <span className="config-monitor-card-value">{fmtHours(usage.totals.open_seconds)}</span>
                  </div>
                  <div className="config-monitor-card">
                    <span className="config-monitor-card-label">Datos totales</span>
                    <span className="config-monitor-card-value">{fmtBytes(usage.totals.bytes_total)}</span>
                  </div>
                  <div className="config-monitor-card">
                    <span className="config-monitor-card-label">Días con actividad</span>
                    <span className="config-monitor-card-value">{usage.days.length}</span>
                  </div>
                </div>

                {usage.days.length === 0 ? (
                  <div className="config-center-empty">Sin registros de uso en este rango</div>
                ) : (
                  <table className="config-monitor-table">
                    <thead>
                      <tr><th>Fecha</th><th>Horas abiertas</th><th>Datos</th><th>Streams</th></tr>
                    </thead>
                    <tbody>
                      {usage.days.map(d => (
                        <tr key={d.date}>
                          <td>{d.date}</td>
                          <td>{fmtHours(d.open_seconds)}</td>
                          <td>{fmtBytes(d.bytes_total)}</td>
                          <td className="config-monitor-streams">
                            {Object.entries(d.bytes_by_stream).sort((a, b) => b[1] - a[1]).map(([name, b]) => (
                              <span key={name} className="config-monitor-stream-pill">
                                {name}: {fmtBytes(b)}
                              </span>
                            ))}
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                )}

                {/* Totales por stream del rango */}
                {Object.keys(usage.totals.bytes_by_stream || {}).length > 0 && (
                  <div className="config-monitor-by-stream">
                    <div className="config-section-subtitle">Consumo por stream (rango)</div>
                    {Object.entries(usage.totals.bytes_by_stream).sort((a, b) => b[1] - a[1]).map(([name, b]) => (
                      <div key={name} className="config-monitor-stream-row">
                        <span className="config-monitor-stream-name">{name}</span>
                        <span className="config-monitor-stream-bytes">{fmtBytes(b)}</span>
                      </div>
                    ))}
                  </div>
                )}
              </>
            )}
          </div>
        </div>
      )}

      {/* ── USUARIOS ── */}
      {configTab === 'users' && (
        <div className="config-main">
          <div className="config-section">
            <div className="config-section-title">Cuentas de Usuario</div>
            <div className="config-info-list">
              {users.map(u => (
                <div key={u.id} className="config-user-row">
                  <span className="config-user-name">{u.username}</span>
                  <select
                    className="config-user-role"
                    value={u.role}
                    onChange={e => changeUserRole(u.id, e.target.value)}
                  >
                    <option value="admin">Administrador</option>
                    <option value="operator">Operador</option>
                  </select>
                  <button className="config-sync-btn" onClick={() => resetUserPassword(u.id)}>Cambiar clave</button>
                  <button className="config-delete-btn" onClick={() => removeUser(u.id)}>Eliminar</button>
                </div>
              ))}
            </div>
          </div>

          <div className="config-section">
            <div className="config-section-title">Nuevo Usuario</div>
            <div className="config-form-row">
              <div className="config-form-group"><label>Usuario</label>
                <input value={newUser.username} onChange={e => setNewUser(u => ({ ...u, username: e.target.value }))} />
              </div>
              <div className="config-form-group"><label>Contraseña</label>
                <input type="password" value={newUser.password} onChange={e => setNewUser(u => ({ ...u, password: e.target.value }))} />
              </div>
              <div className="config-form-group"><label>Rol</label>
                <select value={newUser.role} onChange={e => setNewUser(u => ({ ...u, role: e.target.value }))}>
                  <option value="operator">Operador</option>
                  <option value="admin">Administrador</option>
                </select>
              </div>
            </div>
            {userError && <div className="config-form-row"><span className="config-sync-err">{userError}</span></div>}
            <div className="config-actions">
              <button className="config-save-btn" onClick={addUser} disabled={!newUser.username || !newUser.password}>Crear Usuario</button>
            </div>
          </div>

          <div className="config-section">
            <p className="config-import-hint" style={{ margin: 0 }}>
              <b>Operador</b>: ve el muro de pantallas y opera zoom/PTZ. <b>Administrador</b>: además accede a toda la Configuración.
              Para el muro 24/7, inicia sesión en el PC de las pantallas marcando <b>"Recordar sesión"</b>.
            </p>
          </div>
        </div>
      )}

      {/* Empty states */}
      {(configTab === 'grids' || configTab === 'streams') && !form && (
        <div className="config-empty">
          {configTab === 'grids' && 'Selecciona una grilla para ver su layout'}
          {configTab === 'streams' && 'Selecciona un stream o crea uno nuevo'}
        </div>
      )}
    </div>
  );
}
