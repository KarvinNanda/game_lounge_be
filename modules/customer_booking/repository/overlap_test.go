package repository

import (
	"testing"

	"game_lounge_be/utils"
)

func TestRangesOverlap_RelatifJamBuka(t *testing.T) {
	open := 9 * 60 // store buka 09:00, tutup 02:00
	cases := []struct {
		name         string
		aStart, aEnd string
		bStart, bEnd string
		want         bool
	}{
		{"siang bentrok", "10:00", "12:00", "11:00", "13:00", true},
		{"siang bersebelahan", "10:00", "12:00", "12:00", "13:00", false},
		{"lintas tengah malam vs setelah tengah malam", "23:00", "02:00", "00:00", "01:00", true},
		{"setelah tengah malam vs malam", "00:00", "02:00", "22:00", "00:00", false},
		{"malam vs setelah tengah malam bentrok", "22:00", "01:00", "00:00", "02:00", true},
		{"pagi vs setelah tengah malam (beda sesi)", "09:00", "10:00", "01:00", "02:00", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			aS, aE := utils.NormalizeRange(tc.aStart, tc.aEnd, open)
			bS, bE := utils.NormalizeRange(tc.bStart, tc.bEnd, open)
			if got := aS < bE && aE > bS; got != tc.want {
				t.Errorf("want %v, got %v (a=%d-%d b=%d-%d)", tc.want, got, aS, aE, bS, bE)
			}
		})
	}
}
