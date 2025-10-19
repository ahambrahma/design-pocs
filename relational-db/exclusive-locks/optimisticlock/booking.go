package optimisticlock

import (
	"database/sql"
	"exclusive-locks/db"
	"fmt"
	"sync"
	"time"
)

const maxAttempts = 50

type OptimisticLock struct {
}

func (o *OptimisticLock) BookSeats() {
	start := time.Now()
	// Implementation for booking seats using optimistic locking
	var wg sync.WaitGroup
	for i := 1; i <= 100; i++ {
		wg.Add(1)
		go o.bookSeat(&wg, i)
	}

	wg.Wait()
	fmt.Printf("Total time taken with optimistic lock: %v seconds\n", time.Since(start).Seconds())
}

func (o *OptimisticLock) bookSeat(wg *sync.WaitGroup, userID int) error {
	// Implementation for booking a seat by userID using optimistic locking
	defer wg.Done()

	conn := db.CommonPool.Get()
	connDB := conn.DB
	defer db.CommonPool.Put(conn)

	var selectedSeatID int
	var selectedSeatNumber string
	var selectedSeatVersion int

	var updateResult sql.Result
	var rowsAffected int64

	var err1 error
	var err2 error

	for j := 1; j <= maxAttempts; j++ {
		// As a best practice, always create a new transaction for each attempt and then rollback / commit
		// Do not try to use the same transaction object across multiple attempts

		fmt.Printf("User %d: Attempt %d to book seat\n", userID, j)

		txn, err := connDB.Begin()
		if err != nil {
			txn.Rollback()
			fmt.Printf("User %d: Failed to begin transaction: %v\n", userID, err)
			return err
		}

		row := txn.QueryRow("SELECT id, number, version FROM seats_v2 WHERE user_id IS NULL LIMIT 1")
		err = row.Scan(&selectedSeatID, &selectedSeatNumber, &selectedSeatVersion)

		if err != nil {
			txn.Rollback()
			fmt.Printf("User %d: DB select error: %v\n", userID, err)
			continue
		}

		updateResult, err1 = txn.Exec("UPDATE seats_v2 SET user_id = ?, version = version + 1 WHERE id = ? AND version = ?", userID, selectedSeatID, selectedSeatVersion)
		if err1 != nil || updateResult == nil {
			txn.Rollback()
			fmt.Printf("User %d: failed to update seat, error: %v\n", userID, err1)
			continue
		}

		rowsAffected, err2 = updateResult.RowsAffected()
		if err2 != nil || rowsAffected == 0 {
			txn.Rollback()
			fmt.Printf("User %d: optimistic lock conflict on seat %d, retrying...\n", userID, selectedSeatID)
			continue
		}

		err = txn.Commit()
		if err != nil {
			txn.Rollback()
			fmt.Printf("User %d: failed to commit transaction: %v\n", userID, err)
			continue
		}

		fmt.Printf("User %d successfully booked seat: %d, version: %d\n", userID, selectedSeatID, selectedSeatVersion)
		return nil
	}

	return nil
}
