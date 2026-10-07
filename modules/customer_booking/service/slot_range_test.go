package service

import "testing"

func TestResolveSlotRange(t *testing.T) {
	// Urutan slot resmi weekend 09:00–02:00 (setelah tengah malam ada di akhir).
	day := []string{"09:00", "10:00", "11:00", "12:00", "13:00", "14:00", "15:00", "16:00",
		"17:00", "18:00", "19:00", "20:00", "21:00", "22:00", "23:00", "00:00", "01:00"}

	ok := []struct {
		name       string
		selected   []string
		start, end string
		hours      int
	}{
		{"1 slot", []string{"10:00"}, "10:00", "11:00", 1},
		{"berurutan, input tidak terurut", []string{"12:00", "10:00", "11:00"}, "10:00", "13:00", 3},
		{"lintas tengah malam", []string{"23:00", "00:00", "01:00"}, "23:00", "02:00", 3},
	}
	for _, tc := range ok {
		t.Run(tc.name, func(t *testing.T) {
			start, end, hours, err := resolveSlotRange(tc.selected, day)
			if err != nil {
				t.Fatalf("tidak boleh error: %v", err)
			}
			if start != tc.start || end != tc.end || hours != tc.hours {
				t.Errorf("want %s-%s (%d jam), got %s-%s (%d jam)", tc.start, tc.end, tc.hours, start, end, hours)
			}
		})
	}

	bad := []struct {
		name     string
		selected []string
	}{
		{"tidak berurutan (bayar 2 jam, blok 11 jam)", []string{"10:00", "20:00"}},
		{"duplikat", []string{"10:00", "10:00"}},
		{"di luar jam operasional", []string{"03:00"}},
		{"format rusak", []string{"9x:00"}},
		{"kosong", []string{}},
	}
	for _, tc := range bad {
		t.Run(tc.name, func(t *testing.T) {
			if _, _, _, err := resolveSlotRange(tc.selected, day); err == nil {
				t.Error("harus error")
			}
		})
	}
}
