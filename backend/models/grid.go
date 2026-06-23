package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type OnDemandMode string

const (
	OnDemandNVR    OnDemandMode = "nvr"
	OnDemandDirect OnDemandMode = "direct"
)

type GridType string

const (
	GridSubmarineType GridType = "submarine"
	GridDomeType      GridType = "dome"
)

type DeviceType string

const (
	DeviceNVR    DeviceType = "nvr"
	DeviceCamera DeviceType = "camera"
)

// Device represents an NVR or a camera (manual entry by installer)
type Device struct {
	ID   primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	Name string             `json:"name" bson:"name"`
	Type DeviceType         `json:"type" bson:"type"`
	IP   string             `json:"ip" bson:"ip"`

	// NVR-specific fields (credentials for RTSP auth)
	User string `json:"user,omitempty" bson:"user,omitempty"`
	Pass string `json:"pass,omitempty" bson:"pass,omitempty"`

	// Camera-specific fields
	NVRID           primitive.ObjectID `json:"nvr_id,omitempty" bson:"nvr_id,omitempty"`
	NVRChannel      int                `json:"nvr_channel,omitempty" bson:"nvr_channel,omitempty"`
	CageID          string             `json:"cage_id,omitempty" bson:"cage_id,omitempty"`
	CageName        string             `json:"cage_name,omitempty" bson:"cage_name,omitempty"`
	CameraType      string             `json:"camera_type,omitempty" bson:"camera_type,omitempty"` // "submarina", "PTZ"
	OnDemandMode    OnDemandMode       `json:"ondemand_mode,omitempty" bson:"ondemand_mode,omitempty"`
	HasPTZ          bool               `json:"has_ptz,omitempty" bson:"has_ptz,omitempty"`
	MediaMTXCamera1 string             `json:"mediamtx_camera1,omitempty" bson:"mediamtx_camera1,omitempty"`
	MediaMTXCamera2 string             `json:"mediamtx_camera2,omitempty" bson:"mediamtx_camera2,omitempty"`
}

// Grid represents a layout template (rows x cols)
type Grid struct {
	ID   primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	Name string             `json:"name" bson:"name"`
	Type GridType           `json:"type" bson:"type"`
	Rows int                `json:"rows" bson:"rows"`
	Cols int                `json:"cols" bson:"cols"`
}

// StreamCell maps a cell position to a camera with activation and scale settings
type StreamCell struct {
	Row      int                `json:"row" bson:"row"`
	Col      int                `json:"col" bson:"col"`
	CameraID primitive.ObjectID `json:"camera_id,omitempty" bson:"camera_id,omitempty"`
	Active   bool               `json:"active" bson:"active"`                         // maps to pontón "on" field
	WFactor  float64            `json:"w_factor,omitempty" bson:"w_factor,omitempty"` // horizontal scale factor (0.5-1.0)
}

// Stream combines a grid + camera assignments + compression settings
// This model produces screen_configuration documents for the pontón compression system
type Stream struct {
	ID   primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	Name string             `json:"name" bson:"name"`

	// Grid reference
	GridID primitive.ObjectID `json:"grid_id" bson:"grid_id"`

	// Stream output (for VMS frontend consumption via WHEP)
	StreamIP string `json:"stream_ip" bson:"stream_ip"` // WHEP endpoint URL for composed stream

	// Pontón screen_configuration fields (sent to compression system)
	FileName         string `json:"file_name" bson:"file_name"`                 // screen name, used for RTSP URL: rtsp://<ipServer>:8554/<fileName>
	IPServer         string `json:"ip_server" bson:"ip_server"`                 // IP of compression server
	IsActive         bool   `json:"is_active" bson:"is_active"`                 // true = transmits, false = stops
	Bitrate          int    `json:"bitrate" bson:"bitrate"`                     // output bitrate in kbps
	HardwareEncoding int    `json:"hardware_encoding" bson:"hardware_encoding"` // 1=CPU, 2=GPU, 3=GPU dec+CPU comp+GPU enc, 4=CPU dec+GPU enc
	WidthResolution  int    `json:"width_resolution" bson:"width_resolution"`   // output width (e.g. 1920)
	HeightResolution int    `json:"height_resolution" bson:"height_resolution"` // output height (e.g. 1080)
	SelectFlow       int    `json:"select_flow" bson:"select_flow"`             // 1/2=NVR, 3/4=MediaMTX, 5/6=direct camera
	FPS              int    `json:"fps" bson:"fps"`                             // frames per second
	GOP              int    `json:"gop" bson:"gop"`                             // group of pictures
	PCID             int    `json:"pc_id,omitempty" bson:"pc_id,omitempty"`     // ID of PC executing this grid

	// Camera cell assignments
	Cells []StreamCell `json:"cells" bson:"cells"`
}

