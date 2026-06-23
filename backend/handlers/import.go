package handlers

// Importación desde la base del centro hacia la base local del cliente:
// NVRs, cámaras, grillas y streams con sus coordenadas. La base local es la
// capa editable (ej. marcar una cámara como PTZ); por eso la importación es
// idempotente y NUNCA pisa los campos editados localmente (has_ptz,
// camera_type, ondemand_mode). El centro jamás se modifica.

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"vms-cermaq/database"
	"vms-cermaq/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var ptzTypeRe = regexp.MustCompile(`(?i)ptz|domo|dome`)

type importNvrDoc struct {
	ID        primitive.ObjectID `bson:"_id"`
	Name      string             `bson:"name"`
	IPAddress string             `bson:"ipAddress"`
	User      string             `bson:"user"`
	Password  string             `bson:"password"`
}

type importCameraDoc struct {
	Name       string             `bson:"name"`
	IPCamera   string             `bson:"ipCamera"`
	IPNvr      string             `bson:"ipNvr"`
	NvrChannel *int               `bson:"nvrChannel"`
	User       string             `bson:"user"`
	Pass       string             `bson:"pass"`
	Type       string             `bson:"type"`
	NvrID      primitive.ObjectID `bson:"nvrId,omitempty"`
}

type importScreenDoc struct {
	FileName string `bson:"fileName"`
	Active   bool   `bson:"active"`
	Bitrate  int    `bson:"bitrate"`
	HwEnc    int    `bson:"hardwareEncoding"`
	Grid     struct {
		Rows       int  `bson:"rows"`
		Cols       int  `bson:"columns"`
		Width      int  `bson:"widthResolution"`
		Height     int  `bson:"heightResolution"`
		SelectFlow int  `bson:"selectFlow"`
		FPS        int  `bson:"fps"`
		GOP        int  `bson:"gop"`
		Cells      []struct {
			IDCell   int    `bson:"idCell"`
			Name     string `bson:"name"`
			Type     string `bson:"type"`
			On       bool   `bson:"on"`
			IPCamera string `bson:"ipCamera"`
			IPNvr    string `bson:"ipNvr"`
			Channel  *int   `bson:"nvrChannel"`
			User     string `bson:"user"`
			Pass     string `bson:"pass"`
			M1       string `bson:"mediamtxCamera1"`
			M2       string `bson:"mediamtxCamera2"`
		} `bson:"gridCells"`
	} `bson:"grid_configuration"`
}

// camSource acumula la información de una cámara (colección cameras del
// centro complementada con los datos de las celdas).
type camSource struct {
	Name    string
	IP      string
	IPNvr   string
	Channel int
	User    string
	Pass    string
	Type    string
	NvrID   string // _id hex del NVR en el centro
	M1, M2  string
}

