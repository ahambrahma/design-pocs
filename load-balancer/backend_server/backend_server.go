package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
)

func main() {
	port := os.Args[1]

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		response := fmt.Sprintf("Response from server on port %s\n", port)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(response))
	})

	fmt.Printf("Backend server starting on port %s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
