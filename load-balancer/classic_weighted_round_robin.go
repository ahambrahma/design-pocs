package main

import (
	"errors"
	"fmt"
	"sort"
	"sync"
)

type ClassicWeightedRoundRobin struct {
	mtx           sync.Mutex
	requestCount  int
	serverWeights map[string]int // map of server id to weight
	keys          []string       // Since maps iterations are not definitive in nature, hence we need to convert them to keys and then sort
}

/*
**

Classic weighted round robin algorithm allocates the requests in the order of the weight i.e if we have 3 servers having weights
50, 25 and 25 respectively - then the first 50 requests would be handled by A, the next 25 by B and the rest by C and then the cycle repeats.

As clear from the above explaination, this approach can lead to unexpected spikes on a single instance.
Hence, there is another algorithm which smooths out the overall traffic.
*
*/
func NewClassicWeightedRoundRobin(serverWeights map[string]int) (*ClassicWeightedRoundRobin, error) {
	if len(serverWeights) == 0 {
		return nil, errors.New("provided empty server weights")
	}

	weightSum := 0
	for _, weight := range serverWeights {
		weightSum += weight
	}

	if weightSum != 100 {
		return nil, errors.New("sum of all weights should be 100")
	}

	keys := make([]string, 0, len(serverWeights))

	for key := range serverWeights {
		keys = append(keys, key)
	}

	sort.Strings(keys)

	return &ClassicWeightedRoundRobin{
		requestCount:  0,
		serverWeights: serverWeights,
		keys:          keys,
	}, nil
}

func (w *ClassicWeightedRoundRobin) getNext(lb *LoadBalancer) (*BackendServerProperties, error) {
	if lb == nil || len(lb.Config.Servers) == 0 {
		return nil, errors.New("backend servers cannot be empty")
	}

	// Figure out which URL to send the request to
	w.mtx.Lock()

	w.requestCount = ((w.requestCount % 100) + 1)

	weightSum := 0
	var backendServerId string
	// Map iteration is not definitive, hence commented below
	// for key, weight := range w.serverWeights {
	for _, key := range w.keys {
		weight := w.serverWeights[key]
		weightSum += weight
		// 50, 25, 25
		// In this case, first 50 requests will go to the first server,
		// then the next 25 will go to the second and the last 25 to the third server

		fmt.Printf("Weightsum: %d, request count: %d", weightSum, w.requestCount)

		if weightSum >= w.requestCount {
			backendServerId = key
			break
		}
	}

	w.mtx.Unlock()

	// Figure out the backend server properties object
	for _, server := range lb.Config.Servers {
		if server.GetID() == backendServerId {
			return server, nil
		}
	}

	return nil, errors.New("unexpected error occurred")
}
