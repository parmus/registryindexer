package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/parmus/registryindexer/internal/api"
	"github.com/parmus/registryindexer/internal/notifications"
	indexing "github.com/parmus/registryindexer/pkg/index"
	_ "github.com/joho/godotenv/autoload"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"github.com/parmus/registryindexer/pkg/registry"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	wg := &sync.WaitGroup{}

	config := parseCommandlineArgument()
	stateStorage, err := config.Indexer.GetStateStorage(ctx)
	if err != nil {
		log.Fatalf("Error while trying to initialize cache: %v", err)
	}
	index, err := stateStorage.LoadIndex()
	if err != nil {
		log.Fatalf("Error while trying to read cache: %v", err)
	}

	registries := make([]*registry.Registry, len(config.Registries))
	for i, r := range config.Registries {
		credentialStore, err := r.GetCredentialStore(ctx)
		if err != nil {
			log.Fatal(err)
		}

		registry, err := registry.NewRegistry(r.BaseURL, r.Prefixes, 50, credentialStore)
		if err != nil {
			log.Fatal(err)
		}
		registries[i] = registry
	}

	indexer, err := indexing.NewIndexer(index, config.Indexer.QueueLength, registries...)
	if err != nil {
		log.Fatalf("Failed to create indexer: %+v", err)
	}

	promauto.NewGaugeFunc(
		prometheus.GaugeOpts{
			Namespace: "registryindexer",
			Name:      "actions_in_queue",
			Help:      "Number of queued up actions",
		},
		func() float64 {
			return float64(len(indexer.ActionQueue()))
		},
	)

	controller := api.NewController(index, config.API.Listen, config.API.CORSAllowAll)
	controller.Serve(ctx, wg)
	log.Printf("Listening on %v", config.API.Listen)

	if config.WebhookListener.Enabled() {
		notifications.NewWebHookLister(indexer.ActionQueue(), config.WebhookListener.Registry, config.WebhookListener.Listen).Serve(ctx, wg)
		log.Printf("Listening for webhook notifications on %v", config.WebhookListener.Listen)
	}
	if config.PubSubListener.Enabled() {
		if pubsublistener, err := notifications.NewPubSubListener(indexer.ActionQueue(), config.PubSubListener.Projects, config.PubSubListener.Prefixes, config.PubSubListener.Subscription); err == nil {
			pubsublistener.Serve(ctx, wg)
		} else {
			log.Fatalf("Failed to create PubSub listener: %+v", err)
		}
		log.Printf("Listening for PubSub notifications")
	}

	promauto.NewGaugeFunc(
		prometheus.GaugeOpts{
			Namespace: "registryindexer",
			Name:      "images_total",
			Help:      "Total number of indexed images",
		},
		func() float64 {
			index.Locker().Lock()
			defer index.Locker().Unlock()

			var total int
			for _, repository := range index.Repositories() {
				total = total + len(index.Repository(repository).Images)
			}
			return float64(total)
		},
	)

	if config.Indexer.IndexOnStartup {
		start := time.Now()
		log.Println("Reindexing started")
		err := indexer.IndexAll()
		if err != nil {
			log.Fatalf("Failed to reindex registry: %v", err)
		}
		log.Printf("Indexed in %.2f seconds\n", time.Since(start).Seconds())
	}

	indexer.Serve(ctx, wg)

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	wg.Add(1)
	go func(cancel context.CancelFunc) {
		defer wg.Done()

		<-sigs
		if err := stateStorage.SaveIndex(index); err != nil {
			log.Fatalf("Failed to store cached index: %v", err)
		}
		cancel()
	}(cancel)

	wg.Wait()
	log.Printf("Shutting down")
}
