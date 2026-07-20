package service

import (
	"testing"

	"game_lounge_be/models"
)

// ── parseTimeToMinutes ────────────────────────────────────────

func TestParseTimeToMinutes(t *testing.T) {
	cases := []struct {
		input string
		want  int
	}{
		{"00:00", 0},
		{"01:00", 60},
		{"10:30", 630},
		{"16:00", 960},
		{"23:59", 1439},
		{"00:00:00", 0},       // MySQL TIME format (HH:MM:SS)
		{"10:30:00", 630},     // MySQL TIME with seconds
		{"", 0},               // empty → 0
	}
	for _, c := range cases {
		got := parseTimeToMinutes(c.input)
		if got != c.want {
			t.Errorf("parseTimeToMinutes(%q) = %d, want %d", c.input, got, c.want)
		}
	}
}

// ── minutesToTime ─────────────────────────────────────────────

func TestMinutesToTime(t *testing.T) {
	cases := []struct {
		input int
		want  string
	}{
		{0, "00:00"},
		{60, "01:00"},
		{630, "10:30"},
		{960, "16:00"},
		{1439, "23:59"},
		{1440, "00:00"}, // wrap modulo 24h
		{1500, "01:00"}, // 1440+60
	}
	for _, c := range cases {
		got := minutesToTime(c.input)
		if got != c.want {
			t.Errorf("minutesToTime(%d) = %q, want %q", c.input, got, c.want)
		}
	}
}

// ── formatHours ───────────────────────────────────────────────

func TestFormatHours(t *testing.T) {
	cases := []struct {
		input float64
		want  string
	}{
		{1.0, "1"},
		{2.0, "2"},
		{1.5, "1.5"},
		{0.5, "0.5"},
		{3.0, "3"},
		{2.5, "2.5"},
	}
	for _, c := range cases {
		got := formatHours(c.input)
		if got != c.want {
			t.Errorf("formatHours(%v) = %q, want %q", c.input, got, c.want)
		}
	}
}

// ── intMax / intMin ───────────────────────────────────────────

func TestIntMax(t *testing.T) {
	cases := []struct{ a, b, want int }{
		{5, 3, 5},
		{3, 5, 5},
		{4, 4, 4},
		{-1, 0, 0},
	}
	for _, c := range cases {
		if got := intMax(c.a, c.b); got != c.want {
			t.Errorf("intMax(%d,%d) = %d, want %d", c.a, c.b, got, c.want)
		}
	}
}

func TestIntMin(t *testing.T) {
	cases := []struct{ a, b, want int }{
		{5, 3, 3},
		{3, 5, 3},
		{4, 4, 4},
		{-1, 0, -1},
	}
	for _, c := range cases {
		if got := intMin(c.a, c.b); got != c.want {
			t.Errorf("intMin(%d,%d) = %d, want %d", c.a, c.b, got, c.want)
		}
	}
}

// ── splitByHappyHour ──────────────────────────────────────────

func scheduleOf(start, end string) models.StoreHappyHourSchedule {
	return models.StoreHappyHourSchedule{StartTime: start, EndTime: end}
}

func TestSplitByHappyHour_NoSchedule(t *testing.T) {
	// Tanpa jadwal HH → seluruh segmen menjadi 1 slot normal
	slots := splitByHappyHour(600, 840, nil) // 10:00 - 14:00
	if len(slots) != 1 {
		t.Fatalf("want 1 slot, got %d", len(slots))
	}
	if slots[0].IsHH {
		t.Error("slot harus normal, bukan HH")
	}
	if slots[0].Start != 600 || slots[0].End != 840 {
		t.Errorf("slot = [%d-%d], want [600-840]", slots[0].Start, slots[0].End)
	}
}

func TestSplitByHappyHour_HHCoversAll(t *testing.T) {
	// HH menutupi seluruh segmen → 1 slot HH
	schedules := []models.StoreHappyHourSchedule{scheduleOf("10:00", "14:00")}
	slots := splitByHappyHour(600, 840, schedules)
	if len(slots) != 1 {
		t.Fatalf("want 1 slot, got %d", len(slots))
	}
	if !slots[0].IsHH {
		t.Error("slot harus HH")
	}
}

