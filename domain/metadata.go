package domain

import "time"

// Metadata holds information about a file's creation time and patch.
type Metadata struct {
	Patch        string    `json:"patch,omitempty"`
	CreationTime time.Time `json:"creation_time"`
}
