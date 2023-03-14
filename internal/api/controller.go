package api

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"sync"

	"github.com/parmus/registryindexer/internal/utils"
	"github.com/parmus/registryindexer/pkg/index"
	"github.com/docker/distribution/reference"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	// DefaultLimit is the default pagination limit
	DefaultLimit = 10
)

//go:embed docs/*
var openAPIContent embed.FS

// The Controller implements the API endpoints
type Controller struct {
	index  *index.Index
	locker sync.Locker
	server *http.Server
}

// NewController creates a new Controller instance fully ready to serve
func NewController(index *index.Index, listen string, CORSAllowAll bool) *Controller {
	router := mux.NewRouter()

	var handler http.Handler = router
	if CORSAllowAll {
		handler = handlers.CORS(
			handlers.AllowCredentials(),
			handlers.AllowedOriginValidator(func(string) bool {
				return true
			}),
		)(router)
	}

	c := &Controller{
		index:  index,
		locker: index.Locker(),
		server: &http.Server{
			Addr:    listen,
			Handler: handler,
		},
	}

	pathComponent := `[a-z0-9]+(?:[._-][a-z0-9]+)*`
	repositoryName := pathComponent + `(?:/` + pathComponent + `)*`

	requestDuration := promauto.NewSummaryVec(
		prometheus.SummaryOpts{
			Namespace:  "registryindexer",
			Name:       "request_duration_seconds",
			Help:       "Latency percentiles for request",
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
		},
		[]string{"endpoint", "method", "code"},
	)

	// Configure router
	router.Handle(
		"/repositories/{repository:"+repositoryName+"}/tags/{imageTag}",
		promhttp.InstrumentHandlerDuration(
			requestDuration.MustCurryWith(
				prometheus.Labels{"endpoint": "/repositories/{repository}/{imageTag}"},
			),
			http.HandlerFunc(c.getImage),
		),
	).Methods("GET")
	router.Handle(
		"/repositories/{repository:"+repositoryName+"}/tags",
		promhttp.InstrumentHandlerDuration(
			requestDuration.MustCurryWith(
				prometheus.Labels{"endpoint": "/repositories/{repository}"},
			),
			http.HandlerFunc(c.searchRepository),
		),
	).Methods("GET", "POST")
	router.Handle(
		"/repositories",
		promhttp.InstrumentHandlerDuration(
			requestDuration.MustCurryWith(
				prometheus.Labels{"endpoint": "/repositories"},
			),
			http.HandlerFunc(c.listRepositories),
		),
	).Methods("GET")

	// Metrics
	router.Handle("/metrics", promhttp.Handler())

	// Documentation
	router.PathPrefix("/docs").Handler(http.FileServer(http.FS(openAPIContent)))
	router.Handle("/", http.RedirectHandler("/docs/", http.StatusMovedPermanently)).Methods("GET", "HEAD")

	return c
}

// Serve starts the controller as a background process until
// the context is cancelled
func (c *Controller) Serve(ctx context.Context, wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := c.server.ListenAndServe()
		if err != http.ErrServerClosed {
			log.Fatalf("Unhandled error while serving the API: %+v", err)
		}
	}()

	go func() {
		select {
		case <-ctx.Done():
			c.server.Shutdown(context.Background())
			return
		}
	}()
}

func (c *Controller) getImage(w http.ResponseWriter, r *http.Request) {
	c.locker.Lock()
	defer c.locker.Unlock()

	vars := mux.Vars(r)
	repositoryRef, err := reference.ParseNamed(vars["repository"])
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	imageRef, err := reference.WithTag(repositoryRef, vars["imageTag"])
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	repository := c.index.Repository(imageRef)
	if repository == nil {
		http.Error(w, "Repository not found", http.StatusNotFound)
		return
	}

	image := repository.GetImage(imageRef)
	if image == nil {
		http.Error(w, "Image not found in repository", http.StatusNotFound)
		return
	}

	w.Header().Set("content-type", "application/json")
	json.NewEncoder(w).Encode(image)
}

func (c *Controller) searchRepository(w http.ResponseWriter, r *http.Request) {
	c.locker.Lock()
	defer c.locker.Unlock()

	vars := mux.Vars(r)
	repositoryRef, err := reference.ParseNamed(vars["repository"])
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid repository name: %v", err), http.StatusBadRequest)
		return
	}

	repository := c.index.Repository(repositoryRef)
	if repository == nil {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	queryParams := r.URL.Query()
	offset := 0
	if offsetStr := queryParams.Get("offset"); offsetStr != "" {
		if offsetParam, err := strconv.ParseUint(offsetStr, 10, 32); err == nil {
			offset = int(offsetParam)
		} else {
			http.Error(w, "Error: offset must be an unsigned integer", http.StatusBadRequest)
			return
		}
	}

	limit := DefaultLimit
	if limitStr := queryParams.Get("limit"); limitStr != "" {
		if limitParam, err := strconv.ParseUint(limitStr, 10, 32); err == nil {
			limit = int(limitParam)
		} else {
			http.Error(w, "Error: limit must be an unsigned integer", http.StatusBadRequest)
			return
		}
	}

	query := SearchQuery{}
	if r.Method == "POST" {
		err := json.NewDecoder(r.Body).Decode(&query)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	}
	images := make([]*index.Image, 0)

SEARCHLOOP:
	for _, image := range repository.Images {
		if !query.CreatedAfter.IsZero() && query.CreatedAfter.After(image.Created) {
			continue
		}
		if !query.CreatedBefore.IsZero() && query.CreatedBefore.Before(image.Created) {
			continue SEARCHLOOP
		}
		for labelKey, labelValue := range query.Labels {
			value, ok := image.Labels[labelKey]
			if !ok || value != labelValue {
				continue SEARCHLOOP
			}
		}
		images = append(images, image)
	}
	start := utils.MinInt(offset, len(images))
	end := utils.MinInt(start+limit, len(images))
	searchResponse := SearchResponse{
		Repository: repository.Name.String(),
		Images:     images[start:end],
		Offset:     start,
		Limit:      limit,
		Count:      len(images),
	}

	w.Header().Set("content-type", "application/json")
	json.NewEncoder(w).Encode(searchResponse)
}

func (c *Controller) listRepositories(w http.ResponseWriter, r *http.Request) {
	c.locker.Lock()
	defer c.locker.Unlock()

	repositories := make([]RepositoryStatus, 0)
	for _, repositoryRef := range c.index.Repositories() {
		repository := c.index.Repository(repositoryRef)
		repositories = append(
			repositories,
			RepositoryStatus{
				Name:   repository.Name.String(),
				Images: len(repository.Images),
			},
		)
	}

	w.Header().Set("content-type", "application/json")
	json.NewEncoder(w).Encode(ListRepositoriesResponse{repositories})
}
