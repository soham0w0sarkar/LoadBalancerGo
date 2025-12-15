package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/soham0w0sarkar/LoadBalancerGo.git/internal/algorithms"
	"github.com/soham0w0sarkar/LoadBalancerGo.git/internal/backend"
	configs "github.com/soham0w0sarkar/LoadBalancerGo.git/internal/config"
	ratelimiter "github.com/soham0w0sarkar/LoadBalancerGo.git/internal/middleware/rateLimiter"
	"github.com/soham0w0sarkar/LoadBalancerGo.git/internal/proxy"
	"github.com/soham0w0sarkar/LoadBalancerGo.git/internal/server"
)

func main() {
	config, err := configs.Load("configs/config.yml")

	if err != nil {
		log.Printf("Error: %v", err)
		os.Exit(1)
	}

	err = config.Validate()
	if err != nil {
		log.Printf("Configuration validation error: %v", err)
		os.Exit(1)
	}

	serverPool := backend.NewServerPool(*config)

	balancer, _ := algorithms.SetAlgorithm(string(config.LoadBalancing.Strategy))

	var handler http.Handler = proxy.NewProxy(serverPool, balancer)

	if config.Middlewares.RateLimiter.Enabled {
		capacity := config.Middlewares.RateLimiter.Size
		refillRate := config.Middlewares.RateLimiter.Rate
		handler = ratelimiter.NewRateLimiter(capacity, refillRate, handler)
	}

	healthChecker := backend.NewHealthCheck(serverPool, config.LoadBalancing.HealthCheck)
	healthChecker.Start()
	defer healthChecker.Stop()

	srv := server.NewServer(&config.Server, handler)

	go func() {
		if err := srv.Start(int(config.Server.Port)); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	changeChan := make(chan struct{ URL []*url.URL })
	watcher := configs.NewWatcher("configs/config.yml", config)
	watcher.Start(changeChan)
	defer watcher.Stop()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	fmt.Println("Shutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Stop(ctx); err != nil {
		fmt.Printf("Server shutdown error: %v", err)
	}

	fmt.Println("Server stopped")
}
