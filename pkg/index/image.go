package index

import (
	"time"

	"github.com/parmus/registryindexer/pkg/registry"
	"github.com/docker/distribution/reference"
)

type Image struct {
	Tag     string            `json:"tag"`
	Created time.Time         `json:"created"`
	Labels  map[string]string `json:"labels"`
}

// FetchImage fetch a single image from a repository in a registry
func FetchImage(registry *registry.Registry, tag reference.NamedTagged) (*Image, error) {
	image, err := registry.GetImageFromTag(tag)
	if err != nil {
		return nil, err
	}
	created, err := time.Parse(time.RFC3339Nano, image.Created)
	if err != nil {
		return nil, err
	}

	return &Image{tag.Tag(), created, image.Config.Labels}, nil
}
