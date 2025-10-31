package main

import (
	"database/sql"
	"encoding/hex"
	"log"
	"math/rand"
	"time"

	_ "github.com/go-sql-driver/mysql" // MySQL driver import
	"github.com/google/uuid"
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

func insertEntriesIntoUUIDBasedTable() {
	db := dbPool

	start := time.Now()
	for range 1000000 {
		id := uuid.New()
		hexString := hex.EncodeToString(id[:])

		age := rand.Intn(100)
		_, err := db.Exec("INSERT INTO uuid_demo(id, age) VALUES(?, ?);", hexString, age)
		if err != nil {
			log.Fatalf("Some error occurred while trying to insert values into UUID table", err)
		}
	}
	log.Printf("Execution completed within %v seconds\n", time.Since(start).Seconds())
}

func insertEntriesIntoIntegerTable() {
	db := dbPool

	start := time.Now()
	for i := range 1000000 {
		age := rand.Intn(100)
		_, err := db.Exec("INSERT INTO int_demo(id, age) VALUES(?, ?);", i, age)
		if err != nil {
			log.Fatalf("Some error occurred while trying to insert values into UUID table")
		}
	}
	log.Printf("Execution completed within %v seconds\n", time.Since(start).Seconds())
}

func main() {
	insertEntriesIntoUUIDBasedTable() // took 229.242056458 seconds
	//insertEntriesIntoIntegerTable() // took 214.075277625 seconds
}
