package jsonstore

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"creation-date-saver/domain"

	"github.com/romdo/go-debounce"
)

// RepoConfig holds configuration for the repository.
type RepoConfig struct {
	SaveDelay   time.Duration // debounce delay for saving metadata file
	DeleteDelay time.Duration // delay before actual metadata deletion
}

// TimeRepo provides access to file metadata stored in JSON.
type TimeRepo struct {
	filePath      string
	config        RepoConfig
	mu            sync.Mutex
	cache         map[string]Metadata
	debouncedSave func()
	deleteTimers  map[string]*time.Timer // timers for delayed deletion
}

// CreateTimeRepo creates a new Repository with the given file path and config.
func CreateTimeRepo(filePath string, config RepoConfig) (*TimeRepo, error) {
	cache, err := loadMetadata(filePath)
	if err != nil {
		return nil, err
	}

	debouncedSave, _ := debounce.New(config.SaveDelay, func() {
		err := saveMetadata(filePath, cache)
		if err != nil {
			fmt.Printf("Error saving metadata: %v\n", err)
		}
	})

	return &TimeRepo{
		filePath:      filePath,
		config:        config,
		cache:         cache,
		debouncedSave: debouncedSave,
		deleteTimers:  make(map[string]*time.Timer),
	}, nil
}

// Get returns metadata for a given relPath, or false if not found.
func (r *TimeRepo) Get(relPath string) (domain.Metadata, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	meta, ok := r.cache[relPath]
	return modelMetadataToDomain(relPath, meta), ok
}

// Upsert inserts or updates metadata for a given relPath.
func (r *TimeRepo) Upsert(meta domain.Metadata) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	// Cancel delete timer if it exists
	if timer, ok := r.deleteTimers[meta.Patch]; ok && timer != nil {
		timer.Stop()
		delete(r.deleteTimers, meta.Patch)
	}
	r.cache[meta.Patch] = domainMetadataToModel(meta)
	r.debouncedSave()
	return nil
}

// Delete schedules metadata removal for a given relPath after DeleteDelay.
func (r *TimeRepo) Delete(relPath string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.delete(relPath)
}

// DeleteByPrefix removes all metadata entries whose relative path starts with the given prefix.
func (r *TimeRepo) DeleteByPrefix(prefix string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for k := range r.cache {
		if len(k) >= len(prefix) && k[:len(prefix)] == prefix {
			r.delete(k)
		}
	}
	r.debouncedSave()
	return nil
}

func (r *TimeRepo) delete(relPath string) error {
	// If a delete timer already exists, do nothing
	if _, exists := r.deleteTimers[relPath]; exists {
		return nil
	}
	// If DeleteDelay is zero, delete immediately
	delay := r.config.DeleteDelay
	if delay == 0 {
		delete(r.cache, relPath)
		r.debouncedSave()
		return nil
	}
	// Start a delete timer using DeleteDelay
	r.deleteTimers[relPath] = time.AfterFunc(delay, func() {
		r.mu.Lock()
		defer r.mu.Unlock()
		delete(r.cache, relPath)
		delete(r.deleteTimers, relPath)
		r.debouncedSave()
	})
	return nil
}

// loadMetadata loads metadata from a JSON file.
func loadMetadata(filePath string) (map[string]Metadata, error) {
	file, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string]Metadata), nil
		}
		return nil, err
	}
	var metadata map[string]Metadata
	err = json.Unmarshal(file, &metadata)
	if err != nil {
		return nil, err
	}
	return metadata, nil
}

// saveMetadata saves metadata to a JSON file.
func saveMetadata(filePath string, metadata map[string]Metadata) error {
	data, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filePath, data, 0644)
}
