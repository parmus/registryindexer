// Package registry implements a simple HTTP client for Docker Registry
package registry

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/url"
	"path"

	"github.com/parmus/registryindexer/internal/utils"
	"github.com/docker/distribution"
	"github.com/docker/distribution/manifest/schema2"
	"github.com/docker/distribution/reference"
	"github.com/docker/distribution/registry/client/auth"
	"github.com/docker/docker/api/types"
	"github.com/pkg/errors"
)

// Registry is a simple HTTP client for Docker Registry. This client only
// subset of the Docker Registry API endpoints.
type Registry struct {
	ctx           context.Context
	baseURL       *url.URL
	prefixes      []string
	clientFactory ClientFactory
}

// NewRegistry creates a new Registry client.
func NewRegistry(baseURL *url.URL, prefixes []string, concurrency int, credentialStore auth.CredentialStore) (*Registry, error) {
	clientFactory, err := NewClientFactory(baseURL, concurrency, credentialStore)
	if err != nil {
		return nil, err
	}

	return &Registry{
		ctx:           context.Background(),
		baseURL:       baseURL,
		prefixes:      prefixes,
		clientFactory: clientFactory,
	}, nil
}

func (r *Registry) Hostname() string {
	return r.baseURL.Hostname()
}

// GetCatalog returns a list of all repositories
func (r *Registry) GetCatalog() ([]reference.Named, error) {
	repositoryNames := make([]string, 1000)

	registry, err := r.clientFactory.GetRegistry()
	if err != nil {
		return nil, err
	}

	n, err := registry.Repositories(r.ctx, repositoryNames, "")
	if err != nil && err != io.EOF {
		return nil, errors.WithStack(err)
	}
	refs := make([]reference.Named, 0, n)
	for i, repositoryName := range repositoryNames {
		if i == n {
			break
		}
		ref, err := reference.WithName(path.Join(r.baseURL.Host, repositoryName))
		if err != nil {
			return nil, fmt.Errorf("could not parse %s as a valid reference: %s", repositoryName, err)
		}
		if len(r.prefixes) > 0 && !utils.HasAnyPrefix(r.prefixes, ref.Name()) {
			log.Printf("Skipping %s", ref.Name())
			continue
		}

		refs = append(refs, ref)
	}

	return refs, nil
}

// GetTags returns a list of all tags for a repository
func (r *Registry) GetTags(repositoryName reference.Named) ([]reference.NamedTagged, error) {
	repository, err := r.clientFactory.GetRepository(repositoryName)
	if err != nil {
		return nil, err
	}

	tags, err := repository.Tags(r.ctx).All(r.ctx)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	refs := make([]reference.NamedTagged, 0, len(tags))
	for _, tag := range tags {
		ref, err := reference.WithTag(repositoryName, tag)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		refs = append(refs, ref)
	}

	return refs, nil
}

// GetManifest returns a schema for a specific tag for a specific repository
func (r *Registry) GetManifest(tagged reference.NamedTagged) (*schema2.Manifest, error) {
	repository, err := r.clientFactory.GetRepository(tagged)
	if err != nil {
		return nil, err
	}

	descriptor, err := repository.Tags(r.ctx).Get(r.ctx, tagged.Tag())
	if err != nil {
		return nil, errors.WithStack(err)
	}

	manifestService, err := repository.Manifests(r.ctx)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	manifestResponse, err := manifestService.Get(r.ctx, descriptor.Digest, distribution.WithManifestMediaTypes([]string{schema2.MediaTypeManifest}))
	if err != nil {
		return nil, errors.WithStack(err)
	}

	_, imageResponse, err := manifestResponse.Payload()
	if err != nil {
		return nil, errors.WithStack(err)
	}

	var manifest schema2.Manifest
	err = json.Unmarshal(imageResponse, &manifest)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return &manifest, nil
}

// GetImage returns a specific blob from a repository
func (r *Registry) GetImage(digest reference.Canonical) (*types.ImageInspect, error) {
	repository, err := r.clientFactory.GetRepository(digest)
	if err != nil {
		return nil, err
	}

	imageResponse, err := repository.Blobs(r.ctx).Get(r.ctx, digest.Digest())
	if err != nil {
		return nil, errors.WithStack(err)
	}

	var image types.ImageInspect
	err = json.Unmarshal(imageResponse, &image)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return &image, nil
}

// GetImageFromTag returns a specific blob from a repository based on tag
func (r *Registry) GetImageFromTag(tag reference.NamedTagged) (*types.ImageInspect, error) {
	manifest, err := r.GetManifest(tag)
	if err != nil {
		return nil, err
	}

	digest, err := reference.WithDigest(tag, manifest.Config.Digest)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return r.GetImage(digest)
}
