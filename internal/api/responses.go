package api

import (
	"github.com/parmus/registryindexer/pkg/index"
)

// SearchResponse contrainer the result of an image search
type SearchResponse struct {
	Repository string         `json:"name"`
	Images     []*index.Image `json:"images"`
	Offset     int            `json:"offset"`
	Limit      int            `json:"limit"`
	Count      int            `json:"count"`
}

// ListRepositoriesResponse contains the response
// from listing all repositories
type ListRepositoriesResponse struct {
	Repositories []RepositoryStatus `json:"repositories"`
}

// RepositoryStatus contains the status of
// a single repository
type RepositoryStatus struct {
	Name   string `json:"name"`
	Images int    `json:"images"`
}
