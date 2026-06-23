import { useEffect, useRef } from 'react';
import { centerBootstrap } from '../api/grids';

// Dispara la secuencia de arranque del centro (reinicia compresión, MediaMTX
// si está activo, y el servicio PTZ) una sola vez al abrir la aplicación.
//
// El backend ya deduplica por ventana de tiempo, así que varias ventanas o un
// F5 no provocan reinicios en cadena; este guard de sesión solo evita llamadas
// HTTP redundantes desde la misma pestaña.
//
// `enabled` debe ser false en las ventanas secundarias del muro (/screen/...):
// solo la ventana principal lanza el arranque.
const SESSION_FLAG = 'omnifish_bootstrap_done';

export function useBootstrap(enabled) {
  const ran = useRef(false);

  useEffect(() => {
    if (!enabled || ran.current) return;
    if (sessionStorage.getItem(SESSION_FLAG)) return;
    ran.current = true;
    sessionStorage.setItem(SESSION_FLAG, '1');

    centerBootstrap()
      .then((res) => {
        if (res?.skipped) {
          console.info('[bootstrap] el centro ya se reinició recientemente');
        } else {
          console.info('[bootstrap] secuencia de arranque enviada', res?.steps ?? res);
        }
      })
      .catch((err) => {
        // No bloquea el uso del wall: si el centro está caído, el operador
        // sigue viendo lo que haya. Se podrá reintentar al recargar.
        sessionStorage.removeItem(SESSION_FLAG);
        console.warn('[bootstrap] no se pudo ejecutar el arranque:', err.message);
      });
  }, [enabled]);
}
