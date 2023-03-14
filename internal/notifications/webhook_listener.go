package notifications

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"path"
	"sync"

	"github.com/docker/distribution/notifications"
	"github.com/docker/distribution/reference"
	"github.com/gorilla/mux"
)

type webhookListener struct {
	registry    string
	actionQueue ActionQueue
	server      *http.Server
}

// NewWebHookLister creates a new Listener for listening to webhook updates
func NewWebHookLister(actionQueue ActionQueue, registry string, listen string) Listener {
	router := mux.NewRouter()
	listener := &webhookListener{
		registry:    registry,
		actionQueue: actionQueue,
		server: &http.Server{
			Addr:    listen,
			Handler: router,
		},
	}
	router.HandleFunc(
		"/event",
		listener.handlerFunc,
	).Methods("POST")

	return listener
}

func (l *webhookListener) Serve(ctx context.Context, wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		l.server.ListenAndServe()
	}()

	go func() {
		select {
		case <-ctx.Done():
			l.server.Shutdown(context.Background())
			return
		}
	}()
}

// NewWebhookHandler creates a new webhook event handler function
func (l *webhookListener) handlerFunc(w http.ResponseWriter, r *http.Request) {
	var envelope notifications.Envelope
	err := json.NewDecoder(r.Body).Decode(&envelope)
	if err != nil {
		log.Printf("Invalid envelope")
		return
	}

	for _, event := range envelope.Events {
		if event.Target.Tag == "" {
			// Silenty skip actions without tags
			continue
		}

		repositoryRef, err := reference.WithName(path.Join(l.registry, event.Target.Repository))
		if err != nil {
			log.Printf("[webhook_listener] Invalid target.repository field in event %s: %s", event.ID, err)
			if out, err := json.Marshal(event); err == nil {
				log.Printf("> %v", string(out))
			}
			continue
		}
		imageRef, err := reference.WithTag(repositoryRef, event.Target.Tag)
		if err != nil {
			log.Printf("[webhook_listener] Invalid target.tag field in event %s: %s", event.ID, err)
			if out, err := json.Marshal(event); err == nil {
				log.Printf("> %v", string(out))
			}
			continue
		}

		switch event.Action {
		case "push":
			if event.Target.Tag != "" {
				log.Printf("[webhook_listener] Reindexing %v", imageRef)
				l.actionQueue <- Action{
					Type:  IndexImageAction,
					Image: imageRef,
				}
			}
		case "delete":
			if event.Target.Tag != "" {
				log.Printf("[webhook_listener] Deleting %v", imageRef)
				l.actionQueue <- Action{
					Type:  DeleteImageAction,
					Image: imageRef,
				}
			}
		case "pull":
			continue
		default:
			log.Printf("Unhandled event type received: %v\n", event.Action)
			if out, err := json.Marshal(event); err == nil {
				log.Printf("> %v", string(out))
			}
		}
	}
}
