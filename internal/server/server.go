package server

import (
	"context"
	"fmt"
	"net/http"

	"github.com/soham0w0sarkar/LoadBalancerGo.git/internal/config"
)

type Handler interface {
	ServeHTTP(http.ResponseWriter, *http.Request)
}

type Server struct {
	httpServer *http.Server
}

func NewServer(cs *config.ServerConfig, handler Handler) *Server {
	return &Server{
		httpServer: &http.Server{
			Addr:         fmt.Sprintf(":%d", cs.Port),
			Handler:      handler,
			ReadTimeout:  cs.ReadTimeout,
			WriteTimeout: cs.WriteTimeout,
		},
	}
}

func (s *Server) Start(port int) error {
	fmt.Printf("LoadBalancer on port: %d\n", port)

	return s.httpServer.ListenAndServe()
}

func (s *Server) Stop(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}
