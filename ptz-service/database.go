package main

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Database struct {
	db *mongo.Database
}

// ConnectDB abre la conexión a la Mongo del centro. El driver conecta lazy y
// reintenta por operación, así que un Mongo que tarda en subir tras un corte
// de energía no impide que el servicio arranque.
func ConnectDB(uri, dbName string) (*Database, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		return nil, err
	}
	if err := client.Ping(ctx, nil); err != nil {
		log.Printf("[db] Warning: MongoDB no responde aún, se reintentará por operación: %v", err)
	} else {
		log.Printf("[db] Conectado a %s/%s", uri, dbName)
	}
	return &Database{db: client.Database(dbName)}, nil
}

func (d *Database) Ping(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	return d.db.Client().Ping(ctx, nil)
}

// rawCamera decodifica el documento con el _id incluido
type rawCamera struct {
	ID     primitive.ObjectID `bson:"_id"`
	Camera `bson:",inline"`
}

var normalizeRe = regexp.MustCompile(`[\s\-_]`)

func normalizeName(s string) string {
	return strings.ToLower(normalizeRe.ReplaceAllString(s, ""))
}

// FindCamera resuelve una cámara por _id (hex) o por nombre, con el mismo
// matching flexible que usa el compresor: exacto case-insensitive y
// normalizado (sin espacios, guiones ni guiones bajos).
func (d *Database) FindCamera(ctx context.Context, idOrName string) (*Camera, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	coll := d.db.Collection("cameras")

	if oid, err := primitive.ObjectIDFromHex(idOrName); err == nil {
		var raw rawCamera
		if err := coll.FindOne(ctx, bson.M{"_id": oid}).Decode(&raw); err == nil {
			raw.Camera.ID = raw.ID.Hex()
			return &raw.Camera, nil
		}
	}

	// Nombre exacto (case-insensitive, anclado)
	var raw rawCamera
	exact := bson.M{"name": bson.M{"$regex": "^" + regexp.QuoteMeta(idOrName) + "$", "$options": "i"}}
	if err := coll.FindOne(ctx, exact).Decode(&raw); err == nil {
		raw.Camera.ID = raw.ID.Hex()
		return &raw.Camera, nil
	}

	// Normalizado: recorrer y comparar (la colección es pequeña)
	cursor, err := coll.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	want := normalizeName(idOrName)
	for cursor.Next(ctx) {
		var rc rawCamera
		if err := cursor.Decode(&rc); err != nil {
			continue
		}
		if normalizeName(rc.Name) == want {
			rc.Camera.ID = rc.ID.Hex()
			return &rc.Camera, nil
		}
	}
	return nil, fmt.Errorf("cámara no encontrada: %s", idOrName)
}

// ListCameras devuelve todas las cámaras (sin credenciales) para diagnóstico.
func (d *Database) ListCameras(ctx context.Context) ([]Camera, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	cursor, err := d.db.Collection("cameras").Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	cameras := []Camera{}
	for cursor.Next(ctx) {
		var rc rawCamera
		if err := cursor.Decode(&rc); err != nil {
			continue
		}
		rc.Camera.ID = rc.ID.Hex()
		cameras = append(cameras, rc.Camera)
	}
	return cameras, cursor.Err()
}
