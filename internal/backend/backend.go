package backend

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"
	"time"

	"github.com/soham0w0sarkar/LoadBalancerGo.git/internal/util"
)

type Backend struct {
	URL          *url.URL
	Alive        bool
	mux          sync.RWMutex
	ReverseProxy *httputil.ReverseProxy
	Timeout      time.Duration
	SuccessCount uint8
	FailureCount uint8
}

func NewBackend(url *url.URL, failureThreshold int, timeout time.Duration) *Backend {
	backend := &Backend{
		URL:     url,
		Alive:   false,
		Timeout: timeout,
	}

	proxy := httputil.NewSingleHostReverseProxy(url)

	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		fmt.Printf("[%s] %s\n", url, err.Error())

		retries := util.GetRetryFromContext(r)
		if retries < failureThreshold {
			time.Sleep(10 * time.Millisecond)
			ctx := context.WithValue(r.Context(), util.CtxRetryKey, retries+1)
			backend.UpdateFailureCount(failureThreshold)
			proxy.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		http.Error(w, "Service Unavailable", http.StatusServiceUnavailable)
	}

	backend.ReverseProxy = proxy
	return backend
}

func (b *Backend) IsAlive() (alive bool) {
	b.mux.RLock()
	alive = b.Alive
	b.mux.RUnlock()
	return
}

func (b *Backend) SetAlive(alive bool) {
	b.mux.Lock()
	b.Alive = alive
	b.mux.Unlock()
}

func (b *Backend) UpdateSuccessCount(threshold int) {
	b.mux.Lock()
	b.SuccessCount++
	b.FailureCount = 0
	b.mux.Unlock()

	if b.SuccessCount >= uint8(threshold) {
		b.SetAlive(true)
		b.ResetCounts()
	}
}

func (b *Backend) UpdateFailureCount(threshold int) {
	b.mux.Lock()
	b.FailureCount++
	b.SuccessCount = 0
	b.mux.Unlock()

	if b.FailureCount >= uint8(threshold) {
		b.SetAlive(false)
		b.ResetCounts()
	}
}

func (b *Backend) ResetCounts() {
	b.mux.Lock()
	b.SuccessCount = 0
	b.FailureCount = 0
	b.mux.Unlock()
}
