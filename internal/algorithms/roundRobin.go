package algorithms

import (
	"fmt"
	"sync/atomic"

	"github.com/soham0w0sarkar/LoadBalancerGo.git/internal/backend"
)

type RoundRobin struct {
	current uint64
}

func (rr *RoundRobin) NextIndex(backends []*backend.Backend) int {
	return int(atomic.AddUint64(&rr.current, uint64(1)) % uint64(len(backends)))
}

func (rr *RoundRobin) Select(backends []*backend.Backend) (*backend.Backend, error) {
	if len(backends) == 0 {
		return nil, fmt.Errorf("no Backend found")
	}

	next := rr.NextIndex(backends)
	l := len(backends) + next

	for i := next; i < l; i++ {
		idx := i % len(backends)

		if backends[idx].IsAlive() {
			if i != next {
				atomic.StoreUint64(&rr.current, uint64(idx))
			}
			return backends[idx], nil
		}
	}
	return nil, fmt.Errorf("no Backend found alive")
}
