package jsonstore

import (
	"encoding/json"
	"os"
	"sync"
	"time"

	"creation-date-saver/domain"

	"github.com/romdo/go-debounce"
)

// RepoConfig holds configuration for the repository, e.g., save delay.
type RepoConfig struct {
	SaveDelay time.Duration // not used yet, but reserved for future use
}

// TimeRepo provides access to file metadata stored in JSON.
type TimeRepo struct {
	filePath      string
	config        RepoConfig
	mu            sync.Mutex
	cache         map[string]Metadata
	debouncedSave func()
}

// CreateTimeRepo creates a new Repository with the given file path and config.
func CreateTimeRepo(filePath string, config RepoConfig) (*TimeRepo, error) {
	cache, err := loadMetadata(filePath)
	if err != nil {
		return nil, err
	}

	debouncedSave, _ := debounce.New(config.SaveDelay, func() {
		saveMetadata(filePath, cache)
	})

	return &TimeRepo{
		filePath:      filePath,
		config:        config,
		cache:         cache,
		debouncedSave: debouncedSave,
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
	r.cache[meta.Patch] = domainMetadataToModel(meta)
	r.debouncedSave()
	return nil
}

// Delete removes metadata for a given relPath.
func (r *TimeRepo) Delete(relPath string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.cache, relPath)
	r.debouncedSave()
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
