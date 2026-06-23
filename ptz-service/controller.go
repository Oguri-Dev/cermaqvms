package main

// Controller orquesta los movimientos: elige protocolo (ISAPI primero,
// ONVIF como fallback — mismo orden que el sistema en producción), cachea
// qué protocolo funcionó por cámara, y mantiene el dead-man switch que
// detiene la cámara si el front deja de renovar el comando.

import (
	"context"
	"fmt"
	"log"
	"math"
	"sync"
	"time"
)

type Controller struct {
	cfg   *Config
	db    *Database
	onvif *onvifClient
	isapi *isapiClient

	mu            sync.Mutex
	deadman       map[string]*time.Timer // key: camera ID — timer de auto-stop
	protoCache    map[string]string      // key: camera ID — "isapi" | "onvif"
	profileCache  map[string]string      // key: camera ID — ProfileToken ONVIF
}

func NewController(cfg *Config, db *Database) *Controller {
	return &Controller{
		cfg:          cfg,
		db:           db,
		onvif:        newONVIFClient(cfg.CameraTimeout),
		isapi:        newISAPIClient(cfg.CameraTimeout),
		deadman:      map[string]*time.Timer{},
		protoCache:   map[string]string{},
		profileCache: map[string]string{},
	}
}

func clamp(v float64) float64 {
	if math.IsNaN(v) {
		return 0
	}
	return math.Max(-1, math.Min(1, v))
}

// onvifProfile resuelve (y cachea) el ProfileToken de la cámara.
func (c *Controller) onvifProfile(ctx context.Context, cam *Camera) (string, error) {
	c.mu.Lock()
	token, ok := c.profileCache[cam.ID]
	c.mu.Unlock()
	if ok {
		return token, nil
	}
	token, err := c.onvif.GetProfiles(ctx, cam.IPCamera, cam.User, cam.Pass)
	if err != nil {
		return "", err
	}
	c.mu.Lock()
	c.profileCache[cam.ID] = token
	c.mu.Unlock()
	return token, nil
}

func (c *Controller) cachedProto(camID string) string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.protoCache[camID]
}

func (c *Controller) rememberProto(camID, proto string) {
	c.mu.Lock()
	c.protoCache[camID] = proto
	c.mu.Unlock()
}

// Move ejecuta movimiento continuo y arma/renueva el dead-man switch.
// Cada llamada del front (incluidos los keepalives) renueva el timer.
func (c *Controller) Move(ctx context.Context, cam *Camera, pan, tilt, zoom float64) (string, error) {
	pan, tilt, zoom = clamp(pan), clamp(tilt), clamp(zoom)

	proto, err := c.dispatch(ctx, cam, func() error {
		return c.isapi.ContinuousMove(cam, pan, tilt, zoom)
	}, func(profile string) error {
		return c.onvif.ContinuousMove(ctx, cam.IPCamera, cam.User, cam.Pass, profile, pan, tilt, zoom)
	})
	if err != nil {
		return "", err
	}

	c.armDeadman(cam)
	return proto, nil
}

// Stop detiene la cámara y desarma el dead-man.
func (c *Controller) Stop(ctx context.Context, cam *Camera) (string, error) {
	c.disarmDeadman(cam.ID)
	return c.dispatch(ctx, cam, func() error {
		return c.isapi.Stop(cam)
	}, func(profile string) error {
		return c.onvif.Stop(ctx, cam.IPCamera, cam.User, cam.Pass, profile)
	})
}

// Presets lista los presets (ISAPI primero, fallback ONVIF).
func (c *Controller) Presets(ctx context.Context, cam *Camera) ([]Preset, error) {
	var result []Preset
	_, err := c.dispatch(ctx, cam, func() error {
		p, err := c.isapi.GetPresets(cam)
		if err != nil {
			return err
		}
		result = p
		return nil
	}, func(profile string) error {
		p, err := c.onvif.GetPresets(ctx, cam.IPCamera, cam.User, cam.Pass, profile)
		if err != nil {
			return err
		}
		result = p
		return nil
	})
	return result, err
}

// Goto mueve a un preset.
func (c *Controller) Goto(ctx context.Context, cam *Camera, preset string) (string, error) {
	return c.dispatch(ctx, cam, func() error {
		return c.isapi.GotoPreset(cam, preset)
	}, func(profile string) error {
		return c.onvif.GotoPreset(ctx, cam.IPCamera, cam.User, cam.Pass, profile, preset)
	})
}

// dispatch ejecuta la acción con el protocolo cacheado, o prueba
// ISAPI → ONVIF y recuerda cuál funcionó.
func (c *Controller) dispatch(ctx context.Context, cam *Camera, viaISAPI func() error, viaONVIF func(profile string) error) (string, error) {
	tryISAPI := func() error {
		if cam.IPNvr == "" && cam.IPCamera == "" {
			return fmt.Errorf("la cámara no tiene IP configurada")
		}
		return viaISAPI()
	}
	tryONVIF := func() error {
		if cam.IPCamera == "" {
			return fmt.Errorf("la cámara no tiene ipCamera para ONVIF")
		}
		profile, err := c.onvifProfile(ctx, cam)
		if err != nil {
			return err
		}
		return viaONVIF(profile)
	}

	switch c.cachedProto(cam.ID) {
	case "isapi":
		if err := tryISAPI(); err == nil {
			return "isapi", nil
		}
	case "onvif":
		if err := tryONVIF(); err == nil {
			return "onvif", nil
		}
	}

	errISAPI := tryISAPI()
	if errISAPI == nil {
		c.rememberProto(cam.ID, "isapi")
		return "isapi", nil
	}
	errONVIF := tryONVIF()
	if errONVIF == nil {
		c.rememberProto(cam.ID, "onvif")
		return "onvif", nil
	}
	return "", fmt.Errorf("ISAPI: %v | ONVIF: %v", errISAPI, errONVIF)
}

// armDeadman programa el auto-stop: si nadie renueva el movimiento dentro
// de StopTimeout, se envía stop a la cámara.
func (c *Controller) armDeadman(cam *Camera) {
	camCopy := *cam
	c.mu.Lock()
	defer c.mu.Unlock()
	if t, ok := c.deadman[cam.ID]; ok {
		t.Stop()
	}
	c.deadman[cam.ID] = time.AfterFunc(c.cfg.StopTimeout, func() {
		log.Printf("[deadman] %s: sin keepalive en %v, enviando stop", camCopy.Name, c.cfg.StopTimeout)
		ctx, cancel := context.WithTimeout(context.Background(), c.cfg.CameraTimeout*2)
		defer cancel()
		if _, err := c.Stop(ctx, &camCopy); err != nil {
			log.Printf("[deadman] %s: error en auto-stop: %v", camCopy.Name, err)
		}
	})
}

func (c *Controller) disarmDeadman(camID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if t, ok := c.deadman[camID]; ok {
		t.Stop()
		delete(c.deadman, camID)
	}
}
