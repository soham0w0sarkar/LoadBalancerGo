package backend

import (
	"net/url"
	"slices"
	"sync"
	"time"

	"github.com/soham0w0sarkar/LoadBalancerGo.git/internal/config"
)

type ServerPool struct {
	Backends           []*Backend
	mux                sync.RWMutex
	unhealthyThreshold int
}

func NewServerPool(cb *config.Config) *ServerPool {
	var backends []*Backend

	for _, b := range cb.Backends {
		backendUrl, _ := url.Parse(b.Url)
		backends = append(backends, NewBackend(backendUrl, int(cb.LoadBalancing.HealthCheck.UnhealthyThreshold), b.Timeout))
	}

	return &ServerPool{Backends: backends, unhealthyThreshold: int(cb.LoadBalancing.HealthCheck.UnhealthyThreshold)}
}

func (sp *ServerPool) AddBackends(b []*Backend) {
	sp.mux.Lock()
	defer sp.mux.Unlock()

	sp.Backends = append(sp.Backends, b...)
}

func (sp *ServerPool) RemoveBackends(urls []string) {
	sp.mux.Lock()

	var targetsToRemove []string
	var maxTimeout time.Duration

	for _, target := range urls {
		index := slices.IndexFunc(sp.Backends, func(existing *Backend) bool {
			return existing.URL.String() == target
		})
		if index == -1 {
			continue
		}
		sp.Backends[index].SetAlive(false)

		targetsToRemove = append(targetsToRemove, sp.Backends[index].URL.String())
		if sp.Backends[index].Timeout > maxTimeout {
			maxTimeout = sp.Backends[index].Timeout
		}
	}

	if len(targetsToRemove) == 0 {
		return
	}

	sp.mux.Unlock()
	time.Sleep(maxTimeout)
	sp.mux.Lock()

	for _, t := range targetsToRemove {
		idx := slices.IndexFunc(sp.Backends, func(existing *Backend) bool {
			return existing.URL.String() == t
		})
		if idx != -1 {
			sp.Backends = append(sp.Backends[:idx], sp.Backends[idx+1:]...)
		}
	}
}

func (sp *ServerPool) GetBackends() []*Backend {
	sp.mux.RLock()
	defer sp.mux.RUnlock()
	copySlice := make([]*Backend, len(sp.Backends))
	copy(copySlice, sp.Backends)
	return copySlice
}
