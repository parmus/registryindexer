package notifications

import (
	"context"
	"sync"
)

// ActionQueue is a channel, which listeners push events to
type ActionQueue chan<- Action

// Listener is an interface for Docker Registry update listeners
type Listener interface {
	// Serve starts the listener as a background process until
	// the context is cancelled
	Serve(context.Context, *sync.WaitGroup)
}
