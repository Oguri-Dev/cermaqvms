package handlers

// Lectura de la configuración que corre en el centro (colección screen_configuration,
// generada en la puesta en marcha) y proxy al API de zoom del compresor GST-Grid
// (:8087). Ver docs/api_http.md. La única escritura permitida al centro es un $set
// quirúrgico del campo `active` (activar/desactivar una pantalla); nunca se reemplaza
// el documento completo.

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"vms-cermaq/database"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/bson"
)

// CenterHost es el host donde corren el GST-Grid (:8087), el WHEP (:8889)
// y el servicio PTZ (:8088). Se setea desde main.
var CenterHost string

// CenterPTZHost permite apuntar el proxy PTZ a otro host (útil en desarrollo,
// cuando el servicio PTZ corre local). Vacío = usar CenterHost.
var CenterPTZHost string

func ptzHost() string {
	if CenterPTZHost != "" {
		return CenterPTZHost
	}
	return CenterHost
}

const (
	compressorPort = 8087
	whepPort       = 8889
	ptzPort        = 8088
)

var compressorClient = &http.Client{Timeout: 8 * time.Second}

// centerScreenDoc decodifica SOLO los campos que el VMS necesita del documento real.
// Los documentos tienen campos adicionales (rowspan, hasOverlay, content, etc.) que
// se ignoran a propósito.
type centerScreenDoc struct {
	FileName string `bson:"fileName"`
	Active   bool   `bson:"active"`
	Grid     struct {
		Name   string `bson:"name"`
		Rows   int    `bson:"rows"`
		Cols   int    `bson:"columns"`
		Width  int    `bson:"widthResolution"`
		Height int    `bson:"heightResolution"`
		Cells  []struct {
			IDCell int    `bson:"idCell"`
			Name   string `bson:"name"`
			Type   string `bson:"type"`
			On     bool   `bson:"on"`
		} `bson:"gridCells"`
	} `bson:"grid_configuration"`
}

type centerCell struct {
	Row    int    `json:"row"`
	Col    int    `json:"col"`
	Name   string `json:"name"`
	Type   string `json:"type"`
	On     bool   `json:"on"`
	HasPTZ bool   `json:"has_ptz"`
}

type centerScreen struct {
	FileName string       `json:"file_name"`
	Active   bool         `json:"active"`
	Rows     int          `json:"rows"`
	Cols     int          `json:"cols"`
	Width    int          `json:"width_resolution"`
	Height   int          `json:"height_resolution"`
	WhepURL  string       `json:"whep_url"`
	Cells    []centerCell `json:"cells"`
}

// localPtzMap devuelve nombre (minúsculas) → has_ptz de las cámaras de la
// base LOCAL: ahí es donde el operador edita qué cámara tiene controles PTZ
// (sección Dispositivos), normalmente tras importar desde el centro.
func localPtzMap(ctx context.Context) map[string]bool {
	m := map[string]bool{}
	cursor, err := database.GetCollection("devices").Find(ctx, bson.M{"type": "camera"})
	if err != nil {
		return m
	}
	defer cursor.Close(ctx)
	for cursor.Next(ctx) {
		var d struct {
			Name   string `bson:"name"`
			HasPTZ bool   `bson:"has_ptz"`
		}
		if err := cursor.Decode(&d); err == nil && d.Name != "" {
			m[strings.ToLower(strings.TrimSpace(d.Name))] = d.HasPTZ
		}
	}
	return m
}

func toCenterScreen(doc centerScreenDoc, ptzMap map[string]bool) centerScreen {
	s := centerScreen{
		FileName: doc.FileName,
		Active:   doc.Active,
		Rows:     doc.Grid.Rows,
		Cols:     doc.Grid.Cols,
		Width:    doc.Grid.Width,
		Height:   doc.Grid.Height,
		WhepURL:  fmt.Sprintf("http://%s:%d/%s/", CenterHost, whepPort, doc.FileName),
		Cells:    []centerCell{},
	}
	// Algunos documentos traen más gridCells que rows*cols (restos de ediciones
	// anteriores); las celdas fuera de rango no se muestran en el mosaico.
	for _, c := range doc.Grid.Cells {
		if doc.Grid.Cols <= 0 || c.IDCell < 0 || c.IDCell >= doc.Grid.Rows*doc.Grid.Cols {
			continue
		}
		// La cámara editada localmente manda; si no está importada aún,
		// se infiere del tipo que traiga la celda del centro.
		hasPtz, edited := ptzMap[strings.ToLower(strings.TrimSpace(c.Name))]
		if !edited {
			hasPtz = ptzTypeRe.MatchString(c.Type)
		}
		s.Cells = append(s.Cells, centerCell{
			Row:    c.IDCell / doc.Grid.Cols,
			Col:    c.IDCell % doc.Grid.Cols,
			Name:   c.Name,
			Type:   c.Type,
			On:     c.On,
			HasPTZ: hasPtz,
		})
	}
	return s
}

