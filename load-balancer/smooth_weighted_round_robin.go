package main

import (
	"errors"
	"math"
	"sort"
	"sync"
)

type SmoothWeightedRoundRobin struct {
	mtx            sync.Mutex
	requestCount   int
	serverWeights  map[string]int // map of server id to weight
	currentWeights map[string]int // map of server id to current weight
	keys           []string
}

/***

Assume we have 3 servers i.e A, B and C with fixed weights denoted by FW with values 50, 25 and 25 each.
The calculations for all requests are handled as follows:
1. Initially current weights denoted by CW will be as: CW(A) = CW(B) = CW(C) = 0
2. Add the corresponding fixed weights to the current weights i.e
	CW(A) = CW(A) + FW(A), CW(B) = CW(B) + FW(B), CW(C) = CW(C) + FW(C)
	In our case, initially these values would become 50, 25, 25 respectively
3. Figure out which current value is the max amongst them i.e CW(A) = 50 in this case.
4. Reduce sum of all fixed weights from the current weight of the max value selected in Step 3 i.e CW(A) = 50 - 100 = -50.

Second iteration:
CW(A) = -50 + 50 = 0, CW(B) = 25 + 25 = 50, CW(C) = 25 + 25 = 50

In this case, B will get the request.
Hence, CW(B) = 50 - 100 = -50

Third iteration:
CW(A) = 0 + 50 = 50, CW(B) = -50 + 25 = -25, CW(C) = 50 + 25 = 75

In this case, C will get the request.
Hence, CW(C) = 75-100 = -25

Fourth iteration:
CW(A) = 50 + 50 = 100, CW(B) = -25 + 25 = 0, CW(C) = -25 + 25 = 0

In this case, A will get the request.
Hence, CW(A) = 100 - 0 = 0

Fifth iteration:
CW(A) = 0 + 50 = 50, CW(B) = 0 + 25 = 25, CW(C) = 0 + 25 = 25

In this case, A will get the request.



Considering this algorithm is able to direct the traffic in the order of weights while at the same time prevent overloading the
server, it is more preferred in production systems by the load balancers

**/

func NewSmoothWeightRoundRobin(serverWeights map[string]int) (*SmoothWeightedRoundRobin, error) {
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

	currentWeights := make(map[string]int, len(serverWeights))
	for key := range serverWeights {
		currentWeights[key] = 0
	}

	keys := make([]string, 0, len(serverWeights))

	for key := range serverWeights {
		keys = append(keys, key)
	}

	sort.Strings(keys)

	return &SmoothWeightedRoundRobin{
		requestCount:   0,
		serverWeights:  serverWeights,
		currentWeights: currentWeights,
		keys:           keys,
	}, nil
}

func (s *SmoothWeightedRoundRobin) getNext(lb *LoadBalancer) (*BackendServerProperties, error) {
	if lb == nil || len(lb.Config.Servers) == 0 {
		return nil, errors.New("backend servers cannot be empty")
	}

	s.mtx.Lock()
	defer s.mtx.Unlock()

	max := math.MinInt
	var maxKey string
	for _, key := range s.keys {
		fixedWeight := s.serverWeights[key]
		s.currentWeights[key] += fixedWeight

		if s.currentWeights[key] > max {
			max = s.currentWeights[key]
			maxKey = key
		}
	}

	var result *BackendServerProperties
	for _, server := range lb.Config.Servers {
		if server.GetID() == maxKey {
			result = server
			break
		}
	}

	if result == nil {
		return nil, errors.New("unexpected error")
	}

	s.currentWeights[maxKey] -= 100 // Since sum of all weights is supposed to be 100
	return result, nil
}
