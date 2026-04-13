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

func ListGrids(w http.ResponseWriter, r *http.Request) {
	col := database.GetCollection("grids")
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	cursor, err := col.Find(ctx, bson.M{})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer cursor.Close(ctx)

	var grids []models.Grid
	if err := cursor.All(ctx, &grids); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if grids == nil {
		grids = []models.Grid{}
	}

	respondJSON(w, http.StatusOK, grids)
}

func GetGrid(w http.ResponseWriter, r *http.Request) {
	id, err := primitive.ObjectIDFromHex(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	col := database.GetCollection("grids")
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var grid models.Grid
	if err := col.FindOne(ctx, bson.M{"_id": id}).Decode(&grid); err != nil {
		http.Error(w, "Grid not found", http.StatusNotFound)
		return
	}

	respondJSON(w, http.StatusOK, grid)
}

func CreateGrid(w http.ResponseWriter, r *http.Request) {
	var grid models.Grid
	if err := json.NewDecoder(r.Body).Decode(&grid); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	col := database.GetCollection("grids")
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	result, err := col.InsertOne(ctx, grid)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	grid.ID = result.InsertedID.(primitive.ObjectID)
	respondJSON(w, http.StatusCreated, grid)
}

func UpdateGrid(w http.ResponseWriter, r *http.Request) {
	id, err := primitive.ObjectIDFromHex(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	var grid models.Grid
	if err := json.NewDecoder(r.Body).Decode(&grid); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	col := database.GetCollection("grids")
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	update := bson.M{
		"$set": bson.M{
			"name": grid.Name,
			"type": grid.Type,
			"rows": grid.Rows,
			"cols": grid.Cols,
		},
	}

	result, err := col.UpdateByID(ctx, id, update)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if result.MatchedCount == 0 {
		http.Error(w, "Grid not found", http.StatusNotFound)
		return
	}

	grid.ID = id
	respondJSON(w, http.StatusOK, grid)
}

func DeleteGrid(w http.ResponseWriter, r *http.Request) {
	id, err := primitive.ObjectIDFromHex(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	col := database.GetCollection("grids")
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	result, err := col.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if result.DeletedCount == 0 {
		http.Error(w, "Grid not found", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
