package main

import (
	"errors"
	"math"
)

type LeastConnectionsAlgorithm struct {
}

func (l *LeastConnectionsAlgorithm) getNext(lb *LoadBalancer) (*BackendServerProperties, error) {
	if lb == nil || len(lb.Config.Servers) == 0 {
		return nil, errors.New("backend servers cannot be empty")
	}

	min := int32(math.MaxInt32)
	var result *BackendServerProperties
	for _, server := range lb.Config.Servers {
		activeConnections := server.GetActiveConnections()
		if activeConnections < min {
			min = activeConnections
			result = server
		}
	}

	return result, nil
}
