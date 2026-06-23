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
		AllowedHeaders:   []string{"Content-Type", "Authorization"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	r.Route("/api", func(r chi.Router) {
		// ── Público: login ──
		r.Post("/auth/login", handlers.Login)

		// ── Autenticado (admin u operador): visor, wall, zoom y PTZ ──
		r.Group(func(r chi.Router) {
			r.Use(handlers.RequireAuth)

			r.Get("/auth/me", handlers.Me)

			// El wall y la pantalla individual leen estos:
			r.Get("/screen-config", handlers.GetScreenConfig)
			r.Get("/streams/{id}/full", handlers.GetStreamFull)

			// Centro: lectura + zoom in-stream (el operador maximiza cámaras)
			r.Route("/center/screens", func(r chi.Router) {
				r.Get("/", handlers.ListCenterScreens)
				r.Route("/{fileName}", func(r chi.Router) {
					r.Get("/", handlers.GetCenterScreen)
					r.Post("/zoom/{cellName}", handlers.ZoomCenterCell)
					r.Post("/unzoom", handlers.UnzoomCenter)
					r.Get("/zoom-status", handlers.CenterZoomStatus)
					r.Get("/health", handlers.CenterHealth)
				})
			})

			// PTZ del centro (el operador mueve domos)
			r.Get("/center/ptz/{camera}/{action}", handlers.CenterPTZ)
			r.Post("/center/ptz/{camera}/{action}", handlers.CenterPTZ)

			// Reinicio de la compresión (admin u operador)
			r.Post("/center/compression/restart", handlers.RestartCompression)

			// Secuencia de arranque del wall (admin u operador): reinicia
			// compresión + MediaMTX (si está activo) + servicio PTZ. Protegido
			// contra disparos repetidos por ventana de tiempo en el backend.
			r.Post("/center/bootstrap", handlers.CenterBootstrap)

			// PTZ por celda (modelo local)
			r.Post("/streams/{id}/ptz/{row}/{col}/move", handlers.PTZMove)
			r.Post("/streams/{id}/ptz/{row}/{col}/preset", handlers.PTZPresetAction)

			// Historial de uso: cada ventana del wall reporta su consumo
			r.Post("/usage/heartbeat", handlers.UsageHeartbeat)
		})

		// ── Solo admin: toda la configuración y gestión ──
		r.Group(func(r chi.Router) {
			r.Use(handlers.RequireAuth)
			r.Use(handlers.RequireAdmin)

			r.Put("/screen-config", handlers.UpdateScreenConfig)

			// Activar/desactivar la transmisión de una pantalla del centro
			r.Put("/center/screens/{fileName}/active", handlers.SetCenterScreenActive)

			r.Get("/center-config", handlers.GetCenterConfig)
			r.Put("/center-config", handlers.UpdateCenterConfig)
			r.Get("/center-config/status", handlers.CenterConnectionStatus)
			r.Post("/center/import", handlers.ImportFromCenter)

			// Reportes de uso (historial de horas + datos por stream)
			r.Get("/usage", handlers.GetUsage)
			r.Get("/usage/export", handlers.ExportUsage)

			// Gestión de usuarios
			r.Get("/users", handlers.ListUsers)
			r.Post("/users", handlers.CreateUser)
			r.Put("/users/{id}", handlers.UpdateUser)
			r.Delete("/users/{id}", handlers.DeleteUser)

			// Devices (visor; PTZ editable)
			r.Route("/devices", func(r chi.Router) {
				r.Get("/", handlers.ListDevices)
				r.Post("/", handlers.CreateDevice)
				r.Route("/{id}", func(r chi.Router) {
					r.Get("/", handlers.GetDevice)
					r.Put("/", handlers.UpdateDevice)
					r.Delete("/", handlers.DeleteDevice)
				})
			})

			// Grids
			r.Route("/grids", func(r chi.Router) {
				r.Get("/", handlers.ListGrids)
				r.Post("/", handlers.CreateGrid)
				r.Route("/{id}", func(r chi.Router) {
					r.Get("/", handlers.GetGrid)
					r.Put("/", handlers.UpdateGrid)
					r.Delete("/", handlers.DeleteGrid)
				})
			})

			// Streams (CRUD y administración)
			r.Route("/streams", func(r chi.Router) {
				r.Get("/", handlers.ListStreams)
				r.Post("/", handlers.CreateStream)
				r.Route("/{id}", func(r chi.Router) {
					r.Get("/", handlers.GetStream)
					r.Put("/", handlers.UpdateStream)
					r.Delete("/", handlers.DeleteStream)
					r.Post("/sync", handlers.SyncStreamHandler)
					r.Get("/screen-config", handlers.PreviewScreenConfigHandler)
				})
			})
		})
	})

	return r
}
