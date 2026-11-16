package service

import (
	"testing"

	"game_lounge_be/money"
)

// Characterization: event price = price_per_day / 24 * hours, rounded to the
// nearest Rp 1.000 (half up). Written against the float64 version; the integer
// version must give the same numbers.
func TestCalculateTotalPrice_RoundingCharacterization(t *testing.T) {
	cases := []struct {
		perDay money.Rupiah
		hours  float64
		want   int64
	}{
		{600_000, 0.5, 13_000},     // 12.500 -> 13.000 (exact half)
		{1_000_000, 1, 42_000},     // 41.666,67
		{1_000_000, 4, 167_000},    // 166.666,67
		{1_500_000, 3.5, 219_000},  // 218.750
		{2_400_000, 24, 2_400_000}, // full day
		{1_234_567, 7.5, 386_000},  // 385.802,19
		{2_000_000, 2.4, 200_000},  // 200.000
		{1_100_000, 6.3, 289_000},  // 288.750
		{960_000, 0.5, 20_000},     // 20.000
		{780_000, 1.5, 49_000},     // 48.750
	}
	for _, c := range cases {
		if got := int64(calculateTotalPrice(c.perDay, c.hours)); got != c.want {
			t.Errorf("perDay %d, %v h: want %d, got %d", c.perDay, c.hours, c.want, got)
		}
	}
}
