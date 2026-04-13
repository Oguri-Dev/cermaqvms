import { useState, useMemo } from 'react';
import './ScreenMapper.css';

export default function ScreenMapper({ grids, screens, onScreensChange }) {
  const [detectedScreens, setDetectedScreens] = useState(null);

  const hasAPI = 'getScreenDetails' in window;

  const detectScreens = async () => {
    try {
      const details = await window.getScreenDetails();
      // Filter out ghost screens (0x0 with no label)
      const real = details.screens.filter(s => s.width > 0 && s.height > 0);
      const mapped = real.map((s, i) => ({
        index: i,
        label: s.label || `Pantalla ${i + 1}`,
        left: s.left,
        top: s.top,
        width: s.width,
        height: s.height,
        isPrimary: s.isPrimary,
      }));
      setDetectedScreens(mapped);
      syncScreenConfig(mapped);
    } catch (err) {
      console.error('Screen detection failed:', err);
    }
  };

  const addScreenManually = () => {
    const current = detectedScreens || [];
    const i = current.length;
    const lastScreen = current[current.length - 1];
    const newScreen = {
      index: i,
      label: `Pantalla ${i + 1}`,
      left: lastScreen ? lastScreen.left + lastScreen.width + 20 : 0,
      top: 0,
      width: 3840,
      height: 2160,
      isPrimary: false,
    };
    const updated = [...current, newScreen];
    setDetectedScreens(updated);
    syncScreenConfig(updated);
  };

  const removeScreen = (index) => {
    const updated = (detectedScreens || [])
      .filter(s => s.index !== index)
      .map((s, i) => ({ ...s, index: i }));
    setDetectedScreens(updated);
    syncScreenConfig(updated);
  };

  const syncScreenConfig = (mapped) => {
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
  };

  // Calculate visual layout for the minimap
  const layoutStyle = useMemo(() => {
    if (!detectedScreens || detectedScreens.length === 0) return null;

    // If all screens share the same position, lay them out side by side
    const positioned = detectedScreens.map(s => ({ ...s }));
    const allSamePos = positioned.every(s => s.left === positioned[0].left && s.top === positioned[0].top);
    if (allSamePos && positioned.length > 1) {
      let offset = 0;
      for (const s of positioned) {
        s.left = offset;
        s.top = 0;
        offset += s.width + 20;
      }
    }

    const minLeft = Math.min(...positioned.map(s => s.left));
    const minTop = Math.min(...positioned.map(s => s.top));
    const maxRight = Math.max(...positioned.map(s => s.left + s.width));
    const maxBottom = Math.max(...positioned.map(s => s.top + s.height));

    const totalW = maxRight - minLeft;
    const totalH = maxBottom - minTop;

    const scale = Math.min(600 / totalW, 250 / totalH);

    return {
      monitors: positioned.map(s => ({
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
            <button className="screen-mapper-detect-btn" onClick={addScreenManually}>
              + Agregar Manual
            </button>
            <span className="screen-mapper-detect-info">
              {detectedScreens
                ? `${detectedScreens.length} pantalla${detectedScreens.length > 1 ? 's' : ''}`
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
                          {g.name}
                        </option>
                      ))}
                    </select>
                    <button className="screen-mapper-remove-btn" onClick={() => removeScreen(s.index)} title="Quitar">×</button>
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