func TestSplitByHappyHour_HHInMiddle(t *testing.T) {
	// Segmen 10:00-22:00, HH 16:00-20:00
	// Expect: normal[10-16], HH[16-20], normal[20-22]
	schedules := []models.StoreHappyHourSchedule{scheduleOf("16:00", "20:00")}
	slots := splitByHappyHour(600, 1320, schedules) // 10:00=600, 22:00=1320
	if len(slots) != 3 {
		t.Fatalf("want 3 slots, got %d: %+v", len(slots), slots)
	}
	checkSlot(t, slots[0], 600, 960, false)  // 10:00 - 16:00 normal
	checkSlot(t, slots[1], 960, 1200, true)  // 16:00 - 20:00 HH
	checkSlot(t, slots[2], 1200, 1320, false) // 20:00 - 22:00 normal
}

func TestSplitByHappyHour_HHAtStart(t *testing.T) {
	// Segmen 14:00-22:00, HH 14:00-16:00
	// Expect: HH[14-16], normal[16-22]
	schedules := []models.StoreHappyHourSchedule{scheduleOf("14:00", "16:00")}
	slots := splitByHappyHour(840, 1320, schedules) // 14:00=840, 22:00=1320
	if len(slots) != 2 {
		t.Fatalf("want 2 slots, got %d: %+v", len(slots), slots)
	}
	checkSlot(t, slots[0], 840, 960, true)   // 14:00-16:00 HH
	checkSlot(t, slots[1], 960, 1320, false) // 16:00-22:00 normal
}

func TestSplitByHappyHour_HHAtEnd(t *testing.T) {
	// Segmen 14:00-20:00, HH 16:00-20:00
	// Expect: normal[14-16], HH[16-20]
	schedules := []models.StoreHappyHourSchedule{scheduleOf("16:00", "20:00")}
	slots := splitByHappyHour(840, 1200, schedules) // 14:00=840, 20:00=1200
	if len(slots) != 2 {
		t.Fatalf("want 2 slots, got %d: %+v", len(slots), slots)
	}
	checkSlot(t, slots[0], 840, 960, false)  // 14:00-16:00 normal
	checkSlot(t, slots[1], 960, 1200, true)  // 16:00-20:00 HH
}

func TestSplitByHappyHour_HHOutsideSegment(t *testing.T) {
	// HH 08:00-10:00 tidak overlap dengan segmen 14:00-20:00
	schedules := []models.StoreHappyHourSchedule{scheduleOf("08:00", "10:00")}
	slots := splitByHappyHour(840, 1200, schedules)
	if len(slots) != 1 {
		t.Fatalf("want 1 slot, got %d", len(slots))
	}
	if slots[0].IsHH {
		t.Error("slot harus normal (HH di luar segmen)")
	}
}

func TestSplitByHappyHour_MultipleHH(t *testing.T) {
	// Segmen 08:00-22:00 dengan 2 window HH: 10:00-12:00 dan 16:00-18:00
	// Expect: normal[8-10], HH[10-12], normal[12-16], HH[16-18], normal[18-22]
	schedules := []models.StoreHappyHourSchedule{
		scheduleOf("10:00", "12:00"),
		scheduleOf("16:00", "18:00"),
	}
	slots := splitByHappyHour(480, 1320, schedules) // 08:00=480, 22:00=1320
	if len(slots) != 5 {
		t.Fatalf("want 5 slots, got %d: %+v", len(slots), slots)
	}
	checkSlot(t, slots[0], 480, 600, false)  // 08:00-10:00
	checkSlot(t, slots[1], 600, 720, true)   // 10:00-12:00 HH
	checkSlot(t, slots[2], 720, 960, false)  // 12:00-16:00
	checkSlot(t, slots[3], 960, 1080, true)  // 16:00-18:00 HH
	checkSlot(t, slots[4], 1080, 1320, false) // 18:00-22:00
}

func checkSlot(t *testing.T, s timeSlot, start, end int, isHH bool) {
	t.Helper()
	if s.Start != start || s.End != end || s.IsHH != isHH {
		t.Errorf("slot = {%d-%d isHH=%v}, want {%d-%d isHH=%v}",
			s.Start, s.End, s.IsHH, start, end, isHH)
	}
}

// ── findCheapestPackage ───────────────────────────────────────

func makePricing(edge2h, edge4h string) *models.StorePricing {
	return &models.StorePricing{EdgeCase2h: edge2h, EdgeCase4h: edge4h}
}

func makePackages(durationPrices map[uint]float64) []models.StorePackagePrice {
	var pkgs []models.StorePackagePrice
	for dur, price := range durationPrices {
		pkgs = append(pkgs, models.StorePackagePrice{DurationHours: dur, Price: price})
	}
	return pkgs
}

