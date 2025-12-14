package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/soham0w0sarkar/LoadBalancerGo.git/internal/algorithms"
	"github.com/soham0w0sarkar/LoadBalancerGo.git/internal/backend"
	"github.com/soham0w0sarkar/LoadBalancerGo.git/internal/config"
	"github.com/soham0w0sarkar/LoadBalancerGo.git/internal/proxy"
	"github.com/soham0w0sarkar/LoadBalancerGo.git/internal/server"
)

func main() {
	config, err := config.Load("configs/config.yml")

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

	proxy := proxy.NewProxy(serverPool, balancer)

	healthChecker := backend.NewHealthCheck(serverPool, config.LoadBalancing.HealthCheck)
	healthChecker.Start()
	defer healthChecker.Stop()

	srv := server.NewServer(&config.Server, proxy)

	go func() {
		if err := srv.Start(int(config.Server.Port)); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

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
