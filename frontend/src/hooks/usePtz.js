import { useState, useRef, useCallback, useEffect } from 'react';
import { centerPtzMove, centerPtzStop, centerPtzGoto, centerPtzPresets } from '../api/grids';

const SPEED = 0.5;
// El servicio PTZ detiene la cámara si no recibe renovación en ~2.5s
// (dead-man switch); renovar cada 800ms deja margen de sobra.
const KEEPALIVE_MS = 800;

const VECTORS = {
  up: { pan: 0, tilt: SPEED, zoom: 0 },
  down: { pan: 0, tilt: -SPEED, zoom: 0 },
  left: { pan: -SPEED, tilt: 0, zoom: 0 },
  right: { pan: SPEED, tilt: 0, zoom: 0 },
  zoom_in: { pan: 0, tilt: 0, zoom: SPEED },
  zoom_out: { pan: 0, tilt: 0, zoom: -SPEED },
};

// Los NVR Hikvision devuelven los 256 slots de preset, incluidos los
// especiales de sistema (Remote reboot, Call OSD menu, scans...). Solo se
// muestran los configurados por el usuario (nombre personalizado).
const DEFAULT_PRESET_RE = /^preset ?\d+$/i;
const SPECIAL_PRESET_RE = /^(call (patrol|pattern|osd)|start .*scan|stop a scan|auto-flip|back to origin|day mode|night mode|one-touch|day\/night|set manual|save manual|remote reboot)/i;

function isUserPreset(p) {
  const name = (p.name || '').trim();
  return name !== '' && !DEFAULT_PRESET_RE.test(name) && !SPECIAL_PRESET_RE.test(name);
}

// Lógica PTZ compartida (overlay de celda y panel modal): movimiento
// continuo con keepalive, stop garantizado al soltar en cualquier parte,
// presets bajo demanda y errores transitorios.
export function usePtz(cameraName) {
  const [presets, setPresets] = useState(null);
  const [error, setError] = useState(null);
  const keepaliveRef = useRef(null);
  const movingRef = useRef(false);
  const errorTimerRef = useRef(null);
  const presetsLoadedRef = useRef(false);

  const showError = useCallback((msg) => {
    setError(msg);
    clearTimeout(errorTimerRef.current);
    errorTimerRef.current = setTimeout(() => setError(null), 4000);
  }, []);

  const stopMove = useCallback(() => {
    if (!movingRef.current) return;
    movingRef.current = false;
    if (keepaliveRef.current) {
      clearInterval(keepaliveRef.current);
      keepaliveRef.current = null;
    }
    centerPtzStop(cameraName).catch(err => {
      // El dead-man del servicio detiene igual la cámara en ~2.5s
      console.error('PTZ stop:', err);
      showError('Stop no confirmado (el auto-stop del servicio actúa en 2.5s)');
    });
  }, [cameraName, showError]);

  const startMove = useCallback((direction) => {
    const vec = VECTORS[direction];
    if (!vec || movingRef.current) return;
    movingRef.current = true;

    const send = () => centerPtzMove(cameraName, vec).catch(err => {
      console.error('PTZ move:', err);
      showError('No se pudo mover la cámara');
      stopMove();
    });

    send();
    // Keepalive: renueva el comando mientras el control siga presionado
    keepaliveRef.current = setInterval(send, KEEPALIVE_MS);
  }, [cameraName, stopMove, showError]);

  const gotoPreset = useCallback((presetId) => {
    centerPtzGoto(cameraName, String(presetId)).catch(err => {
      console.error('PTZ goto:', err);
      showError('No se pudo ir al preset');
    });
  }, [cameraName, showError]);

  const loadPresets = useCallback(() => {
    if (presetsLoadedRef.current) return;
    presetsLoadedRef.current = true;
    centerPtzPresets(cameraName)
      .then(res => setPresets((res.presets || []).filter(isUserPreset)))
      .catch(() => setPresets([]));
  }, [cameraName]);

  // Soltar el botón FUERA del control también detiene (pointerup global)
  useEffect(() => {
    window.addEventListener('pointerup', stopMove);
    return () => {
      window.removeEventListener('pointerup', stopMove);
      clearTimeout(errorTimerRef.current);
      stopMove();
    };
  }, [stopMove]);

  return { startMove, stopMove, gotoPreset, presets, loadPresets, error };
}
