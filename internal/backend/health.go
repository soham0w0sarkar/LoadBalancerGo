package backend

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/soham0w0sarkar/LoadBalancerGo.git/internal/config"
)

type HealthCheck struct {
	ServerPool *ServerPool
	config     config.HealthCheckConfig
	stopChan   chan struct{}
	client     *http.Client
}

func NewHealthCheck(pool *ServerPool, cfg config.HealthCheckConfig) *HealthCheck {
	return &HealthCheck{
		ServerPool: pool,
		config:     cfg,
		client: &http.Client{
			Timeout: cfg.Timeout,
		},
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
	backends := hc.ServerPool.Backends

	for _, backend := range backends {
		go hc.check(backend)
	}
}

func (hc *HealthCheck) check(backend *Backend) {
	healthURL := backend.URL.String() + "/health"

	ctx, cancel := context.WithTimeout(context.Background(), hc.config.Timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", healthURL, nil)
	if err != nil {
		backend.UpdateFailureCount(int(hc.config.UnhealthyThreshold))
		return
	}

	resp, err := hc.client.Do(req)
	if err != nil {
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
	if hc.stopChan != nil {
		close(hc.stopChan)
	}
}
