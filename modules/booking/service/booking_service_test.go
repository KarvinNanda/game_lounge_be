package service

import (
	"testing"
	"time"

	"game_lounge_be/models"
)

// ── parseTimeToMins ───────────────────────────────────────────

func TestParseTimeToMins_Normal(t *testing.T) {
	cases := []struct {
		input string
		want  int
	}{
		{"00:00", 0},
		{"01:00", 60},
		{"10:30", 630},
		{"16:00", 960},
		{"23:59", 1439},
	}
	for _, c := range cases {
		got := parseTimeToMins(c.input)
		if got != c.want {
			t.Errorf("parseTimeToMins(%q) = %d, want %d", c.input, got, c.want)
		}
	}
}

func TestParseTimeToMins_StringPendek(t *testing.T) {
	// String kurang dari 5 karakter → 0
	cases := []string{"", "1", "10:", "1:0"}
	for _, s := range cases {
		if got := parseTimeToMins(s); got != 0 {
			t.Errorf("parseTimeToMins(%q) = %d, want 0", s, got)
		}
	}
}

// ── computeStatus ─────────────────────────────────────────────

func makeBooking(status, date, start, end string) models.Booking {
	d, _ := time.Parse("2006-01-02", date)
	return models.Booking{
		Status:      status,
		BookingDate: d,
		StartTime:   start,
		EndTime:     end,
	}
}

func TestComputeStatus_StatusFinal(t *testing.T) {
	// Booking yang sudah cancelled/completed tidak dihitung ulang
	b1 := makeBooking("cancelled", "2030-01-01", "10:00", "12:00")
	if got := computeStatus(b1); got != "cancelled" {
		t.Errorf("want 'cancelled', got %q", got)
	}
	b2 := makeBooking("completed", "2030-01-01", "10:00", "12:00")
	if got := computeStatus(b2); got != "completed" {
		t.Errorf("want 'completed', got %q", got)
	}
}

func TestComputeStatus_TanggalLampau(t *testing.T) {
	// Booking kemarin (atau lebih lama) selalu "completed"
	yesterday := time.Now().In(jakartaLoc).AddDate(0, 0, -1).Format("2006-01-02")
	b := makeBooking("upcoming", yesterday, "10:00", "12:00")
	if got := computeStatus(b); got != "completed" {
		t.Errorf("tanggal lampau: want 'completed', got %q", got)
	}
}

func TestComputeStatus_TanggalMendatang(t *testing.T) {
	// Booking besok (atau lebih jauh) selalu "upcoming"
	tomorrow := time.Now().In(jakartaLoc).AddDate(0, 0, 1).Format("2006-01-02")
	b := makeBooking("upcoming", tomorrow, "10:00", "12:00")
	if got := computeStatus(b); got != "upcoming" {
		t.Errorf("tanggal mendatang: want 'upcoming', got %q", got)
	}
}

func TestComputeStatus_HariIni_SudahSelesai(t *testing.T) {
	// Booking hari ini, jam selesai sudah lewat → "completed"
	// Gunakan jam 00:01 - 00:02 (hampir pasti sudah lewat kecuali tes jalan tengah malam)
	todayStr := time.Now().In(jakartaLoc).Format("2006-01-02")
	b := makeBooking("upcoming", todayStr, "00:01", "00:02")

	now := time.Now().In(jakartaLoc)
	nowMins := now.Hour()*60 + now.Minute()
	if nowMins < 2 {
		t.Skip("skip: tes jalan terlalu dekat tengah malam")
	}

	if got := computeStatus(b); got != "completed" {
		t.Errorf("jam sudah lewat hari ini: want 'completed', got %q", got)
	}
}

func TestComputeStatus_HariIni_BelumMulai(t *testing.T) {
	// Booking hari ini, jam masih sangat jauh ke depan (23:50-23:59) → "upcoming"
	todayStr := time.Now().In(jakartaLoc).Format("2006-01-02")
	b := makeBooking("upcoming", todayStr, "23:50", "23:59")

	now := time.Now().In(jakartaLoc)
	nowMins := now.Hour()*60 + now.Minute()
	if nowMins >= 23*60+50 {
		t.Skip("skip: tes jalan terlalu malam (>= 23:50)")
	}

	if got := computeStatus(b); got != "upcoming" {
		t.Errorf("belum dimulai: want 'upcoming', got %q", got)
	}
}
