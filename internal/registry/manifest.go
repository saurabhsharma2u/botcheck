package registry

import "time"

type Manifest struct {
	Version   int       `json:"version"`
	UpdatedAt time.Time `json:"updated_at"`
	Stats     Stats     `json:"stats"`
}
