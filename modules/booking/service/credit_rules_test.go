package service

import (
	"testing"
	"time"

	"game_lounge_be/models"
)

func TestDurationFromRange(t *testing.T) {
	ok := []struct {
		start, end string
		want       float64
	}{
		{"10:00", "13:00", 3},
		{"10:00", "10:30", 0.5},
		{"23:00", "02:00", 3}, // lintas tengah malam
		{"14:00:00", "16:30:00", 2.5},
	}
	for _, tc := range ok {
		got, err := durationFromRange(tc.start, tc.end)
		if err != nil || got != tc.want {
			t.Errorf("%s-%s: want %.1f, got %.1f (%v)", tc.start, tc.end, tc.want, got, err)
		}
	}
	for _, bad := range [][2]string{{"10:00", "10:00"}, {"9x:00", "10:00"}, {"10:00", ""}, {"25:00", "26:00"}} {
		if _, err := durationFromRange(bad[0], bad[1]); err == nil {
			t.Errorf("%v harus error", bad)
		}
	}
}

func TestCheckCreditUsable(t *testing.T) {
	now := time.Now()
	base := func() *models.CustomerPlayCredit {
		return &models.CustomerPlayCredit{
			CustomerID: "cust-A", RemainingHours: 5, IsActive: true, ExpiresAt: now.Add(24 * time.Hour),
			Package: models.PlayCreditsPackage{ApplyToAllStores: true},
		}
	}
	cases := []struct {
		name   string
		mutate func(c *models.CustomerPlayCredit)
		cust   string
		store  string
		hours  float64
		ok     bool
	}{
		{"valid", func(*models.CustomerPlayCredit) {}, "cust-A", "store-1", 3, true},
		{"milik customer lain", func(*models.CustomerPlayCredit) {}, "cust-B", "store-1", 3, false},
		{"tanpa customer (walk-in)", func(*models.CustomerPlayCredit) {}, "", "store-1", 3, false},
		{"jam tidak cukup", func(*models.CustomerPlayCredit) {}, "cust-A", "store-1", 6, false},
		{"tidak aktif", func(c *models.CustomerPlayCredit) { c.IsActive = false }, "cust-A", "store-1", 1, false},
		{"expired", func(c *models.CustomerPlayCredit) { c.ExpiresAt = now.Add(-time.Hour) }, "cust-A", "store-1", 1, false},
		{"paket khusus store lain", func(c *models.CustomerPlayCredit) {
			c.Package.ApplyToAllStores = false
			c.Package.PackageStores = []models.PlayCreditsPackageStore{{StoreID: "store-2"}}
		}, "cust-A", "store-1", 1, false},
		{"paket khusus store ini", func(c *models.CustomerPlayCredit) {
			c.Package.ApplyToAllStores = false
			c.Package.PackageStores = []models.PlayCreditsPackageStore{{StoreID: "store-1"}}
		}, "cust-A", "store-1", 1, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := base()
			tc.mutate(c)
			err := checkCreditUsable(c, tc.cust, tc.store, tc.hours, now)
			if (err == nil) != tc.ok {
				t.Errorf("want ok=%v, got err=%v", tc.ok, err)
			}
		})
	}
}
