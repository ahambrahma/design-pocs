package main

import (
	"database/sql"
	"encoding/json"
	"errors" // Necessary for checking sql.ErrNoRows more robustly, even if not strictly used here
	"fmt"
	"log"
	"net/http"

	_ "github.com/go-sql-driver/mysql" // MySQL driver import
)

var dbPool *sql.DB

func init() {
	// 1. Initialize Global Connection Pool
	var err error
	dbPool, err = sql.Open("mysql", "root@tcp(127.0.0.1:3306)/id_generator?charset=utf8")
	if err != nil {
		log.Fatalf("FATAL: Error establishing connection pool: %v", err)
	}

	dbPool.SetMaxOpenConns(20)
	dbPool.SetMaxIdleConns(10)

	if err = dbPool.Ping(); err != nil {
		log.Fatalf("FATAL: Failed to ping database: %v", err)
	}

	log.Println("Database connection pool initialized successfully.")
}

// Corrected JSON tags using backticks (` `)
type IDRange struct {
	Min int64 `json:"min"`
	Max int64 `json:"max"`
}

const ServiceName = "service_name"
const BatchSize = int64(100)

func homeHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, "ID Generator Service is running.")
}

func writeJSONResponse(w http.ResponseWriter, result IDRange) {
	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(result); err != nil {
		log.Printf("Error writing JSON response: %v", err)
		http.Error(w, "Error writing response", http.StatusInternalServerError)
	}
}

func generateHandler(w http.ResponseWriter, r *http.Request) {
	queryParams := r.URL.Query()
	if queryParams == nil || queryParams.Get(ServiceName) == "" {
		fmt.Println("Invalid query params found in request")
		http.Error(w, "Missing service_name parameter", http.StatusBadRequest)
		return
	}

	svcName := queryParams.Get(ServiceName)
	fmt.Println("Service name:", svcName)

	db := dbPool

	txn, err := db.Begin()
	if err != nil {
		log.Printf("Error happened while trying to start transaction: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// CRITICAL FIX: Ensure transaction is rolled back on any exit,
	// unless explicitly committed below. This prevents deadlocks/leaks.
	defer txn.Rollback()

	// Optional: Recovery from panic, still needed for robustness
	// The deferred Rollback above handles the main cleanup.
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Entered recovery handler")
			panic(r)
		}
	}()

	var svcCounter int64

	// 1. Query the row with FOR UPDATE lock
	row := txn.QueryRow("SELECT value FROM ids WHERE service_name = ? FOR UPDATE", svcName)
	err = row.Scan(&svcCounter)

	// --- Error Handling Block ---
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) { // Using errors.Is for robust check
			// Case 1: Row not found. Perform initial INSERT.
			fmt.Printf("Service '%s' not found. Inserting initial batch.\n", svcName)

			initialCounterValue := BatchSize // The DB holds the max ID generated

			_, err1 := txn.Exec("INSERT INTO ids(service_name, value) VALUES(?, ?)", svcName, initialCounterValue)
			if err1 != nil {
				log.Printf("Error inserting initial value: %v", err1)
				w.WriteHeader(http.StatusInternalServerError)
				return // defer Rollback() runs here
			}

			// Commit the initial insertion
			if err = txn.Commit(); err != nil {
				log.Printf("Error committing initial transaction: %v", err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			// NOTE: Since Commit succeeded, the deferred Rollback will gracefully fail.

			result := IDRange{
				Min: 1,
				Max: BatchSize,
			}

			w.WriteHeader(http.StatusOK)
			writeJSONResponse(w, result)
			return

		} else {
			// Case 2: Other database error (e.g., connection lost).
			log.Printf("Database query error for service '%s': %v", svcName, err)
			w.WriteHeader(http.StatusInternalServerError)
			return // defer Rollback() runs here
		}
	}
	// --- End Error Handling Block ---

	// Row found: Proceed with update.
	lowerValue := svcCounter + 1
	upperValue := svcCounter + BatchSize

	_, err = txn.Exec("UPDATE ids SET value = ? WHERE service_name = ?", upperValue, svcName)
	if err != nil {
		log.Printf("Error updating counter: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return // defer Rollback() runs here
	}

	// Explicitly commit the transaction
	if err = txn.Commit(); err != nil {
		log.Printf("Error committing transaction: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	// NOTE: Commit succeeded. The deferred Rollback is now harmless.

	result := IDRange{
		Min: lowerValue,
		Max: upperValue,
	}

	w.WriteHeader(http.StatusOK)
	writeJSONResponse(w, result)
}

func main() {
	mux := http.NewServeMux()

	mux.HandleFunc("/", homeHandler)
	mux.HandleFunc("/generate", generateHandler)

	log.Println("Starting ID Generator server on :8080...")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatalf("Error occurred while trying to bring up the http server: %v", err)
	}
}
