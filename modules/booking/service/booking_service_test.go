package service

import (
	"testing"
	"time"

	"game_lounge_be/models"
)

// Parsing jam (dulu parseTimeToMins) kini di utils.ClockMins — dites di utils/session_test.go.

func makeBooking(status, date, start, end string) models.Booking {
	d, _ := time.Parse("2006-01-02", date)
	return models.Booking{
		Status:      status,
		BookingDate: d,
		StartTime:   start,
		EndTime:     end,
	}
}

// ── computeStatusAt ───────────────────────────────────────────
// Waktu "sekarang" dan jam buka diberikan eksplisit supaya test tidak bergantung
// pada jam saat test dijalankan. Store buka 10:00, tutup 02:00.

const testOpen = 10 * 60

func at(date string, h, m int) time.Time {
	d, _ := time.ParseInLocation("2006-01-02", date, jakartaLoc)
	return d.Add(time.Duration(h)*time.Hour + time.Duration(m)*time.Minute)
}

func TestComputeStatus(t *testing.T) {
	cases := []struct {
		name string
		b    models.Booking
		now  time.Time
		want string
	}{
		{"cancelled tidak dihitung ulang", makeBooking("cancelled", "2030-01-01", "10:00", "12:00"), at("2030-01-01", 11, 0), "cancelled"},
		{"completed tidak dihitung ulang", makeBooking("completed", "2030-01-01", "10:00", "12:00"), at("2029-12-31", 9, 0), "completed"},
		{"tanggal lampau", makeBooking("upcoming", "2030-01-01", "10:00", "12:00"), at("2030-01-02", 15, 0), "completed"},
		{"tanggal mendatang", makeBooking("upcoming", "2030-01-02", "10:00", "12:00"), at("2030-01-01", 15, 0), "upcoming"},
		{"hari ini sudah selesai", makeBooking("upcoming", "2030-01-01", "10:00", "12:00"), at("2030-01-01", 13, 0), "completed"},
		{"hari ini belum mulai", makeBooking("upcoming", "2030-01-01", "20:00", "22:00"), at("2030-01-01", 13, 0), "upcoming"},
		{"hari ini sedang berjalan", makeBooking("upcoming", "2030-01-01", "12:00", "14:00"), at("2030-01-01", 13, 0), "ongoing"},
		{"23:00–02:00, jam 00:30 besoknya masih berjalan", makeBooking("upcoming", "2030-01-01", "23:00", "02:00"), at("2030-01-02", 0, 30), "ongoing"},
		{"sesi 00:30–01:30 tgl 1 terjadi tgl 2 dini hari", makeBooking("upcoming", "2030-01-01", "00:30", "01:30"), at("2030-01-01", 23, 0), "upcoming"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := computeStatusAt(tc.b, testOpen, tc.now); got != tc.want {
				t.Errorf("want %q, got %q", tc.want, got)
			}
		})
	}
}
