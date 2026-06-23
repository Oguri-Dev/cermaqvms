package main

// Cliente ONVIF mínimo (perfiles, presets, movimiento continuo, stop, goto)
// con WS-Security UsernameToken digest. Los templates SOAP provienen del
// servicio PTZ que corre en producción (ptzControll), más el Stop estándar.

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type onvifClient struct {
	http *http.Client
}

func newONVIFClient(timeout time.Duration) *onvifClient {
	return &onvifClient{http: &http.Client{Timeout: timeout}}
}

func wsseHeader(password string) (nonceB64, created, digestB64 string, err error) {
	nonce := make([]byte, 16)
	if _, err = rand.Read(nonce); err != nil {
		return
	}
	nonceB64 = base64.StdEncoding.EncodeToString(nonce)
	created = time.Now().UTC().Format(time.RFC3339)

	h := sha1.New()
	h.Write(nonce)
	h.Write([]byte(created))
	h.Write([]byte(password))
	digestB64 = base64.StdEncoding.EncodeToString(h.Sum(nil))
	return
}

func (c *onvifClient) soapCall(ctx context.Context, url, envelope string) ([]byte, int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBufferString(envelope))
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Content-Type", "application/soap+xml; charset=utf-8")
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	return body, resp.StatusCode, nil
}

const wsseBlock = `<wsse:Security s:mustUnderstand="1">
      <wsse:UsernameToken wsu:Id="Token-1">
        <wsse:Username>%s</wsse:Username>
        <wsse:Password Type="http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-username-token-profile-1.0#PasswordDigest">%s</wsse:Password>
        <wsse:Nonce EncodingType="http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-soap-message-security-1.0#Base64Binary">%s</wsse:Nonce>
        <wsu:Created>%s</wsu:Created>
      </wsse:UsernameToken>
    </wsse:Security>`

const envelopeTpl = `<?xml version="1.0" encoding="UTF-8"?>
<s:Envelope
  xmlns:s="http://www.w3.org/2003/05/soap-envelope"
  xmlns:wsse="http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-wssecurity-secext-1.0.xsd"
  xmlns:wsu="http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-wssecurity-utility-1.0.xsd"
  xmlns:wsa="http://www.w3.org/2005/08/addressing"
  xmlns:trt="http://www.onvif.org/ver10/media/wsdl"
  xmlns:tptz="http://www.onvif.org/ver20/ptz/wsdl">
  <s:Header>
    <wsa:Action>%s</wsa:Action>
    <wsa:To>%s</wsa:To>
    %s
  </s:Header>
  <s:Body>
    %s
  </s:Body>
</s:Envelope>`

func (c *onvifClient) buildEnvelope(action, to, user, pass, body string) (string, error) {
	nonce, created, digest, err := wsseHeader(pass)
	if err != nil {
		return "", err
	}
	security := fmt.Sprintf(wsseBlock, user, digest, nonce, created)
	return fmt.Sprintf(envelopeTpl, action, to, security, body), nil
}

// GetProfiles devuelve el primer ProfileToken del media service.
func (c *onvifClient) GetProfiles(ctx context.Context, ip, user, pass string) (string, error) {
	to := fmt.Sprintf("http://%s/onvif/media_service", ip)
	env, err := c.buildEnvelope("http://www.onvif.org/ver10/media/wsdl/GetProfiles", to, user, pass, "<trt:GetProfiles/>")
	if err != nil {
		return "", err
	}
	raw, status, err := c.soapCall(ctx, to, env)
	if err != nil {
		return "", err
	}
	if status >= 400 {
		return "", fmt.Errorf("GetProfiles HTTP %d", status)
	}

	type profile struct {
		Token string `xml:"token,attr"`
	}
	var out struct {
		Profiles []profile `xml:"Body>GetProfilesResponse>Profiles"`
	}
	if err := xml.Unmarshal(raw, &out); err != nil {
		return "", fmt.Errorf("parseando GetProfiles: %w", err)
	}
	if len(out.Profiles) == 0 {
		return "", fmt.Errorf("la cámara no expone perfiles ONVIF")
	}
	return out.Profiles[0].Token, nil
}

