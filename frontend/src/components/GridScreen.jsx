import { useRef, useEffect, useState, useCallback } from 'react';
import { connectWHEP } from '../api/whep';
import { useUsageReporter } from '../hooks/useUsageReporter';
import GridOverlay from './GridOverlay';
import './GridScreen.css';

const RETRY_BASE_MS = 2000;
const RETRY_MAX_MS = 15000;
// El mosaico compuesto emite frames continuamente (20-25 fps, celdas negras
// incluidas): si no llega ninguno en este lapso, la sesión quedó muda
// (ej. reinicio del servicio de compresión) y hay que renegociar.
const STALL_TIMEOUT_MS = 10000;

export default function GridScreen({ grid, aspect, osd, hideOverlay, streamKey, reportUsage, onCellDoubleClick, onCellRightClick }) {
  const videoRef = useRef(null);
  const [status, setStatus] = useState('idle');
  // Referencia al pc vivo para que el reporter de uso lea sus stats sin
  // acoplarse al ciclo de reconexión.
  const pcRef = useRef(null);
  const getPc = useCallback(() => pcRef.current, []);
  // Solo la ventana principal / pantallas reales reportan; reportUsage activa el hook.
  useUsageReporter(reportUsage ? getPc : null, reportUsage ? streamKey : null);

  useEffect(() => {
    const video = videoRef.current;
    if (!grid?.stream_ip || !video) return;

    let cancelled = false;
    let pc = null;
    let retryTimer = null;
    let retryDelay = RETRY_BASE_MS;
    let lastFrameTs = Date.now();
    let watchdog = null;
    let rvfcHandle = null;

    const onFrame = () => {
      lastFrameTs = Date.now();
      if (!cancelled) rvfcHandle = video.requestVideoFrameCallback(onFrame);
    };

    const closePc = () => {
      if (pc) {
        pc.onconnectionstatechange = null;
        pc.close();
        pc = null;
        pcRef.current = null;
      }
    };

    const scheduleReconnect = () => {
      if (cancelled || retryTimer) return;
      setStatus('reconnecting');
      closePc();
      retryTimer = setTimeout(() => {
        retryTimer = null;
        connect();
      }, retryDelay);
      retryDelay = Math.min(retryDelay * 2, RETRY_MAX_MS);
    };

    const connect = () => {
      if (cancelled) return;
      connectWHEP(grid.stream_ip)
        .then(({ pc: newPc, stream }) => {
          if (cancelled) {
            newPc.close();
            return;
          }
          closePc();
          pc = newPc;
          pcRef.current = newPc;
          // Detener los tracks viejos recién al tener la sesión nueva: así el
          // último frame queda visible durante la reconexión
          if (video.srcObject) video.srcObject.getTracks().forEach(t => t.stop());
          video.srcObject = stream;
          video.play().catch(() => {});
          lastFrameTs = Date.now();
          retryDelay = RETRY_BASE_MS;
          setStatus('connected');

          pc.onconnectionstatechange = () => {
            if (!pc) return;
            if (['failed', 'disconnected', 'closed'].includes(pc.connectionState)) {
              scheduleReconnect();
            }
          };
        })
        .catch((err) => {
          if (!cancelled) {
            console.warn('WHEP connection error, reintentando:', err.message);
            scheduleReconnect();
          }
        });
    };

    setStatus('connecting');
    connect();

    // Watchdog de frames congelados (solo con la ventana visible: en segundo
    // plano el navegador deja de componer frames y daría falsos positivos)
    if (video.requestVideoFrameCallback) {
      rvfcHandle = video.requestVideoFrameCallback(onFrame);
      watchdog = setInterval(() => {
        if (cancelled || document.hidden || retryTimer) return;
        if (Date.now() - lastFrameTs > STALL_TIMEOUT_MS) {
          console.warn('Stream congelado, renegociando WHEP');
          scheduleReconnect();
        }
      }, 3000);
    }

    return () => {
      cancelled = true;
      if (retryTimer) clearTimeout(retryTimer);
      if (watchdog) clearInterval(watchdog);
      if (rvfcHandle && video.cancelVideoFrameCallback) video.cancelVideoFrameCallback(rvfcHandle);
      closePc();
      if (video.srcObject) {
        video.srcObject.getTracks().forEach(t => t.stop());
        video.srcObject = null;
      }
    };
  }, [grid?.stream_ip]);

  if (!grid) return null;

  const typeLabel = grid.type === 'submarine' ? 'Submarinas' : 'Domos';

  // La resolución real del mosaico define la proporción; sin ella se mantiene
  // el 16:9 de las hojas de estilo.
  const aspectStyle = aspect
    ? {
        aspectRatio: `${aspect.width} / ${aspect.height}`,
        width: `min(100vw, calc(100vh * ${aspect.width} / ${aspect.height}))`,
        height: `min(100vh, calc(100vw * ${aspect.height} / ${aspect.width}))`,
      }
    : undefined;

  return (
    <div className="grid-screen" style={aspectStyle}>
      {grid.stream_ip ? (
        <>
          <video ref={videoRef} autoPlay muted playsInline />
          {status === 'connecting' && (
            <div className="grid-screen-placeholder">Conectando...</div>
          )}
          {status === 'reconnecting' && (
            <div className="grid-screen-reconnecting">
              <div className="reconnect-spinner">
                {Array.from({ length: 12 }, (_, i) => (
                  <span
                    key={i}
                    style={{
                      transform: `rotate(${i * 30}deg) translateY(-22px)`,
                      animationDelay: `${-(11 - i) / 12}s`,
                    }}
                  />
                ))}
              </div>
              <span className="reconnect-text">Reconnecting</span>
            </div>
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

      {!hideOverlay && (
        <GridOverlay
          rows={grid.rows}
          cols={grid.cols}
          cells={grid.cells}
          osd={osd}
          onCellDoubleClick={(cell, row, col) => onCellDoubleClick?.(grid, cell, row, col)}
          onCellRightClick={(cell, row, col, e) => onCellRightClick?.(grid, cell, row, col, e)}
        />
      )}

      <div className="grid-screen-header">
        <span className="grid-screen-name">{grid.name}</span>
        <span className="grid-screen-type">{typeLabel} {grid.rows}x{grid.cols}</span>
      </div>
    </div>
  );
}
