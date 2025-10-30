package main

import "testing"

func BenchmarkGenerateIDWithHardPersistence(b *testing.B) {
	for i := 0; i < b.N; i++ {
		generateIDWithDiskPersistence()
	}
}

func BenchmarkGenerateIDWithPeriodicPersistence(b *testing.B) {
	for i := 0; i < b.N; i++ {
		generateIDWithPeriodicDiskPersistence(100)
	}
}
