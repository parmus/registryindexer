package notifications

import (
	"github.com/docker/distribution/reference"
)

// ActionType describes a type of update action an index can perform
type ActionType int

// Action codes indicates the type of update the index should perform
const (
	IndexAllAction ActionType = iota
	IndexRepositoryAction
	IndexImageAction
	DeleteImageAction
)

// Action describes a desired update the index should perform
type Action struct {
	Type       ActionType
	Repository reference.Named
	Image      reference.NamedTagged
}
