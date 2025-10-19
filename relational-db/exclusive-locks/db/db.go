package db

import (
	"database/sql"
	"fmt"
	"sync"

	_ "github.com/go-sql-driver/mysql"
)

type conn struct {
	DB *sql.DB
}

type cpool struct {
	mu      sync.Mutex
	channel chan interface{}
	conns   []*conn
	maxConn int
}

var CommonPool *cpool

func newPool(maxConn int, dbName string) *cpool {
	pool := &cpool{
		mu:      sync.Mutex{},
		conns:   make([]*conn, 0, maxConn),
		maxConn: maxConn,
		channel: make(chan interface{}, maxConn),
	}

	for i := 0; i < maxConn; i++ {
		pool.conns = append(pool.conns, &conn{newConn(dbName)})
		pool.channel <- nil
	}

	return pool
}

func (pool *cpool) Put(c *conn) {
	pool.mu.Lock()
	pool.conns = append(pool.conns, c)
	pool.mu.Unlock()

	pool.channel <- nil
}

func (pool *cpool) Get() *conn {
	<-pool.channel

	pool.mu.Lock()
	// LIFO Pop: Get the last element (O(1))
	lastIndex := len(pool.conns) - 1
	conn := pool.conns[lastIndex]
	pool.conns = pool.conns[:lastIndex]
	pool.mu.Unlock()

	return conn
}

func newConn(dbName string) *sql.DB {
	db, err := sql.Open("mysql", fmt.Sprintf("root@tcp(127.0.0.1:3306)/%s?charset=utf8", dbName))
	if err != nil {
		panic(err)
	}
	return db
}

func Init() {
	CommonPool = newPool(101, "airline_booking")
	resetSeats()
}

func resetSeats() {
	conn := CommonPool.Get()
	db := conn.DB
	defer CommonPool.Put(conn)
	_, err := db.Exec("UPDATE seats SET user_id = NULL")
	if err != nil {
		panic(err)
	}

	fmt.Println("Reset all seats to unbooked state")

}
