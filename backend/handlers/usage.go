package handlers

// Historial de uso del wall (colección local "usage_daily"): horas de operación
// y consumo de datos por stream, para vigilar el enlace Starlink del centro.
//
// La captura nace en el navegador: cada ventana lee el bytesReceived de su
// RTCPeerConnection (WHEP) y envía heartbeats periódicos con el INCREMENTO de
// bytes desde el último envío y el slot de 30s en que ocurrió. El backend solo
// acumula ($inc por stream) y registra slots únicos de uptime ($addToSet), de
// modo que varias ventanas en el mismo instante no inflan las horas.

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"time"

	"vms-cermaq/database"
	"vms-cermaq/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// slotSeconds: tamaño del slot de uptime. 30s da resolución de medio minuto con
// un costo acotado de slots por día (2880 como máximo).
const slotSeconds = 30

var dateRe = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)

// usageHeartbeat es el cuerpo que envía el front cada ~30s.
type usageHeartbeat struct {
	Date  string           `json:"date"`  // YYYY-MM-DD local del wall
	Slot  int              `json:"slot"`  // índice de slot del día (segundos-del-día / 30)
	Bytes map[string]int64 `json:"bytes"` // incremento de bytes por fileName desde el último envío
}

// UsageHeartbeat — POST /api/usage/heartbeat  (autenticado)
func UsageHeartbeat(w http.ResponseWriter, r *http.Request) {
	var hb usageHeartbeat
	if err := json.NewDecoder(r.Body).Decode(&hb); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "JSON inválido"})
		return
	}
	if !dateRe.MatchString(hb.Date) {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "fecha inválida (se espera YYYY-MM-DD)"})
		return
	}
	if hb.Slot < 0 || hb.Slot >= 86400/slotSeconds {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "slot fuera de rango"})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 6*time.Second)
	defer cancel()

	update := bson.M{
		"$set":      bson.M{"updated_at": time.Now()},
		"$addToSet": bson.M{"uptime_slots": hb.Slot},
	}
	// $inc solo los streams con incremento positivo (ignora ruido y deltas a 0).
	inc := bson.M{}
	for name, delta := range hb.Bytes {
		if delta > 0 && name != "" {
			inc["bytes_by_stream."+name] = delta
		}
	}
	if len(inc) > 0 {
		update["$inc"] = inc
	}

	_, err := database.GetCollection("usage_daily").UpdateOne(
		ctx, bson.M{"_id": hb.Date}, update, options.Update().SetUpsert(true),
	)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	respondJSON(w, http.StatusOK, map[string]bool{"success": true})
}

// usageDay es la forma expuesta al front (slots ya traducidos a segundos/total).
type usageDay struct {
	Date          string           `json:"date"`
	OpenSeconds   int              `json:"open_seconds"`
	BytesTotal    int64            `json:"bytes_total"`
	BytesByStream map[string]int64 `json:"bytes_by_stream"`
}

// queryUsage carga y normaliza los días dentro del rango [from, to] inclusive.
func queryUsage(ctx context.Context, from, to string) ([]usageDay, error) {
	cursor, err := database.GetCollection("usage_daily").Find(
		ctx,
		bson.M{"_id": bson.M{"$gte": from, "$lte": to}},
		options.Find().SetSort(bson.M{"_id": 1}),
	)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	days := []usageDay{}
	for cursor.Next(ctx) {
		var d models.UsageDaily
		if err := cursor.Decode(&d); err != nil {
			continue
		}
		day := usageDay{
			Date:          d.Date,
			OpenSeconds:   len(d.UptimeSlots) * slotSeconds,
			BytesByStream: d.BytesByStream,
		}
		if day.BytesByStream == nil {
			day.BytesByStream = map[string]int64{}
		}
		for _, b := range day.BytesByStream {
			day.BytesTotal += b
		}
		days = append(days, day)
	}
	return days, nil
}

// rangeParams lee y valida from/to del query string (default: hoy..hoy según el
// front no aplica aquí; si faltan se usa un rango amplio del mes en curso).
func rangeParams(r *http.Request) (string, string, bool) {
	from := r.URL.Query().Get("from")
	to := r.URL.Query().Get("to")
	if from == "" || to == "" || !dateRe.MatchString(from) || !dateRe.MatchString(to) {
		return "", "", false
	}
	if from > to {
		from, to = to, from
	}
	return from, to, true
}

// GetUsage — GET /api/usage?from=YYYY-MM-DD&to=YYYY-MM-DD  (admin)
func GetUsage(w http.ResponseWriter, r *http.Request) {
	from, to, ok := rangeParams(r)
	if !ok {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "se requieren from y to (YYYY-MM-DD)"})
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
	defer cancel()

	days, err := queryUsage(ctx, from, to)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	// Totales del rango
	var totalSeconds int
	var totalBytes int64
	byStream := map[string]int64{}
	for _, d := range days {
		totalSeconds += d.OpenSeconds
		totalBytes += d.BytesTotal
		for name, b := range d.BytesByStream {
			byStream[name] += b
		}
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"from": from,
		"to":   to,
		"days": days,
		"totals": map[string]interface{}{
			"open_seconds":    totalSeconds,
			"bytes_total":     totalBytes,
			"bytes_by_stream": byStream,
		},
	})
}

// ExportUsage — GET /api/usage/export?from=...&to=...  (admin)
// Devuelve un CSV con una fila por día: fecha, horas abiertas y MB por stream.
func ExportUsage(w http.ResponseWriter, r *http.Request) {
	from, to, ok := rangeParams(r)
	if !ok {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "se requieren from y to (YYYY-MM-DD)"})
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
	defer cancel()

	days, err := queryUsage(ctx, from, to)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	// Conjunto ordenado de todos los streams presentes en el rango (columnas).
	streamSet := map[string]struct{}{}
	for _, d := range days {
		for name := range d.BytesByStream {
			streamSet[name] = struct{}{}
		}
	}
	streams := make([]string, 0, len(streamSet))
	for name := range streamSet {
		streams = append(streams, name)
	}
	sort.Strings(streams)

	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"uso_%s_a_%s.csv\"", from, to))

	cw := csv.NewWriter(w)
	defer cw.Flush()

	// Cabecera: Fecha, Horas abiertas, Total (MB), <stream...> (MB)
	header := []string{"Fecha", "Horas abiertas", "Total (MB)"}
	for _, s := range streams {
		header = append(header, s+" (MB)")
	}
	cw.Write(header)

	for _, d := range days {
		row := []string{
			d.Date,
			fmt.Sprintf("%.2f", float64(d.OpenSeconds)/3600.0),
			mbStr(d.BytesTotal),
		}
		for _, s := range streams {
			row = append(row, mbStr(d.BytesByStream[s]))
		}
		cw.Write(row)
	}
}

// mbStr convierte bytes a megabytes (decimal, 1 MB = 1e6 bytes) con 2 decimales.
func mbStr(b int64) string {
	return strconv.FormatFloat(float64(b)/1e6, 'f', 2, 64)
}