// GetPresets lista los presets con nombre configurado (ignora los de fábrica).
func (c *onvifClient) GetPresets(ctx context.Context, ip, user, pass, profile string) ([]Preset, error) {
	to := fmt.Sprintf("http://%s/onvif/PTZ", ip)
	body := fmt.Sprintf("<tptz:GetPresets><tptz:ProfileToken>%s</tptz:ProfileToken></tptz:GetPresets>", profile)
	env, err := c.buildEnvelope("http://www.onvif.org/ver20/ptz/wsdl/GetPresets", to, user, pass, body)
	if err != nil {
		return nil, err
	}
	raw, status, err := c.soapCall(ctx, to, env)
	if err != nil {
		return nil, err
	}
	if status >= 400 {
		return nil, fmt.Errorf("GetPresets HTTP %d", status)
	}

	type preset struct {
		Token string `xml:"token,attr"`
		Name  string `xml:"Name"`
	}
	var out struct {
		Presets []preset `xml:"Body>GetPresetsResponse>Preset"`
	}
	if err := xml.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("parseando GetPresets: %w", err)
	}

	presets := []Preset{}
	for _, p := range out.Presets {
		name := strings.TrimSpace(p.Name)
		if name == "" || strings.HasPrefix(name, "Preset") {
			continue
		}
		presets = append(presets, Preset{ID: p.Token, Name: name})
	}
	return presets, nil
}

// ContinuousMove inicia movimiento continuo. Velocidades en [-1, 1].
func (c *onvifClient) ContinuousMove(ctx context.Context, ip, user, pass, profile string, pan, tilt, zoom float64) error {
	to := fmt.Sprintf("http://%s/onvif/PTZ", ip)
	body := fmt.Sprintf(`<tptz:ContinuousMove>
      <tptz:ProfileToken>%s</tptz:ProfileToken>
      <tptz:Velocity>
        <tt:PanTilt xmlns:tt="http://www.onvif.org/ver10/schema" x="%.2f" y="%.2f"/>
        <tt:Zoom xmlns:tt="http://www.onvif.org/ver10/schema" x="%.2f"/>
      </tptz:Velocity>
    </tptz:ContinuousMove>`, profile, pan, tilt, zoom)
	env, err := c.buildEnvelope("http://www.onvif.org/ver20/ptz/wsdl/ContinuousMove", to, user, pass, body)
	if err != nil {
		return err
	}
	_, status, err := c.soapCall(ctx, to, env)
	if err != nil {
		return err
	}
	if status >= 400 {
		return fmt.Errorf("ContinuousMove HTTP %d", status)
	}
	return nil
}

// Stop detiene pan/tilt y zoom.
func (c *onvifClient) Stop(ctx context.Context, ip, user, pass, profile string) error {
	to := fmt.Sprintf("http://%s/onvif/PTZ", ip)
	body := fmt.Sprintf(`<tptz:Stop>
      <tptz:ProfileToken>%s</tptz:ProfileToken>
      <tptz:PanTilt>true</tptz:PanTilt>
      <tptz:Zoom>true</tptz:Zoom>
    </tptz:Stop>`, profile)
	env, err := c.buildEnvelope("http://www.onvif.org/ver20/ptz/wsdl/Stop", to, user, pass, body)
	if err != nil {
		return err
	}
	_, status, err := c.soapCall(ctx, to, env)
	if err != nil {
		return err
	}
	if status >= 400 {
		return fmt.Errorf("Stop HTTP %d", status)
	}
	return nil
}

// GotoPreset mueve la cámara a un preset.
func (c *onvifClient) GotoPreset(ctx context.Context, ip, user, pass, profile, preset string) error {
	to := fmt.Sprintf("http://%s/onvif/PTZ", ip)
	body := fmt.Sprintf(`<tptz:GotoPreset>
      <tptz:ProfileToken>%s</tptz:ProfileToken>
      <tptz:PresetToken>%s</tptz:PresetToken>
    </tptz:GotoPreset>`, profile, preset)
	env, err := c.buildEnvelope("http://www.onvif.org/ver20/ptz/wsdl/GotoPreset", to, user, pass, body)
	if err != nil {
		return err
	}
	_, status, err := c.soapCall(ctx, to, env)
	if err != nil {
		return err
	}
	if status >= 400 {
		return fmt.Errorf("GotoPreset HTTP %d", status)
	}
	return nil
}
