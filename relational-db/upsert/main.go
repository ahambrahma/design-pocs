package main

import (
	"database/sql"
	"fmt"
	"sync"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

func newConn(dbName string) *sql.DB {
	db, err := sql.Open("mysql", fmt.Sprintf("root@tcp(127.0.0.1:3306)/%s?charset=utf8", dbName))
	if err != nil {
		panic(err)
	}
	return db
}

func insertIntoTable(db *sql.DB) {
	query := `INSERT INTO upsert_testing (name, age) VALUES (?, ?);`
	wg := sync.WaitGroup{}
	wg.Add(100)
	for i := 1; i <= 100; i++ {
		go func(iteration int) {
			defer wg.Done()
			for i := 1; i <= 10000; i++ {
				_, err := db.Exec(query, fmt.Sprintf("name_%d_%d", iteration, i), i)
				if err != nil {
					fmt.Printf("Error occurred while trying to write to table: %v", err)
					return
				}
			}
		}(i)
	}
	wg.Wait()

}

func benchmarkOnDuplicateKeyUpdate(db *sql.DB) {
	query := `INSERT INTO upsert_testing (name, age) VALUES (?, ?)
			  ON DUPLICATE KEY UPDATE age = VALUES(age);`
	wg := sync.WaitGroup{}
	wg.Add(100)
	for i := 1; i <= 100; i++ {
		go func(iteration int) {
			defer wg.Done()
			for i := 1; i <= 10000; i++ {
				_, err := db.Exec(query, fmt.Sprintf("name_%d_%d", iteration, i), i+1)
				if err != nil {
					fmt.Printf("Error occurred while trying to upsert to table: %v", err)
					return
				}
			}
		}(i)
	}
	wg.Wait()
}

func benchmarkReplaceInto(db *sql.DB) {
	query := `REPLACE INTO upsert_testing (name, age) VALUES (?, ?);`
	wg := sync.WaitGroup{}
	wg.Add(100)
	for i := 1; i <= 100; i++ {
		go func(iteration int) {
			defer wg.Done()
			for i := 1; i <= 10000; i++ {
				// Note: REPLACE INTO will increment the ID or create gaps
				_, err := db.Exec(query, fmt.Sprintf("name_%d_%d", iteration, i), i+1)
				if err != nil {
					fmt.Printf("Error in REPLACE INTO: %v\n", err)
					return
				}
			}
		}(i)
	}
	wg.Wait()
}

func main() {
	conn := newConn("upsert")
	insertIntoTable(conn)
	start := time.Now()

	// benchmarkOnDuplicateKeyUpdate(conn)
	// fmt.Printf("ON DUPLICATE KEY UPDATE took: %v\n", time.Since(start))

	// ON DUPLICATE KEY UPDATE took: 2m10.506238709s

	// 4. Benchmark REPLACE INTO
	// Note: This changes IDs, so we reset if we want a fair comparison of "update" speed
	// But for raw throughput testing on the same dataset:
	// start = time.Now()
	benchmarkReplaceInto(conn)
	fmt.Printf("REPLACE INTO took: %v\n", time.Since(start))
	// Takes way more longer than ON DUPLICATE KEY UPDATE due to delete + insert
}
