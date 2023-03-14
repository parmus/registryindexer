package config

import (
	"gopkg.in/yaml.v3"
)

type PubSubListenerOpts struct {
	Projects     []string
	Prefixes     []string
	Subscription string
}

func (p *PubSubListenerOpts) UnmarshalYAML(value *yaml.Node) error {
	in := struct {
		Projects     []string
		Prefixes     []string
		Subscription *string
	}{
		Projects:     p.Projects,
		Prefixes:     p.Prefixes,
		Subscription: &p.Subscription,
	}

	if err := value.Decode(&in); err != nil {
		return err
	}

	if in.Subscription != nil {
		p.Subscription = *in.Subscription
	}

	p.Projects = in.Projects
	p.Prefixes = in.Prefixes
	return nil
}

func (p *PubSubListenerOpts) Enabled() bool {
	return len(p.Projects) > 0
}
