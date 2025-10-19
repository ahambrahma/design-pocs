package update

import (
	"database/sql"
	"exclusive-locks/db"
	"fmt"
	"sync"
	"time"
)

type SeatBookingWithUpdate struct {
}

func (s *SeatBookingWithUpdate) BookSeats() {
	start := time.Now()
	var wg sync.WaitGroup
	for i := 1; i <= 100; i++ {
		wg.Add(1)
		go s.bookSeat(&wg, i)

	}

	wg.Wait()
	fmt.Printf("Total time taken with UPDATE lock: %v seconds\n", time.Since(start).Seconds())
}

func (s *SeatBookingWithUpdate) bookSeat(wg *sync.WaitGroup, userID int) error {
	defer wg.Done()
	// Implementation for booking a seat by userID
	conn := db.CommonPool.Get()
	connDB := conn.DB
	defer db.CommonPool.Put(conn)
	txn, err := connDB.Begin()
	if err != nil {
		fmt.Printf("User %d: Failed to begin transaction: %v\n", userID, err)
		return err
	}

	var selectedSeatID int
	var selectedSeatNumber string
	var selectedSeatUserID *int

	row := txn.QueryRow("SELECT id, number, user_id FROM seats WHERE user_id IS NULL LIMIT 1 FOR UPDATE")
	err = row.Scan(&selectedSeatID, &selectedSeatNumber, &selectedSeatUserID)

	if err == sql.ErrNoRows {
		// No available seats found
		txn.Rollback()
		fmt.Printf("User %d: no available seats to book\n", userID)
		return fmt.Errorf("user %d: no available seats to book", userID)
	}
	if err != nil {
		// Database or scanning error
		txn.Rollback()
		fmt.Printf("User %d: got DB scanning error\n", userID)
		return fmt.Errorf("user %d: failed to select seat: %w", userID, err)
	}

	updateResult, err := txn.Exec("UPDATE seats SET user_id = ? WHERE id = ?", userID, selectedSeatID)
	if err != nil {
		fmt.Printf("User %d: got error while trying to update seat\n", userID)
		txn.Rollback()
		return fmt.Errorf("user %d: failed to update seat: %w", userID, err)
	}

	rowsAffected, err := updateResult.RowsAffected()
	if err != nil {
		fmt.Printf("User %d: got error while checking rows affected\n", userID)
		txn.Rollback()
		return fmt.Errorf("user %d: failed to check rows affected for seat ID %d: %w", userID, selectedSeatID, err)
	}

	if rowsAffected == 0 {
		fmt.Printf("User %d: no rows were affected when trying to update seat ID %d. Booking failed.\n", userID, selectedSeatID)
		txn.Rollback()
		return fmt.Errorf("user %d: no rows were affected when trying to update seat ID %d. Booking failed", userID, selectedSeatID)
	}

	// 3. Commit the transaction
	err = txn.Commit()
	if err != nil {
		fmt.Printf("User %d: error occurred while committing transaction\n", userID)
		return fmt.Errorf("user %d: failed to commit transaction for seat ID %d: %w", userID, selectedSeatID, err)
	}

	fmt.Printf("User %d successfully booked seat: %v\n", userID, selectedSeatID)
	return nil
}
