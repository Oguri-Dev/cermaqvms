package main

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

func respond(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func respondErr(w http.ResponseWriter, status int, msg string) {
	respond(w, status, map[string]interface{}{"success": false, "error": msg})
}

func SetupRoutes(ctrl *Controller, db *Database) *chi.Mux {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders: []string{"Content-Type"},
	}))

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		mongoOK := db.Ping(r.Context()) == nil
		respond(w, http.StatusOK, map[string]interface{}{
			"success": true,
			"service": "vms-ptz",
			"mongo":   mongoOK,
		})
	})

	// Lista de cámaras (sin credenciales) para diagnóstico
	r.Get("/cameras", func(w http.ResponseWriter, r *http.Request) {
		cameras, err := db.ListCameras(r.Context())
		if err != nil {
			respondErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		respond(w, http.StatusOK, map[string]interface{}{"success": true, "cameras": cameras})
	})

	// Reinicio del servicio de compresión (corre en este mismo equipo)
	r.Post("/compression/restart", restartCompressionHandler(ctrl.cfg))

	// Secuencia de arranque del wall: reinicia compresión, MediaMTX (si está
	// activo) y el propio servicio PTZ.
	r.Post("/bootstrap", bootstrapHandler(ctrl.cfg))

	r.Route("/ptz/{camera}", func(r chi.Router) {
		// Resuelve la cámara por _id o nombre antes de cada acción
		withCamera := func(next func(w http.ResponseWriter, r *http.Request, cam *Camera)) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				cam, err := db.FindCamera(r.Context(), chi.URLParam(r, "camera"))
				if err != nil {
					respondErr(w, http.StatusNotFound, err.Error())
					return
				}
				next(w, r, cam)
			}
		}

		// Movimiento continuo. El front reenvía este comando periódicamente
		// mientras el control esté presionado: cada llamada renueva el
		// dead-man switch. Sin renovación, el servicio detiene la cámara solo.
		r.Post("/move", withCamera(func(w http.ResponseWriter, r *http.Request, cam *Camera) {
			var req MoveRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				respondErr(w, http.StatusBadRequest, "JSON inválido")
				return
			}
			proto, err := ctrl.Move(r.Context(), cam, req.Pan, req.Tilt, req.Zoom)
			if err != nil {
				respondErr(w, http.StatusBadGateway, err.Error())
				return
			}
			respond(w, http.StatusOK, map[string]interface{}{
				"success": true, "camera": cam.Name, "protocol": proto,
				"pan": req.Pan, "tilt": req.Tilt, "zoom": req.Zoom,
			})
		}))

		r.Post("/stop", withCamera(func(w http.ResponseWriter, r *http.Request, cam *Camera) {
			proto, err := ctrl.Stop(r.Context(), cam)
			if err != nil {
				respondErr(w, http.StatusBadGateway, err.Error())
				return
			}
			respond(w, http.StatusOK, map[string]interface{}{"success": true, "camera": cam.Name, "protocol": proto})
		}))

		r.Get("/presets", withCamera(func(w http.ResponseWriter, r *http.Request, cam *Camera) {
			presets, err := ctrl.Presets(r.Context(), cam)
			if err != nil {
				respondErr(w, http.StatusBadGateway, err.Error())
				return
			}
			respond(w, http.StatusOK, map[string]interface{}{"success": true, "camera": cam.Name, "presets": presets})
		}))

		r.Post("/goto", withCamera(func(w http.ResponseWriter, r *http.Request, cam *Camera) {
			var req GotoRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Preset == "" {
				respondErr(w, http.StatusBadRequest, "se requiere preset")
				return
			}
			proto, err := ctrl.Goto(r.Context(), cam, req.Preset)
			if err != nil {
				respondErr(w, http.StatusBadGateway, err.Error())
				return
			}
			respond(w, http.StatusOK, map[string]interface{}{"success": true, "camera": cam.Name, "preset": req.Preset, "protocol": proto})
		}))
	})

	return r
}
