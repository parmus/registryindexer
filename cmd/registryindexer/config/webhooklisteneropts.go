package config

import (
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

type WebhookListenerOpts struct {
	Registry string
	Listen   string
}

func (w *WebhookListenerOpts) UnmarshalYAML(value *yaml.Node) error {
	var in struct {
		Registry string
		Listen   *string
	}

	if err := value.Decode(&in); err != nil {
		return err
	}
	if in.Listen != nil && in.Registry == "" {
		return errors.Errorf("webhook-listener must include registry")
	}
	if in.Listen != nil {
		w.Listen = *in.Listen
	}
	w.Registry = in.Registry
	return nil
}

func (w *WebhookListenerOpts) Enabled() bool {
	return w.Listen != ""
}
