package service

import (
	"testing"

	"game_lounge_be/models"
)

// Per-hour price is compared by cross-multiplication instead of division.
// On a tie the package seen first wins, as in the float version.
func TestFindBestPackageForRemaining_CrossMultiply(t *testing.T) {
	pkgs := []models.StorePackagePrice{
		{DurationHours: 3, Price: 120000}, // 40.000/h
		{DurationHours: 2, Price: 80000},  // 40.000/h, tie with 3h
		{DurationHours: 1, Price: 50000},  // 50.000/h
	}
	if got := findBestPackageForRemaining(5, pkgs); got.DurationHours != 3 {
		t.Errorf("tie must keep the first package (3h), got %dh", got.DurationHours)
	}
	if got := findBestPackageForRemaining(2, pkgs); got.DurationHours != 2 {
		t.Errorf("3h does not fit in 2h remaining, want 2h, got %dh", got.DurationHours)
	}

	odd := []models.StorePackagePrice{
		{DurationHours: 3, Price: 100000}, // 33.333,33/h
		{DurationHours: 7, Price: 233331}, // 33.333,00/h (cheaper by a fraction)
	}
	if got := findBestPackageForRemaining(10, odd); got.DurationHours != 7 {
		t.Errorf("want the cheaper per-hour 7h package, got %dh", got.DurationHours)
	}
	if got := findBestPackageForRemaining(0, pkgs); got != nil {
		t.Errorf("nothing fits in 0h, got %+v", got)
	}
}
