package routes

import (
	"vms-cermaq/handlers"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

func Setup() *chi.Mux {
	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	r.Route("/api", func(r chi.Router) {
		// Screen config (singleton)
		r.Get("/screen-config", handlers.GetScreenConfig)
		r.Put("/screen-config", handlers.UpdateScreenConfig)

		// Devices CRUD (NVRs and cameras)
		r.Route("/devices", func(r chi.Router) {
			r.Get("/", handlers.ListDevices)
			r.Post("/", handlers.CreateDevice)
			r.Route("/{id}", func(r chi.Router) {
				r.Get("/", handlers.GetDevice)
				r.Put("/", handlers.UpdateDevice)
				r.Delete("/", handlers.DeleteDevice)
			})
		})

		// Grids CRUD (layout only)
		r.Route("/grids", func(r chi.Router) {
			r.Get("/", handlers.ListGrids)
			r.Post("/", handlers.CreateGrid)
			r.Route("/{id}", func(r chi.Router) {
				r.Get("/", handlers.GetGrid)
				r.Put("/", handlers.UpdateGrid)
				r.Delete("/", handlers.DeleteGrid)
			})
		})

		// Streams CRUD (grid + camera assignments + stream IP)
		r.Route("/streams", func(r chi.Router) {
			r.Get("/", handlers.ListStreams)
			r.Post("/", handlers.CreateStream)
			r.Route("/{id}", func(r chi.Router) {
				r.Get("/", handlers.GetStream)
				r.Put("/", handlers.UpdateStream)
				r.Delete("/", handlers.DeleteStream)
				r.Get("/full", handlers.GetStreamFull)
				r.Post("/sync", handlers.SyncStreamHandler)
				r.Get("/screen-config", handlers.PreviewScreenConfigHandler)

				// PTZ controls for a specific cell
				r.Post("/ptz/{row}/{col}/move", handlers.PTZMove)
				r.Post("/ptz/{row}/{col}/preset", handlers.PTZPresetAction)
			})
		})
	})

	return r
}
