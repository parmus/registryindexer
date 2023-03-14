package config

import (
	"context"

	"github.com/parmus/registryindexer/pkg/index"
	"gopkg.in/yaml.v3"
)

type IndexerOpts struct {
	QueueLength    uint64 `yaml:"queue-length"`
	StateFile      string `yaml:"state-file"`
	IndexOnStartup bool   `yaml:"index-on-startup"`
}

func (i *IndexerOpts) UnmarshalYAML(value *yaml.Node) error {
	var in struct {
		QueueLength    *uint64 `yaml:"queue-length,omitempty"`
		StateFile      *string `yaml:"state-file"`
		IndexOnStartup *bool   `yaml:"index-on-startup"`
	}

	if err := value.Decode(&in); err != nil {
		return err
	}
	if in.QueueLength != nil {
		i.QueueLength = *in.QueueLength
	}
	if in.StateFile != nil {
		i.StateFile = *in.StateFile
	}
	if in.IndexOnStartup != nil {
		i.IndexOnStartup = *in.IndexOnStartup
	}
	return nil
}

func (i *IndexerOpts) GetStateStorage(ctx context.Context) (index.StateStorage, error) {
	return index.NewStateStorage(i.StateFile, ctx)
}
