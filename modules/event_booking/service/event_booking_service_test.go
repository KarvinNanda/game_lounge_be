package service

import (
	"testing"
	"time"

	"game_lounge_be/models"
)

// ── calculateTotalPrice ───────────────────────────────────────

func TestCalculateTotalPrice_Proporsional(t *testing.T) {
	// 1.000.000 / hari → per jam = 41.666,67 → 4 jam = 166.666,67 → dibulatkan ke 167.000
	got := calculateTotalPrice(1_000_000, 4)
	want := 167_000.0
	if got != want {
		t.Errorf("calculateTotalPrice(1000000, 4) = %.0f, want %.0f", got, want)
	}
}

func TestCalculateTotalPrice_SebuahHari(t *testing.T) {
	// 1 hari penuh (24 jam) → total = price_per_day itu sendiri
	got := calculateTotalPrice(2_400_000, 24)
	want := 2_400_000.0
	if got != want {
		t.Errorf("calculateTotalPrice(2400000, 24) = %.0f, want %.0f", got, want)
	}
}

func TestCalculateTotalPrice_SetengahHari(t *testing.T) {
	// 2.400.000 / hari, 12 jam = 1.200.000
	got := calculateTotalPrice(2_400_000, 12)
	want := 1_200_000.0
	if got != want {
		t.Errorf("calculateTotalPrice(2400000, 12) = %.0f, want %.0f", got, want)
	}
}

func TestCalculateTotalPrice_DibulatkanKeRibuanBawah(t *testing.T) {
	// 500.000 / hari, 1 jam = 20.833,33 → dibulatkan ke 21.000
	got := calculateTotalPrice(500_000, 1)
	want := 21_000.0
	if got != want {
		t.Errorf("calculateTotalPrice(500000, 1) = %.0f, want %.0f", got, want)
	}
}

func TestCalculateTotalPrice_DibulatkanKeRibuanAtas(t *testing.T) {
	// 720.000 / hari, 1 jam = 30.000 tepat → tidak ada pembulatan
	got := calculateTotalPrice(720_000, 1)
	want := 30_000.0
	if got != want {
		t.Errorf("calculateTotalPrice(720000, 1) = %.0f, want %.0f", got, want)
	}
}

func TestCalculateTotalPrice_DurasiPecahan(t *testing.T) {
	// 2.400.000 / hari, 1.5 jam = 150.000
	got := calculateTotalPrice(2_400_000, 1.5)
	want := 150_000.0
	if got != want {
		t.Errorf("calculateTotalPrice(2400000, 1.5) = %.0f, want %.0f", got, want)
	}
}

// ── computeStatus (EventBooking) ──────────────────────────────

func makeEventBooking(status, date, start, end string) models.EventBooking {
	d, _ := time.Parse("2006-01-02", date)
	return models.EventBooking{
		Status:      status,
		BookingDate: d,
		StartTime:   start,
		EndTime:     end,
	}
}

// Waktu "sekarang" & jam buka eksplisit supaya test tidak bergantung jam berjalan.
// Store buka 09:00, tutup 02:00.
const testOpen = 9 * 60

func atWIB(date string, h, m int) time.Time {
	loc, _ := time.LoadLocation("Asia/Jakarta")
	d, _ := time.ParseInLocation("2006-01-02", date, loc)
	return d.Add(time.Duration(h)*time.Hour + time.Duration(m)*time.Minute)
}

func TestComputeStatus(t *testing.T) {
	cases := []struct {
		name string
		b    models.EventBooking
		now  time.Time
		want string
	}{
		{"cancelled", makeEventBooking("cancelled", "2030-01-01", "10:00", "12:00"), atWIB("2030-01-01", 11, 0), "cancelled"},
		{"completed", makeEventBooking("completed", "2030-01-01", "10:00", "12:00"), atWIB("2029-12-31", 9, 0), "completed"},
		{"tanggal lampau", makeEventBooking("upcoming", "2030-01-01", "10:00", "22:00"), atWIB("2030-01-02", 23, 0), "completed"},
		{"tanggal mendatang", makeEventBooking("upcoming", "2030-01-02", "10:00", "22:00"), atWIB("2030-01-01", 12, 0), "upcoming"},
		{"hari ini selesai", makeEventBooking("upcoming", "2030-01-01", "10:00", "12:00"), atWIB("2030-01-01", 13, 0), "completed"},
		{"hari ini belum mulai", makeEventBooking("upcoming", "2030-01-01", "20:00", "22:00"), atWIB("2030-01-01", 13, 0), "upcoming"},
		{"full day 09:00–02:00, jam 01:00 besoknya masih berjalan", makeEventBooking("upcoming", "2030-01-01", "09:00", "02:00"), atWIB("2030-01-02", 1, 0), "ongoing"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := computeStatusAt(tc.b, testOpen, tc.now); got != tc.want {
				t.Errorf("want %q, got %q", tc.want, got)
			}
		})
	}
}
