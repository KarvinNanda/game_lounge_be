package service

import (
	"testing"

	"game_lounge_be/money"
)

// Characterization: exact rupiah results of the percentage discount, including
// results that land on, just above and just below .5. Written against the float64
// version; the integer version must give the same numbers.
func TestCalculateDiscount_RoundingCharacterization(t *testing.T) {
	cases := []struct {
		amount  money.Rupiah
		percent float64
		want    int64
	}{
		{10, 15, 2},          // 1.5 -> 2 (half rounds up)
		{30, 25, 8},          // 7.5 -> 8
		{3333, 15, 500},      // 499.95 -> 500
		{10001, 15, 1500},    // 1500.15 -> 1500
		{33333, 15, 5000},    // 4999.95 -> 5000
		{66667, 15, 10000},   // 10000.05 -> 10000
		{1005, 15, 151},      // 150.75 -> 151
		{123456, 10, 12346},  // 12345.6 -> 12346
		{999999, 33, 330000}, // 329999.67 -> 330000
		{1, 50, 1},           // 0.5 -> 1
		{1, 49, 0},           // 0.49 -> 0
		{0, 20, 0},
	}
	for _, c := range cases {
		v := makeVoucher("percentage", c.percent, nil)
		if got := int64(calculateDiscount(c.amount, v)); got != c.want {
			t.Errorf("%v%% of %d: want %d, got %d", c.percent, c.amount, c.want, got)
		}
	}
}

func TestCalculateDiscount_NominalCappedAtAmount(t *testing.T) {
	v := makeVoucher("nominal", 25000, nil)
	if got := int64(calculateDiscount(10000, v)); got != 10000 {
		t.Errorf("nominal above amount must cap at amount, got %d", got)
	}
	if got := int64(calculateDiscount(30000, v)); got != 25000 {
		t.Errorf("nominal below amount, got %d", got)
	}
}
