package jsonstore

import "time"

type Metadata struct {
	CreationTime time.Time `json:"creation_time"`
}
