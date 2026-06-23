// Package auth implementa sesiones por token (JWT HS256) y hashing de
// contraseñas con bcrypt. JWT se construye con la stdlib para no agregar
// dependencias; bcrypt viene de golang.org/x/crypto (ya indirecta del driver).
package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

var secret []byte

// SetSecret configura el secreto de firma. Debe llamarse al arrancar.
func SetSecret(s string) {
	secret = []byte(s)
}

// Claims es el cuerpo del token.
type Claims struct {
	Sub  string `json:"sub"`  // user id
	User string `json:"user"` // username
	Role string `json:"role"`
	Exp  int64  `json:"exp"` // unix seconds
}

func b64(b []byte) string {
	return base64.RawURLEncoding.EncodeToString(b)
}

func sign(signingInput string) string {
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(signingInput))
	return b64(mac.Sum(nil))
}

// Issue crea un token firmado válido por la duración indicada.
func Issue(c Claims, ttl time.Duration) (string, error) {
	c.Exp = time.Now().Add(ttl).Unix()
	header := b64([]byte(`{"alg":"HS256","typ":"JWT"}`))
	payloadJSON, err := json.Marshal(c)
	if err != nil {
		return "", err
	}
	payload := b64(payloadJSON)
	signingInput := header + "." + payload
	return signingInput + "." + sign(signingInput), nil
}

var (
	ErrMalformed = errors.New("token malformado")
	ErrSignature = errors.New("firma inválida")
	ErrExpired   = errors.New("token expirado")
)

// Verify valida la firma y la expiración, devolviendo los claims.
func Verify(token string) (*Claims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, ErrMalformed
	}
	signingInput := parts[0] + "." + parts[1]
	expected := sign(signingInput)
	// Comparación en tiempo constante
	if !hmac.Equal([]byte(expected), []byte(parts[2])) {
		return nil, ErrSignature
	}
	payloadJSON, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, ErrMalformed
	}
	var c Claims
	if err := json.Unmarshal(payloadJSON, &c); err != nil {
		return nil, ErrMalformed
	}
	if time.Now().Unix() > c.Exp {
		return nil, ErrExpired
	}
	return &c, nil
}
