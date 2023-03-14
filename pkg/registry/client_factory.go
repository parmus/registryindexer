// Package registry implements a simple HTTP client for Docker Registry
package registry

import (
	"net/http"
	"net/url"
	"sync"

	"github.com/docker/distribution"
	"github.com/docker/distribution/reference"
	"github.com/docker/distribution/registry/client"
	"github.com/docker/distribution/registry/client/auth"
	"github.com/docker/distribution/registry/client/auth/challenge"
	"github.com/docker/distribution/registry/client/transport"
	"github.com/docker/docker/registry"
	"github.com/pkg/errors"
)

const (
	// AuthClientID is the client ID used by the client created by GetV2AuthHTTPClient
	AuthClientID = "registryindexer"
)

// ClientFactory produces client.Registry and distribution.Repository instances while
// hiding authentication complexity
type ClientFactory interface {
	// GetTransport creates a new http.RoundTripper authenticated with a specific scope.
	// GetTransport is not memoized, and will create a new http.RoundTripper instance
	// every time.
	GetTransport(...auth.Scope) http.RoundTripper

	// GetRegistry creates a client.Registry. GetRegistry is memoized and safe to call
	// naively whenever needed.
	GetRegistry() (client.Registry, error)

	// GetRepository creates a distribution.Repository for a specific repository.
	// GetRepository is memoized and safe to call naively whenever needed.
	GetRepository(reference.Named) (distribution.Repository, error)
}

type clientFactory struct {
	baseURL             *url.URL
	credentialStore     auth.CredentialStore
	challengeManager    challenge.Manager
	tokenHandlerOptions auth.TokenHandlerOptions

	// memoized instances
	registry     client.Registry
	repositories map[reference.Named]distribution.Repository

	// Mutex
	mutex sync.Mutex
}

// NewClientFactory produces a new ClientFactory
func NewClientFactory(baseURL *url.URL, concurrency int, credentialStore auth.CredentialStore) (ClientFactory, error) {
	baseTransport := registry.NewTransport(nil)
	baseTransport.MaxConnsPerHost = concurrency
	baseTransport.DisableKeepAlives = false

	challengeManager, foundV2, err := registry.PingV2Registry(baseURL, baseTransport)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	if !foundV2 {
		return nil, errors.Errorf("%v is not a Docker Registry v2 API", baseURL)
	}
	tokenHandlerOptions := auth.TokenHandlerOptions{
		Transport:     baseTransport,
		Credentials:   credentialStore,
		OfflineAccess: true,
		ClientID:      AuthClientID,
		Scopes:        nil,
	}

	return &clientFactory{
		baseURL:             baseURL,
		credentialStore:     credentialStore,
		challengeManager:    challengeManager,
		tokenHandlerOptions: tokenHandlerOptions,
		registry:            nil,
		repositories:        make(map[reference.Named]distribution.Repository),
	}, nil
}

func (f *clientFactory) GetTransport(scopes ...auth.Scope) http.RoundTripper {
	if f.credentialStore == nil {
		return f.tokenHandlerOptions.Transport
	}

	tokenHandlerOptions := f.tokenHandlerOptions
	tokenHandlerOptions.Scopes = scopes
	tokenHandler := auth.NewTokenHandlerWithOptions(tokenHandlerOptions)
	return transport.NewTransport(f.tokenHandlerOptions.Transport, auth.NewAuthorizer(f.challengeManager, tokenHandler))
}

func (f *clientFactory) GetRegistry() (client.Registry, error) {
	var err error
	if f.registry == nil {
		f.registry, err = client.NewRegistry(f.baseURL.String(), f.GetTransport())
	}
	return f.registry, errors.WithStack(err)
}

func (f *clientFactory) GetRepository(repositoryName reference.Named) (distribution.Repository, error) {
	var err error
	var ok bool
	var repository distribution.Repository

	if reference.Domain(repositoryName) != f.baseURL.Hostname() {
		return nil, errors.Errorf("Domain name mismatch: %v does not belong in %v", repositoryName, f.baseURL)
	}

	repositoryName, err = reference.WithName(reference.Path(repositoryName))
	if err != nil {
		return nil, errors.WithStack(err)
	}

	f.mutex.Lock()
	defer f.mutex.Unlock()
	if repository, ok = f.repositories[repositoryName]; !ok {
		scope := auth.RepositoryScope{
			Repository: repositoryName.String(),
			Actions:    []string{"pull"},
		}

		var err error
		repository, err = client.NewRepository(repositoryName, f.baseURL.String(), f.GetTransport(scope))
		if err != nil {
			return nil, errors.WithStack(err)
		}
		f.repositories[repositoryName] = repository
	}
	return repository, nil
}
