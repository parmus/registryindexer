package api

import "time"

// SearchQuery contains the parameters for an image search
type SearchQuery struct {
	Labels        map[string]string `json:"labels"`
	CreatedAfter  time.Time         `json:"created_after"`
	CreatedBefore time.Time         `json:"created_before"`
}
