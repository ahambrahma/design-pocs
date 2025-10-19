package main

import (
	"exclusive-locks/db"
	"exclusive-locks/optimisticlock"
)

func main() {
	db.Init()

	// seatBookingWithoutUpdate := &noupdate.SeatBookingWithoutUpdate{}
	// seatBookingWithoutUpdate.BookSeats()

	// seatBookingWithUpdate := &update.SeatBookingWithUpdate{}
	// seatBookingWithUpdate.BookSeats()

	// SeatBookingWithUpdateSkip := &updateskip.SeatBookingWithUpdateSkip{}
	// SeatBookingWithUpdateSkip.BookSeats()

	seatBookingWithOptimisticLock := &optimisticlock.OptimisticLock{}
	seatBookingWithOptimisticLock.BookSeats()
}
