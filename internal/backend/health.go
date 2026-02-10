package backend

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/soham0w0sarkar/LoadBalancerGo.git/internal/config"
)

type HealthCheck struct {
	ServerPool *ServerPool
	config     config.HealthCheckConfig
	stopChan   chan struct{}
	client     *http.Client
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
}

func NewHealthCheck(pool *ServerPool, cfg config.HealthCheckConfig) *HealthCheck {
	ctx, cancel := context.WithCancel(context.Background())
	return &HealthCheck{
		ServerPool: pool,
		config:     cfg,
		client: &http.Client{
			Timeout: cfg.Timeout,
		},
		ctx:    ctx,
		cancel: cancel,
	}
}

func (hc *HealthCheck) Start() {
	hc.stopChan = make(chan struct{})
	go hc.run()
}

func (hc *HealthCheck) run() {
	ticker := time.NewTicker(hc.config.Interval)
	defer ticker.Stop()

	hc.checkAll()

	for {
		select {
		case <-ticker.C:
			hc.checkAll()

		case <-hc.stopChan:
			fmt.Println("Health checker stopped")
			return
		}
	}
}

func (hc *HealthCheck) checkAll() {
	// Fix race condition: Use GetBackends() which returns a safe copy
	backends := hc.ServerPool.GetBackends()

	for _, backend := range backends {
		// Track goroutine to prevent leaks
		hc.wg.Add(1)
		go hc.check(backend)
	}
}

func (hc *HealthCheck) check(backend *Backend) {
	defer hc.wg.Done()

	// Fix goroutine leak: Use context that can be cancelled
	ctx, cancel := context.WithTimeout(hc.ctx, hc.config.Timeout)
	defer cancel()

	// Check if context was cancelled before starting
	select {
	case <-hc.ctx.Done():
		return
	default:
	}

	healthURL := backend.URL.String() + "/health"

	req, err := http.NewRequestWithContext(ctx, "GET", healthURL, nil)
	if err != nil {
		backend.UpdateFailureCount(int(hc.config.UnhealthyThreshold))
		return
	}

	resp, err := hc.client.Do(req)
	if err != nil {
		// Don't update failure count if context was cancelled (backend removed)
		if ctx.Err() == context.Canceled {
			return
		}
		backend.UpdateFailureCount(int(hc.config.UnhealthyThreshold))
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		backend.UpdateSuccessCount(int(hc.config.HealthyThreshold))
	} else {
		backend.UpdateFailureCount(int(hc.config.UnhealthyThreshold))
	}
}

func (hc *HealthCheck) Stop() {
	// Cancel all health check contexts to stop running goroutines
	if hc.cancel != nil {
		hc.cancel()
	}

	// Stop the main health check loop
	if hc.stopChan != nil {
		close(hc.stopChan)
	}

	// Wait for all health check goroutines to finish
	hc.wg.Wait()
}
