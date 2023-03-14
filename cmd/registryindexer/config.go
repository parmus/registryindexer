package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/parmus/registryindexer/cmd/registryindexer/config"
	"gopkg.in/yaml.v3"
)

func dumpConfigAndExit(config *config.Config) {
	if configYaml, err := yaml.Marshal(config); err == nil {
		fmt.Print(string(configYaml))
		os.Exit(0)
	} else {
		log.Fatal(err)
	}
}

func parseCommandlineArgument() *config.Config {
	const (
		helpUsage                   = "Show usage"
		configUsage                 = "Use configuration file"
		noReindexUsage              = "Don't reindex on startup, even if configured in the configuration file"
		disableWebhookListenerUsage = "Disable Webhook Listener, even if configured in the configuration file"
		disablePubSubListenerUsage  = "Disable PubSub Listener, even if configured in the configuration file"
		showConfigUsage             = "Show effective configuration"
		showDefaultConfigUsage      = "Show default configuration (before loading configuration file)"
	)

	var help bool
	configFile := os.Getenv("REGISTRYINDEXER_CONFIGFILE")
	if configFile == "" {
		configFile = "config.yaml"
	}

	flag.BoolVar(&help, "help", false, helpUsage)
	flag.BoolVar(&help, "h", false, helpUsage)

	flag.StringVar(
		&configFile,
		"config",
		configFile,
		configUsage,
	)

	var noReindex bool
	var disableWebhookListener bool
	var disablePubSubListener bool

	flag.BoolVar(&noReindex, "no-reindex", false, noReindexUsage)
	flag.BoolVar(&disableWebhookListener, "disable-webhook-listener", false, disableWebhookListenerUsage)
	flag.BoolVar(&disablePubSubListener, "disable-pubsub-listener", false, disablePubSubListenerUsage)

	var showConfig bool
	var showDefaultConfig bool
	flag.BoolVar(&showConfig, "show-config", false, showConfigUsage)
	flag.BoolVar(&showDefaultConfig, "show-default-config", false, showDefaultConfigUsage)

	flag.Parse()
	if help {
		flag.Usage()
		os.Exit(0)
	}

	config := config.DefaultConfig()
	if showDefaultConfig {
		dumpConfigAndExit(config)
	}

	if bytebuf, err := ioutil.ReadFile(configFile); err != nil {
		log.Fatalf("Failed to read configuration file: %v", err)
	} else {
		if err := yaml.Unmarshal(bytebuf, config); err != nil {
			log.Fatalf("%v", err)
		}
	}

	if noReindex {
		config.Indexer.IndexOnStartup = false
	}

	if disableWebhookListener {
		config.WebhookListener.Listen = ""
	}

	if disablePubSubListener {
		config.PubSubListener.Projects = make([]string, 0)
	}

	if showConfig {
		dumpConfigAndExit(config)
	}

	if len(config.Registries) == 0 {
		log.Fatal("You must configure at least one registry to index")
	}

	return config
}
