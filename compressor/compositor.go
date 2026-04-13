package main

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// Compositor watches the screen_configuration collection in MongoDB and manages
// one FFmpeg Pipeline per active fileName. It detects config changes via polling
// and starts/stops/restarts pipelines as needed.
type Compositor struct {
	mu        sync.RWMutex
	pipelines map[string]*Pipeline // fileName -> Pipeline
	hashes    map[string]string    // fileName -> config hash (for change detection)
	db        *mongo.Database
	cfg       *Config
}

func NewCompositor(db *mongo.Database, cfg *Config) *Compositor {
	return &Compositor{
		pipelines: make(map[string]*Pipeline),
		hashes:    make(map[string]string),
		db:        db,
		cfg:       cfg,
	}
}

// Start performs an initial config sync and begins the polling loop.
func (c *Compositor) Start(ctx context.Context) {
	log.Println("[compositor] starting...")
	c.syncConfigs(ctx)

	active := 0
	c.mu.RLock()
	active = len(c.pipelines)
	c.mu.RUnlock()
	log.Printf("[compositor] %d active pipeline(s)", active)

	go c.pollLoop(ctx)
}

// GetPipeline returns the pipeline for a given fileName, or nil if not found.
func (c *Compositor) GetPipeline(fileName string) *Pipeline {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.pipelines[fileName]
}

// ListPipelines returns the status of all active pipelines.
func (c *Compositor) ListPipelines() []PipelineStatus {
	c.mu.RLock()
	defer c.mu.RUnlock()

	statuses := []PipelineStatus{}
	for _, p := range c.pipelines {
		statuses = append(statuses, p.Status())
	}
	return statuses
}

// pollLoop periodically syncs configs from MongoDB.
func (c *Compositor) pollLoop(ctx context.Context) {
	ticker := time.NewTicker(time.Duration(c.cfg.PollInterval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			c.stopAll()
			return
		case <-ticker.C:
			c.syncConfigs(ctx)
		}
	}
}

// syncConfigs fetches all screen_configuration documents and creates, updates,
// or removes pipelines to match the current state in MongoDB.
func (c *Compositor) syncConfigs(ctx context.Context) {
	col := c.db.Collection("screen_configuration")

	findCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	cursor, err := col.Find(findCtx, bson.M{})
	if err != nil {
		log.Printf("[compositor] error fetching configs: %v", err)
		return
	}
	defer cursor.Close(findCtx)

	var configs []PontonScreenConfiguration
	if err := cursor.All(findCtx, &configs); err != nil {
		log.Printf("[compositor] error decoding configs: %v", err)
		return
	}

	seen := map[string]bool{}

	for _, config := range configs {
		if config.FileName == "" {
			continue
		}
		seen[config.FileName] = true
		hash := hashConfig(config)

		c.mu.RLock()
		oldHash := c.hashes[config.FileName]
		c.mu.RUnlock()

		if hash == oldHash {
			continue // No change
		}

		log.Printf("[compositor] config changed for %q", config.FileName)
		c.updatePipeline(config, hash)
	}

	// Stop pipelines whose configs were deleted from MongoDB
	c.mu.Lock()
	for name, p := range c.pipelines {
		if !seen[name] {
			log.Printf("[compositor] stopping pipeline %q (config removed)", name)
			p.Stop()
			delete(c.pipelines, name)
			delete(c.hashes, name)
		}
	}
	c.mu.Unlock()
}

// updatePipeline creates, updates, or deactivates a pipeline based on config state.
func (c *Compositor) updatePipeline(config PontonScreenConfiguration, hash string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	fileName := config.FileName

	// Existing pipeline — update or deactivate
	if p, exists := c.pipelines[fileName]; exists {
		if !config.Active {
			log.Printf("[compositor] deactivating pipeline %q", fileName)
			p.Stop()
			delete(c.pipelines, fileName)
			c.hashes[fileName] = hash
			return
		}
		log.Printf("[compositor] updating pipeline %q", fileName)
		p.UpdateConfig(config)
		c.hashes[fileName] = hash
		return
	}

	// Not active — just track the hash
	if !config.Active {
		c.hashes[fileName] = hash
		return
	}

	// New active pipeline
	gc := config.GridConfiguration
	log.Printf("[compositor] creating pipeline %q (%dx%d grid, %d kbps, %d fps)",
		fileName, gc.Rows, gc.Columns, config.Bitrate, gc.FPS)

	p := NewPipeline(config, c.cfg)
	c.pipelines[fileName] = p
	c.hashes[fileName] = hash
	p.Start()
}

// stopAll terminates all running pipelines.
func (c *Compositor) stopAll() {
	c.mu.Lock()
	defer c.mu.Unlock()

	for name, p := range c.pipelines {
		log.Printf("[compositor] stopping pipeline %q", name)
		p.Stop()
	}
}

// hashConfig generates a short hash of the config for change detection.
func hashConfig(config PontonScreenConfiguration) string {
	data, _ := json.Marshal(config)
	h := sha256.Sum256(data)
	return fmt.Sprintf("%x", h[:8])
}
