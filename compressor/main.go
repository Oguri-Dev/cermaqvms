package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
)

func main() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)

	cfg := LoadConfig()

	// Verify FFmpeg is available
	ffmpegPath, err := exec.LookPath(cfg.FFmpegPath)
	if err != nil {
		log.Fatalf("[main] FFmpeg not found at %q — install FFmpeg or set FFMPEG_PATH", cfg.FFmpegPath)
	}
	cfg.FFmpegPath = ffmpegPath
	log.Printf("[main] FFmpeg: %s", ffmpegPath)

	// Connect to MongoDB
	db := ConnectMongo(cfg.MongoURI, cfg.MongoDBName)

	// Start compositor (watches MongoDB, manages FFmpeg pipelines)
	ctx, cancel := context.WithCancel(context.Background())
	comp := NewCompositor(db, cfg)
	comp.Start(ctx)

	// Setup HTTP API (pontón-compatible)
	api := SetupAPI(comp)

	// Graceful shutdown on Ctrl+C
	srv := &http.Server{Addr: ":" + cfg.Port, Handler: api}
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, os.Interrupt)
		<-sigCh
		log.Println("[main] shutting down...")
		cancel()
		srv.Close()
	}()

	log.Println("==============================================")
	log.Println("  VMS Compressor — Omnifish")
	log.Println("==============================================")
	log.Printf("  API:       http://localhost:%s", cfg.Port)
	log.Printf("  RTSP out:  %s/<fileName>", cfg.MediaMTXRTSP)
	log.Printf("  MongoDB:   %s/%s", cfg.MongoURI, cfg.MongoDBName)
	log.Printf("  Poll:      every %ds", cfg.PollInterval)
	log.Println("==============================================")

	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatalf("[main] server error: %v", err)
	}
}
