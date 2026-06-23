package main

import (
	"log"
	"net/http"

	"vms-cermaq/config"
	"vms-cermaq/database"
	"vms-cermaq/handlers"
	"vms-cermaq/routes"
)

func main() {
	cfg := config.Load()

	database.Connect(cfg.MongoURI, cfg.MongoDBName)

	if cfg.PontonMongoURI != "" {
		database.ConnectPonton(cfg.PontonMongoURI, cfg.PontonDBName)
	}

	handlers.InitCenter(cfg.CenterMongoURI, cfg.CenterDBName, cfg.CenterHost)
	handlers.CenterPTZHost = cfg.CenterPTZHost
	handlers.InitAuth(cfg.AuthSecret)

	router := routes.Setup()

	log.Printf("VMS Cermaq server starting on :%s", cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, router); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
