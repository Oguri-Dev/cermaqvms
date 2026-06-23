package main

// VMS PTZ Service — corre en el equipo de compresión del centro.
// Resuelve cámaras y credenciales desde la MongoDB del centro; el front
// solo envía el id/nombre de la cámara. Protocolos: ISAPI (Hikvision,
// digest) con fallback ONVIF (WS-Security), y dead-man switch que detiene
// la cámara si el comando de movimiento deja de renovarse.

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	cfg := LoadConfig()

	db, err := ConnectDB(cfg.MongoURI, cfg.MongoDBName)
	if err != nil {
		log.Fatalf("Error creando cliente MongoDB: %v", err)
	}

	ctrl := NewController(cfg, db)
	router := SetupRoutes(ctrl, db)

	server := &http.Server{Addr: ":" + cfg.Port, Handler: router}

	go func() {
		log.Printf("VMS PTZ service en :%s (db %s, stop-timeout %v)", cfg.Port, cfg.MongoDBName, cfg.StopTimeout)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	server.Shutdown(ctx)
	log.Println("Servicio detenido")
}
