package index

import (
	"encoding/json"
	"sort"
	"sync"

	"github.com/docker/distribution/reference"
)

// Index contains a index of a Docker registry
type Index struct {
	repositories map[reference.Named]*Repository
	rwmutex      sync.RWMutex
}

// NewIndex creates a new empty Index
func NewIndex() *Index {
	return &Index{
		repositories: make(map[reference.Named]*Repository),
	}
}

// Locker returns the read lock need to access the index thread-safely
func (i *Index) Locker() sync.Locker {
	return i.rwmutex.RLocker()
}

// ReplaceAllRepositories atomically replaces all repositories
func (i *Index) ReplaceAllRepositories(repositories map[reference.Named]*Repository) {
	i.rwmutex.Lock()
	defer i.rwmutex.Unlock()
	i.repositories = repositories
}

// ReplaceRepository atomically replaces a single repository
func (i *Index) ReplaceRepository(repository *Repository) {
	i.rwmutex.Lock()
	defer i.rwmutex.Unlock()
	i.repositories[repository.Name] = repository
}

// ReplaceImage atomically replaces a single image
func (i *Index) ReplaceImage(imageRef reference.NamedTagged, image *Image) {
	i.rwmutex.Lock()
	defer i.rwmutex.Unlock()

	if repository, ok := i.repositories[reference.TrimNamed(imageRef)]; ok {
		repository.UpdateImage(image)
	} else {
		repository := RepositoryFromImages(imageRef, image)
		i.repositories[repository.Name] = repository
	}
}

// DeleteImage deletes an image from a repository
func (i *Index) DeleteImage(imageRef reference.NamedTagged) {
	i.rwmutex.Lock()
	defer i.rwmutex.Unlock()
	if repository, ok := i.repositories[reference.TrimNamed(imageRef)]; ok {
		repository.DeleteImage(imageRef)
	}
}

// Repositories returns a list of all the repository names
func (i *Index) Repositories() []reference.Named {
	var result = make([]reference.Named, 0, len(i.repositories))
	for key := range i.repositories {
		result = append(result, key)
	}
	sort.Sort(referenceList(result))
	return result
}

// MarshalJSON handles JSON serialization of an Index
func (i *Index) MarshalJSON() ([]byte, error) {
	out := make(map[string]*([]*Image))
	for repositoryRef, repository := range i.repositories {
		out[repositoryRef.String()] = &repository.Images
	}
	return json.Marshal(out)
}

// UnmarshalJSON handles JSON deserialization of an Index
func (i *Index) UnmarshalJSON(b []byte) error {
	in := make(map[string]*([]*Image))
	if err := json.Unmarshal(b, &in); err != nil {
		return err
	}

	repositories := make(map[reference.Named]*Repository)
	for repositoryName, images := range in {
		repositoryRef, err := reference.ParseNamed(repositoryName)
		if err != nil {
			return err
		}

		repositories[reference.TrimNamed(repositoryRef)] = RepositoryFromImages(repositoryRef, *images...)
	}
	i.ReplaceAllRepositories(repositories)
	return nil
}

// Repository returns a single repository
func (i *Index) Repository(repositoryRef reference.Named) *Repository {
	return i.repositories[reference.TrimNamed(repositoryRef)]
}
