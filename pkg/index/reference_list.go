package index

import (
	"github.com/docker/distribution/reference"
)

type referenceList []reference.Named

func (r referenceList) Len() int {
	return len(r)
}

func (r referenceList) Swap(i, j int) {
	r[i], r[j] = r[j], r[i]
}

func (r referenceList) Less(i, j int) bool {
	return r[i].Name() < r[j].Name()
}