func TestFindCheapestPackage_Basic(t *testing.T) {
	pkgs := makePackages(map[uint]float64{
		1: 50000,
		2: 90000,
		3: 120000,
	})
	pricing := makePricing("two_x_1h", "3h_plus_1h")

	// 1 jam → Paket 1 Jam
	price, _ := findCheapestPackage(1, pkgs, pricing)
	if price != 50000 {
		t.Errorf("1 jam: want 50000, got %.0f", price)
	}

	// 3 jam → Paket 3 Jam (120000, lebih murah dari 3× Paket 1 Jam = 150000)
	price, _ = findCheapestPackage(3, pkgs, pricing)
	if price != 120000 {
		t.Errorf("3 jam: want 120000, got %.0f", price)
	}
}

func TestFindCheapestPackage_EdgeCase2h_TwoX1h(t *testing.T) {
	pkgs := makePackages(map[uint]float64{1: 50000, 3: 120000})
	pricing := makePricing("two_x_1h", "3h_plus_1h")
	price, desc := findCheapestPackage(2, pkgs, pricing)
	if price != 100000 {
		t.Errorf("edge 2h two_x_1h: want 100000, got %.0f", price)
	}
	if desc != "2× Paket 1 Jam" {
		t.Errorf("desc: want '2× Paket 1 Jam', got %q", desc)
	}
}

func TestFindCheapestPackage_EdgeCase2h_Force3h(t *testing.T) {
	pkgs := makePackages(map[uint]float64{1: 50000, 3: 120000})
	pricing := makePricing("force_3h", "3h_plus_1h")
	price, desc := findCheapestPackage(2, pkgs, pricing)
	if price != 120000 {
		t.Errorf("edge 2h force_3h: want 120000, got %.0f", price)
	}
	if desc != "Paket 3 Jam" {
		t.Errorf("desc: want 'Paket 3 Jam', got %q", desc)
	}
}

func TestFindCheapestPackage_EdgeCase4h_3hPlus1h(t *testing.T) {
	pkgs := makePackages(map[uint]float64{1: 50000, 3: 120000, 5: 200000})
	pricing := makePricing("two_x_1h", "3h_plus_1h")
	price, desc := findCheapestPackage(4, pkgs, pricing)
	if price != 170000 { // 120000 + 50000
		t.Errorf("edge 4h 3h_plus_1h: want 170000, got %.0f", price)
	}
	if desc != "Paket 3 Jam + Paket 1 Jam" {
		t.Errorf("desc: want 'Paket 3 Jam + Paket 1 Jam', got %q", desc)
	}
}

func TestFindCheapestPackage_EdgeCase4h_Force5h(t *testing.T) {
	pkgs := makePackages(map[uint]float64{1: 50000, 3: 120000, 5: 200000})
	pricing := makePricing("two_x_1h", "force_5h")
	price, desc := findCheapestPackage(4, pkgs, pricing)
	if price != 200000 {
		t.Errorf("edge 4h force_5h: want 200000, got %.0f", price)
	}
	if desc != "Paket 5 Jam" {
		t.Errorf("desc: want 'Paket 5 Jam', got %q", desc)
	}
}

func TestFindCheapestPackage_GreedyOptimal(t *testing.T) {
	// 5 jam dengan paket {1:50000, 3:120000}:
	// Greedy pilih 3 jam (40k/jam) → sisa 2 jam → pilih 1 jam dua kali (50k/jam)
	// Total = 120000 + 50000 + 50000 = 220000
	// (lebih murah dari 5× Paket 1 Jam = 250000)
	pkgs := makePackages(map[uint]float64{1: 50000, 3: 120000})
	pricing := makePricing("two_x_1h", "3h_plus_1h")
	price, _ := findCheapestPackage(5, pkgs, pricing)
	if price != 220000 {
		t.Errorf("5 jam greedy: want 220000, got %.0f", price)
	}
}

func TestFindCheapestPackage_GreedyWith5hPackage(t *testing.T) {
	// 5 jam dengan paket {1:50000, 3:120000, 5:200000}:
	// Greedy: 5h (40k/jam) vs 3h (40k/jam) — 5h lebih hemat
	// 5h = 200000
	pkgs := makePackages(map[uint]float64{1: 50000, 3: 120000, 5: 200000})
	pricing := makePricing("two_x_1h", "3h_plus_1h")
	price, _ := findCheapestPackage(5, pkgs, pricing)
	if price != 200000 {
		t.Errorf("5 jam greedy with 5h pkg: want 200000, got %.0f", price)
	}
}

