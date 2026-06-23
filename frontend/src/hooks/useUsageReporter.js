import { useEffect, useRef } from 'react';
import { usageHeartbeat } from '../api/grids';

// Reporta el consumo de datos de un stream WHEP para el historial de uso.
//
// Lee bytesReceived de la RTCPeerConnection vía getStats() y, cada SAMPLE_MS,
// envía al backend el INCREMENTO de bytes desde la última muestra junto con la
// fecha local y el slot de 30s actual. El backend acumula por stream y marca el
// slot como "wall abierto".
//
// getStats devuelve contadores monótonos POR conexión: al renegociar (la
// reconexión WHEP crea un pc nuevo) el contador vuelve a empezar. Por eso se
// trabaja con deltas y, ante un valor menor al anterior (reset), se cuenta el
// nuevo valor como incremento desde cero en vez de un delta negativo.
const SAMPLE_MS = 30000;
const SLOT_SECONDS = 30;

// Suma bytesReceived de todos los inbound-rtp (video + audio) del pc.
async function readBytes(pc) {
  let total = 0;
  const stats = await pc.getStats();
  stats.forEach((report) => {
    if (report.type === 'inbound-rtp' && typeof report.bytesReceived === 'number') {
      total += report.bytesReceived;
    }
  });
  return total;
}

function localDateStr(d) {
  // YYYY-MM-DD en hora local (no UTC): el wall y el backend están en el mismo huso.
  const y = d.getFullYear();
  const m = String(d.getMonth() + 1).padStart(2, '0');
  const day = String(d.getDate()).padStart(2, '0');
  return `${y}-${m}-${day}`;
}

function slotOfDay(d) {
  const secondsOfDay = d.getHours() * 3600 + d.getMinutes() * 60 + d.getSeconds();
  return Math.floor(secondsOfDay / SLOT_SECONDS);
}

// getPc: función que devuelve el RTCPeerConnection vivo (o null).
// streamKey: identificador del stream (file_name) para agrupar el consumo.
export function useUsageReporter(getPc, streamKey) {
  const lastBytesRef = useRef(0); // última lectura cruda del pc actual
  const pcRef = useRef(null);     // pc al que pertenece lastBytes (detecta cambio de pc)

  useEffect(() => {
    if (!streamKey) return;
    let cancelled = false;

    const tick = async () => {
      if (cancelled) return;
      const pc = getPc?.();
      let delta = 0;

      if (pc && pc.connectionState !== 'closed') {
        try {
          const current = await readBytes(pc);
          if (pc !== pcRef.current) {
            // pc nuevo (reconexión): el contador parte de cero, todo es incremento.
            pcRef.current = pc;
            delta = current;
          } else if (current >= lastBytesRef.current) {
            delta = current - lastBytesRef.current;
          } else {
            // El contador retrocedió sin cambiar de pc (raro): tratar como reset.
            delta = current;
          }
          lastBytesRef.current = current;
        } catch {
          /* getStats puede fallar durante la renegociación: omitir esta muestra */
        }
      }

      const now = new Date();
      const payload = {
        date: localDateStr(now),
        slot: slotOfDay(now),
        bytes: delta > 0 ? { [streamKey]: delta } : {},
      };
      // Aunque no haya datos, el heartbeat marca el slot como "wall abierto".
      usageHeartbeat(payload).catch(() => { /* no bloquear el wall por el historial */ });
    };

    const interval = setInterval(tick, SAMPLE_MS);
    // Una primera marca rápida para no perder el arranque si la ventana se cierra pronto.
    const warmup = setTimeout(tick, 3000);

    return () => {
      cancelled = true;
      clearInterval(interval);
      clearTimeout(warmup);
    };
  }, [getPc, streamKey]);
}
