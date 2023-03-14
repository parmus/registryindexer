package config

import (
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Registries      []*RegistryOpts     `yaml:"registries"`
	WebhookListener WebhookListenerOpts `yaml:"webhook-listener"`
	PubSubListener  PubSubListenerOpts  `yaml:"pubsub-listener"`
	Indexer         IndexerOpts         `yaml:"indexer"`
	API             APIOpts             `yaml:"api"`
}

func (c *Config) UnmarshalYAML(value *yaml.Node) error {
	in := struct {
		Registries      []*RegistryOpts      `yaml:"registries"`
		WebhookListener *WebhookListenerOpts `yaml:"webhook-listener"`
		PubSubListener  *PubSubListenerOpts  `yaml:"pubsub-listener"`
		Indexer         *IndexerOpts         `yaml:"indexer"`
		API             *APIOpts             `yaml:"api"`
	}{
		Registries:      c.Registries,
		WebhookListener: &c.WebhookListener,
		PubSubListener:  &c.PubSubListener,
		Indexer:         &c.Indexer,
		API:             &c.API,
	}
	err := value.Decode(&in)
	if err != nil {
		return err
	}

	seen := make(map[string]bool)
	for _, registry := range in.Registries {
		if seen[registry.BaseURL.String()] {
			return errors.Errorf("Duplicate registry entry: %v", registry.BaseURL)
		}
		seen[registry.BaseURL.String()] = true
	}
	c.Registries = in.Registries

	if in.WebhookListener != nil {
		c.WebhookListener = *in.WebhookListener
	}

	if in.PubSubListener != nil {
		c.PubSubListener = *in.PubSubListener
	}
	if in.Indexer != nil {
		c.Indexer = *in.Indexer
	}
	if in.API != nil {
		c.API = *in.API
	}
	return nil
}

type APIOpts struct {
	Listen       string `yaml:"listen"`
	CORSAllowAll bool   `yaml:"cors-allow-all"`
}

func (a *APIOpts) UnmarshalYAML(value *yaml.Node) error {
	var in struct {
		Listen       string `yaml:"listen"`
		CORSAllowAll bool   `yaml:"cors-allow-all"`
	}

	if err := value.Decode(&in); err != nil {
		return err
	}
	if in.Listen != "" {
		a.Listen = in.Listen
	}
	a.CORSAllowAll = in.CORSAllowAll
	return nil
}

// DefaultConfig creates a default configuration with sane defaults
func DefaultConfig() *Config {
	return &Config{
		Registries:      make([]*RegistryOpts, 0),
		WebhookListener: WebhookListenerOpts{},
		PubSubListener: PubSubListenerOpts{
			Projects:     make([]string, 0),
			Prefixes:     make([]string, 0),
			Subscription: "registryindexer",
		},
		Indexer: IndexerOpts{
			QueueLength:    1024,
			StateFile:      "",
			IndexOnStartup: true,
		},
		API: APIOpts{
			Listen: ":5010",
		},
	}
}