// ── priceNormalSegment ────────────────────────────────────────

func TestPriceNormalSegment_NoHH(t *testing.T) {
	pkgs := makePackages(map[uint]float64{1: 50000, 3: 120000})
	pricing := makePricing("two_x_1h", "3h_plus_1h")

	// 3 jam normal, tidak ada HH → Paket 3 Jam
	price, items := priceNormalSegment(600, 780, false, pricing, pkgs, nil, nil) // 10:00-13:00
	if price != 120000 {
		t.Errorf("want 120000, got %.0f", price)
	}
	if len(items) != 1 {
		t.Fatalf("want 1 breakdown item, got %d", len(items))
	}
	if items[0].Type != "Normal Hour" {
		t.Errorf("type: want 'Normal Hour', got %q", items[0].Type)
	}
	if items[0].TimeRange != "10:00 - 13:00" {
		t.Errorf("time range: want '10:00 - 13:00', got %q", items[0].TimeRange)
	}
}

func TestPriceNormalSegment_WithHH_SplitMiddle(t *testing.T) {
	pkgs := makePackages(map[uint]float64{1: 50000, 3: 120000})
	pricing := &models.StorePricing{
		IsHappyHourEnabled: true,
		EdgeCase2h:         "two_x_1h",
		EdgeCase4h:         "3h_plus_1h",
	}
	hhPrice := &models.StoreHappyHourPrice{PricePerHour: 30000}
	schedules := []models.StoreHappyHourSchedule{scheduleOf("16:00", "20:00")}

	// Segmen 14:00-22:00: normal[14-16]=2jam, HH[16-20]=4jam, normal[20-22]=2jam
	price, items := priceNormalSegment(840, 1320, true, pricing, pkgs, schedules, hhPrice)

	// Normal 14-16 = 2× Paket 1 Jam = 100000
	// HH 16-20 = 4jam × 30000 = 120000
	// Normal 20-22 = 2× Paket 1 Jam = 100000
	// Total = 320000
	if price != 320000 {
		t.Errorf("want 320000, got %.0f", price)
	}
	if len(items) != 3 {
		t.Fatalf("want 3 breakdown items, got %d: %+v", len(items), items)
	}
	if items[0].Type != "Normal Hour" {
		t.Errorf("items[0] type: want 'Normal Hour', got %q", items[0].Type)
	}
	if items[1].Type != "Happy Hour" {
		t.Errorf("items[1] type: want 'Happy Hour', got %q", items[1].Type)
	}
	if items[2].Type != "Normal Hour" {
		t.Errorf("items[2] type: want 'Normal Hour', got %q", items[2].Type)
	}
}

func TestPriceNormalSegment_IsWeekdayFalse_IgnoresHH(t *testing.T) {
	// Meskipun ada jadwal HH, jika isWeekday=false maka HH tidak berlaku
	pkgs := makePackages(map[uint]float64{1: 50000, 2: 90000})
	pricing := &models.StorePricing{
		IsHappyHourEnabled: true,
		EdgeCase2h:         "two_x_1h", // 2 jam → 2× Paket 1 Jam
		EdgeCase4h:         "3h_plus_1h",
	}
	hhPrice := &models.StoreHappyHourPrice{PricePerHour: 30000}
	schedules := []models.StoreHappyHourSchedule{scheduleOf("16:00", "20:00")}

	// Segmen 16:00-18:00 (2 jam, masuk HH window), tapi isWeekday=false → pure normal
	// EdgeCase2h = two_x_1h → 2× Paket 1 Jam (50000) = 100000
	price, items := priceNormalSegment(960, 1080, false, pricing, pkgs, schedules, hhPrice)
	if len(items) != 1 || items[0].Type != "Normal Hour" {
		t.Errorf("harus Normal Hour saat isWeekday=false, got items=%+v", items)
	}
	if price != 100000 {
		t.Errorf("want 100000 (2× Paket 1 Jam via EdgeCase2h=two_x_1h), got %.0f", price)
	}
}

func TestPriceNormalSegment_EmptySegment(t *testing.T) {
	pkgs := makePackages(map[uint]float64{1: 50000})
	pricing := makePricing("two_x_1h", "3h_plus_1h")
	price, items := priceNormalSegment(600, 600, false, pricing, pkgs, nil, nil)
	if price != 0 || len(items) != 0 {
		t.Errorf("segmen kosong harus return 0, got price=%.0f items=%d", price, len(items))
	}
}
