package notifications

import (
	"context"
	"encoding/json"
	"log"
	"sync"

	"cloud.google.com/go/pubsub"
	"github.com/parmus/registryindexer/internal/utils"
	"github.com/docker/distribution/reference"
	"github.com/pkg/errors"
)

// TODO: Remove hardcodings
const (
	DefaultSubID = "registryindexer"
)

type pubsubevent struct {
	Action string
	Digest string `json:",omitempty"`
	Tag    string `json:",omitempty"`
}

type pubsublistener struct {
	subscriptions []*pubsub.Subscription
	prefixes      []string
	actionQueue   ActionQueue
}

// NewPubSubListener creates a new Listener for listening to Google Pub/Sub updates
func NewPubSubListener(actionQueue ActionQueue, projectIDs []string, prefixes []string, subscriptionID string) (Listener, error) {
	if subscriptionID == "" {
		subscriptionID = DefaultSubID
	}

	subscriptions := make([]*pubsub.Subscription, 0, len(projectIDs))
	for _, projectID := range projectIDs {
		if subscription, err := getSubscription(context.Background(), projectID, subscriptionID); err == nil {
			subscriptions = append(subscriptions, subscription)
		} else {
			return nil, err
		}
	}

	return &pubsublistener{
		subscriptions: subscriptions,
		prefixes:      prefixes,
		actionQueue:   actionQueue,
	}, nil

}

func (l *pubsublistener) Serve(ctx context.Context, wg *sync.WaitGroup) {
	for _, subscription := range l.subscriptions {
		wg.Add(1)
		go func(subscription *pubsub.Subscription) {
			defer wg.Done()

			err := subscription.Receive(ctx, func(ctx context.Context, msg *pubsub.Message) {
				var event pubsubevent
				if err := json.Unmarshal(msg.Data, &event); err != nil {
					log.Printf("[Pubsub message %v] Invalid event", msg.ID)
					return
				}
				defer msg.Ack()

				if event.Tag == "" {
					// TODO: Count in monitoring
					return
				}

				distributionRef, err := reference.ParseNamed(event.Tag)
				if err != nil {
					log.Printf("[Pubsub message %v] %v", msg.ID, err)
					return
				}

				if len(l.prefixes) > 0 && !utils.HasAnyPrefix(l.prefixes, distributionRef.Name()) {
					// TODO: Count in monitoring
					log.Printf("[Pubsub message %v] %s doesn't match any of the allowed prefixes", msg.ID, distributionRef.Name())
					return
				}

				tagged, ok := distributionRef.(reference.NamedTagged)
				if !ok {
					log.Printf("[Pubsub message %v] Ignored because tag is missing", msg.ID)
					return
				}

				switch event.Action {
				case "INSERT":
					l.actionQueue <- Action{
						Type:  IndexImageAction,
						Image: tagged,
					}
				case "DELETE":
					l.actionQueue <- Action{
						Type:  DeleteImageAction,
						Image: tagged,
					}
				default:
					log.Printf("Unhandled event type received: %v\n", event.Action)
				}
			})
			if err != nil {
				log.Printf("PubSub subscription %v failed: %+v", subscription, err)
			} else {
				log.Printf("Shutting down pubsub subscription")
			}
		}(subscription)
	}
}

func getSubscription(ctx context.Context, projectID string, subID string) (*pubsub.Subscription, error) {
	client, err := pubsub.NewClient(ctx, projectID)
	if err != nil {
		return nil, errors.Errorf("Can't create PubSub client: %+v", err) // TODO: Replace with proper error handler
	}

	return client.Subscription(subID), nil
}
