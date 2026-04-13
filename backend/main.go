package main

import (
	"log"
	"net/http"

	"vms-cermaq/config"
	"vms-cermaq/database"
	"vms-cermaq/routes"
)

func main() {
	cfg := config.Load()

	database.Connect(cfg.MongoURI, cfg.MongoDBName)

	router := routes.Setup()

	log.Printf("VMS Cermaq server starting on :%s", cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, router); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
