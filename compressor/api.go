package main

import (
	"encoding/json"
	"net/http"
	"net/url"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

// SetupAPI creates the HTTP router with pontón-compatible API endpoints.
//
// Endpoints:
//   GET /health                              — Health check
//   GET /status                              — List all active pipelines
//   GET /{fileName}/start/{cameraName}       — Zoom into a single camera
//   GET /{fileName}/stop                     — Restore grid view (unzoom)
//   GET /{fileName}/status                   — Pipeline status
//   GET /{fileName}/cameras                  — List cameras in pipeline
//   GET /{fileName}/width-factor/{factor}    — Change horizontal scale (placeholder)
func SetupAPI(comp *Compositor) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type"},
		AllowCredentials: false,
	}))

	// Health check
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	// Global status — list all active pipelines
	r.Get("/status", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, comp.ListPipelines())
	})

	// Per-pipeline routes (pontón-compatible)
	r.Route("/{fileName}", func(r chi.Router) {
		// Zoom: maximize a single camera to full screen
		r.Get("/start/{cameraName}", func(w http.ResponseWriter, r *http.Request) {
			fileName := chi.URLParam(r, "fileName")
			cameraName, _ := url.PathUnescape(chi.URLParam(r, "cameraName"))

			p := comp.GetPipeline(fileName)
			if p == nil {
				http.Error(w, "pipeline not found", http.StatusNotFound)
				return
			}

			if err := p.Zoom(cameraName); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			writeJSON(w, http.StatusOK, map[string]string{
				"status": "zoomed",
				"camera": cameraName,
			})
		})

		// Unzoom: restore grid composition
		r.Get("/stop", func(w http.ResponseWriter, r *http.Request) {
			fileName := chi.URLParam(r, "fileName")

			p := comp.GetPipeline(fileName)
			if p == nil {
				http.Error(w, "pipeline not found", http.StatusNotFound)
				return
			}

			p.Unzoom()
			writeJSON(w, http.StatusOK, map[string]string{"status": "grid_restored"})
		})

		// Status for a specific pipeline
		r.Get("/status", func(w http.ResponseWriter, r *http.Request) {
			fileName := chi.URLParam(r, "fileName")

			p := comp.GetPipeline(fileName)
			if p == nil {
				http.Error(w, "pipeline not found", http.StatusNotFound)
				return
			}

			writeJSON(w, http.StatusOK, p.Status())
		})

		// List cameras in the pipeline's grid
		r.Get("/cameras", func(w http.ResponseWriter, r *http.Request) {
			fileName := chi.URLParam(r, "fileName")

			p := comp.GetPipeline(fileName)
			if p == nil {
				http.Error(w, "pipeline not found", http.StatusNotFound)
				return
			}

			cameras := p.Cameras()
			if cameras == nil {
				cameras = []CameraStatus{}
			}
			writeJSON(w, http.StatusOK, cameras)
		})

		// Width factor (placeholder — requires pipeline restart with modified w_factor)
		r.Get("/width-factor/{factor}", func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, http.StatusOK, map[string]string{"status": "not_implemented"})
		})
	})

	return r
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
