package utils

import (
	"testing"
	"time"
)

func TestSessionStatus(t *testing.T) {
	loc := JakartaLoc()
	d := time.Date(2026, 10, 10, 0, 0, 0, 0, loc) // tanggal booking
	at := func(day, h, m int) time.Time { return time.Date(2026, 10, day, h, m, 0, 0, loc) }
	open := 9 * 60 // store buka 09:00, tutup 02:00

	cases := []struct {
		name       string
		current    string
		start, end string
		now        time.Time
		want       string
	}{
		{"status final tidak dihitung ulang", "cancelled", "10:00", "12:00", at(10, 11, 0), "cancelled"},
		{"siang: sebelum mulai", "upcoming", "10:00", "12:00", at(10, 9, 59), "upcoming"},
		{"siang: berjalan", "upcoming", "10:00", "12:00", at(10, 11, 0), "ongoing"},
		{"siang: selesai", "upcoming", "10:00", "12:00", at(10, 12, 0), "completed"},
		{"23:00-02:00 jam 23:30 → berjalan", "upcoming", "23:00", "02:00", at(10, 23, 30), "ongoing"},
		{"23:00-02:00 jam 00:30 besoknya → masih berjalan", "upcoming", "23:00", "02:00", at(11, 0, 30), "ongoing"},
		{"23:00-02:00 jam 02:00 besoknya → selesai", "ongoing", "23:00", "02:00", at(11, 2, 0), "completed"},
		{"sesi 00:00-02:00 milik tgl 10 = tgl 11 dini hari; jam 22:00 tgl 10 → belum", "upcoming", "00:00", "02:00", at(10, 22, 0), "upcoming"},
		{"sesi 00:00-02:00 jam 01:00 tgl 11 → berjalan", "upcoming", "00:00", "02:00", at(11, 1, 0), "ongoing"},
		{"tanggal kemarin → selesai", "upcoming", "10:00", "12:00", at(11, 15, 0), "completed"},
		{"tanggal besok → belum", "upcoming", "10:00", "12:00", at(9, 15, 0), "upcoming"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := SessionStatus(tc.current, d, tc.start, tc.end, open, tc.now); got != tc.want {
				t.Errorf("want %s, got %s", tc.want, got)
			}
		})
	}
}

func TestClockMins(t *testing.T) {
	for in, want := range map[string]int{"00:00": 0, "09:30": 570, "9:30": 570, "10:30:00": 630, "23:59:59": 1439} {
		if got, ok := ClockMins(in); !ok || got != want {
			t.Errorf("%s: want %d, got %d (%v)", in, want, got, ok)
		}
	}
	for _, bad := range []string{"", "1", "10:", "1:0", "9x:00", "24:00", "12:60"} {
		if _, ok := ClockMins(bad); ok {
			t.Errorf("%q harus tidak valid", bad)
		}
	}
}

func TestMinsOfDanMinsToClock(t *testing.T) {
	if MinsOf("16:00") != 960 || MinsOf("rusak") != 0 {
		t.Error("MinsOf")
	}
	for in, want := range map[int]string{0: "00:00", 630: "10:30", 1439: "23:59", 1440: "00:00", 1500: "01:00", -60: "23:00"} {
		if got := MinsToClock(in); got != want {
			t.Errorf("MinsToClock(%d) = %q, want %q", in, got, want)
		}
	}
}