func centerAvailable(w http.ResponseWriter) bool {
	if database.CenterDB == nil {
		respondJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "sin conexión a la base de datos del centro"})
		return false
	}
	return true
}

// ListCenterScreens — GET /api/center/screens
func ListCenterScreens(w http.ResponseWriter, r *http.Request) {
	if !centerAvailable(w) {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
	defer cancel()

	cursor, err := database.CenterDB.Collection("screen_configuration").Find(ctx, bson.M{})
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	defer cursor.Close(ctx)

	ptzMap := localPtzMap(ctx)
	screens := []centerScreen{}
	for cursor.Next(ctx) {
		var doc centerScreenDoc
		if err := cursor.Decode(&doc); err != nil {
			continue
		}
		screens = append(screens, toCenterScreen(doc, ptzMap))
	}
	respondJSON(w, http.StatusOK, screens)
}

// GetCenterScreen — GET /api/center/screens/{fileName}
func GetCenterScreen(w http.ResponseWriter, r *http.Request) {
	if !centerAvailable(w) {
		return
	}
	fileName := chi.URLParam(r, "fileName")
	ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
	defer cancel()

	var doc centerScreenDoc
	err := database.CenterDB.Collection("screen_configuration").FindOne(ctx, bson.M{"fileName": fileName}).Decode(&doc)
	if err != nil {
		respondJSON(w, http.StatusNotFound, map[string]string{"error": "pantalla no encontrada en el centro: " + fileName})
		return
	}
	respondJSON(w, http.StatusOK, toCenterScreen(doc, localPtzMap(ctx)))
}

// SetCenterScreenActive — PUT /api/center/screens/{fileName}/active  body: {"active":true}
// Activa o desactiva la transmisión de una pantalla. Hace un $set QUIRÚRGICO de
// un solo campo en el documento del centro: no toca id_center, gridCells ni
// ningún otro campo (a diferencia de un ReplaceOne, que sería destructivo).
// El GST-Grid recoge el cambio y arranca/detiene el stream.
func SetCenterScreenActive(w http.ResponseWriter, r *http.Request) {
	if !centerAvailable(w) {
		return
	}
	fileName := chi.URLParam(r, "fileName")
	var body struct {
		Active bool `json:"active"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "JSON inválido"})
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
	defer cancel()

	res, err := database.CenterDB.Collection("screen_configuration").
		UpdateOne(ctx, bson.M{"fileName": fileName}, bson.M{"$set": bson.M{"active": body.Active}})
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if res.MatchedCount == 0 {
		respondJSON(w, http.StatusNotFound, map[string]string{"error": "pantalla no encontrada en el centro: " + fileName})
		return
	}
	respondJSON(w, http.StatusOK, map[string]interface{}{"success": true, "file_name": fileName, "active": body.Active})
}

// proxyCompressor reenvía un GET al API del GST-Grid y retransmite la respuesta JSON tal cual.
func proxyCompressor(w http.ResponseWriter, r *http.Request, path string) {
	target := fmt.Sprintf("http://%s:%d%s", CenterHost, compressorPort, path)
	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, target, nil)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	resp, err := compressorClient.Do(req)
	if err != nil {
		respondJSON(w, http.StatusBadGateway, map[string]string{"error": "compresor no disponible: " + err.Error()})
		return
	}
	defer resp.Body.Close()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

// ZoomCenterCell — POST /api/center/screens/{fileName}/zoom/{cellName}
// Maximiza la cámara dentro del mismo stream compuesto (instantáneo, sin reconexión).
func ZoomCenterCell(w http.ResponseWriter, r *http.Request) {
	fileName := chi.URLParam(r, "fileName")
	cellName, err := url.PathUnescape(chi.URLParam(r, "cellName"))
	if err != nil || cellName == "" {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "nombre de celda inválido"})
		return
	}
	// El matching del compresor normaliza quitando espacios y guiones; enviar el
	// nombre ya normalizado evita depender de que su servidor HTTP decodifique
	// percent-encoding (ej. "PTZ 1" → "ptz1").
	normalized := strings.ToLower(strings.NewReplacer(" ", "", "-", "").Replace(cellName))
	proxyCompressor(w, r, fmt.Sprintf("/%s/start/%s", fileName, url.PathEscape(normalized)))
}

// UnzoomCenter — POST /api/center/screens/{fileName}/unzoom
func UnzoomCenter(w http.ResponseWriter, r *http.Request) {
	proxyCompressor(w, r, fmt.Sprintf("/%s/stop", chi.URLParam(r, "fileName")))
}

// CenterZoomStatus — GET /api/center/screens/{fileName}/zoom-status
func CenterZoomStatus(w http.ResponseWriter, r *http.Request) {
	proxyCompressor(w, r, fmt.Sprintf("/%s/status", chi.URLParam(r, "fileName")))
}

// CenterHealth — GET /api/center/screens/{fileName}/health
// Estado por cámara (connected/inactive) para marcar celdas caídas en el overlay.
func CenterHealth(w http.ResponseWriter, r *http.Request) {
	proxyCompressor(w, r, fmt.Sprintf("/%s/cameras", chi.URLParam(r, "fileName")))
}

// proxyPTZ reenvía la petición (método y body incluidos) al servicio PTZ del centro.
// Las credenciales de las cámaras viven en ese servicio; aquí solo viaja el id.
func proxyPTZ(w http.ResponseWriter, r *http.Request, path string) {
	target := fmt.Sprintf("http://%s:%d%s", ptzHost(), ptzPort, path)
	req, err := http.NewRequestWithContext(r.Context(), r.Method, target, r.Body)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := compressorClient.Do(req)
	if err != nil {
		respondJSON(w, http.StatusBadGateway, map[string]string{"error": "servicio PTZ no disponible: " + err.Error()})
		return
	}
	defer resp.Body.Close()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

// RestartCompression — POST /api/center/compression/restart
// Proxy al servicio del centro que reinicia GST-Grid (systemctl restart).
func RestartCompression(w http.ResponseWriter, r *http.Request) {
	proxyPTZ(w, r, "/compression/restart")
}

// bootstrapWindow es el tiempo mínimo entre secuencias de arranque. El front
// dispara el bootstrap al abrir la app; varias ventanas del wall y los F5
// llegan casi a la vez, así que se ejecuta una sola vez por ventana de tiempo.
const bootstrapWindow = 5 * time.Minute

var (
	bootstrapMu   sync.Mutex
	bootstrapLast time.Time // cero = nunca ejecutado
)

// CenterBootstrap — POST /api/center/bootstrap
// Reinicia compresión, MediaMTX (si está activo) y el servicio PTZ del centro.
// Protegido contra disparos repetidos: si ya se ejecutó dentro de la ventana,
// responde "skipped" sin volver a llamar al centro (evita reinicios en cadena
// por F5 o por abrir varias ventanas del wall).
func CenterBootstrap(w http.ResponseWriter, r *http.Request) {
	bootstrapMu.Lock()
	now := time.Now()
	if !bootstrapLast.IsZero() && now.Sub(bootstrapLast) < bootstrapWindow {
		next := bootstrapWindow - now.Sub(bootstrapLast)
		bootstrapMu.Unlock()
		respondJSON(w, http.StatusOK, map[string]interface{}{
			"success":          true,
			"skipped":          true,
			"message":          "arranque ya ejecutado recientemente",
			"retry_in_seconds": int(next.Seconds()),
		})
		return
	}
	// Marca optimista: se reserva la ventana antes de llamar al centro para que
	// peticiones concurrentes (varias ventanas) no disparen reinicios paralelos.
	bootstrapLast = now
	bootstrapMu.Unlock()

	// El reinicio de servicios puede tardar; el proxy general usa 8s.
	target := fmt.Sprintf("http://%s:%d/bootstrap", ptzHost(), ptzPort)
	req, err := http.NewRequestWithContext(r.Context(), http.MethodPost, target, nil)
	if err != nil {
		bootstrapReset(now)
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		// Si el centro no respondió, libera la ventana para poder reintentar.
		bootstrapReset(now)
		respondJSON(w, http.StatusBadGateway, map[string]string{"error": "servicio del centro no disponible: " + err.Error()})
		return
	}
	defer resp.Body.Close()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

// bootstrapReset libera la ventana de dedupe si el arranque falló, pero solo
// si nadie más la reclamó mientras tanto (compara la marca propia).
func bootstrapReset(claimed time.Time) {
	bootstrapMu.Lock()
	if bootstrapLast.Equal(claimed) {
		bootstrapLast = time.Time{}
	}
	bootstrapMu.Unlock()
}

// CenterPTZ — /api/center/ptz/{camera}/(move|stop|goto|presets)
func CenterPTZ(w http.ResponseWriter, r *http.Request) {
	camera := chi.URLParam(r, "camera")
	action := chi.URLParam(r, "action")
	switch action {
	case "move", "stop", "goto", "presets":
		proxyPTZ(w, r, fmt.Sprintf("/ptz/%s/%s", url.PathEscape(camera), action))
	default:
		respondJSON(w, http.StatusNotFound, map[string]string{"error": "acción PTZ desconocida: " + action})
	}
}
