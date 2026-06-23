import PTZCellOverlay from './PTZCellOverlay';
import './GridOverlay.css';

export default function GridOverlay({ rows, cols, cells, osd, onCellDoubleClick, onCellRightClick }) {
  const gridStyle = {
    gridTemplateRows: `repeat(${rows}, 1fr)`,
    gridTemplateColumns: `repeat(${cols}, 1fr)`,
  };

  // OSD configurable: tamaño y posición de la etiqueta de celda
  const pos = osd?.position || 'top-left';
  const labelStyle = {};
  if (osd?.size) labelStyle.fontSize = osd.size;
  if (pos.startsWith('bottom')) {
    labelStyle.top = 'auto';
    labelStyle.bottom = 4;
  }
  if (pos.endsWith('right')) {
    labelStyle.left = 'auto';
    labelStyle.right = 6;
  }
  // Si la etiqueta ocupa la esquina superior derecha, el badge PTZ se corre
  const badgeStyle = pos === 'top-right' ? { right: 'auto', left: 6 } : undefined;

  // Color de líneas configurable (con sombra oscura sutil para que la línea
  // se lea tanto sobre agua iluminada como sobre celdas oscuras)
  const cellStyle = osd?.gridColor && osd.gridColor !== '#ffffff'
    ? {
        borderColor: `${osd.gridColor}cc`,
        boxShadow: 'inset 0 0 0 1px rgba(0, 0, 0, 0.35)',
      }
    : undefined;

  return (
    <div className="grid-overlay" style={gridStyle}>
      {Array.from({ length: rows * cols }, (_, i) => {
        const row = Math.floor(i / cols);
        const col = i % cols;
        const cell = cells?.find(c => c.row === row && c.col === col);

        return (
          <div
            key={`${row}-${col}`}
            className={cell?.offline ? 'grid-cell grid-cell-offline' : 'grid-cell'}
            style={cellStyle}
            onDoubleClick={() => onCellDoubleClick?.(cell, row, col)}
            onContextMenu={(e) => {
              e.preventDefault();
              onCellRightClick?.(cell, row, col, e);
            }}
          >
            {cell?.cage_name && (
              <span className="cell-label" style={labelStyle}>{cell.cage_name}</span>
            )}
            {cell?.has_ptz && (
              <span className="cell-ptz-badge" style={badgeStyle}>PTZ</span>
            )}
            {cell?.offline && (
              <span className="cell-offline-badge">SIN SEÑAL</span>
            )}
            {cell?.has_ptz && !cell?.offline && (cell?.name || cell?.cage_name) && (
              <PTZCellOverlay cameraName={cell.name || cell.cage_name} />
            )}
          </div>
        );
      })}
    </div>
  );
}
