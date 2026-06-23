package main

import (
	"os"
	"strconv"
	"strings"
	"time"
)

// Config del servicio PTZ. Corre en el equipo de compresión del centro,
// junto a la base de datos que tiene las cámaras y sus credenciales.
type Config struct {
	Port        string
	MongoURI    string
	MongoDBName string
	// Dead-man switch: si el front deja de renovar el comando de movimiento
	// (keepalive), el servicio detiene la cámara pasado este tiempo.
	StopTimeout time.Duration
	// Timeout de cada llamada HTTP hacia cámaras/NVRs.
	CameraTimeout time.Duration
	// Comando para reiniciar el servicio de compresión (configurable por si
	// cambia el nombre del servicio o el mecanismo). Se ejecuta tal cual.
	CompressionRestartCmd []string
	// Comando para reiniciar MediaMTX (solo si el servicio existe/está activo).
	MediaMTXRestartCmd []string
	// Comando para comprobar si MediaMTX está activo (systemctl is-active).
	// Si devuelve "active", el bootstrap intenta reiniciarlo; si no, lo omite.
	MediaMTXCheckCmd []string
	// Comando para reiniciar el propio servicio PTZ. Se ejecuta en background
	// tras responder (systemd lo levanta de nuevo por Restart=always).
	PTZRestartCmd []string
	// Timeout de cada comando de reinicio.
	RestartTimeout time.Duration
}

func LoadConfig() *Config {
	return &Config{
		Port:          getEnv("PTZ_PORT", "8088"),
		MongoURI:      getEnv("MONGO_URI", "mongodb://localhost:27017"),
		MongoDBName:   getEnv("MONGO_DB_NAME", "camancha_vsmweb"),
		StopTimeout:   time.Duration(getEnvInt("STOP_TIMEOUT_MS", 2500)) * time.Millisecond,
		CameraTimeout: time.Duration(getEnvInt("CAMERA_TIMEOUT_MS", 4000)) * time.Millisecond,
		// Default: systemctl restart gst-grid. Override con COMPRESSION_RESTART_CMD
		// (separado por espacios) si cambia el nombre del servicio.
		CompressionRestartCmd: strings.Fields(getEnv("COMPRESSION_RESTART_CMD", "systemctl restart gst-grid")),
		// MediaMTX: solo se reinicia si el servicio EXISTE en el equipo (el
		// compresor nuevo lo eliminó, pero algunos centros aún lo corren). Se
		// detecta por la presencia de la unit, no por su estado, para cubrir
		// también un mediamtx atascado en activating/failed (que un restart
		// justamente arregla).
		MediaMTXCheckCmd:   strings.Fields(getEnv("MEDIAMTX_CHECK_CMD", "systemctl list-unit-files mediamtx.service")),
		MediaMTXRestartCmd: strings.Fields(getEnv("MEDIAMTX_RESTART_CMD", "systemctl restart mediamtx")),
		// Reinicio del propio servicio PTZ (en background tras responder).
		PTZRestartCmd:  strings.Fields(getEnv("PTZ_RESTART_CMD", "systemctl restart vms-ptz")),
		RestartTimeout: time.Duration(getEnvInt("RESTART_TIMEOUT_MS", 15000)) * time.Millisecond,
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}
