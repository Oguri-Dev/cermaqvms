package main

// Cliente ISAPI (Hikvision) con digest auth. El movimiento usa
// /ISAPI/PTZCtrl/channels/{ch}/continuous con velocidades -100..100;
// el stop es el mismo comando con velocidades 0 (igual que el sistema
// en producción). Funciona contra el NVR (ipNvr + canal) o directo a
// la cámara (canal 1).

import (
	"encoding/xml"
	"fmt"
	"io"
	"math"
	"net"
	"regexp"
	"strings"
	"time"

	dac "github.com/xinsnake/go-http-digest-auth-client"
)

type isapiClient struct {
	timeout time.Duration
}

func newISAPIClient(timeout time.Duration) *isapiClient {
	return &isapiClient{timeout: timeout}
}

// target resuelve host y canal ISAPI: vía NVR si la cámara tiene uno,
// directo a la cámara (canal 1) si no.
func isapiTarget(cam *Camera) (host string, channel int) {
	if cam.IPNvr != "" {
		ch := cam.NvrChannel
		if ch <= 0 {
			ch = 1
		}
		return cam.IPNvr, ch
	}
	return cam.IPCamera, 1
}

// reachable evita que el digest client quede colgado contra un host caído:
// el dial corto falla rápido y permite el fallback al otro protocolo.
func (c *isapiClient) reachable(host string) error {
	conn, err := net.DialTimeout("tcp", net.JoinHostPort(host, "80"), c.timeout/2)
	if err != nil {
		return fmt.Errorf("ISAPI %s inaccesible: %w", host, err)
	}
	conn.Close()
	return nil
}

func (c *isapiClient) do(user, pass, method, url, body string) (int, []byte, error) {
	dr := dac.NewRequest(user, pass, method, url, body)
	if body != "" {
		dr.Header.Set("Content-Type", "application/xml")
	}
	resp, err := dr.Execute()
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, data, nil
}

// ContinuousMove con velocidades en [-1, 1] (se escalan a -100..100).
func (c *isapiClient) ContinuousMove(cam *Camera, pan, tilt, zoom float64) error {
	host, ch := isapiTarget(cam)
	if err := c.reachable(host); err != nil {
		return err
	}
	url := fmt.Sprintf("http://%s/ISAPI/PTZCtrl/channels/%d/continuous", host, ch)
	body := fmt.Sprintf("<PTZData><pan>%d</pan><tilt>%d</tilt><zoom>%d</zoom></PTZData>",
		toISAPISpeed(pan), toISAPISpeed(tilt), toISAPISpeed(zoom))
	status, _, err := c.do(cam.User, cam.Pass, "PUT", url, body)
	if err != nil {
		return err
	}
	if status >= 400 {
		return fmt.Errorf("ISAPI continuous HTTP %d", status)
	}
	return nil
}

// Stop = continuous con velocidades 0.
func (c *isapiClient) Stop(cam *Camera) error {
	return c.ContinuousMove(cam, 0, 0, 0)
}

// GetPresets lista los presets configurados.
func (c *isapiClient) GetPresets(cam *Camera) ([]Preset, error) {
	host, ch := isapiTarget(cam)
	if err := c.reachable(host); err != nil {
		return nil, err
	}
	url := fmt.Sprintf("http://%s/ISAPI/PTZCtrl/channels/%d/presets", host, ch)
	status, data, err := c.do(cam.User, cam.Pass, "GET", url, "")
	if err != nil {
		return nil, err
	}
	if status != 200 {
		return nil, fmt.Errorf("ISAPI presets HTTP %d", status)
	}

	var list struct {
		Presets []struct {
			Enabled    bool   `xml:"enabled"`
			ID         int    `xml:"id"`
			PresetName string `xml:"presetName"`
		} `xml:"PTZPreset"`
	}
	if err := xml.Unmarshal(data, &list); err != nil {
		return nil, fmt.Errorf("parseando presets ISAPI: %w", err)
	}

	presets := []Preset{}
	for _, p := range list.Presets {
		// Los NVR Hikvision devuelven los 256 slots, incluidos los presets
		// especiales de sistema (Remote reboot, Call OSD menu, scans...).
		// Solo se exponen los configurados por el usuario: habilitados y
		// con nombre personalizado.
		if !p.Enabled || !isUserPresetName(p.PresetName) {
			continue
		}
		presets = append(presets, Preset{ID: fmt.Sprintf("%d", p.ID), Name: p.PresetName})
	}
	return presets, nil
}

// GotoPreset mueve la cámara a un preset por id.
func (c *isapiClient) GotoPreset(cam *Camera, presetID string) error {
	host, ch := isapiTarget(cam)
	if err := c.reachable(host); err != nil {
		return err
	}
	url := fmt.Sprintf("http://%s/ISAPI/PTZCtrl/channels/%d/presets/%s/goto", host, ch, presetID)
	status, _, err := c.do(cam.User, cam.Pass, "PUT", url, "")
	if err != nil {
		return err
	}
	if status >= 400 {
		return fmt.Errorf("ISAPI goto HTTP %d", status)
	}
	return nil
}

var defaultPresetRe = regexp.MustCompile(`(?i)^preset ?\d+$`)
var specialPresetRe = regexp.MustCompile(`(?i)^(call (patrol|pattern|osd)|start .*scan|stop a scan|auto-flip|back to origin|day mode|night mode|one-touch|day/night|set manual|save manual|remote reboot)`)

func isUserPresetName(name string) bool {
	name = strings.TrimSpace(name)
	return name != "" && !defaultPresetRe.MatchString(name) && !specialPresetRe.MatchString(name)
}

func toISAPISpeed(v float64) int {
	if math.IsNaN(v) {
		return 0
	}
	scaled := int(math.Round(v * 100))
	if scaled > 100 {
		return 100
	}
	if scaled < -100 {
		return -100
	}
	return scaled
}
