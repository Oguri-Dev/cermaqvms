package config

import (
	"os"
)

type Config struct {
	Port           string
	MongoURI       string
	MongoDBName    string
	PontonMongoURI string // Remote pontón MongoDB (optional, for sync)
	PontonDBName   string // Remote pontón DB name (e.g. "VMSWeb")
	CenterMongoURI string // MongoDB del centro: lectura de screen_configuration (solo lectura)
	CenterDBName   string // Base del centro (e.g. "camancha_vsmweb")
	CenterHost     string // Host del centro: API zoom GST-Grid (:8087) y WHEP (:8889)
	CenterPTZHost  string // Override del host del servicio PTZ (:8088); vacío = CenterHost
	AuthSecret     string // Secreto para firmar los tokens de sesión (JWT)
}

func Load() *Config {
	return &Config{
		Port:           getEnv("PORT", "8080"),
		MongoURI:       getEnv("MONGO_URI", "mongodb://localhost:27017"),
		MongoDBName:    getEnv("MONGO_DB_NAME", "vms_cermaq"),
		PontonMongoURI: getEnv("PONTON_MONGO_URI", ""),
		PontonDBName:   getEnv("PONTON_DB_NAME", "VMSWeb"),
		CenterMongoURI: getEnv("CENTER_MONGO_URI", "mongodb://10.1.1.229:27017"),
		CenterDBName:   getEnv("CENTER_DB_NAME", "camancha_vsmweb"),
		CenterHost:     getEnv("CENTER_HOST", "10.1.1.229"),
		CenterPTZHost:  getEnv("CENTER_PTZ_HOST", ""),
		AuthSecret:     getEnv("AUTH_SECRET", ""),
	}
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}
