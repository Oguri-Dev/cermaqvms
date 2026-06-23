package handlers

// Autenticación: login, usuario actual, CRUD de usuarios y middleware de
// roles. Las cuentas viven en la colección local "users". Roles:
// "admin" (todo) y "operator" (ver + operar, sin Configuración).

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"vms-cermaq/auth"
	"vms-cermaq/database"
	"vms-cermaq/models"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	sessionTTL   = 12 * time.Hour      // sesión normal
	rememberTTL  = 365 * 24 * time.Hour // "recordar sesión" (wall 24/7)
	ctxRoleKey   = ctxKey("role")
	ctxUserKey   = ctxKey("user")
)

type ctxKey string

// InitAuth fija el secreto de firma (de env o persistido en Mongo) y
// asegura que exista al menos un admin (semilla admin/admin).
func InitAuth(envSecret string) {
	secret := envSecret
	if secret == "" {
		secret = loadOrCreateSecret()
	}
	auth.SetSecret(secret)
	ensureSeedAdmin()
}

// loadOrCreateSecret guarda un secreto aleatorio en la colección app_secrets
// para que los tokens sobrevivan reinicios del backend (clave para el wall
// con sesión recordada).
func loadOrCreateSecret() string {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	coll := database.GetCollection("app_secrets")

	var doc struct {
		Value string `bson:"value"`
	}
	if err := coll.FindOne(ctx, bson.M{"_id": "auth_secret"}).Decode(&doc); err == nil && doc.Value != "" {
		return doc.Value
	}

	buf := make([]byte, 32)
	rand.Read(buf)
	secret := hex.EncodeToString(buf)
	coll.ReplaceOne(ctx, bson.M{"_id": "auth_secret"}, bson.M{"_id": "auth_secret", "value": secret}, options.Replace().SetUpsert(true))
	log.Println("[auth] Secreto de sesión generado y persistido")
	return secret
}

func ensureSeedAdmin() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	coll := database.GetCollection("users")
	count, err := coll.CountDocuments(ctx, bson.M{})
	if err != nil {
		log.Printf("[auth] No se pudo verificar usuarios: %v", err)
		return
	}
	if count > 0 {
		return
	}
	hash, _ := auth.HashPassword("admin")
	coll.InsertOne(ctx, bson.M{"username": "admin", "password_hash": hash, "role": "admin"})
	log.Println("[auth] Usuario semilla creado: admin / admin (cámbialo)")
}

// ── Login ──

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Remember bool   `json:"remember"`
}

// Login — POST /api/auth/login
func Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "JSON inválido"})
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var u models.User
	err := database.GetCollection("users").
		FindOne(ctx, bson.M{"username": bson.M{"$regex": "^" + regexpQuote(req.Username) + "$", "$options": "i"}}).
		Decode(&u)
	if err != nil || !auth.CheckPassword(u.PasswordHash, req.Password) {
		respondJSON(w, http.StatusUnauthorized, map[string]string{"error": "usuario o contraseña incorrectos"})
		return
	}

	ttl := sessionTTL
	if req.Remember {
		ttl = rememberTTL
	}
	token, err := auth.Issue(auth.Claims{Sub: u.ID.Hex(), User: u.Username, Role: u.Role}, ttl)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "no se pudo emitir el token"})
		return
	}
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"token": token,
		"user":  map[string]string{"username": u.Username, "role": u.Role},
	})
}

// Me — GET /api/auth/me (devuelve el usuario del token)
func Me(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]string{
		"username": r.Context().Value(ctxUserKey).(string),
		"role":     r.Context().Value(ctxRoleKey).(string),
	})
}

// ── Middleware ──

func bearerToken(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if strings.HasPrefix(h, "Bearer ") {
		return strings.TrimPrefix(h, "Bearer ")
	}
	return ""
}

// RequireAuth valida el token y mete user/role en el contexto.
func RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, err := auth.Verify(bearerToken(r))
		if err != nil {
			respondJSON(w, http.StatusUnauthorized, map[string]string{"error": "no autenticado"})
			return
		}
		ctx := context.WithValue(r.Context(), ctxUserKey, claims.User)
		ctx = context.WithValue(ctx, ctxRoleKey, claims.Role)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequireAdmin exige rol admin (asume RequireAuth antes).
func RequireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if role, _ := r.Context().Value(ctxRoleKey).(string); role != "admin" {
			respondJSON(w, http.StatusForbidden, map[string]string{"error": "se requiere rol administrador"})
			return
		}
		next.ServeHTTP(w, r)
	})
}

// ── CRUD de usuarios (solo admin) ──

type userInput struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Role     string `json:"role"`
}

// ListUsers — GET /api/users
func ListUsers(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	cursor, err := database.GetCollection("users").Find(ctx, bson.M{})
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	var users []models.User
	cursor.All(ctx, &users)
	respondJSON(w, http.StatusOK, users)
}

// CreateUser — POST /api/users
func CreateUser(w http.ResponseWriter, r *http.Request) {
	var in userInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "JSON inválido"})
		return
	}
	in.Username = strings.TrimSpace(in.Username)
	if in.Username == "" || in.Password == "" {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "usuario y contraseña son obligatorios"})
		return
	}
	if in.Role != "admin" && in.Role != "operator" {
		in.Role = "operator"
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	coll := database.GetCollection("users")
	n, _ := coll.CountDocuments(ctx, bson.M{"username": bson.M{"$regex": "^" + regexpQuote(in.Username) + "$", "$options": "i"}})
	if n > 0 {
		respondJSON(w, http.StatusConflict, map[string]string{"error": "ya existe un usuario con ese nombre"})
		return
	}
	hash, _ := auth.HashPassword(in.Password)
	res, err := coll.InsertOne(ctx, bson.M{"username": in.Username, "password_hash": hash, "role": in.Role})
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	respondJSON(w, http.StatusCreated, map[string]interface{}{
		"id": res.InsertedID, "username": in.Username, "role": in.Role,
	})
}

// UpdateUser — PUT /api/users/{id} (cambia rol y/o contraseña)
func UpdateUser(w http.ResponseWriter, r *http.Request) {
	id, err := primitive.ObjectIDFromHex(chi.URLParam(r, "id"))
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "id inválido"})
		return
	}
	var in userInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "JSON inválido"})
		return
	}
	set := bson.M{}
	if in.Role == "admin" || in.Role == "operator" {
		set["role"] = in.Role
	}
	if in.Password != "" {
		set["password_hash"], _ = auth.HashPassword(in.Password)
	}
	if len(set) == 0 {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "nada que actualizar"})
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	if _, err := database.GetCollection("users").UpdateOne(ctx, bson.M{"_id": id}, bson.M{"$set": set}); err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	respondJSON(w, http.StatusOK, map[string]bool{"success": true})
}

// DeleteUser — DELETE /api/users/{id} (no permite quedarse sin admins)
func DeleteUser(w http.ResponseWriter, r *http.Request) {
	id, err := primitive.ObjectIDFromHex(chi.URLParam(r, "id"))
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "id inválido"})
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	coll := database.GetCollection("users")

	var target models.User
	if err := coll.FindOne(ctx, bson.M{"_id": id}).Decode(&target); err != nil {
		respondJSON(w, http.StatusNotFound, map[string]string{"error": "usuario no encontrado"})
		return
	}
	if target.Role == "admin" {
		admins, _ := coll.CountDocuments(ctx, bson.M{"role": "admin"})
		if admins <= 1 {
			respondJSON(w, http.StatusConflict, map[string]string{"error": "no se puede eliminar el último administrador"})
			return
		}
	}
	coll.DeleteOne(ctx, bson.M{"_id": id})
	respondJSON(w, http.StatusOK, map[string]bool{"success": true})
}

func regexpQuote(s string) string {
	// Escape mínimo para el nombre de usuario en regex de Mongo
	r := strings.NewReplacer(
		`\`, `\\`, `.`, `\.`, `*`, `\*`, `+`, `\+`, `?`, `\?`,
		`(`, `\(`, `)`, `\)`, `[`, `\[`, `]`, `\]`, `{`, `\{`, `}`, `\}`,
		`^`, `\^`, `$`, `\$`, `|`, `\|`,
	)
	return r.Replace(s)
}
