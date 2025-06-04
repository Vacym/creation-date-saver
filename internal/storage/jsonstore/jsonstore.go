package jsonstore

import (
	"encoding/json"
	"os"
	"sync"
	"time"

	"creation-date-saver/domain"
)

// RepositoryConfig holds configuration for the repository, e.g., save delay.
type RepositoryConfig struct {
	SaveDelay time.Duration // not used yet, but reserved for future use
}

// Repository provides access to file metadata stored in JSON.
type Repository struct {
	filePath string
	config   RepositoryConfig
	mu       sync.Mutex
	cache    map[string]Metadata
}

// NewRepository creates a new Repository with the given file path and config.
func NewRepository(filePath string, config RepositoryConfig) (*Repository, error) {
	cache, err := loadMetadata(filePath)
	if err != nil {
		return nil, err
	}
	return &Repository{
		filePath: filePath,
		config:   config,
		cache:    cache,
	}, nil
}

// Get returns metadata for a given relPath, or false if not found.
func (r *Repository) Get(relPath string) (domain.Metadata, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	meta, ok := r.cache[relPath]
	return modelMetadataToDomain(relPath, meta), ok
}

// Upsert inserts or updates metadata for a given relPath.
func (r *Repository) Upsert(meta domain.Metadata) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.cache[meta.Patch] = domainMetadataToModel(meta)
	return saveMetadata(r.filePath, r.cache)
}

// Delete removes metadata for a given relPath.
func (r *Repository) Delete(relPath string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.cache, relPath)
	return saveMetadata(r.filePath, r.cache)
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
