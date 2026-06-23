import { useEffect } from 'react';
import { usePtz } from '../hooks/usePtz';
import './PTZCellOverlay.css';

// Controles PTZ embebidos en la celda del mosaico (o sobre el video
// maximizado): aparecen al pasar el mouse encima. El doble click sobre los
// controles no burbujea, para no disparar el zoom/unzoom de la celda.
export default function PTZCellOverlay({ cameraName }) {
  const { startMove, stopMove, gotoPreset, presets, loadPresets, error } = usePtz(cameraName);

  useEffect(() => { loadPresets(); }, [loadPresets]);

  const btn = (direction, label) => (
    <button
      className="ptz-cell-btn"
      onPointerDown={() => startMove(direction)}
      onPointerUp={stopMove}
      onPointerLeave={stopMove}
      onDoubleClick={(e) => e.stopPropagation()}
      onContextMenu={(e) => e.preventDefault()}
    >
      {label}
    </button>
  );

  return (
    <div className="ptz-cell-overlay">
      <div className="ptz-cell-cluster" onDoubleClick={(e) => e.stopPropagation()}>
        <div className="ptz-cell-dpad">
          <div />
          {btn('up', '▲')}
          <div />
          {btn('left', '◀')}
          <div className="ptz-cell-center" />
          {btn('right', '▶')}
          <div />
          {btn('down', '▼')}
          <div />
        </div>
        <div className="ptz-cell-zoom">
          {btn('zoom_in', '+')}
          {btn('zoom_out', '−')}
        </div>
      </div>

      {presets?.length > 0 && (
        <div className="ptz-cell-presets" onDoubleClick={(e) => e.stopPropagation()}>
          {presets.map(p => (
            <button
              key={p.id}
              className="ptz-cell-preset"
              onClick={(e) => { e.stopPropagation(); gotoPreset(p.id); }}
              onDoubleClick={(e) => e.stopPropagation()}
              title={p.name}
            >
              {p.name || p.id}
            </button>
          ))}
        </div>
      )}

      {error && <div className="ptz-cell-error">{error}</div>}
    </div>
  );
}
