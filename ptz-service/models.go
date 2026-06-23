package main

// Camera replica los campos relevantes de la colección "cameras" de la base
// del centro (mismo esquema que usa el compresor GST-Grid). Las credenciales
// nunca salen de este servicio.
type Camera struct {
	ID         string `bson:"-" json:"id"`
	Name       string `bson:"name" json:"name"`
	IPCamera   string `bson:"ipCamera" json:"ip_camera"`
	IPNvr      string `bson:"ipNvr" json:"ip_nvr"`
	NvrChannel int    `bson:"nvrChannel" json:"nvr_channel"`
	User       string `bson:"user" json:"-"`
	Pass       string `bson:"pass" json:"-"`
	Type       string `bson:"type" json:"type"`
}

// MoveRequest es el comando de movimiento continuo. Valores en [-1, 1].
// El front lo reenvía periódicamente mientras el operador mantiene presionado
// el control: cada reenvío renueva el dead-man switch.
type MoveRequest struct {
	Pan  float64 `json:"pan"`
	Tilt float64 `json:"tilt"`
	Zoom float64 `json:"zoom"`
}

// GotoRequest mueve la cámara a un preset.
type GotoRequest struct {
	Preset string `json:"preset"`
}

// Preset es un preset configurado en la cámara.
type Preset struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}
