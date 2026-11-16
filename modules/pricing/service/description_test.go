package service

import (
	"testing"

	"game_lounge_be/models"
	"game_lounge_be/money"
)

// The simulator shows Description as is, so amounts use thousand separators.
func TestPriceNormalSegment_HappyHourDescriptionUsesThousands(t *testing.T) {
	pricing := makePricing("two_x_1h", "3h_plus_1h")
	pricing.IsHappyHourEnabled = true
	pkgs := makePackages(map[uint]money.Rupiah{1: 50000})
	schedules := []models.StoreHappyHourSchedule{scheduleOf("14:00", "16:00")}
	hhPrice := &models.StoreHappyHourPrice{PricePerHour: 31000}

	_, items := priceNormalSegment(14*60, 16*60, true, pricing, pkgs, schedules, hhPrice)
	if len(items) != 1 || items[0].Description != "2 Jam × Rp 31.000" {
		t.Fatalf("want one item '2 Jam × Rp 31.000', got %+v", items)
	}
}
