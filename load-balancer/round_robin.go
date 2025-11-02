package main

import (
	"errors"
	"sync"
)

type RoundRobinLoadBalancingAlgorithm struct {
	mu      sync.Mutex
	counter int
}

func (r *RoundRobinLoadBalancingAlgorithm) getNext(lb *LoadBalancer) (*BackendServerProperties, error) {

	if lb == nil || len(lb.Config.Servers) == 0 {
		return nil, errors.New("backend servers cannot be empty")
	}

	servers := lb.Config.Servers

	r.mu.Lock()
	defer r.mu.Unlock()

	result := servers[r.counter%len(servers)]
	r.counter = (r.counter + 1) % len(servers) // Simplified increment logic
	return result, nil
}
