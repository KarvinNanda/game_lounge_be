// Package jobs berisi pekerjaan latar belakang yang berjalan selama server hidup.
package jobs

import (
	"log"
	"time"

	customerBookingRepo "game_lounge_be/modules/customer_booking/repository"
	eventRepo "game_lounge_be/modules/event_booking/repository"
	"game_lounge_be/utils"
)

// StartCleanup menjalankan pembersihan setiap interval:
//   - event customer yang tidak dibayar dalam jendela bayar → cancelled
//   - hold booking yang sudah lama expired → dihapus
func StartCleanup(interval time.Duration) {
	utils.SafeGo(func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for range ticker.C {
			runCleanup()
		}
	})
}

func runCleanup() {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[JOB] cleanup panic: %v", r)
		}
	}()
	if n, err := eventRepo.CancelExpiredUnpaid(); err != nil {
		log.Printf("[JOB] cancel event belum dibayar: %v", err)
	} else if n > 0 {
		log.Printf("[JOB] %d event belum dibayar dibatalkan", n)
	}
	if err := customerBookingRepo.CleanExpiredHolds(); err != nil {
		log.Printf("[JOB] hapus hold expired: %v", err)
	}
}
