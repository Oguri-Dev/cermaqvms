import './GridOverlay.css';

export default function GridOverlay({ rows, cols, cells, onCellDoubleClick, onCellRightClick }) {
  const gridStyle = {
    gridTemplateRows: `repeat(${rows}, 1fr)`,
    gridTemplateColumns: `repeat(${cols}, 1fr)`,
  };

  return (
    <div className="grid-overlay" style={gridStyle}>
      {Array.from({ length: rows * cols }, (_, i) => {
        const row = Math.floor(i / cols);
        const col = i % cols;
        const cell = cells?.find(c => c.row === row && c.col === col);

        return (
          <div
            key={`${row}-${col}`}
            className="grid-cell"
            onDoubleClick={() => onCellDoubleClick?.(cell, row, col)}
            onContextMenu={(e) => {
              e.preventDefault();
              onCellRightClick?.(cell, row, col, e);
            }}
          >
            {cell?.cage_name && (
              <span className="cell-label">{cell.cage_name}</span>
            )}
            {cell?.has_ptz && (
              <span className="cell-ptz-badge">PTZ</span>
            )}
          </div>
        );
      })}
    </div>
  );
}
