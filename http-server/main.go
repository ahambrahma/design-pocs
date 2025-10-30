package main

import (
	"fmt"
	"log"
	"net/http"
	"sync"
)

var count int
var mu sync.Mutex

func handler(w http.ResponseWriter, r *http.Request) {
	mu.Lock()
	currentCount := count
	count += 1
	mu.Unlock()

	fmt.Printf("Received request for path: %s, count: %d\n", r.URL.Path, currentCount)
	if r.URL.Path != "/" {
		fmt.Println("Non home path")
		http.NotFound(w, r)
		return
	}

	fmt.Fprintf(w, "Hello from Golang server\n")
}

func faviconHandler(w http.ResponseWriter, r *http.Request) {
	http.NotFound(w, r)
	fmt.Println("Received request for favicon")
}

func main() {
	mux := http.NewServeMux()

	mux.HandleFunc("/", handler)
	mux.HandleFunc("/favicon.ico", faviconHandler) // Explicitly handle the favicon

	addr := ":8443"
	certFile := "./certs/server.crt" // Path to your self-signed certificate - removed for safety reasons
	keyFile := "./certs/server.key"  // Path to your self-signed private key - removed for safety reasons

	log.Println("Starting https go server on port 8081")
	if err := http.ListenAndServeTLS(addr, certFile, keyFile, mux); err != nil {
		log.Fatal("ListenAndServe error: ", err)
	}
}
