import { useState, useEffect, useRef, useCallback } from 'react';
import { useParams } from 'react-router-dom';
import {
  getScreenConfig,
  getStreamFull,
  getCenterScreen,
  zoomCenterCell,
  unzoomCenter,
  getCenterZoomStatus,
  getCenterHealth,
} from '../api/grids';
import GridScreen from '../components/GridScreen';
import OnDemandViewer from '../components/OnDemandViewer';
import PTZControls from '../components/PTZControls';
import PTZCellOverlay from '../components/PTZCellOverlay';
import './Screen.css';

const HEALTH_POLL_MS = 10000;

export default function Screen() {
  const { slot } = useParams();
  const slotNum = parseInt(slot);
  const [streamData, setStreamData] = useState(null);
  const [centerData, setCenterData] = useState(null);
  const [loading, setLoading] = useState(true);
  const [onDemand, setOnDemand] = useState(null);
  const [ptzTarget, setPtzTarget] = useState(null);
  const [isFullscreen, setIsFullscreen] = useState(false);
  const [zoomedCell, setZoomedCell] = useState(null);
  const [offlineCells, setOfflineCells] = useState(new Set());
  const [osd, setOsd] = useState(null);
  const [actionError, setActionError] = useState(null);
  const zoomedRef = useRef(null);
  zoomedRef.current = zoomedCell;
  // Época de acciones de zoom: las respuestas de polls lanzados antes de la
  // última acción del usuario se descartan para no pisar el estado fresco.
  const zoomSeqRef = useRef(0);

  const showActionError = useCallback((msg) => {
    setActionError(msg);
    setTimeout(() => setActionError(null), 4000);
  }, []);

  useEffect(() => {
    let active = true;
    getScreenConfig()
      .then(config => {
        setOsd({ size: config.osd_size, position: config.osd_position, gridColor: config.grid_color });
        const screenSlot = config.screens?.find(s => s.slot === slotNum);
        if (screenSlot?.stream_id) {
          return getStreamFull(screenSlot.stream_id);
        }
        return null;
      })
      .then(async data => {
        if (!active) return;
        if (data) {
          setStreamData(data);
          document.title = `${data.name} - Omnifish VMS`;
          // La grilla real (filas, columnas, celdas) viene de la configuración
          // que corre en el centro, no de la configuración local.
          if (data.file_name) {
            try {
              const center = await getCenterScreen(data.file_name);
              if (active) setCenterData(center);
            } catch (err) {
              console.warn('Centro no disponible, usando grilla local:', err.message);
            }
          }
        }
        setLoading(false);
      })
      .catch(() => active && setLoading(false));
    return () => { active = false; };
  }, [slotNum]);

  // Sincronizar estado de zoom y salud de cámaras con el compresor
  useEffect(() => {
    if (!centerData?.file_name) return;
    let active = true;

    // El índice de cámara del compresor recorre solo las celdas activas en orden idCell
    const activeNames = centerData.cells.filter(c => c.on).map(c => c.name);

    const poll = async () => {
      const seq = zoomSeqRef.current;
      try {
        const status = await getCenterZoomStatus(centerData.file_name);
        if (active && seq === zoomSeqRef.current && status?.data) {
          if (status.data.zoomActive) {
            // currentCameraName no está documentado en el API; derivar del índice como respaldo
            const name = status.data.currentCameraName
              || activeNames[status.data.currentCameraIndex]
              || zoomedRef.current
              || 'Cámara';
            setZoomedCell(name);
          } else {
            setZoomedCell(null);
          }
        }
      } catch { /* compresor no disponible: mantener último estado */ }
      try {
        const health = await getCenterHealth(centerData.file_name);
        if (active && health?.data?.activeCameras) {
          const offline = new Set(
            health.data.activeCameras
              .filter(c => c.inactive || !c.connected)
              .map(c => (c.cameraName || '').toLowerCase())
          );
          setOfflineCells(offline);
        }
      } catch { /* idem */ }
    };

    poll();
    const interval = setInterval(poll, HEALTH_POLL_MS);
    return () => { active = false; clearInterval(interval); };
  }, [centerData?.file_name]);

  const handleUnzoom = useCallback(() => {
    if (!centerData?.file_name) return;
    zoomSeqRef.current++;
    unzoomCenter(centerData.file_name)
      .then(() => setZoomedCell(null))
      .catch(err => {
        console.error('Error al restaurar grilla:', err);
        showActionError('No se pudo restaurar la grilla (compresor no disponible)');
      });
  }, [centerData?.file_name, showActionError]);

  useEffect(() => {
    const onFsChange = () => setIsFullscreen(!!document.fullscreenElement);
    document.addEventListener('fullscreenchange', onFsChange);

    const handleKey = (e) => {
      if (e.key === 'Escape') {
        setOnDemand(null);
        setPtzTarget(null);
        // Si está en pantalla completa, el primer ESC solo sale de fullscreen
        // (el navegador lo hace solo); no des-zoomear el stream compartido.
        if (!document.fullscreenElement && zoomedRef.current) handleUnzoom();
      }
      if (e.key === 'F11') {
        e.preventDefault();
        if (!document.fullscreenElement) {
          document.documentElement.requestFullscreen();
        } else {
          document.exitFullscreen();
        }
      }
    };
    window.addEventListener('keydown', handleKey);
    return () => {
      window.removeEventListener('keydown', handleKey);
      document.removeEventListener('fullscreenchange', onFsChange);
    };
  }, [handleUnzoom]);

  const goFullscreen = () => {
    document.documentElement.requestFullscreen().catch(() => {});
  };

  const handleCellDoubleClick = useCallback((grid, cell, row, col) => {
    if (centerData?.file_name) {
      // Zoom dentro del mismo stream compuesto vía el compresor del centro
      if (zoomedRef.current) {
        handleUnzoom();
        return;
      }
      if (cell?.offline) {
        showActionError(`${cell.name}: sin señal, no se puede maximizar`);
        return;
      }
      if (cell?.on && cell?.name) {
        zoomSeqRef.current++;
        zoomCenterCell(centerData.file_name, cell.name)
          .then(res => setZoomedCell(res?.data?.realCameraName || cell.name))
          .catch(err => {
            console.error('Error al hacer zoom:', err);
            showActionError('No se pudo maximizar la cámara (compresor no disponible)');
          });
      }
      return;
    }
    // Sin conexión al centro: comportamiento anterior (visor on-demand)
    if (cell?.camera) {
      setOnDemand({ streamId: streamData.id, cell, row, col, camera: cell.camera });
    }
  }, [centerData?.file_name, streamData, handleUnzoom]);

  const handleCellRightClick = useCallback((grid, cell) => {
    if (centerData?.file_name) {
      // En modo centro las celdas PTZ se identifican por su tipo y el
      // servicio PTZ resuelve la cámara por nombre
      if (cell?.on && cell?.name && cell?.has_ptz) {
        setPtzTarget({ cameraName: cell.name });
      }
      return;
    }
    if (cell?.camera?.has_ptz) {
      setPtzTarget({ cameraName: cell.camera.cage_name || cell.camera.name });
    }
  }, [centerData?.file_name]);

  if (loading) {
    return <div className="screen"><span className="screen-loading">Cargando...</span></div>;
  }

  if (!streamData) {
    return <div className="screen"><span className="screen-loading">Sin stream asignado a pantalla {slotNum + 1}</span></div>;
  }

  // Build grid-like object for GridScreen component
  const gridForScreen = centerData
    ? {
        id: streamData.id,
        name: streamData.name || centerData.file_name,
        type: streamData.grid?.type,
        rows: centerData.rows,
        cols: centerData.cols,
        stream_ip: streamData.stream_ip || centerData.whep_url,
        cells: centerData.cells.map(c => ({
          row: c.row,
          col: c.col,
          name: c.name,
          on: c.on,
          cage_name: c.on ? c.name : '',
          // El backend resuelve has_ptz mezclando la edición local
          // (Dispositivos) con el tipo que traiga la celda del centro
          has_ptz: !!c.has_ptz,
          offline: c.on && offlineCells.has((c.name || '').toLowerCase()),
        })),
      }
    : {
        id: streamData.id,
        name: streamData.name,
        type: streamData.grid.type,
        rows: streamData.grid.rows,
        cols: streamData.grid.cols,
        stream_ip: streamData.stream_ip,
        cells: streamData.cells.map(c => ({
          row: c.row,
          col: c.col,
          cage_name: c.camera?.cage_name || '',
          has_ptz: c.camera?.has_ptz || false,
          camera: c.camera,
        })),
      };

  const aspect = centerData?.width_resolution > 0 && centerData?.height_resolution > 0
    ? { width: centerData.width_resolution, height: centerData.height_resolution }
    : null;

  // Si la cámara maximizada es PTZ, sus controles se muestran sobre el video
  // (el overlay de grilla está oculto durante el zoom)
  const zoomedPtzName = zoomedCell
    ? gridForScreen.cells?.find(c =>
        c.has_ptz && (c.name || c.cage_name || '').toLowerCase() === zoomedCell.toLowerCase()
      ) && zoomedCell
    : null;

  return (
    <div className="screen" onDoubleClick={zoomedCell ? handleUnzoom : undefined}>
      {!isFullscreen && (
        <div className="screen-fullscreen-overlay">
          <button className="screen-fullscreen-btn" onClick={goFullscreen}>Pantalla Completa</button>
        </div>
      )}
      <GridScreen
        grid={gridForScreen}
        aspect={aspect}
        osd={osd}
        hideOverlay={!!zoomedCell}
        reportUsage
        streamKey={streamData.file_name || gridForScreen.name}
        onCellDoubleClick={handleCellDoubleClick}
        onCellRightClick={handleCellRightClick}
      />

      {zoomedPtzName && (
        <div className="screen-ptz-zoomed">
          <PTZCellOverlay cameraName={zoomedPtzName} />
        </div>
      )}

      {zoomedCell && (
        <div className="screen-zoom-banner">
          <span className="screen-zoom-name">{zoomedCell}</span>
          <span className="screen-zoom-hint">Doble click o ESC para volver a la grilla</span>
        </div>
      )}

      {actionError && (
        <div className="screen-zoom-banner screen-error-banner">
          <span className="screen-error-text">{actionError}</span>
        </div>
      )}

      {onDemand && (
        <OnDemandViewer
          gridId={onDemand.streamId}
          cell={onDemand.cell}
          row={onDemand.row}
          col={onDemand.col}
          onClose={() => setOnDemand(null)}
        />
      )}

      {ptzTarget && (
        <PTZControls
          cameraName={ptzTarget.cameraName}
          onClose={() => setPtzTarget(null)}
        />
      )}
    </div>
  );
}
