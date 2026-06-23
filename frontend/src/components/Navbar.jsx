import { useState, useEffect } from 'react';
import { NavLink } from 'react-router-dom';
import { getScreenConfig, restartCompression } from '../api/grids';
import { useAuth } from '../auth/AuthContext';
import './Navbar.css';

export default function Navbar() {
  const [screenConfig, setScreenConfig] = useState(null);
  const [restarting, setRestarting] = useState(false);
  const { user, isAdmin, logout } = useAuth();

  useEffect(() => {
    getScreenConfig().then(setScreenConfig).catch(console.error);
  }, []);

  const handleRestartCompression = async () => {
    if (!confirm('Reiniciar la compresión cortará el video de todas las pantallas durante unos segundos. ¿Continuar?')) return;
    setRestarting(true);
    try {
      await restartCompression();
      alert('Compresión reiniciada. El video se restablecerá en unos segundos.');
    } catch (err) {
      alert('No se pudo reiniciar la compresión: ' + err.message);
    } finally {
      setRestarting(false);
    }
  };

  const handleStart = async () => {
    if (!screenConfig) return;

    const assignedScreens = (screenConfig.screens || []).filter(s => s.stream_id);
    if (assignedScreens.length === 0) return;

    let physicalScreens = null;
    if ('getScreenDetails' in window) {
      try {
        const details = await window.getScreenDetails();
        physicalScreens = details.screens.filter(s => s.width > 0 && s.height > 0);
      } catch (e) {
        console.warn('Screen details not available:', e);
      }
    }

    for (const s of assignedScreens) {
      const url = `${window.location.origin}/screen/${s.slot}`;
      let left = s.screen_left || 0;
      let top = s.screen_top || 0;
      let width = s.screen_width || screen.width;
      let height = s.screen_height || screen.height;

      if (physicalScreens && physicalScreens[s.slot]) {
        const ps = physicalScreens[s.slot];
        left = ps.availLeft;
        top = ps.availTop;
        width = ps.availWidth;
        height = ps.availHeight;
      }

      const features = `popup=yes,left=${left},top=${top},width=${width},height=${height},menubar=no,toolbar=no,location=no,status=no`;
      const win = window.open(url, `omnifish_screen_${s.slot}`, features);
      if (win) {
        // Request fullscreen once the page loads
        win.addEventListener('load', () => {
          try { win.document.documentElement.requestFullscreen(); } catch (e) {}
        });
      }
    }
  };

  return (
    <nav className="navbar">
      <div className="navbar-brand">
        <img src="/logo.png" alt="Cermaq" className="navbar-logo" />
        <span className="navbar-title">OMNIFISH VMS</span>
      </div>

      <div className="navbar-nav">
        <button className="navbar-start-btn" onClick={handleStart}>
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
            <polygon points="5 3 19 12 5 21 5 3" />
          </svg>
          Abrir Pantallas
        </button>

        <button className="navbar-link navbar-restart-btn" onClick={handleRestartCompression} disabled={restarting} title="Reiniciar el servicio de compresión del centro">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
            <path d="M23 4v6h-6M1 20v-6h6" />
            <path d="M3.51 9a9 9 0 0 1 14.85-3.36L23 10M1 14l4.64 4.36A9 9 0 0 0 20.49 15" />
          </svg>
          {restarting ? 'Reiniciando...' : 'Reiniciar compresión'}
        </button>

        {isAdmin && (
          <NavLink to="/config" className={({ isActive }) => `navbar-link ${isActive ? 'active' : ''}`}>
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
              <circle cx="12" cy="12" r="3" />
              <path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1-2.83 2.83l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-4 0v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83-2.83l.06-.06A1.65 1.65 0 0 0 4.68 15a1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1 0-4h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 2.83-2.83l.06.06A1.65 1.65 0 0 0 9 4.68a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 4 0v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 2.83l-.06.06A1.65 1.65 0 0 0 19.4 9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 0 4h-.09a1.65 1.65 0 0 0-1.51 1z" />
            </svg>
            Configuración
          </NavLink>
        )}
      </div>

      <div className="navbar-spacer" />

      <div className="navbar-user">
        <span className="navbar-user-name">{user?.username}</span>
        <span className="navbar-user-role">{isAdmin ? 'Administrador' : 'Operador'}</span>
        <button className="navbar-logout" onClick={logout} title="Cerrar sesión">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
            <path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4" />
            <polyline points="16 17 21 12 16 7" />
            <line x1="21" y1="12" x2="9" y2="12" />
          </svg>
        </button>
      </div>
    </nav>
  );
}
