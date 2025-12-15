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

func (sp *ServerPool) AddBackends(b []*Backend) {
	sp.mux.Lock()
	defer sp.mux.Unlock()

	sp.Backends = append(sp.Backends, b...)
}

func (sp *ServerPool) RemoveBackends(b []*Backend) {
	sp.mux.Lock()
	defer sp.mux.Unlock()

	for _, rb := range b {
		index := slices.IndexFunc(sp.Backends, func(existing *Backend) bool {
			return existing.URL.String() == rb.URL.String()
		})
		if index != -1 {
			sp.Backends = append(sp.Backends[:index], sp.Backends[index+1:]...)
		}
	}
}
