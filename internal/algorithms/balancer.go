package algorithms

import (
	"fmt"

	"github.com/soham0w0sarkar/LoadBalancerGo.git/internal/backend"
)

type Balancer interface {
	Select([]*backend.Backend) (*backend.Backend, error)
}

func SetAlgorithm(strategy string) (Balancer, error) {
	switch strategy {
	case "round_robin":
		return &RoundRobin{}, nil
	}

	return nil, fmt.Errorf("unkown strategy: %s", strategy)
}
