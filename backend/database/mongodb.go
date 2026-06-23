package database

import (
	"context"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var DB *mongo.Database
var PontonDB *mongo.Database
var CenterDB *mongo.Database
var centerClient *mongo.Client

func Connect(uri, dbName string) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		log.Fatalf("Error connecting to MongoDB: %v", err)
	}

	if err := client.Ping(ctx, nil); err != nil {
		log.Fatalf("Error pinging MongoDB: %v", err)
	}

	DB = client.Database(dbName)
	log.Printf("Connected to MongoDB: %s/%s", uri, dbName)
}

func ConnectPonton(uri, dbName string) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		log.Printf("[pontón] Warning: cannot connect to remote MongoDB: %v", err)
		return
	}

	if err := client.Ping(ctx, nil); err != nil {
		log.Printf("[pontón] Warning: cannot ping remote MongoDB: %v", err)
		return
	}

	PontonDB = client.Database(dbName)
	log.Printf("[pontón] Connected to remote MongoDB: %s/%s", uri, dbName)
}

func ConnectCenter(uri, dbName string) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		log.Printf("[centro] Warning: cannot connect to center MongoDB: %v", err)
		return
	}

	// Reconexión en caliente: cerrar el cliente anterior si existía
	if centerClient != nil {
		centerClient.Disconnect(ctx)
	}
	centerClient = client

	// El ping inicial puede fallar (enlace al centro caído durante el arranque);
	// se asigna igual: el driver reintenta por operación y la conexión se
	// recupera sola cuando vuelve el enlace.
	if err := client.Ping(ctx, nil); err != nil {
		log.Printf("[centro] Warning: center MongoDB unreachable at startup (will retry per request): %v", err)
	} else {
		log.Printf("[centro] Connected to center MongoDB: %s/%s", uri, dbName)
	}

	CenterDB = client.Database(dbName)
}

func GetCollection(name string) *mongo.Collection {
	return DB.Collection(name)
}
