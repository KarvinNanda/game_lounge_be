package utils

import (
	"fmt"
	"strings"
	"time"
)

// JakartaLoc mengembalikan zona Asia/Jakarta (fallback UTC+7 jika tzdata tidak ada).
func JakartaLoc() *time.Location {
	if loc, err := time.LoadLocation("Asia/Jakarta"); err == nil {
		return loc
	}
	return time.FixedZone("WIB", 7*60*60)
}

// ClockMins mengubah "HH:MM", "H:MM", atau "HH:MM:SS" (format MySQL TIME)
// menjadi menit sejak 00:00. ok=false untuk format atau nilai yang tidak valid.
func ClockMins(t string) (mins int, ok bool) {
	if strings.Count(t, ":") == 2 {
		t = t[:strings.LastIndex(t, ":")] // buang detik
	}
	parsed, err := time.Parse("15:04", t)
	if err != nil {
		return 0, false
	}
	return parsed.Hour()*60 + parsed.Minute(), true
}

// MinsOf = ClockMins tanpa flag; input tidak valid → 0.
func MinsOf(t string) int {
	m, _ := ClockMins(t)
	return m
}

// MinsToClock mengubah menit menjadi "HH:MM", wrap 24 jam (1500 → "01:00").
func MinsToClock(m int) string {
	m %= 24 * 60
	if m < 0 {
		m += 24 * 60
	}
	return fmt.Sprintf("%02d:%02d", m/60, m%60)
}

// NormalizeRange mengubah jam mulai/selesai menjadi menit dalam 1 sesi operasional
// yang dimulai pada jam buka store (openMins). Jam sebelum jam buka (mis. 01:00
// untuk store 09:00–02:00) adalah setelah tengah malam → +24 jam.
func NormalizeRange(start, end string, openMins int) (int, int) {
	s, _ := ClockMins(start)
	e, _ := ClockMins(end)
	if s < openMins {
		s += 24 * 60
	}
	if e < openMins {
		e += 24 * 60
	}
	if e <= s {
		e += 24 * 60
	}
	return s, e
}

// SessionWindow mengembalikan waktu mulai & selesai absolut (WIB) sebuah sesi
// pada tanggal operasional date.
func SessionWindow(date time.Time, start, end string, openMins int) (time.Time, time.Time) {
	loc := JakartaLoc()
	y, m, d := date.Date()
	base := time.Date(y, m, d, 0, 0, 0, 0, loc)
	s, e := NormalizeRange(start, end, openMins)
	return base.Add(time.Duration(s) * time.Minute), base.Add(time.Duration(e) * time.Minute)
}

// SessionStatus menghitung status sesi (upcoming/ongoing/completed) dari waktu
// absolut, sehingga sesi lintas tengah malam (23:00–02:00) tetap "ongoing"
// sampai 02:00 hari berikutnya. Status final (cancelled/completed) tidak diubah.
func SessionStatus(current string, date time.Time, start, end string, openMins int, now time.Time) string {
	if current == "cancelled" || current == "completed" {
		return current
	}
	from, to := SessionWindow(date, start, end, openMins)
	switch {
	case now.Before(from):
		return "upcoming"
	case now.Before(to):
		return "ongoing"
	default:
		return "completed"
	}
}
