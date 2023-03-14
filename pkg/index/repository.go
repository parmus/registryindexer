package index

import (
	"log"
	"sort"
	"sync"

	"github.com/parmus/registryindexer/pkg/registry"
	"github.com/docker/distribution/reference"
)

type Repository struct {
	Name       reference.Named
	Images     []*Image
	imageByTag map[string]*Image
}

// RepositoryFromImages creates a new Repository from a single Image
func RepositoryFromImages(repositoryRef reference.Named, images ...*Image) *Repository {
	repository := Repository{
		Name:       reference.TrimNamed(repositoryRef),
		Images:     make([]*Image, 0, len(images)),
		imageByTag: make(map[string]*Image),
	}
	repository.Images = images
	for _, image := range images {
		repository.imageByTag[image.Tag] = image
	}
	repository.sort()

	return &repository
}

// GetImage gets an image by it's tag name
func (r *Repository) GetImage(imageRef reference.NamedTagged) *Image {
	return r.imageByTag[imageRef.Tag()]
}

// UpdateImage adds or updates an image in the repository
func (r *Repository) UpdateImage(image *Image) {
	if _, ok := r.imageByTag[image.Tag]; ok {
		images := make([]*Image, 0, len(r.Images)+1)
		for _, i := range r.Images {
			if i.Tag == image.Tag {
				images = append(images, image)
			} else {
				images = append(images, i)
			}
		}
		r.Images = images
	} else {
		r.Images = append(r.Images, image)
	}
	r.sort()
	r.imageByTag[image.Tag] = image
}

// DeleteImage deletes an image from the repository
func (r *Repository) DeleteImage(imageRef reference.NamedTagged) {
	imageTag := imageRef.Tag()
	if _, ok := r.imageByTag[imageTag]; ok {
		images := make([]*Image, 0, len(r.Images)-1)
		for _, image := range r.Images {
			if image.Tag != imageTag {
				images = append(images, image)
			}
		}
		r.Images = images
	}
	delete(r.imageByTag, imageTag)
}

// FetchRepository fetch a whole repository from a registry
func FetchRepository(registry *registry.Registry, repositoryRef reference.Named) (*Repository, error) {
	ch := make(chan *Image)
	tags, err := registry.GetTags(repositoryRef)
	if err != nil {
		return nil, err
	}

	var wg sync.WaitGroup
	for _, tag := range tags {
		wg.Add(1)
		go func(tag reference.NamedTagged) {
			defer wg.Done()
			image, err := FetchImage(registry, tag)
			if err != nil {
				log.Fatalf("Failed to fetch image: %+v", err)
			}
			ch <- image
		}(tag)
	}

	go func() {
		wg.Wait()
		close(ch)
	}()

	images := make([]*Image, 0, len(tags))
	for image := range ch {
		images = append(images, image)
	}

	return RepositoryFromImages(repositoryRef, images...), nil
}

// FetchRepositories fetch all repositories from a registry
func FetchRepositories(registry *registry.Registry) (map[reference.Named]*Repository, error) {
	ch := make(chan *Repository)

	var wg sync.WaitGroup
	repos, err := registry.GetCatalog()
	if err != nil {
		return nil, err
	}

	for _, repositoryName := range repos {
		wg.Add(1)
		go func(repositoryName reference.Named) {
			defer wg.Done()
			repository, err := FetchRepository(registry, repositoryName)
			if err != nil {
				log.Fatalf("Failed to fetch repository: %+v", err)
			}
			ch <- repository
		}(repositoryName)
	}

	go func() {
		wg.Wait()
		close(ch)
	}()

	repositories := make(map[reference.Named]*Repository)
	for repository := range ch {
		repositories[repository.Name] = repository
	}
	return repositories, nil
}

func (r *Repository) sort() {
	sort.Slice(r.Images, func(i, j int) bool {
		return r.Images[i].Created.After(r.Images[j].Created)
	})
}
