package main

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

const MaxNumber = 500000000
const NumWorkers = 10

var numPrimes int32 = 0
var primeStart int32 = 1

func isPrime(n int) bool {
	if n&1 == 0 {
		return false
	}

	for i := 3; i*i <= n; i++ {
		if n%i == 0 {
			return false
		}
	}

	atomic.AddInt32(&numPrimes, 1)

	return true
}

func calculatePrimesInRange(wg *sync.WaitGroup, numBatch, nstart, nend int) {
	start := time.Now()

	for i := nstart; i < nend; i++ {
		isPrime(i)
	}

	fmt.Printf("Batch %d: time taken: %v\n", numBatch, time.Since(start).Seconds())
	wg.Done()
}

func nonFairPrimeCalculator() {
	start := time.Now()
	nstart := 2
	nend := MaxNumber

	batchSize := MaxNumber / NumWorkers

	var wg sync.WaitGroup

	for i := 0; i < NumWorkers; i++ {
		batchStart := nstart + i*batchSize
		batchEnd := batchStart + batchSize
		if i == NumWorkers-1 {
			batchEnd = nend
		}
		wg.Add(1)
		go calculatePrimesInRange(&wg, i, batchStart, batchEnd)
	}

	wg.Wait()

	fmt.Printf("Overall: Total primes found: %d, total time taken: %v\n", numPrimes, time.Since(start).Seconds())

}

func calculatePrimesFairly(wg *sync.WaitGroup, numBatch int) {
	start := time.Now()
	for {
		currentNumber := atomic.AddInt32(&primeStart, 1)
		if int(currentNumber) > MaxNumber {
			fmt.Printf("Batch %d: time taken: %v\n", numBatch, time.Since(start).Seconds())
			wg.Done()
			break
		}
		isPrime(int(currentNumber))
	}
}

func fairPrimeCalculator() {
	start := time.Now()

	var wg sync.WaitGroup
	for i := 0; i < NumWorkers; i++ {
		wg.Add(1)
		go calculatePrimesFairly(&wg, i)
	}

	wg.Wait()
	fmt.Printf("Overall: Total primes found: %d, total time taken: %v\n", numPrimes, time.Since(start).Seconds())
}

func main() {
	// nonFairPrimeCalculator()
	fairPrimeCalculator()
}
