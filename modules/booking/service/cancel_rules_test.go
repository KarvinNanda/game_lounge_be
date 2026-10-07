package service

import (
	"testing"
	"time"

	"game_lounge_be/models"
)

func TestCheckCancellable(t *testing.T) {
	now := time.Date(2026, 10, 7, 15, 0, 0, 0, jakartaLoc)
	day := func(offset int) time.Time { return now.AddDate(0, 0, offset) }
	cases := []struct {
		name string
		b    models.Booking
		ok   bool
	}{
		{"besok, upcoming", models.Booking{Status: "upcoming", BookingDate: day(1), StartTime: "10:00", EndTime: "12:00"}, true},
		{"hari ini belum mulai", models.Booking{Status: "upcoming", BookingDate: day(0), StartTime: "18:00", EndTime: "20:00"}, true},
		{"hari ini sedang berjalan", models.Booking{Status: "upcoming", BookingDate: day(0), StartTime: "14:00", EndTime: "16:00"}, false},
		{"kemarin, di DB masih upcoming", models.Booking{Status: "upcoming", BookingDate: day(-1), StartTime: "10:00", EndTime: "12:00"}, false},
		{"sudah dibatalkan", models.Booking{Status: "cancelled", BookingDate: day(1), StartTime: "10:00", EndTime: "12:00"}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := checkCancellable(tc.b, testOpen, now)
			if (err == nil) != tc.ok {
				t.Errorf("want ok=%v, got %v", tc.ok, err)
			}
		})
	}
}
