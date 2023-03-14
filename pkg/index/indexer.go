package index

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/parmus/registryindexer/internal/notifications"
	"github.com/parmus/registryindexer/pkg/registry"
	"github.com/docker/distribution/reference"
	"github.com/pkg/errors"
)

type Indexer struct {
	registryByHost map[string]*registry.Registry
	index          *Index
	actionQueue    chan notifications.Action
}

// NewIndexer creates a new Indexer
func NewIndexer(index *Index, actionQueueLength uint64, registries ...*registry.Registry) (*Indexer, error) {
	registryByHost := make(map[string]*registry.Registry)
	for _, registry := range registries {
		registryByHost[registry.Hostname()] = registry
	}

	return &Indexer{
		registryByHost: registryByHost,
		index:          index,
		actionQueue:    make(chan notifications.Action, actionQueueLength),
	}, nil
}

// ActionQueue returns an action queue for use by notification listeners
func (i *Indexer) ActionQueue() notifications.ActionQueue {
	return i.actionQueue
}

// IndexAll performs a complete reindexing
func (i *Indexer) IndexAll() error {
	allRepositories := make(map[reference.Named]*Repository)
	for _, registry := range i.registryByHost {
		repositories, err := FetchRepositories(registry)
		if err != nil {
			return err
		}
		for key, value := range repositories {
			allRepositories[key] = value
		}
	}

	i.index.ReplaceAllRepositories(allRepositories)
	return nil
}

// IndexRepository performs a reindexing of a single repository
func (i *Indexer) IndexRepository(repositoryRef reference.Named) error {
	registry := i.registryByHost[reference.Domain(repositoryRef)]
	if registry == nil {
		return errors.Errorf("Failed to index %v: no such registry configured", repositoryRef)
	}
	repository, err := FetchRepository(registry, repositoryRef)
	if err != nil {
		return err
	}

	i.index.ReplaceRepository(repository)
	return nil
}

// IndexImage reindexes a single image
func (i *Indexer) IndexImage(imageRef reference.NamedTagged) error {
	registry := i.registryByHost[reference.Domain(imageRef)]
	if registry == nil {
		return errors.Errorf("Failed to index %v: no such registry configured", imageRef)
	}
	image, err := FetchImage(registry, imageRef)
	if err != nil {
		return err
	}

	i.index.ReplaceImage(imageRef, image)
	return nil
}

// DeleteImage deletes an image from a repository
func (i *Indexer) DeleteImage(imageRef reference.NamedTagged) {
	i.index.DeleteImage(imageRef)
}

// Serve starts serving the action queue
func (i *Indexer) Serve(ctx context.Context, wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		defer wg.Done()

		taintedImages := make([]reference.NamedTagged, 0)

		for {
			select {
			case action := <-i.actionQueue:
				switch action.Type {
				case notifications.IndexAllAction:
					log.Printf("[indexer] Reindexing all registries")
					if err := i.IndexAll(); err != nil {
						log.Printf("Unable to reindex registry: %v", err)
					}
				case notifications.IndexRepositoryAction:
					if _, ok := i.registryByHost[reference.Domain(action.Repository)]; !ok {
						log.Printf("[indexer] Skipping %v; registry not configured", action.Repository)
						continue
					}
					log.Printf("[indexer] Reindexing %v", action.Repository)
					if err := i.IndexRepository(action.Repository); err != nil {
						log.Printf("Unable to reindex repository %v: %v", action.Repository, err)
					}
				case notifications.IndexImageAction:
					if _, ok := i.registryByHost[reference.Domain(action.Image)]; !ok {
						log.Printf("[indexer] Skipping %v; registry not configured", action.Image)
						continue
					}
					log.Printf("[indexer] Reindexing %v", action.Image)
					if err := i.IndexImage(action.Image); err != nil {
						log.Printf("Unable to reindex image %v: %v", action.Image, err)
						taintedImages = append(taintedImages, action.Image)
					}
				case notifications.DeleteImageAction:
					log.Printf("[indexer] Deleting %v", action.Image)
					i.DeleteImage(action.Image)
				}
			case <-time.After(10 * time.Second):
				if len(taintedImages) > 0 {
					stillTainted := make([]reference.NamedTagged, 0)
					for _, image := range taintedImages {
						log.Printf("[indexer][retry] Reindexing %v", image)
						if err := i.IndexImage(image); err != nil {
							log.Printf("Unable to reindex image %v: %+v", image, err)
							stillTainted = append(stillTainted, image)
						}
					}
					taintedImages = stillTainted
					if len(taintedImages) > 0 {
						log.Printf("[Indexer] %v tainted images remains", len(taintedImages))
					}
				}
			case <-ctx.Done():
				log.Printf("Shutting down indexer")
				return
			}
		}
	}()
}
