package internal

import (
	"encoding/json"
	"os"
	"time"
)

// Metadata holds information about a file's creation time.
type Metadata struct {
	CreationTime time.Time `json:"creation_time"`
}

// MetadataMap maps file paths to their metadata.
type MetadataMap map[string]Metadata

// LoadMetadata loads the metadata from a JSON file.
func LoadMetadata(filePath string) (MetadataMap, error) {
	file, err := os.ReadFile(filePath)
	if err != nil {
		// If the file does not exist, return an empty map.
		if os.IsNotExist(err) {
			return make(MetadataMap), nil
		}
		return nil, err
	}

	var metadata MetadataMap
	err = json.Unmarshal(file, &metadata)
	if err != nil {
		return nil, err
	}

	return metadata, nil
}

// SaveMetadata saves the metadata to a JSON file.
func SaveMetadata(filePath string, metadata MetadataMap) error {
	data, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return err
	}

	err = os.WriteFile(filePath, data, 0644)
	if err != nil {
		return err
	}

	return nil
}

// UpdateCreationTime updates the creation time of a file in the metadata.
func UpdateCreationTime(metadata MetadataMap, relPath string, creationTime time.Time) {
	// Check if the file already exists in metadata.
	if meta, exists := metadata[relPath]; exists {
		// Update the creation time only if it's earlier than the stored one.
		if creationTime.Before(meta.CreationTime) {
			meta.CreationTime = creationTime
			metadata[relPath] = meta
		}
	} else {
		// Add a new record for the file.
		metadata[relPath] = Metadata{
			CreationTime: creationTime,
		}
	}
}

// DeleteMetadata completely removes the metadata entry for a file.
func DeleteMetadata(metadata MetadataMap, relPath string) {
	delete(metadata, relPath)
}
