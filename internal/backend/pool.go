package backend

import (
	"net/url"

	"github.com/soham0w0sarkar/LoadBalancerGo.git/internal/config"
)

type ServerPool struct {
	Backends []*Backend
}

func NewServerPool(cb config.Config) *ServerPool {
	var backends []*Backend

	for _, backend := range cb.Backends {
		backendUrl, _ := url.Parse(backend.Url)
		backends = append(backends, NewBackend(backendUrl, int(cb.LoadBalancing.HealthCheck.UnhealthyThreshold)))
	}

	return &ServerPool{Backends: backends}
}
