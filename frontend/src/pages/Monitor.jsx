import { useState, useEffect, useCallback, useRef } from 'react';
import { listGrids, getScreenConfig, getGrid } from '../api/grids';
import GridScreen from '../components/GridScreen';
import OnDemandViewer from '../components/OnDemandViewer';
import PTZControls from '../components/PTZControls';
import './Monitor.css';

export default function Monitor() {
  const [grids, setGrids] = useState([]);
  const [layout, setLayout] = useState(1);
  const [selectedGrids, setSelectedGrids] = useState([null, null, null, null]);
  const [onDemand, setOnDemand] = useState(null);
  const [ptzTarget, setPtzTarget] = useState(null);
  const [isFullscreen, setIsFullscreen] = useState(false);
  const monitorRef = useRef(null);

  useEffect(() => {
    Promise.all([listGrids(), getScreenConfig()])
      .then(([allGrids, config]) => {
        setGrids(allGrids);
        if (config && config.layout) {
          setLayout(config.layout);
          const assigned = [null, null, null, null];
          (config.screens || []).forEach(s => {
            const grid = allGrids.find(g => g.id === s.grid_id);
            if (grid && s.slot < 4) {
              assigned[s.slot] = grid;
            }
          });
          setSelectedGrids(assigned);
        }
      })
      .catch(console.error);
  }, []);

  useEffect(() => {
    const handleKey = (e) => {
      if (e.key === 'Escape') {
        setOnDemand(null);
        setPtzTarget(null);
      }
    };
    window.addEventListener('keydown', handleKey);
    return () => window.removeEventListener('keydown', handleKey);
  }, []);

  const assignGrid = (slotIndex, gridId) => {
    setSelectedGrids(prev => {
      const next = [...prev];
      next[slotIndex] = grids.find(g => g.id === gridId) || null;
      return next;
    });
  };

  const handleCellDoubleClick = useCallback((grid, cell, row, col) => {
    setOnDemand({ gridId: grid.id, cell, row, col });
  }, []);

  const handleCellRightClick = useCallback((grid, cell, row, col) => {
    if (cell?.has_ptz) {
      setPtzTarget({ gridId: grid.id, cell, row, col });
    }
  }, []);

  const toggleFullscreen = useCallback(() => {
    if (!document.fullscreenElement) {
      monitorRef.current?.requestFullscreen();
    } else {
      document.exitFullscreen();
    }
  }, []);

  useEffect(() => {
    const onChange = () => setIsFullscreen(!!document.fullscreenElement);
    document.addEventListener('fullscreenchange', onChange);
    return () => document.removeEventListener('fullscreenchange', onChange);
  }, []);

  const visibleSlots = layout;

  return (
    <div className="monitor" ref={monitorRef}>
      <div className="monitor-toolbar">
        <span className="monitor-toolbar-label">Pantallas</span>
        {[1, 2, 3, 4].map(n => (
          <button
            key={n}
            className={`monitor-layout-btn ${layout === n ? 'active' : ''}`}
            onClick={() => setLayout(n)}
          >
            {n}
          </button>
        ))}

        <span style={{ width: 1, height: 20, background: 'var(--border-color)', margin: '0 8px' }} />

        {Array.from({ length: visibleSlots }, (_, i) => (
          <select
            key={i}
            className="monitor-grid-select"
            value={selectedGrids[i]?.id || ''}
            onChange={(e) => assignGrid(i, e.target.value)}
          >
            <option value="">Pantalla {i + 1}</option>
            {grids.map(g => (
              <option key={g.id} value={g.id}>{g.name}</option>
            ))}
          </select>
        ))}

        <span style={{ flex: 1 }} />

        <button className="monitor-fullscreen-btn" onClick={toggleFullscreen} title={isFullscreen ? 'Salir de pantalla completa' : 'Pantalla completa'}>
          {isFullscreen ? (
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
              <path d="M8 3v3a2 2 0 0 1-2 2H3m18 0h-3a2 2 0 0 1-2-2V3m0 18v-3a2 2 0 0 1 2-2h3M3 16h3a2 2 0 0 1 2 2v3" />
            </svg>
          ) : (
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
              <path d="M8 3H5a2 2 0 0 0-2 2v3m18 0V5a2 2 0 0 0-2-2h-3m0 18h3a2 2 0 0 0 2-2v-3M3 16v3a2 2 0 0 0 2 2h3" />
            </svg>
          )}
        </button>
      </div>

      <div className={`monitor-content layout-${layout}`}>
        {Array.from({ length: visibleSlots }, (_, i) => {
          const grid = selectedGrids[i];
          return grid ? (
            <GridScreen
              key={grid.id}
              grid={grid}
              onCellDoubleClick={handleCellDoubleClick}
              onCellRightClick={handleCellRightClick}
            />
          ) : (
            <div key={i} className="monitor-empty">
              Seleccionar grilla
            </div>
          );
        })}
      </div>

      {onDemand && (
        <OnDemandViewer
          gridId={onDemand.gridId}
          cell={onDemand.cell}
          row={onDemand.row}
          col={onDemand.col}
          onClose={() => setOnDemand(null)}
        />
      )}

      {ptzTarget && (
        <PTZControls
          gridId={ptzTarget.gridId}
          cell={ptzTarget.cell}
          row={ptzTarget.row}
          col={ptzTarget.col}
          onClose={() => setPtzTarget(null)}
        />
      )}
    </div>
  );
}
