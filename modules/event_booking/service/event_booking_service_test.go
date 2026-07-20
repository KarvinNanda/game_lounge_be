package service

import (
	"testing"
	"time"

	"game_lounge_be/models"
)

// ── parseMins ──────────────────────────────────────────────────

func TestParseMins_Normal(t *testing.T) {
	cases := []struct {
		input string
		want  int
	}{
		{"00:00", 0},
		{"01:00", 60},
		{"08:30", 510},
		{"16:00", 960},
		{"23:59", 1439},
	}
	for _, c := range cases {
		got := parseMins(c.input)
		if got != c.want {
			t.Errorf("parseMins(%q) = %d, want %d", c.input, got, c.want)
		}
	}
}

func TestParseMins_StringPendek(t *testing.T) {
	// String < 5 karakter → 0
	cases := []string{"", "1", "10:", "1:0"}
	for _, s := range cases {
		if got := parseMins(s); got != 0 {
			t.Errorf("parseMins(%q) = %d, want 0", s, got)
		}
	}
}

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

func jakartaNow() time.Time {
	loc, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		loc = time.UTC
	}
	return time.Now().In(loc)
}

func TestComputeStatus_Cancelled(t *testing.T) {
	b := makeEventBooking("cancelled", "2030-01-01", "10:00", "12:00")
	if got := computeStatus(b); got != "cancelled" {
		t.Errorf("want 'cancelled', got %q", got)
	}
}

func TestComputeStatus_Completed(t *testing.T) {
	b := makeEventBooking("completed", "2030-01-01", "10:00", "12:00")
	if got := computeStatus(b); got != "completed" {
		t.Errorf("want 'completed', got %q", got)
	}
}

func TestComputeStatus_TanggalLampau(t *testing.T) {
	yesterday := jakartaNow().AddDate(0, 0, -1).Format("2006-01-02")
	b := makeEventBooking("upcoming", yesterday, "10:00", "22:00")
	if got := computeStatus(b); got != "completed" {
		t.Errorf("tanggal lampau: want 'completed', got %q", got)
	}
}

func TestComputeStatus_TanggalMendatang(t *testing.T) {
	tomorrow := jakartaNow().AddDate(0, 0, 1).Format("2006-01-02")
	b := makeEventBooking("upcoming", tomorrow, "10:00", "22:00")
	if got := computeStatus(b); got != "upcoming" {
		t.Errorf("tanggal mendatang: want 'upcoming', got %q", got)
	}
}

func TestComputeStatus_HariIni_SudahSelesai(t *testing.T) {
	now := jakartaNow()
	nowMins := now.Hour()*60 + now.Minute()
	if nowMins < 2 {
		t.Skip("skip: tes jalan terlalu dekat tengah malam")
	}
	todayStr := now.Format("2006-01-02")
	b := makeEventBooking("upcoming", todayStr, "00:01", "00:02")
	if got := computeStatus(b); got != "completed" {
		t.Errorf("jam sudah lewat: want 'completed', got %q", got)
	}
}

func TestComputeStatus_HariIni_BelumMulai(t *testing.T) {
	now := jakartaNow()
	nowMins := now.Hour()*60 + now.Minute()
	if nowMins >= 23*60+50 {
		t.Skip("skip: tes jalan terlalu malam (>= 23:50)")
	}
	todayStr := now.Format("2006-01-02")
	b := makeEventBooking("upcoming", todayStr, "23:50", "23:59")
	if got := computeStatus(b); got != "upcoming" {
		t.Errorf("belum mulai: want 'upcoming', got %q", got)
	}
}
