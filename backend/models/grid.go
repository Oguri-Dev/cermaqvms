package models

import "go.mongodb.org/mongo-driver/bson/primitive"

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
	Active   bool               `json:"active" bson:"active"`            // maps to pontón "on" field
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
	FileName         string `json:"file_name" bson:"file_name"`                   // screen name, used for RTSP URL: rtsp://<ipServer>:8554/<fileName>
	IPServer         string `json:"ip_server" bson:"ip_server"`                   // IP of compression server
	IsActive         bool   `json:"is_active" bson:"is_active"`                   // true = transmits, false = stops
	Bitrate          int    `json:"bitrate" bson:"bitrate"`                       // output bitrate in kbps
	HardwareEncoding int    `json:"hardware_encoding" bson:"hardware_encoding"`   // 1=CPU, 2=GPU, 3=GPU dec+CPU comp+GPU enc, 4=CPU dec+GPU enc
	WidthResolution  int    `json:"width_resolution" bson:"width_resolution"`     // output width (e.g. 1920)
	HeightResolution int    `json:"height_resolution" bson:"height_resolution"`   // output height (e.g. 1080)
	SelectFlow       int    `json:"select_flow" bson:"select_flow"`               // 1/2=NVR, 3/4=MediaMTX, 5/6=direct camera
	FPS              int    `json:"fps" bson:"fps"`                               // frames per second
	GOP              int    `json:"gop" bson:"gop"`                               // group of pictures
	PCID             int    `json:"pc_id,omitempty" bson:"pc_id,omitempty"`       // ID of PC executing this grid

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
type ScreenConfig struct {
	ID         primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	CenterName string             `json:"center_name" bson:"center_name"`
	Layout     int                `json:"layout" bson:"layout"`
	Screens    []ScreenSlot       `json:"screens" bson:"screens"`
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
