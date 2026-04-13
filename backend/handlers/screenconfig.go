package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"vms-cermaq/database"
	"vms-cermaq/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// GetScreenConfig returns the singleton screen configuration
func GetScreenConfig(w http.ResponseWriter, r *http.Request) {
	col := database.GetCollection("screen_config")
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var config models.ScreenConfig
	err := col.FindOne(ctx, bson.M{}).Decode(&config)
	if err != nil {
		// Return default if none exists
		config = models.ScreenConfig{
			CenterName: "Centro Omnifish",
			Layout:     1,
			Screens:    []models.ScreenSlot{},
		}
	}

	respondJSON(w, http.StatusOK, config)
}

// UpdateScreenConfig upserts the singleton screen configuration
func UpdateScreenConfig(w http.ResponseWriter, r *http.Request) {
	var config models.ScreenConfig
	if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	col := database.GetCollection("screen_config")
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	opts := options.Replace().SetUpsert(true)
	filter := bson.M{}
	if !config.ID.IsZero() {
		filter = bson.M{"_id": config.ID}
	}

	result, err := col.ReplaceOne(ctx, filter, config, opts)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if result.UpsertedID != nil {
		if oid, ok := result.UpsertedID.(interface{ Hex() string }); ok {
			_ = oid
		}
	}

	respondJSON(w, http.StatusOK, config)
}
