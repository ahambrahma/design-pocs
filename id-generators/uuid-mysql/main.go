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

	/***

	int_demo - 25 MB
	uuid_demo - 85 MB



	**/
}

/***
SELECT
    TABLE_NAME AS `Table`,
    ROUND((DATA_LENGTH + INDEX_LENGTH) / 1024 / 1024, 2) AS `Size (MB)`,
    ROUND(DATA_LENGTH / 1024 / 1024, 2) AS `Data Size (MB)`,
    ROUND(INDEX_LENGTH / 1024 / 1024, 2) AS `Index Size (MB)`
FROM information_schema.TABLES
WHERE TABLE_SCHEMA = 'id_generator'
ORDER BY (DATA_LENGTH + INDEX_LENGTH) DESC;

SELECT
    TABLE_NAME AS `Table`,
    ROUND((DATA_LENGTH + INDEX_LENGTH) / 1024 / 1024, 2) AS `Size (MB)`,
    ROUND(INDEX_LENGTH / 1024 / 1024, 2) AS `Index Size (MB)`
FROM information_schema.TABLES
WHERE TABLE_SCHEMA = 'id_generator'
AND TABLE_NAME = 'uuid_demo';

SELECT
    database_name,
    table_name,
    index_name,
    ROUND(stat_value * @@innodb_page_size / 1024, 2) AS `Size (KB)`
FROM mysql.innodb_index_stats
WHERE stat_name = 'size'
AND database_name = 'id_generator'
AND table_name = 'int_demo'
ORDER BY stat_value DESC;

SELECT
    database_name,
    table_name,
    index_name,
    ROUND(stat_value * @@innodb_page_size / 1024, 2) AS `Size (KB)`
FROM mysql.innodb_index_stats
WHERE stat_name = 'size'
AND database_name = 'id_generator'
AND table_name = 'uuid_demo'
ORDER BY stat_value DESC;

-- ---------------+------------+------------+-----------+
| database_name | table_name | index_name | Size (KB) |
+---------------+------------+------------+-----------+
| id_generator  | uuid_demo  | PRIMARY    |  86800.00 |
| id_generator  | uuid_demo  | idx_age    |  48784.00 |
-- ---------------+------------+------------+-----------+




**/
