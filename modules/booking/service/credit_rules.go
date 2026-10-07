package service

import (
	"errors"
	"fmt"
	"time"

	"game_lounge_be/models"
)

// durationFromRange menghitung durasi (jam) dari jam mulai & selesai di server.
// Dulu durasi diambil dari request (duration_hours), sehingga booking 4 jam bisa
// memotong credits hanya 0.5 jam.
func durationFromRange(start, end string) (float64, error) {
	s, err1 := parseClock(start)
	e, err2 := parseClock(end)
	if err1 != nil || err2 != nil {
		return 0, errors.New("format jam tidak valid (HH:MM)")
	}
	if s == e {
		return 0, errors.New("jam mulai dan jam selesai tidak boleh sama")
	}
	if e < s {
		e += 24 * 60 // lintas tengah malam
	}
	return float64(e-s) / 60, nil
}

func parseClock(t string) (int, error) {
	if len(t) < 5 {
		return 0, errors.New("format jam")
	}
	parsed, err := time.Parse("15:04", t[:5])
	if err != nil {
		return 0, err
	}
	return parsed.Hour()*60 + parsed.Minute(), nil
}

// checkCreditUsable memastikan play credit boleh dipakai untuk booking ini:
// milik customer yang sama, aktif, belum expired, berlaku di store, dan jam cukup.
func checkCreditUsable(c *models.CustomerPlayCredit, customerID, storeID string, hours float64, now time.Time) error {
	if customerID == "" || c.CustomerID != customerID {
		return errors.New("play credits bukan milik customer ini")
	}
	if !c.IsActive || c.DeletedAt != nil {
		return errors.New("play credits tidak aktif")
	}
	if !now.Before(c.ExpiresAt) {
		return errors.New("play credits sudah kadaluwarsa")
	}
	if !c.Package.ApplyToAllStores {
		allowed := false
		for _, ps := range c.Package.PackageStores {
			if ps.StoreID == storeID {
				allowed = true
				break
			}
		}
		if !allowed {
			return errors.New("play credits tidak berlaku di cabang ini")
		}
	}
	if c.RemainingHours < hours {
		return fmt.Errorf("sisa jam credits (%.1f jam) tidak cukup untuk booking ini (%.1f jam)", c.RemainingHours, hours)
	}
	return nil
}
