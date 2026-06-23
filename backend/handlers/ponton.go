package handlers

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"vms-cermaq/database"
	"vms-cermaq/models"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// buildScreenConfiguration converts a VMS Stream into the pontón's screen_configuration format.
// It resolves the grid, cameras, and NVRs to populate every field the compression system expects.
func buildScreenConfiguration(ctx context.Context, stream models.Stream) (*models.PontonScreenConfiguration, error) {
	// 1. Resolve grid for rows/columns
	var grid models.Grid
	if err := database.GetCollection("grids").FindOne(ctx, bson.M{"_id": stream.GridID}).Decode(&grid); err != nil {
		return nil, fmt.Errorf("grid not found: %w", err)
	}

	// 2. Collect all camera IDs from cells
	cameraIDs := []primitive.ObjectID{}
	for _, c := range stream.Cells {
		if !c.CameraID.IsZero() {
			cameraIDs = append(cameraIDs, c.CameraID)
		}
	}

	// 3. Fetch cameras
	camerasMap := map[primitive.ObjectID]models.Device{}
	if len(cameraIDs) > 0 {
		cursor, err := database.GetCollection("devices").Find(ctx, bson.M{"_id": bson.M{"$in": cameraIDs}})
		if err != nil {
			return nil, fmt.Errorf("fetching cameras: %w", err)
		}
		var cameras []models.Device
		if err := cursor.All(ctx, &cameras); err != nil {
			return nil, fmt.Errorf("decoding cameras: %w", err)
		}
		for _, cam := range cameras {
			camerasMap[cam.ID] = cam
		}
	}

	// 4. Fetch NVRs referenced by cameras
	nvrIDs := []primitive.ObjectID{}
	for _, cam := range camerasMap {
		if !cam.NVRID.IsZero() {
			nvrIDs = append(nvrIDs, cam.NVRID)
		}
	}
	nvrsMap := map[primitive.ObjectID]models.Device{}
	if len(nvrIDs) > 0 {
		cursor, err := database.GetCollection("devices").Find(ctx, bson.M{"_id": bson.M{"$in": nvrIDs}})
		if err != nil {
			return nil, fmt.Errorf("fetching NVRs: %w", err)
		}
		var nvrs []models.Device
		if err := cursor.All(ctx, &nvrs); err != nil {
			return nil, fmt.Errorf("decoding NVRs: %w", err)
		}
		for _, nvr := range nvrs {
			nvrsMap[nvr.ID] = nvr
		}
	}

	// 5. Build cell lookup from stream cells (row,col → StreamCell)
	cellMap := map[string]models.StreamCell{}
	for _, c := range stream.Cells {
		key := fmt.Sprintf("%d,%d", c.Row, c.Col)
		cellMap[key] = c
	}

	// 6. Generate ALL gridCells (rows * columns), including empty ones
	totalCells := grid.Rows * grid.Cols
	gridCells := make([]models.PontonGridCell, totalCells)

	for i := 0; i < totalCells; i++ {
		row := i / grid.Cols
		col := i % grid.Cols
		key := fmt.Sprintf("%d,%d", row, col)

		cell := models.PontonGridCell{
			IDCell:  i,
			On:      false,
			WFactor: 1.0,
		}

		if sc, ok := cellMap[key]; ok && !sc.CameraID.IsZero() {
			if cam, ok := camerasMap[sc.CameraID]; ok {
				cell.Name = cam.Name
				cell.IPCamera = cam.IP
				cell.NVRChannel = cam.NVRChannel
				cell.MediaMTXCamera1 = cam.MediaMTXCamera1
				cell.MediaMTXCamera2 = cam.MediaMTXCamera2
				cell.On = sc.Active

				if sc.WFactor > 0 {
					cell.WFactor = sc.WFactor
				}

				// Resolve credentials: NVR first, fall back to camera's own
				if nvr, ok := nvrsMap[cam.NVRID]; ok {
					cell.IPNVR = nvr.IP
					cell.User = nvr.User
					cell.Pass = nvr.Pass
				} else {
					// Direct camera — use its own credentials
					cell.User = cam.User
					cell.Pass = cam.Pass
				}
			}
		}

		gridCells[i] = cell
	}

	// 7. Build the final pontón document
	doc := &models.PontonScreenConfiguration{
		FileName:         stream.FileName,
		Active:           stream.IsActive,
		Bitrate:          stream.Bitrate,
		HardwareEncoding: stream.HardwareEncoding,
		IPServer:         stream.IPServer,
		GridConfiguration: models.PontonGridConfiguration{
			Rows:             grid.Rows,
			Columns:          grid.Cols,
			WidthResolution:  stream.WidthResolution,
			HeightResolution: stream.HeightResolution,
			SelectFlow:       stream.SelectFlow,
			FPS:              stream.FPS,
			GOP:              stream.GOP,
			PCID:             stream.PCID,
			GridCells:        gridCells,
			UnionCells:       []interface{}{},
		},
	}

	return doc, nil
}

// SyncScreenConfiguration generates and upserts the pontón screen_configuration document.
// It matches by fileName (unique per stream/screen in the compression system).
func SyncScreenConfiguration(ctx context.Context, stream models.Stream) error {
	doc, err := buildScreenConfiguration(ctx, stream)
	if err != nil {
		return err
	}

	filter := bson.M{"fileName": stream.FileName}
	opts := options.Replace().SetUpsert(true)

	// Write to local MongoDB
	col := database.GetCollection("screen_configuration")
	_, err = col.ReplaceOne(ctx, filter, doc, opts)
	if err != nil {
		return fmt.Errorf("upserting screen_configuration (local): %w", err)
	}
	log.Printf("[pontón] screen_configuration synced locally for fileName=%q", stream.FileName)

	// Write to remote pontón MongoDB if configured
	if database.PontonDB != nil {
		remoteCol := database.PontonDB.Collection("screen_configuration")
		_, err = remoteCol.ReplaceOne(ctx, filter, doc, opts)
		if err != nil {
			log.Printf("[pontón] WARNING: remote sync failed for fileName=%q: %v", stream.FileName, err)
		} else {
			log.Printf("[pontón] screen_configuration synced to REMOTE for fileName=%q", stream.FileName)
		}
	}

	return nil
}

// SyncStreamHandler is the HTTP handler for manual sync: POST /api/streams/{id}/sync
func SyncStreamHandler(w http.ResponseWriter, r *http.Request) {
	id, err := primitive.ObjectIDFromHex(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	var stream models.Stream
	if err := database.GetCollection("streams").FindOne(ctx, bson.M{"_id": id}).Decode(&stream); err != nil {
		http.Error(w, "Stream not found", http.StatusNotFound)
		return
	}

	if err := SyncScreenConfiguration(ctx, stream); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{
		"status":   "synced",
		"fileName": stream.FileName,
	})
}

// PreviewScreenConfigHandler returns the pontón document without saving: GET /api/streams/{id}/screen-config
func PreviewScreenConfigHandler(w http.ResponseWriter, r *http.Request) {
	id, err := primitive.ObjectIDFromHex(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	var stream models.Stream
	if err := database.GetCollection("streams").FindOne(ctx, bson.M{"_id": id}).Decode(&stream); err != nil {
		http.Error(w, "Stream not found", http.StatusNotFound)
		return
	}

	doc, err := buildScreenConfiguration(ctx, stream)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusOK, doc)
}
