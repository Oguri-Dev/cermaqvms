import { ptzMove, ptzPreset } from '../api/grids';
import './PTZControls.css';

export default function PTZControls({ gridId, cell, row, col, onClose }) {
  const sendMove = (action) => {
    ptzMove(gridId, row, col, { action, speed: 0.5 }).catch(console.error);
  };

  const sendStop = () => {
    ptzMove(gridId, row, col, { action: 'stop', speed: 0 }).catch(console.error);
  };

  const gotoPreset = (presetId) => {
    ptzPreset(gridId, row, col, { preset_id: presetId, action: 'goto' }).catch(console.error);
  };

  return (
    <div className="ptz-overlay" onClick={onClose}>
      <div className="ptz-panel" onClick={(e) => e.stopPropagation()}>
        <div className="ptz-header">
          <div>
            <div className="ptz-title">Control PTZ</div>
            <div className="ptz-subtitle">{cell?.cage_name || `Celda [${row},${col}]`}</div>
          </div>
          <button className="ptz-close" onClick={onClose}>&times;</button>
        </div>

        <div className="ptz-dpad">
          <div />
          <button className="ptz-btn" onMouseDown={() => sendMove('up')} onMouseUp={sendStop}>&#9650;</button>
          <div />
          <button className="ptz-btn" onMouseDown={() => sendMove('left')} onMouseUp={sendStop}>&#9664;</button>
          <button className="ptz-btn center">PTZ</button>
          <button className="ptz-btn" onMouseDown={() => sendMove('right')} onMouseUp={sendStop}>&#9654;</button>
          <div />
          <button className="ptz-btn" onMouseDown={() => sendMove('down')} onMouseUp={sendStop}>&#9660;</button>
          <div />
        </div>

        <div>
          <div className="ptz-zoom-label">Zoom</div>
          <div className="ptz-zoom">
            <button className="ptz-zoom-btn" onMouseDown={() => sendMove('zoom_out')} onMouseUp={sendStop}>&minus;</button>
            <button className="ptz-zoom-btn" onMouseDown={() => sendMove('zoom_in')} onMouseUp={sendStop}>+</button>
          </div>
        </div>

        <div className="ptz-presets">
          <div className="ptz-presets-label">Presets</div>
          <div className="ptz-presets-grid">
            {Array.from({ length: 8 }, (_, i) => (
              <button
                key={i + 1}
                className="ptz-preset-btn"
                onClick={() => gotoPreset(i + 1)}
              >
                {i + 1}
              </button>
            ))}
          </div>
        </div>
      </div>
    </div>
  );
}
