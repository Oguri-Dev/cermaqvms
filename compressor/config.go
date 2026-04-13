package main

import (
	"os"
	"strconv"
)

// Config holds all environment-driven configuration for the compressor service.
type Config struct {
	Port         string // HTTP API port (default: 8087)
	MongoURI     string // MongoDB connection URI
	MongoDBName  string // MongoDB database name
	MediaMTXRTSP string // MediaMTX RTSP publish base URL (ffmpeg outputs here)
	FFmpegPath   string // Path to ffmpeg binary
	PollInterval int    // Seconds between MongoDB config polls
	RTSPPort     string // NVR RTSP port (default: 554)
}

func LoadConfig() *Config {
	return &Config{
		Port:         getEnv("COMPRESSOR_PORT", "8087"),
		MongoURI:     getEnv("MONGO_URI", "mongodb://localhost:27017"),
		MongoDBName:  getEnv("MONGO_DB_NAME", "vms_cermaq"),
		MediaMTXRTSP: getEnv("MEDIAMTX_RTSP", "rtsp://localhost:8554"),
		FFmpegPath:   getEnv("FFMPEG_PATH", "ffmpeg"),
		PollInterval: getEnvInt("POLL_INTERVAL", 5),
		RTSPPort:     getEnv("NVR_RTSP_PORT", "554"),
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
