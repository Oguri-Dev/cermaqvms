package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"vms-cermaq/database"
	"vms-cermaq/models"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func ListDevices(w http.ResponseWriter, r *http.Request) {
	col := database.GetCollection("devices")
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	cursor, err := col.Find(ctx, bson.M{})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer cursor.Close(ctx)

	var devices []models.Device
	if err := cursor.All(ctx, &devices); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if devices == nil {
		devices = []models.Device{}
	}

	respondJSON(w, http.StatusOK, devices)
}

func GetDevice(w http.ResponseWriter, r *http.Request) {
	id, err := primitive.ObjectIDFromHex(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	col := database.GetCollection("devices")
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var device models.Device
	if err := col.FindOne(ctx, bson.M{"_id": id}).Decode(&device); err != nil {
		http.Error(w, "Device not found", http.StatusNotFound)
		return
	}

	respondJSON(w, http.StatusOK, device)
}

func CreateDevice(w http.ResponseWriter, r *http.Request) {
	var device models.Device
	if err := json.NewDecoder(r.Body).Decode(&device); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	col := database.GetCollection("devices")
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	result, err := col.InsertOne(ctx, device)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	device.ID = result.InsertedID.(primitive.ObjectID)
	respondJSON(w, http.StatusCreated, device)
}

func UpdateDevice(w http.ResponseWriter, r *http.Request) {
	id, err := primitive.ObjectIDFromHex(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	var device models.Device
	if err := json.NewDecoder(r.Body).Decode(&device); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	col := database.GetCollection("devices")
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	update := bson.M{
		"$set": bson.M{
			"name":              device.Name,
			"type":              device.Type,
			"ip":                device.IP,
			"user":              device.User,
			"pass":              device.Pass,
			"nvr_id":            device.NVRID,
			"nvr_channel":       device.NVRChannel,
			"cage_id":           device.CageID,
			"cage_name":         device.CageName,
			"camera_type":       device.CameraType,
			"ondemand_mode":     device.OnDemandMode,
			"has_ptz":           device.HasPTZ,
			"mediamtx_camera1":  device.MediaMTXCamera1,
			"mediamtx_camera2":  device.MediaMTXCamera2,
		},
	}

	result, err := col.UpdateByID(ctx, id, update)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if result.MatchedCount == 0 {
		http.Error(w, "Device not found", http.StatusNotFound)
		return
	}

	device.ID = id
	respondJSON(w, http.StatusOK, device)
}

func DeleteDevice(w http.ResponseWriter, r *http.Request) {
	id, err := primitive.ObjectIDFromHex(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	col := database.GetCollection("devices")
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	result, err := col.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if result.DeletedCount == 0 {
		http.Error(w, "Device not found", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
