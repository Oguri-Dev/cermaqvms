import { useEffect } from 'react';
import { usePtz } from '../hooks/usePtz';
import './PTZControls.css';

// Panel PTZ modal (right-click sobre una celda PTZ). Comparte la lógica de
// movimiento/keepalive con el overlay de hover (hooks/usePtz.js).
export default function PTZControls({ cameraName, onClose }) {
  const { startMove, stopMove, gotoPreset, presets, loadPresets, error } = usePtz(cameraName);

  useEffect(() => { loadPresets(); }, [loadPresets]);

  const moveBtn = (direction, label, className = 'ptz-btn') => (
    <button
      className={className}
      onPointerDown={() => startMove(direction)}
      onPointerUp={stopMove}
      onPointerLeave={stopMove}
      onContextMenu={(e) => e.preventDefault()}
    >
      {label}
    </button>
  );

  return (
    <div className="ptz-overlay" onClick={onClose}>
      <div className="ptz-panel" onClick={(e) => e.stopPropagation()}>
        <div className="ptz-header">
          <div>
            <div className="ptz-title">Control PTZ</div>
            <div className="ptz-subtitle">{cameraName}</div>
          </div>
          <button className="ptz-close" onClick={onClose}>&times;</button>
        </div>

        <div className="ptz-dpad">
          <div />
          {moveBtn('up', '▲')}
          <div />
          {moveBtn('left', '◀')}
          <button className="ptz-btn center">PTZ</button>
          {moveBtn('right', '▶')}
          <div />
          {moveBtn('down', '▼')}
          <div />
        </div>

        <div>
          <div className="ptz-zoom-label">Zoom</div>
          <div className="ptz-zoom">
            {moveBtn('zoom_out', '−', 'ptz-zoom-btn')}
            {moveBtn('zoom_in', '+', 'ptz-zoom-btn')}
          </div>
        </div>

        <div className="ptz-presets">
          <div className="ptz-presets-label">Presets</div>
          {presets === null && <div className="ptz-presets-empty">Cargando...</div>}
          {presets?.length === 0 && <div className="ptz-presets-empty">Sin presets configurados</div>}
          {presets?.length > 0 && (
            <div className="ptz-presets-grid">
              {presets.map(p => (
                <button key={p.id} className="ptz-preset-btn" onClick={() => gotoPreset(p.id)} title={p.name}>
                  {p.name || p.id}
                </button>
              ))}
            </div>
          )}
        </div>

        {error && <div className="ptz-error">{error}</div>}
      </div>
    </div>
  );
}
