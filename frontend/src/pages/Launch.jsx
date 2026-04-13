import { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { getScreenConfig } from '../api/grids';
import './Launch.css';

export default function Launch() {
  const navigate = useNavigate();
  const [centerName, setCenterName] = useState('');
  const [screenConfig, setScreenConfig] = useState(null);
  const [opened, setOpened] = useState(false);

  useEffect(() => {
    getScreenConfig()
      .then(config => {
        setCenterName(config.center_name || 'Centro Omnifish');
        setScreenConfig(config);
      })
      .catch(() => setCenterName('Centro Omnifish'));
  }, []);

  const handleStart = async () => {
    if (!screenConfig || opened) return;

    const assignedScreens = (screenConfig.screens || []).filter(s => s.stream_id);
    if (assignedScreens.length === 0) return;

    // Try to use Multi-Screen API for precise placement
    let physicalScreens = null;
    if ('getScreenDetails' in window) {
      try {
        const details = await window.getScreenDetails();
        physicalScreens = details.screens;
      } catch (e) {
        console.warn('Screen details not available:', e);
      }
    }

    for (const s of assignedScreens) {
      const url = `${window.location.origin}/screen/${s.slot}`;

      if (physicalScreens && physicalScreens[s.slot]) {
        // Open on the exact physical screen
        const ps = physicalScreens[s.slot];
        window.open(
          url,
          `omnifish_screen_${s.slot}`,
          `left=${ps.availLeft},top=${ps.availTop},width=${ps.availWidth},height=${ps.availHeight}`
        );
      } else if (s.screen_left !== undefined && s.screen_width) {
        // Use saved screen positions
        window.open(
          url,
          `omnifish_screen_${s.slot}`,
          `left=${s.screen_left},top=${s.screen_top},width=${s.screen_width},height=${s.screen_height}`
        );
      } else {
        // Fallback: open popup
        window.open(url, `omnifish_screen_${s.slot}`, 'popup=yes');
      }
    }

    setOpened(true);
  };

  const assignedCount = (screenConfig?.screens || []).filter(s => s.stream_id).length;

  return (
    <div className="launch">
      <img src="/logo.png" alt="Omnifish" className="launch-logo" />
      <div className="launch-center-name">{centerName}</div>
      <div className="launch-subtitle">Sistema de Video Monitoreo</div>
      <button className="launch-btn" onClick={handleStart} disabled={opened || assignedCount === 0}>
        {opened ? 'Pantallas abiertas' : 'Iniciar'}
      </button>
      {!opened && assignedCount > 0 && (
        <div className="launch-hint">
          {assignedCount} pantalla{assignedCount > 1 ? 's' : ''} configurada{assignedCount > 1 ? 's' : ''}
        </div>
      )}
      {opened && (
        <div className="launch-hint">
          Presione F11 en cada ventana para pantalla completa
        </div>
      )}
      {assignedCount === 0 && screenConfig && (
        <div className="launch-hint" style={{ color: 'var(--warning)' }}>
          No hay pantallas configuradas
        </div>
      )}
      <button className="launch-config-link" onClick={() => navigate('/config')}>
        Configuración
      </button>
      <div className="launch-footer">Omnifish VMS</div>
    </div>
  );
}
