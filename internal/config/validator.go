package config

import (
	"fmt"
	"net/url"
)

func (c *Config) Validate() error {
	if c.Server.Port == 0 {
		return fmt.Errorf("port cannot be 0")
	}
	if c.Server.ReadTimeout <= 0 {
		return fmt.Errorf("read timeout must be positive")
	}
	if c.Server.WriteTimeout <= 0 {
		return fmt.Errorf("write timeout must be positive")
	}

	if len(c.Backends) == 0 {
		return fmt.Errorf("at least one backend must be specified")
	}
	for i, backend := range c.Backends {
		_, err := url.Parse(backend.Url)
		if err != nil {
			return fmt.Errorf("backend[%d]: invalid URL: %w", i, err)
		}
		if backend.Timeout <= 0 {
			return fmt.Errorf("backend timeout must be positive")
		}
	}

	switch c.LoadBalancing.Strategy {
	case RoundRobin, Weighted, LeastConnection, ConsistentHash:
	default:
		return fmt.Errorf("unrecognized load balancing strategy: %s", c.LoadBalancing.Strategy)
	}

	hc := c.LoadBalancing.HealthCheck
	if hc.Interval <= 0 {
		return fmt.Errorf("health check interval must be positive")
	}
	if hc.Timeout <= 0 {
		return fmt.Errorf("health check timeout must be positive")
	}
	if hc.Timeout >= hc.Interval {
		return fmt.Errorf("health check timeout must be less than interval")
	}
	if hc.UnhealthyThreshold == 0 {
		return fmt.Errorf("unhealthy threshold must be positive")
	}
	if hc.HealthyThreshold == 0 {
		return fmt.Errorf("healthy threshold must be positive")
	}

	rl := c.Middlewares.RateLimiter
	if rl.Enabled {
		if rl.Rate == 0 {
			return fmt.Errorf("rate limiter refill rate must be positive when enabled")
		}
	}

	return nil
}
