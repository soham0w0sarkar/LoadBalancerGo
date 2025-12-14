package proxy

import (
	"context"
	"fmt"
	"net/http"

	"github.com/soham0w0sarkar/LoadBalancerGo.git/internal/algorithms"
	"github.com/soham0w0sarkar/LoadBalancerGo.git/internal/backend"
	"github.com/soham0w0sarkar/LoadBalancerGo.git/internal/util"
)

type Proxy struct {
	ServerPool *backend.ServerPool
	Balancer   algorithms.Balancer
}

func NewProxy(s *backend.ServerPool, b algorithms.Balancer) *Proxy {
	return &Proxy{
		ServerPool: s,
		Balancer:   b,
	}
}

func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	backends := p.ServerPool.Backends

	attempts := util.GetAttemptsFromContext(r)
	if attempts > 3 {
		fmt.Printf("%s(%s) Max attempts reached, terminating\n", r.RemoteAddr, r.URL.Path)
		http.Error(w, "Service not available", http.StatusServiceUnavailable)
		return
	}

	backend, err := p.Balancer.Select(backends)

	if err != nil {
		http.Error(w, "Failed to select backend", http.StatusInternalServerError)
		return
	}

	ctx := context.WithValue(r.Context(), util.CtxAttemptsKey, attempts+1)
	backend.ReverseProxy.ServeHTTP(w, r.WithContext(ctx))
}
