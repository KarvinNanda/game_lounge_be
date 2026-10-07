package service

import (
	"testing"

	"game_lounge_be/utils"
)

// ── utils.MinsOf (dulu timeToMins) ────────────────────────────────────────────────

func TestTimeToMins_Normal(t *testing.T) {
	cases := []struct {
		input string
		want  int
	}{
		{"00:00", 0},
		{"01:00", 60},
		{"08:30", 510},
		{"10:00", 600},
		{"16:00", 960},
		{"23:59", 1439},
	}
	for _, c := range cases {
		got := utils.MinsOf(c.input)
		if got != c.want {
			t.Errorf("utils.MinsOf(%q) = %d, want %d", c.input, got, c.want)
		}
	}
}

func TestTimeToMins_StringPendek(t *testing.T) {
	// String < 5 karakter → 0
	cases := []string{"", "1", "10:", "1:0"}
	for _, s := range cases {
		if got := utils.MinsOf(s); got != 0 {
			t.Errorf("utils.MinsOf(%q) = %d, want 0", s, got)
		}
	}
}

// ── utils.MinsToClock (dulu minsToTime) ────────────────────────────────────────────────

func TestMinsToTime_Normal(t *testing.T) {
	cases := []struct {
		input int
		want  string
	}{
		{0, "00:00"},
		{60, "01:00"},
		{510, "08:30"},
		{600, "10:00"},
		{960, "16:00"},
		{1439, "23:59"},
	}
	for _, c := range cases {
		got := utils.MinsToClock(c.input)
		if got != c.want {
			t.Errorf("utils.MinsToClock(%d) = %q, want %q", c.input, got, c.want)
		}
	}
}

func TestMinsToTime_Modulo24h(t *testing.T) {
	// Menit >= 1440 (lebih dari 24 jam) harus wrap ke modulo 24h
	cases := []struct {
		input int
		want  string
	}{
		{1440, "00:00"}, // tepat 24h
		{1500, "01:00"}, // 24h + 1h
		{1560, "02:00"}, // 24h + 2h
		{2880, "00:00"}, // tepat 48h
	}
	for _, c := range cases {
		got := utils.MinsToClock(c.input)
		if got != c.want {
			t.Errorf("utils.MinsToClock(%d) = %q, want %q", c.input, got, c.want)
		}
	}
}

func TestMinsToTime_RoundTrip(t *testing.T) {
	// utils.MinsOf(utils.MinsToClock(x)) == x untuk 0 <= x < 1440
	for _, m := range []int{0, 60, 120, 240, 480, 600, 720, 960, 1200, 1320, 1439} {
		s := utils.MinsToClock(m)
		back := utils.MinsOf(s)
		if back != m {
			t.Errorf("round-trip %d → %q → %d (want %d)", m, s, back, m)
		}
	}
}
