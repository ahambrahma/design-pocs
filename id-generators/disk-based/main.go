package main

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"
)

var mu sync.Mutex
var counter int64

const counterFilePath = "counter_value.txt"

func loadCounter() {
	mu.Lock()
	defer mu.Unlock()

	data, err := os.ReadFile(counterFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			counter = 0
			os.Create(counterFilePath)
			persistCounter(counterFilePath, 0)

			log.Printf("Counter file not found. Starting counter at 0")
			return
		}
		log.Fatalf("Error reading counter file: %v. Considering counter as 0", err)
		counter = 0
		return
	}

	val, err := strconv.ParseInt(string(data), 10, 64)
	if err != nil {
		log.Fatalf("Error reading counter data: %v. Considering counter as 0", err)
		counter = 0
		return
	}

	counter = val
	log.Printf("Counter successfully loaded from disk: %d\n", counter)
}

func generateIDWithDiskPersistence() (int64, error) {
	mu.Lock()
	defer mu.Unlock()
	counter++
	value := counter
	// Persist value to disk
	if err := persistCounter(counterFilePath, value); err != nil {
		log.Fatalf("FATAL: Failed to persist counter value %d to disk: %v", value, err)
		return value, errors.New("Failed to persist counter to disk")
	}

	return value, nil
}

func persistCounter(filePath string, value int64) error {
	// Convert int64 to string
	data := []byte(strconv.FormatInt(value, 10))

	// Write the data back to the file
	return os.WriteFile(filePath, data, 0644)
}

func handler(w http.ResponseWriter, r *http.Request) {

	if r.URL.Path != "/" {
		// fmt.Printf("Base handler skipping non-root path: %s\n", r.URL.Path)
		http.NotFound(w, r)
		return
	}

	fmt.Println("Entered base handler")
	newID, err := generateIDWithDiskPersistence()
	if err == nil {
		fmt.Fprintf(w, "Generated ID: %d\n", newID)
		fmt.Printf("Generated ID: %d\n", newID)
	} else {
		fmt.Println("Error while generating ID: ", err)
	}
}

func periodicHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Entered periodic handler")

	newID, err := generateIDWithPeriodicDiskPersistence(50)
	if err == nil {
		fmt.Fprintf(w, "Generated ID: %d\n", newID)
		fmt.Printf("Generated ID: %d\n", newID)
	} else {
		fmt.Println("Error while generating ID: ", err)
	}
}

func generateIDWithPeriodicDiskPersistence(iterations int64) (int64, error) {
	mu.Lock()
	defer mu.Unlock()
	counter++
	value := counter
	if value%iterations == 0 {
		fmt.Println("Entered block")
		if err := persistCounter(counterFilePath, value); err != nil {
			log.Fatalf("FATAL: Failed to persist counter value %d to disk: %v", value, err)
			return value, errors.New("Failed to persist counter to disk")
		}
	}

	return value, nil
}

// func generateIDWithPeriodicDiskPersistenceAndBuffer() int64 {

// }

func faviconHandler(w http.ResponseWriter, r *http.Request) {
	http.NotFound(w, r)
	fmt.Println("Request came to favicon handler")
}

func main() {
	mux := http.NewServeMux()

	loadCounter()

	mux.HandleFunc("/", handler)
	mux.HandleFunc("/periodic", periodicHandler)
	mux.HandleFunc("/favicon.ico", faviconHandler)

	log.Println("Starting server on :8080...")
	log.Fatal(http.ListenAndServe(":8080", mux))
}
