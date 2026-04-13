package models

import "go.mongodb.org/mongo-driver/bson/primitive"

// PontonGridCell matches the pontón compression system's gridCells format exactly.
// Field names (BSON tags) MUST NOT be changed — the compression system is immutable.
type PontonGridCell struct {
	IDCell          int     `json:"idCell" bson:"idCell"`
	Name            string  `json:"name" bson:"name"`
	IPCamera        string  `json:"ipCamera" bson:"ipCamera"`
	IPNVR           string  `json:"ipNvr" bson:"ipNvr"`
	NVRChannel      int     `json:"nvrChannel" bson:"nvrChannel"`
	User            string  `json:"user" bson:"user"`
	Pass            string  `json:"pass" bson:"pass"`
	MediaMTXCamera1 string  `json:"mediamtxCamera1" bson:"mediamtxCamera1"`
	MediaMTXCamera2 string  `json:"mediamtxCamera2" bson:"mediamtxCamera2"`
	On              bool    `json:"on" bson:"on"`
	WFactor         float64 `json:"w_factor" bson:"w_factor"`
}

// PontonGridConfiguration matches the pontón's grid_configuration embedded object.
type PontonGridConfiguration struct {
	Rows             int              `json:"rows" bson:"rows"`
	Columns          int              `json:"columns" bson:"columns"`
	WidthResolution  int              `json:"widthResolution" bson:"widthResolution"`
	HeightResolution int              `json:"heightResolution" bson:"heightResolution"`
	SelectFlow       int              `json:"selectFlow" bson:"selectFlow"`
	FPS              int              `json:"fps" bson:"fps"`
	GOP              int              `json:"gop" bson:"gop"`
	PCID             int              `json:"pc_id" bson:"pc_id"`
	GridCells        []PontonGridCell `json:"gridCells" bson:"gridCells"`
	UnionCells       []interface{}    `json:"unionCells" bson:"unionCells"`
}

// PontonScreenConfiguration is the root document the pontón compression system reads from MongoDB.
// The compression system queries the "screen_configuration" collection and uses these exact field names.
type PontonScreenConfiguration struct {
	ID                primitive.ObjectID      `json:"_id,omitempty" bson:"_id,omitempty"`
	IDCenter          primitive.ObjectID      `json:"id_center,omitempty" bson:"id_center,omitempty"`
	FileName          string                  `json:"fileName" bson:"fileName"`
	Active            bool                    `json:"active" bson:"active"`
	Bitrate           int                     `json:"bitrate" bson:"bitrate"`
	HardwareEncoding  int                     `json:"hardwareEncoding" bson:"hardwareEncoding"`
	IPServer          string                  `json:"ipServer" bson:"ipServer"`
	GridConfiguration PontonGridConfiguration `json:"grid_configuration" bson:"grid_configuration"`
}
