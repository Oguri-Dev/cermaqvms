import { useState, useMemo } from 'react';
import './ScreenMapper.css';

export default function ScreenMapper({ grids, screens, onScreensChange }) {
  const [detectedScreens, setDetectedScreens] = useState(null);

  const hasAPI = 'getScreenDetails' in window;

  const detectScreens = async () => {
    try {
      const details = await window.getScreenDetails();
      const mapped = details.screens.map((s, i) => ({
        index: i,
        label: s.label || `Pantalla ${i + 1}`,
        left: s.left,
        top: s.top,
        width: s.width,
        height: s.height,
        isPrimary: s.isPrimary,
      }));
      setDetectedScreens(mapped);

      // Auto-update screen config layout count
      if (mapped.length !== screens.length) {
        // Preserve existing assignments
        const newScreens = mapped.map((s, i) => {
          const existing = screens.find(sc => sc.slot === i);
          return {
            slot: i,
            stream_id: existing?.stream_id || '',
            screen_label: s.label,
            screen_left: s.left,
            screen_top: s.top,
            screen_width: s.width,
            screen_height: s.height,
          };
        });
        onScreensChange(newScreens, mapped.length);
      }
    } catch (err) {
      console.error('Screen detection failed:', err);
    }
  };

  // Calculate visual layout for the minimap
  const layoutStyle = useMemo(() => {
    if (!detectedScreens || detectedScreens.length === 0) return null;

    const minLeft = Math.min(...detectedScreens.map(s => s.left));
    const minTop = Math.min(...detectedScreens.map(s => s.top));
    const maxRight = Math.max(...detectedScreens.map(s => s.left + s.width));
    const maxBottom = Math.max(...detectedScreens.map(s => s.top + s.height));

    const totalW = maxRight - minLeft;
    const totalH = maxBottom - minTop;

    // Scale to fit in the canvas (max 500px wide, 250px tall)
    const scale = Math.min(500 / totalW, 250 / totalH);

    return {
      monitors: detectedScreens.map(s => ({
        ...s,
        x: (s.left - minLeft) * scale,
        y: (s.top - minTop) * scale,
        w: s.width * scale,
        h: s.height * scale,
      })),
      totalW: totalW * scale,
      totalH: totalH * scale,
    };
  }, [detectedScreens]);

  const assignGrid = (slot, gridId) => {
    const newScreens = screens.map(s =>
      s.slot === slot ? { ...s, stream_id: gridId } : s
    );
    onScreensChange(newScreens);
  };

  const getAssignedGridName = (slot) => {
    const s = screens.find(sc => sc.slot === slot);
    if (!s?.stream_id) return null;
    const g = grids.find(g => g.id === s.stream_id);
    return g?.name || null;
  };

  return (
    <div className="screen-mapper">
      {hasAPI ? (
        <>
          <div className="screen-mapper-detect">
            <button className="screen-mapper-detect-btn" onClick={detectScreens}>
              Detectar Pantallas
            </button>
            <span className="screen-mapper-detect-info">
              {detectedScreens
                ? `${detectedScreens.length} pantalla${detectedScreens.length > 1 ? 's' : ''} detectada${detectedScreens.length > 1 ? 's' : ''}`
                : 'Haz click para detectar los monitores conectados'}
            </span>
          </div>

          {layoutStyle && (
            <div className="screen-mapper-canvas">
              <div className="screen-mapper-layout" style={{ width: layoutStyle.totalW, height: layoutStyle.totalH }}>
                {layoutStyle.monitors.map((m) => {
                  const gridName = getAssignedGridName(m.index);
                  return (
                    <div
                      key={m.index}
                      className={`screen-mapper-monitor ${gridName ? 'assigned' : ''} ${m.isPrimary ? 'primary' : ''}`}
                      style={{
                        left: m.x,
                        top: m.y,
                        width: m.w,
                        height: m.h,
                      }}
                    >
                      {m.isPrimary && <span className="screen-mapper-monitor-primary-badge">Principal</span>}
                      <span className="screen-mapper-monitor-number">{m.index + 1}</span>
                      <span className="screen-mapper-monitor-label">{m.label}</span>
                      <span className="screen-mapper-monitor-res">{m.width}x{m.height}</span>
                      {gridName && <span className="screen-mapper-monitor-grid">{gridName}</span>}
                    </div>
                  );
                })}
              </div>
            </div>
          )}

          {detectedScreens && (
            <div className="screen-mapper-assign">
              {detectedScreens.map((s) => {
                const current = screens.find(sc => sc.slot === s.index);
                return (
                  <div key={s.index} className={`screen-mapper-assign-row ${current?.stream_id ? 'active' : ''}`}>
                    <span className="screen-mapper-assign-num">{s.index + 1}</span>
                    <div className="screen-mapper-assign-info">
                      <div className="screen-mapper-assign-name">{s.label}</div>
                      <div className="screen-mapper-assign-meta">{s.width}x{s.height} {s.isPrimary ? '(Principal)' : ''}</div>
                    </div>
                    <select
                      className="screen-mapper-assign-select"
                      value={current?.stream_id || ''}
                      onChange={e => assignGrid(s.index, e.target.value)}
                    >
                      <option value="">-- Sin asignar --</option>
                      {grids.map(g => (
                        <option key={g.id} value={g.id}>
                          {g.name} ({g.type === 'submarine' ? 'Sub' : 'Domo'} {g.rows}x{g.cols})
                        </option>
                      ))}
                    </select>
                  </div>
                );
              })}
            </div>
          )}
        </>
      ) : (
        <div className="screen-mapper-noapi">
          Tu navegador no soporta la detección de múltiples pantallas.<br />
          Usa Chrome o Edge para esta funcionalidad.
        </div>
      )}
    </div>
  );
}