func nameKey(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

// ImportFromCenter — POST /api/center/import
func ImportFromCenter(w http.ResponseWriter, r *http.Request) {
	if !centerAvailable(w) {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	// ── 1. Leer todo del centro (solo lectura) ──
	var nvrs []importNvrDoc
	if cur, err := database.CenterDB.Collection("nvr").Find(ctx, bson.M{}); err == nil {
		cur.All(ctx, &nvrs)
	}

	var centerCams []importCameraDoc
	if cur, err := database.CenterDB.Collection("cameras").Find(ctx, bson.M{}); err == nil {
		cur.All(ctx, &centerCams)
	}

	cur, err := database.CenterDB.Collection("screen_configuration").Find(ctx, bson.M{})
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	var screens []importScreenDoc
	if err := cur.All(ctx, &screens); err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "leyendo screen_configuration: " + err.Error()})
		return
	}

	devices := database.GetCollection("devices")
	counts := map[string]int{"nvrs": 0, "cameras": 0, "grids": 0, "streams": 0}

	// ── 2. NVRs → devices locales (match por nombre) ──
	nvrLocalID := map[string]primitive.ObjectID{} // _id hex centro -> _id local
	for _, n := range nvrs {
		if n.Name == "" {
			continue
		}
		filter := bson.M{"type": "nvr", "name": exactNameFilter(n.Name)}
		update := bson.M{"$set": bson.M{
			"name": n.Name, "type": "nvr", "ip": n.IPAddress,
			"user": n.User, "pass": n.Password,
		}}
		id, err := upsertReturningID(ctx, devices, filter, update)
		if err != nil {
			continue
		}
		nvrLocalID[n.ID.Hex()] = id
		counts["nvrs"]++
	}

	// ── 3. Consolidar cámaras: colección cameras + datos de celdas ──
	camMap := map[string]*camSource{}
	order := []string{}
	for _, c := range centerCams {
		if c.Name == "" {
			continue
		}
		key := nameKey(c.Name)
		ch := 0
		if c.NvrChannel != nil {
			ch = *c.NvrChannel
		}
		camMap[key] = &camSource{
			Name: c.Name, IP: c.IPCamera, IPNvr: c.IPNvr, Channel: ch,
			User: c.User, Pass: c.Pass, Type: c.Type, NvrID: c.NvrID.Hex(),
		}
		order = append(order, key)
	}
	for _, s := range screens {
		for _, cell := range s.Grid.Cells {
			if cell.Name == "" || !cell.On {
				continue
			}
			key := nameKey(cell.Name)
			src, ok := camMap[key]
			if !ok {
				ch := 0
				if cell.Channel != nil {
					ch = *cell.Channel
				}
				src = &camSource{
					Name: cell.Name, IP: cell.IPCamera, IPNvr: cell.IPNvr, Channel: ch,
					User: cell.User, Pass: cell.Pass, Type: cell.Type,
				}
				camMap[key] = src
				order = append(order, key)
			}
			// Los paths MediaMTX solo existen en las celdas
			if src.M1 == "" {
				src.M1 = cell.M1
			}
			if src.M2 == "" {
				src.M2 = cell.M2
			}
		}
	}

	// ── 4. Cámaras → devices locales (preservando ediciones locales) ──
	camLocalID := map[string]primitive.ObjectID{}
	for _, key := range order {
		src := camMap[key]
		set := bson.M{
			"name": src.Name, "type": "camera", "ip": src.IP,
			"user": src.User, "pass": src.Pass,
			"nvr_channel": src.Channel,
			"mediamtx_camera1": src.M1, "mediamtx_camera2": src.M2,
		}
		if id, ok := nvrLocalID[src.NvrID]; ok {
			set["nvr_id"] = id
		}
		isPtz := ptzTypeRe.MatchString(src.Type)
		camType := "submarina"
		if isPtz {
			camType = "PTZ"
		}
		// Campos editables localmente: solo se fijan al crear
		setOnInsert := bson.M{
			"has_ptz": isPtz, "camera_type": camType,
			"ondemand_mode": "nvr", "cage_name": src.Name,
		}
		filter := bson.M{"type": "camera", "name": exactNameFilter(src.Name)}
		update := bson.M{"$set": set, "$setOnInsert": setOnInsert}
		id, err := upsertReturningID(ctx, devices, filter, update)
		if err != nil {
			continue
		}
		camLocalID[key] = id
		counts["cameras"]++
	}

	// ── 5. Grillas y streams por screen_configuration ──
	grids := database.GetCollection("grids")
	streamsColl := database.GetCollection("streams")
	for _, s := range screens {
		if s.FileName == "" || s.Grid.Rows <= 0 || s.Grid.Cols <= 0 {
			continue
		}
		gridUpdate := bson.M{"$set": bson.M{
			"name": s.FileName, "type": "submarine",
			"rows": s.Grid.Rows, "cols": s.Grid.Cols,
		}}
		gridID, err := upsertReturningID(ctx, grids, bson.M{"name": s.FileName}, gridUpdate)
		if err != nil {
			continue
		}
		counts["grids"]++

		// Celdas con coordenadas (idCell row-major) y cámara asignada
		cells := []models.StreamCell{}
		total := s.Grid.Rows * s.Grid.Cols
		assigned := map[int]models.StreamCell{}
		for _, cell := range s.Grid.Cells {
			if cell.IDCell < 0 || cell.IDCell >= total {
				continue
			}
			sc := models.StreamCell{
				Row:    cell.IDCell / s.Grid.Cols,
				Col:    cell.IDCell % s.Grid.Cols,
				Active: cell.On,
			}
			if id, ok := camLocalID[nameKey(cell.Name)]; ok && cell.On {
				sc.CameraID = id
			}
			assigned[cell.IDCell] = sc
		}
		for i := 0; i < total; i++ {
			if sc, ok := assigned[i]; ok {
				cells = append(cells, sc)
			} else {
				cells = append(cells, models.StreamCell{Row: i / s.Grid.Cols, Col: i % s.Grid.Cols})
			}
		}

		streamUpdate := bson.M{"$set": bson.M{
			"name": s.FileName, "grid_id": gridID,
			"file_name": s.FileName, "ip_server": CenterHost,
			"stream_ip": fmt.Sprintf("http://%s:%d/%s/", CenterHost, whepPort, s.FileName),
			"is_active": s.Active, "bitrate": s.Bitrate, "hardware_encoding": s.HwEnc,
			"width_resolution": s.Grid.Width, "height_resolution": s.Grid.Height,
			"select_flow": s.Grid.SelectFlow, "fps": s.Grid.FPS, "gop": s.Grid.GOP,
			"cells": cells,
		}}
		if _, err := upsertReturningID(ctx, streamsColl, bson.M{"file_name": s.FileName}, streamUpdate); err == nil {
			counts["streams"]++
		}
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"counts":  counts,
	})
}

func exactNameFilter(name string) bson.M {
	return bson.M{"$regex": "^" + regexp.QuoteMeta(name) + "$", "$options": "i"}
}

func upsertReturningID(ctx context.Context, coll *mongo.Collection, filter, update bson.M) (primitive.ObjectID, error) {
	opts := options.FindOneAndUpdate().SetUpsert(true).SetReturnDocument(options.After)
	var doc struct {
		ID primitive.ObjectID `bson:"_id"`
	}
	if err := coll.FindOneAndUpdate(ctx, filter, update, opts).Decode(&doc); err != nil {
		return primitive.NilObjectID, err
	}
	return doc.ID, nil
}
