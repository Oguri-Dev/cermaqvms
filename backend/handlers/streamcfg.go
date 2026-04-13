package handlers

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"vms-cermaq/database"
	"vms-cermaq/models"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func ListStreams(w http.ResponseWriter, r *http.Request) {
	col := database.GetCollection("streams")
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	cursor, err := col.Find(ctx, bson.M{})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer cursor.Close(ctx)

	var streams []models.Stream
	if err := cursor.All(ctx, &streams); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if streams == nil {
		streams = []models.Stream{}
	}

	respondJSON(w, http.StatusOK, streams)
}

func GetStream(w http.ResponseWriter, r *http.Request) {
	id, err := primitive.ObjectIDFromHex(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	col := database.GetCollection("streams")
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var stream models.Stream
	if err := col.FindOne(ctx, bson.M{"_id": id}).Decode(&stream); err != nil {
		http.Error(w, "Stream not found", http.StatusNotFound)
		return
	}

	respondJSON(w, http.StatusOK, stream)
}

func CreateStream(w http.ResponseWriter, r *http.Request) {
	var stream models.Stream
	if err := json.NewDecoder(r.Body).Decode(&stream); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	col := database.GetCollection("streams")
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	result, err := col.InsertOne(ctx, stream)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	stream.ID = result.InsertedID.(primitive.ObjectID)

	// Auto-sync to pontón screen_configuration
	if stream.FileName != "" {
		if err := SyncScreenConfiguration(ctx, stream); err != nil {
			log.Printf("[pontón] sync error on create: %v", err)
		}
	}

	respondJSON(w, http.StatusCreated, stream)
}

func UpdateStream(w http.ResponseWriter, r *http.Request) {
	id, err := primitive.ObjectIDFromHex(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	var stream models.Stream
	if err := json.NewDecoder(r.Body).Decode(&stream); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	col := database.GetCollection("streams")
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	update := bson.M{
		"$set": bson.M{
			"name":              stream.Name,
			"grid_id":           stream.GridID,
			"stream_ip":         stream.StreamIP,
			"file_name":         stream.FileName,
			"ip_server":         stream.IPServer,
			"is_active":         stream.IsActive,
			"bitrate":           stream.Bitrate,
			"hardware_encoding": stream.HardwareEncoding,
			"width_resolution":  stream.WidthResolution,
			"height_resolution": stream.HeightResolution,
			"select_flow":       stream.SelectFlow,
			"fps":               stream.FPS,
			"gop":               stream.GOP,
			"pc_id":             stream.PCID,
			"cells":             stream.Cells,
		},
	}

	result, err := col.UpdateByID(ctx, id, update)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if result.MatchedCount == 0 {
		http.Error(w, "Stream not found", http.StatusNotFound)
		return
	}

	stream.ID = id

	// Auto-sync to pontón screen_configuration
	if stream.FileName != "" {
		if err := SyncScreenConfiguration(ctx, stream); err != nil {
			log.Printf("[pontón] sync error on update: %v", err)
		}
	}

	respondJSON(w, http.StatusOK, stream)
}

func DeleteStream(w http.ResponseWriter, r *http.Request) {
	id, err := primitive.ObjectIDFromHex(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	col := database.GetCollection("streams")
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	result, err := col.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if result.DeletedCount == 0 {
		http.Error(w, "Stream not found", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GetStreamFull returns a stream with its grid and cameras resolved
func GetStreamFull(w http.ResponseWriter, r *http.Request) {
	id, err := primitive.ObjectIDFromHex(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	// Get stream
	var stream models.Stream
	if err := database.GetCollection("streams").FindOne(ctx, bson.M{"_id": id}).Decode(&stream); err != nil {
		http.Error(w, "Stream not found", http.StatusNotFound)
		return
	}

	// Get grid
	var grid models.Grid
	if err := database.GetCollection("grids").FindOne(ctx, bson.M{"_id": stream.GridID}).Decode(&grid); err != nil {
		http.Error(w, "Grid not found", http.StatusNotFound)
		return
	}

	// Get devices (cameras) referenced in cells
	deviceIDs := []primitive.ObjectID{}
	for _, c := range stream.Cells {
		if !c.CameraID.IsZero() {
			deviceIDs = append(deviceIDs, c.CameraID)
		}
	}

	devicesMap := map[primitive.ObjectID]models.Device{}
	if len(deviceIDs) > 0 {
		cursor, err := database.GetCollection("devices").Find(ctx, bson.M{"_id": bson.M{"$in": deviceIDs}})
		if err == nil {
			var devices []models.Device
			cursor.All(ctx, &devices)
			for _, dev := range devices {
				devicesMap[dev.ID] = dev
			}
		}
	}

	// Resolve NVR IPs for cameras that reference an NVR
	nvrIDs := []primitive.ObjectID{}
	for _, dev := range devicesMap {
		if !dev.NVRID.IsZero() {
			nvrIDs = append(nvrIDs, dev.NVRID)
		}
	}
	nvrsMap := map[primitive.ObjectID]models.Device{}
	if len(nvrIDs) > 0 {
		cursor, err := database.GetCollection("devices").Find(ctx, bson.M{"_id": bson.M{"$in": nvrIDs}})
		if err == nil {
			var nvrs []models.Device
			cursor.All(ctx, &nvrs)
			for _, nvr := range nvrs {
				nvrsMap[nvr.ID] = nvr
			}
		}
	}

	// Build response
	type CellFull struct {
		Row    int            `json:"row"`
		Col    int            `json:"col"`
		Camera *models.Device `json:"camera,omitempty"`
		NVR    *models.Device `json:"nvr,omitempty"`
	}

	cells := []CellFull{}
	for _, c := range stream.Cells {
		cf := CellFull{Row: c.Row, Col: c.Col}
		if dev, ok := devicesMap[c.CameraID]; ok {
			cf.Camera = &dev
			if nvr, ok := nvrsMap[dev.NVRID]; ok {
				cf.NVR = &nvr
			}
		}
		cells = append(cells, cf)
	}

	response := map[string]interface{}{
		"id":        stream.ID,
		"name":      stream.Name,
		"stream_ip": stream.StreamIP,
		"grid": map[string]interface{}{
			"id":   grid.ID,
			"name": grid.Name,
			"type": grid.Type,
			"rows": grid.Rows,
			"cols": grid.Cols,
		},
		"cells": cells,
	}

	respondJSON(w, http.StatusOK, response)
}
