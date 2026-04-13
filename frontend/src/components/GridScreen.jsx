import { useRef, useEffect, useState } from 'react';
import { connectWHEP } from '../api/whep';
import GridOverlay from './GridOverlay';
import './GridScreen.css';

export default function GridScreen({ grid, onCellDoubleClick, onCellRightClick }) {
  const videoRef = useRef(null);
  const pcRef = useRef(null);
  const [status, setStatus] = useState('idle');

  useEffect(() => {
    if (!grid?.stream_ip || !videoRef.current) return;

    let cancelled = false;
    setStatus('connecting');

    connectWHEP(grid.stream_ip)
      .then(({ pc, stream }) => {
        if (cancelled) {
          pc.close();
          return;
        }
        pcRef.current = pc;
        videoRef.current.srcObject = stream;
        setStatus('connected');

        pc.onconnectionstatechange = () => {
          if (pc.connectionState === 'failed' || pc.connectionState === 'disconnected') {
            setStatus('disconnected');
          }
        };
      })
      .catch((err) => {
        if (!cancelled) {
          console.error('WHEP connection error:', err);
          setStatus('error');
        }
      });

    return () => {
      cancelled = true;
      if (pcRef.current) {
        pcRef.current.close();
        pcRef.current = null;
      }
      if (videoRef.current?.srcObject) {
        videoRef.current.srcObject.getTracks().forEach(t => t.stop());
        videoRef.current.srcObject = null;
      }
    };
  }, [grid?.stream_ip]);

  if (!grid) return null;

  const typeLabel = grid.type === 'submarine' ? 'Submarinas' : 'Domos';

  return (
    <div className="grid-screen">
      {grid.stream_ip ? (
        <>
          <video ref={videoRef} autoPlay muted playsInline />
          {status === 'connecting' && (
            <div className="grid-screen-placeholder">Conectando...</div>
          )}
          {status === 'error' && (
            <div className="grid-screen-placeholder">Error de conexión</div>
          )}
          {status === 'disconnected' && (
            <div className="grid-screen-placeholder">Desconectado</div>
          )}
        </>
      ) : (
        <div className="grid-screen-placeholder">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5">
            <path d="M15.6 11.6L22 7v10l-6.4-4.6M2 5h13a2 2 0 0 1 2 2v10a2 2 0 0 1-2 2H2z" />
          </svg>
          Sin stream configurado
        </div>
      )}

      <GridOverlay
        rows={grid.rows}
        cols={grid.cols}
        cells={grid.cells}
        onCellDoubleClick={(cell, row, col) => onCellDoubleClick?.(grid, cell, row, col)}
        onCellRightClick={(cell, row, col, e) => onCellRightClick?.(grid, cell, row, col, e)}
      />

      <div className="grid-screen-header">
        <span className="grid-screen-name">{grid.name}</span>
        <span className="grid-screen-type">{typeLabel} {grid.rows}x{grid.cols}</span>
      </div>
    </div>
  );
}
