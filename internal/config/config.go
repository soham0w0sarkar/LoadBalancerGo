package config

import (
	"time"
)

type ServerConfig struct {
	Port         uint16        `yaml:"port"`
	ReadTimeout  time.Duration `yaml:"read_timeout"`
	WriteTimeout time.Duration `yaml:"write_timeout"`
}

type BackendConfig struct {
	Url     string        `yaml:"url"`
	Timeout time.Duration `yaml:"timeout"`
}

type HealthCheckConfig struct {
	Interval           time.Duration `yaml:"interval"`
	Timeout            time.Duration `yaml:"timeout"`
	UnhealthyThreshold uint8         `yaml:"unhealthy_threshold"`
	HealthyThreshold   uint8         `yaml:"healthy_threshold"`
}

type Strategy string

const (
	RoundRobin      Strategy = "round_robin"
	Weighted        Strategy = "weighted"
	LeastConnection Strategy = "least_conn"
	ConsistentHash  Strategy = "consistent_hash"
)

type LoadBalancingConfig struct {
	Strategy    Strategy          `yaml:"strategy"`
	HealthCheck HealthCheckConfig `yaml:"health_check"`
}

type RateLimiterConfig struct {
	Enabled bool    `yaml:"enabled"`
	Rate    float64 `yaml:"rate"`
	Size    uint    `yaml:"size"`
}

type StickySessionConfig struct {
	Enabled    bool          `yaml:"enabled"`
	CookieName string        `yaml:"cookie_name"`
	TTL        time.Duration `yaml:"ttl"`
}

type MiddlewareConfig struct {
	RateLimiter   RateLimiterConfig   `yaml:"rate_limiter"`
	StickySession StickySessionConfig `yaml:"sticky_session"`
}

type Config struct {
	Server        ServerConfig        `yaml:"server"`
	Backends      []BackendConfig     `yaml:"backends"`
	LoadBalancing LoadBalancingConfig `yaml:"load_balancing"`
	Middlewares   MiddlewareConfig    `yaml:"middlewares"`
}
