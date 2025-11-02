package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

type BackendServerProperties struct {
	IP              string
	BaseUrl         string
	APIPort         uint
	HealthCheckPort uint
}

type LoadBalancingConfig struct {
	Servers   []*BackendServerProperties
	Algorithm LoadBalancingAlgorithm // TODO: Replace with a factory pattern
}

type LoadBalancer struct {
	Config                  LoadBalancingConfig
	HealthCheckFrequencySec uint
}

func NewLoadBalancer(config LoadBalancingConfig, healthCheckFrequency uint) *LoadBalancer {
	lb := &LoadBalancer{
		Config:                  config,
		HealthCheckFrequencySec: healthCheckFrequency,
	}
	return lb
}

func (lb *LoadBalancer) requestHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("Request received: %s %s from %s\n", r.Method, r.URL.Path, r.RemoteAddr)

	// Handle logic for request handling
	nextServer, err := lb.Config.Algorithm.getNext(lb)
	if err != nil {
		fmt.Println("Error occurred while getting the next server", err)
		w.WriteHeader(500)
		w.Write([]byte("Error while processing the request"))
		return
	}

	// Establish connection with the server and forward the request
	lb.forwardRequest(w, r, nextServer)
}

func (lb *LoadBalancer) forwardRequest(w http.ResponseWriter, r *http.Request, server *BackendServerProperties) {
	// 1. Construct the backend URL
	backendURL := fmt.Sprintf("%s:%d%s", server.BaseUrl, server.APIPort, r.URL.Path)

	// Add query parameters if they exist
	if r.URL.RawQuery != "" {
		backendURL += "?" + r.URL.RawQuery
	}

	fmt.Printf("Forwarding to: %s\n", backendURL)

	// 2. Create a new request to the backend
	backendReq, err := http.NewRequest(r.Method, backendURL, r.Body)
	if err != nil {
		fmt.Println("Error creating backend request:", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Failed to create backend request"))
		return
	}

	// 3. Copy headers from original request
	for key, values := range r.Header {
		for _, value := range values {
			backendReq.Header.Add(key, value)
		}
	}

	// 4. Add X-Forwarded headers (important for tracking original request)
	backendReq.Header.Set("X-Forwarded-For", r.RemoteAddr)
	backendReq.Header.Set("X-Forwarded-Host", r.Host)
	if r.TLS != nil {
		backendReq.Header.Set("X-Forwarded-Proto", "https")
	} else {
		backendReq.Header.Set("X-Forwarded-Proto", "http")
	}

	// 5. Make the HTTP call to backend
	client := &http.Client{
		Timeout: 30 * time.Second, // Set a reasonable timeout
	}

	backendResp, err := client.Do(backendReq)
	if err != nil {
		fmt.Println("Error calling backend:", err)
		w.WriteHeader(http.StatusBadGateway)
		w.Write([]byte("Backend server unavailable"))
		return
	}
	defer backendResp.Body.Close()

	// 6. Copy response headers from backend to client
	for key, values := range backendResp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	// 7. Copy status code
	w.WriteHeader(backendResp.StatusCode)

	// 8. Copy response body from backend to client
	_, err = io.Copy(w, backendResp.Body)
	if err != nil {
		fmt.Println("Error copying response body:", err)
	}
}

func healthCheck(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(200)
	w.Write([]byte("Hello"))
}

func (lb *LoadBalancer) startServer() {

	mux := http.NewServeMux()
	mux.HandleFunc("/health", healthCheck)
	mux.HandleFunc("/", lb.requestHandler)

	// Start the HTTP server in a goroutine
	go func() {
		port := ":8080"
		fmt.Printf("Load balancer starting on port %s\n", port)
		if err := http.ListenAndServe(port, mux); err != nil {
			log.Fatal("Failed to start load balancer:", err)
		}
	}()
}

type LoadBalancingAlgorithm interface {
	getNext(lb *LoadBalancer) (*BackendServerProperties, error)
}

func main() {
	// Define backend servers
	servers := []*BackendServerProperties{
		{
			IP:              "127.0.0.1",
			BaseUrl:         "http://localhost",
			APIPort:         9001,
			HealthCheckPort: 9001,
		},
		{
			IP:              "127.0.0.1",
			BaseUrl:         "http://localhost",
			APIPort:         9002,
			HealthCheckPort: 9002,
		},
		{
			IP:              "127.0.0.1",
			BaseUrl:         "http://localhost",
			APIPort:         9003,
			HealthCheckPort: 9003,
		},
	}

	// Create round-robin algorithm instance
	algorithm := &RoundRobinLoadBalancingAlgorithm{
		counter: 0,
	}

	// Configure the load balancer
	config := LoadBalancingConfig{
		Servers:   servers,
		Algorithm: algorithm,
	}

	// Initialize and start the load balancer
	lb := NewLoadBalancer(config, 10)
	lb.startServer()

	fmt.Println("Load balancer initialized with", len(servers), "backend servers")
	fmt.Println("Press Ctrl+C to stop")

	// Keep the main goroutine running
	select {}
}
