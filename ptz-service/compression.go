package main

// Control del servicio de compresión (GST-Grid) que corre en este mismo
// equipo Linux. El reinicio corta el video de todas las pantallas del
// equipo unos segundos; el front lo confirma antes de llamar.

import (
	"context"
	"log"
	"net/http"
	"os/exec"
	"strings"
	"time"
)

// restartCompressionHandler ejecuta el comando configurado (por defecto:
// systemctl restart gst-grid). Devuelve un handler que captura la config.
func restartCompressionHandler(cfg *Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cmd := cfg.CompressionRestartCmd
		if len(cmd) == 0 {
			respondErr(w, http.StatusInternalServerError, "comando de reinicio no configurado")
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), cfg.RestartTimeout)
		defer cancel()

		log.Printf("[compression] reiniciando: %s", strings.Join(cmd, " "))
		out, err := exec.CommandContext(ctx, cmd[0], cmd[1:]...).CombinedOutput()
		if err != nil {
			log.Printf("[compression] error: %v | salida: %s", err, string(out))
			respondErr(w, http.StatusBadGateway, "no se pudo reiniciar la compresión: "+strings.TrimSpace(string(out)))
			return
		}

		log.Printf("[compression] reinicio OK")
		respond(w, http.StatusOK, map[string]interface{}{
			"success": true,
			"message": "compresión reiniciada",
		})
	}
}

// runCmd ejecuta un comando con timeout y devuelve su salida combinada.
func runCmd(ctx context.Context, cmd []string) (string, error) {
	if len(cmd) == 0 {
		return "", nil
	}
	out, err := exec.CommandContext(ctx, cmd[0], cmd[1:]...).CombinedOutput()
	return strings.TrimSpace(string(out)), err
}

// bootstrapHandler reinicia los servicios del equipo en orden al arrancar el
// wall: compresión (gst-grid), MediaMTX (solo si está activo) y el propio
// servicio PTZ. Devuelve el resultado de cada paso. El reinicio del PTZ se
// agenda en background tras responder, porque mataría este proceso (systemd
// lo levanta de nuevo por Restart=always).
func bootstrapHandler(cfg *Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		steps := []map[string]interface{}{}

		// 1) Compresión (gst-grid): reinicia todos los mosaicos de una vez.
		func() {
			ctx, cancel := context.WithTimeout(r.Context(), cfg.RestartTimeout)
			defer cancel()
			step := map[string]interface{}{"service": "compression"}
			if len(cfg.CompressionRestartCmd) == 0 {
				step["status"] = "skipped"
				step["detail"] = "comando no configurado"
			} else {
				log.Printf("[bootstrap] reiniciando compresión: %s", strings.Join(cfg.CompressionRestartCmd, " "))
				out, err := runCmd(ctx, cfg.CompressionRestartCmd)
				if err != nil {
					step["status"] = "error"
					step["detail"] = strings.TrimSpace(err.Error() + " " + out)
				} else {
					step["status"] = "ok"
				}
			}
			steps = append(steps, step)
		}()

		// 2) MediaMTX: solo si el servicio existe en el equipo. El compresor
		//    nuevo lo eliminó, pero algunos centros aún lo corren.
		func() {
			ctx, cancel := context.WithTimeout(r.Context(), cfg.RestartTimeout)
			defer cancel()
			step := map[string]interface{}{"service": "mediamtx"}
			// El check lista la unit: si systemd no la conoce, la salida no
			// menciona "mediamtx" (o el comando falla) y se omite el reinicio.
			out, _ := runCmd(ctx, cfg.MediaMTXCheckCmd)
			if !strings.Contains(out, "mediamtx") {
				step["status"] = "skipped"
				step["detail"] = "el servicio no existe en este equipo"
				steps = append(steps, step)
				return
			}
			log.Printf("[bootstrap] reiniciando mediamtx: %s", strings.Join(cfg.MediaMTXRestartCmd, " "))
			out, err := runCmd(ctx, cfg.MediaMTXRestartCmd)
			if err != nil {
				step["status"] = "error"
				step["detail"] = strings.TrimSpace(err.Error() + " " + out)
			} else {
				step["status"] = "ok"
			}
			steps = append(steps, step)
		}()

		// 3) Servicio PTZ: se reinicia a sí mismo en background tras responder.
		ptzStep := map[string]interface{}{"service": "ptz"}
		if len(cfg.PTZRestartCmd) == 0 {
			ptzStep["status"] = "skipped"
			ptzStep["detail"] = "comando no configurado"
		} else {
			ptzStep["status"] = "scheduled"
			ptzStep["detail"] = "se reiniciará tras responder"
			go func(cmd []string, timeout time.Duration) {
				// Pequeño respiro para que la respuesta HTTP salga antes de morir.
				time.Sleep(500 * time.Millisecond)
				ctx, cancel := context.WithTimeout(context.Background(), timeout)
				defer cancel()
				log.Printf("[bootstrap] reiniciando servicio PTZ: %s", strings.Join(cmd, " "))
				if out, err := runCmd(ctx, cmd); err != nil {
					log.Printf("[bootstrap] error reiniciando PTZ: %v | %s", err, out)
				}
			}(cfg.PTZRestartCmd, cfg.RestartTimeout)
		}
		steps = append(steps, ptzStep)

		respond(w, http.StatusOK, map[string]interface{}{
			"success": true,
			"message": "secuencia de arranque ejecutada",
			"steps":   steps,
		})
	}
}
