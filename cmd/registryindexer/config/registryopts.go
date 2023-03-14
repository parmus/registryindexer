package config

import (
	"context"
	"net/url"
	"strings"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"

	"github.com/parmus/registryindexer/pkg/auth"
	docker_auth "github.com/docker/distribution/registry/client/auth"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/registry"
)

type RegistryOpts struct {
	BaseURL     *url.URL
	Prefixes    []string
	Credentials *Credentials
}

type Credentials struct {
	Username string
	Password string
}

func (r *RegistryOpts) MarshalYAML() (interface{}, error) {
	out := struct {
		BaseURL     string
		Prefixes    []string
		Credentials *Credentials
	}{
		BaseURL:     r.BaseURL.String(),
		Prefixes:    r.Prefixes,
		Credentials: r.Credentials,
	}
	return out, nil
}

func (c *RegistryOpts) UnmarshalYAML(value *yaml.Node) error {
	var in struct {
		BaseURL     string
		Prefixes    []string
		Credentials *Credentials
	}
	var err error
	err = value.Decode(&in)
	if err != nil {
		return err
	}
	var baseurl *url.URL
	baseurl, err = url.Parse(in.BaseURL)
	if err != nil {
		return err
	}

	if baseurl.Host == "" && baseurl.Path != "" {
		if strings.HasPrefix(baseurl.Path, "/") {
			return errors.Errorf("Invalid URL: %v", baseurl)
		}
		baseurl.Host = baseurl.Path
		baseurl.Path = ""
	}
	if baseurl.Scheme == "" {
		baseurl.Scheme = "https"
	}

	c.BaseURL = baseurl
	c.Prefixes = in.Prefixes
	c.Credentials = in.Credentials

	return nil
}

func (c *RegistryOpts) GetCredentialStore(context context.Context) (docker_auth.CredentialStore, error) {
	if c.Credentials != nil {
		return registry.NewStaticCredentialStore(&types.AuthConfig{
			Username: c.Credentials.Username,
			Password: c.Credentials.Password,
		}), nil
	}

	return auth.NewApplicationDefaultCredentialStore(context)
}
