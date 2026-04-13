import { useState, useEffect } from 'react';
import {
  listDevices, createDevice, updateDevice, deleteDevice,
  listGrids, createGrid, updateGrid, deleteGrid,
  listStreams, createStream, updateStream, deleteStream, syncStream,
  getScreenConfig, updateScreenConfig,
} from '../api/grids';
import ScreenMapper from '../components/ScreenMapper';
import './Config.css';

export default function Config() {
  const [configTab, setConfigTab] = useState('cameras');

  // Data
  const [devices, setDevices] = useState([]);
  const [grids, setGrids] = useState([]);
  const [streams, setStreams] = useState([]);
  const [screenConfig, setScreenConfig] = useState({ center_name: '', layout: 1, screens: [] });

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
    createStream({ name: `Stream ${streams.length + 1}`, grid_id: grid.id, stream_ip: '', cells })
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
        c.row === row && c.col === col ? { ...c, camera_id: cameraId || undefined } : c
      );
      return { ...prev, cells };
    });
  };

  // ─── SCREEN CONFIG ────────────────────────────────
  const saveScreenConfig = () => {
    updateScreenConfig(screenConfig).then(loadAll).catch(console.error);
  };

  // ─── RENDER ───────────────────────────────────────
  const selectItem = (item) => { setSelectedId(item.id); setForm({ ...item }); };

  const navItems = [
    { key: 'devices', label: 'Dispositivos', icon: 'M15.6 11.6L22 7v10l-6.4-4.6M2 5h13a2 2 0 0 1 2 2v10a2 2 0 0 1-2 2H2z' },
    { key: 'grids', label: 'Grillas', icon: 'M3 3h7v7H3zM14 3h7v7h-7zM3 14h7v7H3zM14 14h7v7h-7z' },
    { key: 'streams', label: 'Streams', icon: 'M2 12s3-7 10-7 10 7 10 7-3 7-10 7-10-7zm0 0M12 9a3 3 0 1 0 0 6 3 3 0 0 0 0-6z' },
    { key: 'screens', label: 'Pantallas', icon: 'M2 3h20v14H2zM8 21h8M12 17v4' },
  ];

  const currentList = configTab === 'devices' ? devices
    : configTab === 'grids' ? grids
    : configTab === 'streams' ? streams
    : [];

  const currentNew = configTab === 'grids' ? newGrid
    : configTab === 'streams' ? newStream
    : null;

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

      {configTab !== 'screens' && (
        <div className="config-sub-sidebar">
          <div className="config-sidebar-header">
            <span className="config-sidebar-title">{navItems.find(n => n.key === configTab)?.label}</span>
            {configTab === 'devices' && (
              <div style={{ display: 'flex', gap: 4 }}>
                <button className="config-add-btn" onClick={() => newDevice('nvr')}>+ NVR</button>
                <button className="config-add-btn" onClick={() => newDevice('camera')}>+ Cam</button>
              </div>
            )}
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

      {/* ── DEVICES EDITOR ── */}
      {configTab === 'devices' && form && (
        <div className="config-main">
          <div className="config-section">
            <div className="config-section-title">{form.type === 'nvr' ? 'NVR' : 'Cámara'}</div>
            <div className="config-form-row">
              <div className="config-form-group"><label>Nombre</label>
                <input value={form.name || ''} onChange={e => setForm(f => ({ ...f, name: e.target.value }))} />
              </div>
              <div className="config-form-group"><label>IP</label>
                <input value={form.ip || ''} onChange={e => setForm(f => ({ ...f, ip: e.target.value }))} placeholder="192.200.60.3" />
              </div>
            </div>

            {/* NVR: credenciales RTSP */}
            {form.type === 'nvr' && (
              <div className="config-form-row">
                <div className="config-form-group"><label>Usuario RTSP</label>
                  <input value={form.user || ''} onChange={e => setForm(f => ({ ...f, user: e.target.value }))} placeholder="admin" />
                </div>
                <div className="config-form-group"><label>Contraseña RTSP</label>
                  <input type="password" value={form.pass || ''} onChange={e => setForm(f => ({ ...f, pass: e.target.value }))} />
                </div>
              </div>
            )}

            {/* CAMERA: todos los campos */}
            {form.type === 'camera' && (
              <>
                <div className="config-form-row">
                  <div className="config-form-group"><label>NVR asociado</label>
                    <select value={form.nvr_id || ''} onChange={e => setForm(f => ({ ...f, nvr_id: e.target.value || undefined }))}>
                      <option value="">-- Sin NVR (directo) --</option>
                      {nvrs.map(n => <option key={n.id} value={n.id}>{n.name} ({n.ip})</option>)}
                    </select>
                  </div>
                  {form.nvr_id && (
                    <div className="config-form-group"><label>Canal NVR</label>
                      <input type="number" min="1" value={form.nvr_channel || 1} onChange={e => setForm(f => ({ ...f, nvr_channel: parseInt(e.target.value) || 1 }))} />
                    </div>
                  )}
                </div>
                <div className="config-form-row">
                  <div className="config-form-group"><label>Tipo de cámara</label>
                    <select value={form.camera_type || 'submarina'} onChange={e => setForm(f => ({ ...f, camera_type: e.target.value }))}>
                      <option value="submarina">Submarina</option>
                      <option value="PTZ">PTZ / Domo</option>
                    </select>
                  </div>
                  <div className="config-form-group"><label>&nbsp;</label>
                    <div className="config-checkbox">
                      <input type="checkbox" checked={form.has_ptz || false} onChange={e => setForm(f => ({ ...f, has_ptz: e.target.checked }))} />
                      <label>Soporta PTZ</label>
                    </div>
                  </div>
                </div>
                <div className="config-form-row">
                  <div className="config-form-group"><label>Jaula</label>
                    <input value={form.cage_name || ''} onChange={e => setForm(f => ({ ...f, cage_name: e.target.value }))} placeholder="J101" />
                  </div>
                  <div className="config-form-group"><label>ID Jaula</label>
                    <input value={form.cage_id || ''} onChange={e => setForm(f => ({ ...f, cage_id: e.target.value }))} />
                  </div>
                </div>
                <div className="config-form-row">
                  <div className="config-form-group"><label>MediaMTX Stream 1</label>
                    <input value={form.mediamtx_camera1 || ''} onChange={e => setForm(f => ({ ...f, mediamtx_camera1: e.target.value }))} placeholder="J101 A-1" />
                  </div>
                  <div className="config-form-group"><label>MediaMTX Stream 2</label>
                    <input value={form.mediamtx_camera2 || ''} onChange={e => setForm(f => ({ ...f, mediamtx_camera2: e.target.value }))} placeholder="J101 A-2" />
                  </div>
                </div>
                <div className="config-form-row">
                  <div className="config-form-group"><label>Modo On-Demand</label>
                    <select value={form.ondemand_mode || 'nvr'} onChange={e => setForm(f => ({ ...f, ondemand_mode: e.target.value }))}>
                      <option value="nvr">Vía NVR</option>
                      <option value="direct">Directo a cámara</option>
                    </select>
                  </div>
                </div>
              </>
            )}
          </div>
          <div className="config-actions">
            <button className="config-delete-btn" onClick={removeDevice}>Eliminar</button>
            <button className="config-save-btn" onClick={saveDevice}>Guardar</button>
          </div>
        </div>
      )}

      {/* ── GRIDS EDITOR ── */}
      {configTab === 'grids' && form && (
        <div className="config-main">
          <div className="config-section">
            <div className="config-section-title">Layout de Grilla</div>
            <div className="config-form-row">
              <div className="config-form-group"><label>Nombre</label>
                <input value={form.name || ''} onChange={e => setForm(f => ({ ...f, name: e.target.value }))} />
              </div>
              <div className="config-form-group"><label>Tipo</label>
                <select value={form.type || 'submarine'} onChange={e => {
                  const t = e.target.value;
                  setForm(f => ({ ...f, type: t, rows: t === 'dome' ? 2 : 4, cols: t === 'dome' ? 2 : 4 }));
                }}>
                  <option value="submarine">Submarinas</option>
                  <option value="dome">Domos</option>
                </select>
              </div>
            </div>
            <div className="config-form-row">
              <div className="config-form-group"><label>Filas</label>
                <input type="number" min="1" max="8" value={form.rows || 4} onChange={e => setForm(f => ({ ...f, rows: parseInt(e.target.value) || 1 }))} />
              </div>
              <div className="config-form-group"><label>Columnas</label>
                <input type="number" min="1" max="8" value={form.cols || 4} onChange={e => setForm(f => ({ ...f, cols: parseInt(e.target.value) || 1 }))} />
              </div>
            </div>
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
          <div className="config-actions">
            <button className="config-delete-btn" onClick={removeGrid}>Eliminar</button>
            <button className="config-save-btn" onClick={saveGrid}>Guardar</button>
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
                <input value={form.file_name || ''} onChange={e => setForm(f => ({ ...f, file_name: e.target.value }))} placeholder="pantalla1" />
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

      {/* Empty states */}
      {configTab !== 'screens' && !form && (
        <div className="config-empty">
          {configTab === 'devices' && 'Selecciona un dispositivo o crea uno nuevo'}
          {configTab === 'grids' && 'Selecciona una grilla o crea una nueva'}
          {configTab === 'streams' && 'Selecciona un stream o crea uno nuevo'}
        </div>
      )}
    </div>
  );
}
