package backend

import (
	"net/url"
	"slices"
	"sync"

	"github.com/soham0w0sarkar/LoadBalancerGo.git/internal/config"
)

type ServerPool struct {
	Backends []*Backend
	mux      sync.RWMutex
}

func NewServerPool(cb config.Config) *ServerPool {
	var backends []*Backend

	for _, backend := range cb.Backends {
		backendUrl, _ := url.Parse(backend.Url)
		backends = append(backends, NewBackend(backendUrl, int(cb.LoadBalancing.HealthCheck.UnhealthyThreshold)))
	}

	return &ServerPool{Backends: backends}
}

func (sp *ServerPool) AddBackend(b *Backend) {
	sp.mux.Lock()
	defer sp.mux.Unlock()

	sp.Backends = append(sp.Backends, b)
}

func (sp *ServerPool) RemoveBackend(b *Backend) {
	sp.mux.Lock()
	defer sp.mux.Unlock()

	for idx, backend := range sp.Backends {
		if backend == b {
			sp.Backends = slices.Delete(sp.Backends, idx, idx+1)
		}
	}
}
