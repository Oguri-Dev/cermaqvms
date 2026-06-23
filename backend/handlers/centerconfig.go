package handlers

// Configuración de la conexión al servidor del centro, editable desde el front.
// Se guarda como singleton en la colección local "center_config" y al guardar
// se reconecta en caliente (Mongo del centro + host del compresor).

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"vms-cermaq/database"
	"vms-cermaq/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Valores efectivos de la conexión al centro (defaults de env o lo guardado en Mongo local)
var (
	CenterMongoURI string
	CenterDBName   string
)

// InitCenter carga la configuración guardada (si existe) sobre los defaults de
// entorno y abre la conexión al Mongo del centro. Llamar después de Connect local.
func InitCenter(defaultURI, defaultDB, defaultHost string) {
	CenterMongoURI, CenterDBName, CenterHost = defaultURI, defaultDB, defaultHost

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var saved models.CenterConfig
	if err := database.GetCollection("center_config").FindOne(ctx, bson.M{}).Decode(&saved); err == nil {
		if saved.MongoURI != "" {
			CenterMongoURI = saved.MongoURI
		}
		if saved.DBName != "" {
			CenterDBName = saved.DBName
		}
		if saved.Host != "" {
			CenterHost = saved.Host
		}
		log.Printf("[centro] Using saved center config: %s/%s (host %s)", CenterMongoURI, CenterDBName, CenterHost)
	}

	if CenterMongoURI != "" {
		database.ConnectCenter(CenterMongoURI, CenterDBName)
	}
}

// GetCenterConfig — GET /api/center-config
func GetCenterConfig(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, models.CenterConfig{
		MongoURI: CenterMongoURI,
		DBName:   CenterDBName,
		Host:     CenterHost,
	})
}

// CenterConnectionStatus — GET /api/center-config/status
// Prueba ambas conexiones sin guardar ni reconectar nada.
func CenterConnectionStatus(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"mongo_connected":      testCenterMongo(r.Context()),
		"compressor_connected": testCompressor(r.Context()),
	})
}

// UpdateCenterConfig — PUT /api/center-config
// Guarda, reconecta y devuelve el resultado de probar ambas conexiones.
func UpdateCenterConfig(w http.ResponseWriter, r *http.Request) {
	var cfg models.CenterConfig
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "JSON inválido"})
		return
	}
	cfg.MongoURI = strings.TrimSpace(cfg.MongoURI)
	cfg.DBName = strings.TrimSpace(cfg.DBName)
	cfg.Host = strings.TrimSpace(cfg.Host)
	if cfg.MongoURI == "" || cfg.DBName == "" || cfg.Host == "" {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "mongo_uri, db_name y host son obligatorios"})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	_, err := database.GetCollection("center_config").
		ReplaceOne(ctx, bson.M{}, cfg, options.Replace().SetUpsert(true))
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	// Reconexión en caliente
	CenterMongoURI, CenterDBName, CenterHost = cfg.MongoURI, cfg.DBName, cfg.Host
	database.ConnectCenter(cfg.MongoURI, cfg.DBName)

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"config":               cfg,
		"mongo_connected":      testCenterMongo(r.Context()),
		"compressor_connected": testCompressor(r.Context()),
	})
}

func testCenterMongo(parent context.Context) bool {
	if database.CenterDB == nil {
		return false
	}
	ctx, cancel := context.WithTimeout(parent, 4*time.Second)
	defer cancel()
	return database.CenterDB.Client().Ping(ctx, nil) == nil
}

func testCompressor(parent context.Context) bool {
	ctx, cancel := context.WithTimeout(parent, 4*time.Second)
	defer cancel()
	// El GST-Grid no tiene endpoint de health global (solo rutas por fileName):
	// cualquier respuesta HTTP prueba que el servicio está arriba.
	url := fmt.Sprintf("http://%s:%d/", CenterHost, compressorPort)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return false
	}
	resp, err := compressorClient.Do(req)
	if err != nil {
		return false
	}
	resp.Body.Close()
	return true
}
