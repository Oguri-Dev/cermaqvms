import { useState, useEffect, useCallback } from 'react';
import { useParams } from 'react-router-dom';
import { getScreenConfig, getStreamFull } from '../api/grids';
import GridScreen from '../components/GridScreen';
import OnDemandViewer from '../components/OnDemandViewer';
import PTZControls from '../components/PTZControls';
import './Screen.css';

export default function Screen() {
  const { slot } = useParams();
  const slotNum = parseInt(slot);
  const [streamData, setStreamData] = useState(null);
  const [loading, setLoading] = useState(true);
  const [onDemand, setOnDemand] = useState(null);
  const [ptzTarget, setPtzTarget] = useState(null);

  useEffect(() => {
    getScreenConfig()
      .then(config => {
        const screenSlot = config.screens?.find(s => s.slot === slotNum);
        if (screenSlot?.stream_id) {
          return getStreamFull(screenSlot.stream_id);
        }
        return null;
      })
      .then(data => {
        if (data) {
          setStreamData(data);
          document.title = `${data.name} - Omnifish VMS`;
        }
        setLoading(false);
      })
      .catch(() => setLoading(false));
  }, [slotNum]);

  useEffect(() => {
    const handleKey = (e) => {
      if (e.key === 'Escape') {
        setOnDemand(null);
        setPtzTarget(null);
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
    return () => window.removeEventListener('keydown', handleKey);
  }, []);

  const handleCellDoubleClick = useCallback((grid, cell, row, col) => {
    if (cell?.camera) {
      setOnDemand({ streamId: streamData.id, cell, row, col, camera: cell.camera });
    }
  }, [streamData]);

  const handleCellRightClick = useCallback((grid, cell, row, col) => {
    if (cell?.camera?.has_ptz) {
      setPtzTarget({ streamId: streamData.id, cell, row, col, camera: cell.camera });
    }
  }, [streamData]);

  if (loading) {
    return <div className="screen"><span className="screen-loading">Cargando...</span></div>;
  }

  if (!streamData) {
    return <div className="screen"><span className="screen-loading">Sin stream asignado a pantalla {slotNum + 1}</span></div>;
  }

  // Build grid-like object for GridScreen component
  const gridForScreen = {
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

  return (
    <div className="screen">
      <GridScreen
        grid={gridForScreen}
        onCellDoubleClick={handleCellDoubleClick}
        onCellRightClick={handleCellRightClick}
      />

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
          gridId={ptzTarget.streamId}
          cell={ptzTarget.cell}
          row={ptzTarget.row}
          col={ptzTarget.col}
          onClose={() => setPtzTarget(null)}
        />
      )}
    </div>
  );
}