// ScreenSlot maps a physical screen to a stream
type ScreenSlot struct {
	Slot         int                `json:"slot" bson:"slot"`
	StreamID     primitive.ObjectID `json:"stream_id" bson:"stream_id"`
	ScreenLabel  string             `json:"screen_label,omitempty" bson:"screen_label,omitempty"`
	ScreenLeft   int                `json:"screen_left" bson:"screen_left"`
	ScreenTop    int                `json:"screen_top" bson:"screen_top"`
	ScreenWidth  int                `json:"screen_width" bson:"screen_width"`
	ScreenHeight int                `json:"screen_height" bson:"screen_height"`
}

// ScreenConfig is a singleton that stores the monitor layout configuration
// and the cell-label OSD preferences (editable from the client)
type ScreenConfig struct {
	ID          primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	CenterName  string             `json:"center_name" bson:"center_name"`
	Layout      int                `json:"layout" bson:"layout"`
	Screens     []ScreenSlot       `json:"screens" bson:"screens"`
	OSDSize     int                `json:"osd_size,omitempty" bson:"osd_size,omitempty"`         // px de la etiqueta de celda
	OSDPosition string             `json:"osd_position,omitempty" bson:"osd_position,omitempty"` // top-left | top-right | bottom-left | bottom-right
	GridColor   string             `json:"grid_color,omitempty" bson:"grid_color,omitempty"`     // color hex de las líneas de la grilla
}

// User represents a VMS account (local DB). Roles: "admin" | "operator".
type User struct {
	ID           primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	Username     string             `json:"username" bson:"username"`
	PasswordHash string             `json:"-" bson:"password_hash"`
	Role         string             `json:"role" bson:"role"`
}

// CenterConfig is a singleton that stores the connection to the center server
// (Mongo with screen_configuration + GST-Grid compressor host)
type CenterConfig struct {
	MongoURI string `json:"mongo_uri" bson:"mongo_uri"`
	DBName   string `json:"db_name" bson:"db_name"`
	Host     string `json:"host" bson:"host"`
}

// UsageDaily es el resumen de uso de un día (colección local "usage_daily"),
// para vigilar el consumo de datos del enlace Starlink y las horas de operación
// del wall. Un documento por día, identificado por la fecha LOCAL (la envía el
// front, que conoce la zona del operador). Se actualiza con $inc/$addToSet desde
// los heartbeats del navegador.
type UsageDaily struct {
	Date string `json:"date" bson:"_id"` // YYYY-MM-DD (fecha local del wall)
	// BytesByStream acumula los bytes recibidos (WebRTC bytesReceived) por
	// stream/fileName. Es la suma de todas las ventanas que reprodujeron ese
	// stream durante el día.
	BytesByStream map[string]int64 `json:"bytes_by_stream" bson:"bytes_by_stream"`
	// UptimeSlots son los índices de slots de 30s del día en que hubo AL MENOS
	// una ventana abierta. Se cuentan únicos para derivar las horas operativas
	// (varias ventanas en el mismo slot cuentan una sola vez).
	UptimeSlots []int     `json:"-" bson:"uptime_slots"`
	UpdatedAt   time.Time `json:"updated_at" bson:"updated_at"`
}

// PTZCommand represents a PTZ movement command
type PTZCommand struct {
	Action string  `json:"action"`
	Speed  float64 `json:"speed"`
}

// PTZPreset represents a PTZ preset call
type PTZPreset struct {
	PresetID int    `json:"preset_id"`
	Action   string `json:"action"`
}
