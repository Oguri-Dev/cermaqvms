package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"vms-cermaq/database"
	"vms-cermaq/models"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// PTZMove handles PTZ movement commands
func PTZMove(w http.ResponseWriter, r *http.Request) {
	cam, err := getDeviceFromStreamCell(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if !cam.HasPTZ {
		http.Error(w, "This device does not support PTZ", http.StatusBadRequest)
		return
	}

	var cmd models.PTZCommand
	if err := json.NewDecoder(r.Body).Decode(&cmd); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	target := cam.IP

	response := map[string]interface{}{
		"status":  "ok",
		"target":  target,
		"channel": cam.NVRChannel,
		"command": cmd,
	}

	respondJSON(w, http.StatusOK, response)
}

// PTZPresetAction handles PTZ preset commands
func PTZPresetAction(w http.ResponseWriter, r *http.Request) {
	cam, err := getDeviceFromStreamCell(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if !cam.HasPTZ {
		http.Error(w, "This device does not support PTZ", http.StatusBadRequest)
		return
	}

	var preset models.PTZPreset
	if err := json.NewDecoder(r.Body).Decode(&preset); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	target := cam.IP

	response := map[string]interface{}{
		"status":  "ok",
		"target":  target,
		"channel": cam.NVRChannel,
		"preset":  preset,
	}

	respondJSON(w, http.StatusOK, response)
}

func getDeviceFromStreamCell(r *http.Request) (*models.Device, error) {
	streamID, err := primitive.ObjectIDFromHex(chi.URLParam(r, "id"))
	if err != nil {
		return nil, fmt.Errorf("invalid stream ID")
	}

	row, err := strconv.Atoi(chi.URLParam(r, "row"))
	if err != nil {
		return nil, fmt.Errorf("invalid row")
	}

	col, err := strconv.Atoi(chi.URLParam(r, "col"))
	if err != nil {
		return nil, fmt.Errorf("invalid col")
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var stream models.Stream
	if err := database.GetCollection("streams").FindOne(ctx, bson.M{"_id": streamID}).Decode(&stream); err != nil {
		return nil, fmt.Errorf("stream not found")
	}

	// Find cell
	var cameraID primitive.ObjectID
	for _, c := range stream.Cells {
		if c.Row == row && c.Col == col {
			cameraID = c.CameraID
			break
		}
	}

	if cameraID.IsZero() {
		return nil, fmt.Errorf("no camera assigned to cell [%d,%d]", row, col)
	}

	var cam models.Device
	if err := database.GetCollection("devices").FindOne(ctx, bson.M{"_id": cameraID}).Decode(&cam); err != nil {
		return nil, fmt.Errorf("device not found")
	}

	return &cam, nil
}
