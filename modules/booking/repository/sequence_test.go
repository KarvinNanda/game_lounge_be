package repository

import (
	"sync"
	"testing"

	"game_lounge_be/config"
)

// GetNextSequence dipanggil bersamaan (mis. 2 webhook PAID sekaligus) tidak boleh
// mengembalikan nomor yang sama — booking_code UNIQUE akan menolak insert kedua.
func TestGetNextSequence_KonkurenTidakDuplikat(t *testing.T) {
	skipIfNoDB(t)
	sqlDB, _ := config.DB.DB()
	sqlDB.SetMaxOpenConns(20)

	const n = 40
	var wg sync.WaitGroup
	var mu sync.Mutex
	seen := map[uint]bool{}
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			seq, err := GetNextSequence()
			if err != nil {
				t.Errorf("GetNextSequence error: %v", err)
				return
			}
			mu.Lock()
			defer mu.Unlock()
			if seen[seq] {
				t.Errorf("nomor duplikat: %d", seq)
			}
			seen[seq] = true
		}()
	}
	wg.Wait()
	if len(seen) != n {
		t.Errorf("want %d nomor unik, got %d", n, len(seen))
	}
}
