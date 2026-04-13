import { useState, useEffect, useRef } from 'react';
import { connectWHEP } from '../api/whep';
import PTZControls from './PTZControls';
import './OnDemandViewer.css';

export default function OnDemandViewer({ gridId, cell, row, col, onClose }) {
  const [showPTZ, setShowPTZ] = useState(false);
  const [status, setStatus] = useState('connecting');
  const videoRef = useRef(null);
  const pcRef = useRef(null);

  const camera = cell?.camera;
  const address = camera?.ondemand_mode === 'direct' ? camera?.camera_ip : camera?.nvr_address;
  const modeLabel = camera?.ondemand_mode === 'nvr' ? 'NVR' : 'Directo';

  useEffect(() => {
    if (!address) { setStatus('error'); return; }

    let cancelled = false;
    const streamUrl = address.startsWith('http') ? address : `http://${address}:8889`;

    connectWHEP(streamUrl)
      .then((result) => {
        if (cancelled) { result.pc.close(); return; }
        pcRef.current = result.pc;
        if (videoRef.current) {
          videoRef.current.srcObject = result.stream;
        }
        setStatus('connected');
      })
      .catch((err) => {
        if (!cancelled) {
          console.error('On-demand stream error:', err);
          setStatus('error');
        }
      });

    return () => {
      cancelled = true;
      if (pcRef.current) {
        pcRef.current.close();
        pcRef.current = null;
      }
    };
  }, [address]);

  return (
    <div className="ondemand-overlay">
      <div className="ondemand-header">
        <div className="ondemand-info">
          <span className="ondemand-cage">{camera?.cage_name || cell?.cage_name || `Celda [${row},${col}]`}</span>
          {address && <span className="ondemand-mode">{modeLabel} - {address}</span>}
        </div>
        <div style={{ display: 'flex', gap: '8px' }}>
          {camera?.has_ptz && (
            <button className="ondemand-close" style={{ background: 'var(--cermaq-blue-dark)' }} onClick={() => setShowPTZ(true)}>
              PTZ
            </button>
          )}
          <button className="ondemand-close" onClick={onClose}>Cerrar (ESC)</button>
        </div>
      </div>

      <div className="ondemand-video">
        <video ref={videoRef} autoPlay muted playsInline />
        {status === 'connecting' && <span className="ondemand-loading">Conectando stream on-demand...</span>}
        {status === 'error' && <span className="ondemand-loading">Error de conexión</span>}
      </div>

      {showPTZ && (
        <PTZControls
          gridId={gridId}
          cell={cell}
          row={row}
          col={col}
          onClose={() => setShowPTZ(false)}
        />
      )}
    </div>
  );
}
