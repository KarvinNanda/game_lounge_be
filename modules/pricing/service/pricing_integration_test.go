package service

import (
	"testing"
	"time"

	"game_lounge_be/config"
	"game_lounge_be/models"
	"game_lounge_be/modules/pricing/dto"

	"github.com/google/uuid"
)

// Tanggal tetap yang dipakai integration test.
// 2026-05-16 = Sabtu  (weekend → HH tidak berlaku)
// 2026-05-18 = Senin  (weekday → HH berlaku)
var (
	saturdayDate = "2026-05-16"
	mondayDate   = "2026-05-18"
)

// ── CalculatePrice ────────────────────────────────────────────

func TestCalculatePrice_NormalHoursWeekend(t *testing.T) {
	skipIfNoDB(t)

	// Sabtu 10:00-13:00 (3 jam) → tidak ada HH → Paket 3 Jam = 120000
	resp, err := CalculatePrice(dto.CalculatePriceRequest{
		StoreID:        testStoreID,
		RoomTemplateID: testRoomTplID,
		BookingDate:    saturdayDate,
		StartTime:      "10:00",
		EndTime:        "13:00",
	})
	if err != nil {
		t.Fatalf("CalculatePrice error: %v", err)
	}
	if resp.BasePrice != 120000 {
		t.Errorf("base_price: want 120000, got %.0f", resp.BasePrice)
	}
	if resp.TotalHours != 3 {
		t.Errorf("total_hours: want 3, got %.1f", resp.TotalHours)
	}
	if resp.HasFlashSale {
		t.Error("has_flash_sale harus false di weekend tanpa FS")
	}
	if len(resp.Breakdown) != 1 || resp.Breakdown[0].Type != "Normal Hour" {
		t.Errorf("breakdown harus 1 item Normal Hour, got %+v", resp.Breakdown)
	}
}

func TestCalculatePrice_NormalHoursWeekdayBeforeHH(t *testing.T) {
	skipIfNoDB(t)

	// Senin 10:00-13:00 (3 jam, sebelum HH 16:00) → Paket 3 Jam = 120000
	resp, err := CalculatePrice(dto.CalculatePriceRequest{
		StoreID:        testStoreID,
		RoomTemplateID: testRoomTplID,
		BookingDate:    mondayDate,
		StartTime:      "10:00",
		EndTime:        "13:00",
	})
	if err != nil {
		t.Fatalf("CalculatePrice error: %v", err)
	}
	if resp.BasePrice != 120000 {
		t.Errorf("base_price: want 120000, got %.0f", resp.BasePrice)
	}
}

func TestCalculatePrice_WithHappyHour(t *testing.T) {
	skipIfNoDB(t)

	// Senin 14:00-20:00 (6 jam):
	// • Normal 14:00-16:00 → 2 jam → EdgeCase2h two_x_1h → 2×50000 = 100000
	// • HH    16:00-20:00 → 4 jam × 30000               = 120000
	// Total                                               = 220000
	resp, err := CalculatePrice(dto.CalculatePriceRequest{
		StoreID:        testStoreID,
		RoomTemplateID: testRoomTplID,
		BookingDate:    mondayDate,
		StartTime:      "14:00",
		EndTime:        "20:00",
	})
	if err != nil {
		t.Fatalf("CalculatePrice error: %v", err)
	}
	if resp.BasePrice != 220000 {
		t.Errorf("base_price: want 220000, got %.0f", resp.BasePrice)
	}
	if len(resp.Breakdown) != 2 {
		t.Fatalf("want 2 breakdown items (Normal + HH), got %d: %+v", len(resp.Breakdown), resp.Breakdown)
	}
	if resp.Breakdown[0].Type != "Normal Hour" {
		t.Errorf("breakdown[0] type: want 'Normal Hour', got %q", resp.Breakdown[0].Type)
	}
	if resp.Breakdown[1].Type != "Happy Hour" {
		t.Errorf("breakdown[1] type: want 'Happy Hour', got %q", resp.Breakdown[1].Type)
	}
}

func TestCalculatePrice_FlashSaleAtEnd(t *testing.T) {
	skipIfNoDB(t)

	// Buat flash sale: Senin 16:00-20:00, harga 25000/jam
	// (tanggal range mencakup mondayDate 2026-05-18)
	fsID := uuid.NewString()
	actor := "test"
	fs := models.StoreFlashSale{
		ID:             fsID,
		StoreID:        testStoreID,
		RoomTemplateID: testRoomTplID,
		Name:           "TEST Flash Sale",
		PricePerHour:   25000,
		DateFrom:       time.Date(2026, 1, 1, 0, 0, 0, 0, time.Local),
		DateTo:         time.Date(2026, 12, 31, 0, 0, 0, 0, time.Local),
		TimeFrom:       "16:00",
		TimeTo:         "20:00",
		IsActive:       true,
		CreatedBy:      &actor,
	}
	config.DB.Create(&fs)
	defer config.DB.Unscoped().Delete(&models.StoreFlashSale{}, "id = ?", fsID)

	// Booking Senin 14:00-20:00:
	// • Normal 14:00-16:00 → 2 jam → EdgeCase2h two_x_1h → 100000
	// • FS     16:00-20:00 → 4 jam × 25000                = 100000
	// Total                                                = 200000
	resp, err := CalculatePrice(dto.CalculatePriceRequest{
		StoreID:        testStoreID,
		RoomTemplateID: testRoomTplID,
		BookingDate:    mondayDate,
		StartTime:      "14:00",
		EndTime:        "20:00",
	})
	if err != nil {
		t.Fatalf("CalculatePrice error: %v", err)
	}
	if resp.BasePrice != 200000 {
		t.Errorf("base_price: want 200000, got %.0f", resp.BasePrice)
	}
	if !resp.HasFlashSale {
		t.Error("has_flash_sale harus true")
	}
	if len(resp.Breakdown) != 2 {
		t.Fatalf("want 2 breakdown items, got %d: %+v", len(resp.Breakdown), resp.Breakdown)
	}
	if resp.Breakdown[0].Type != "Normal Hour" {
		t.Errorf("breakdown[0]: want 'Normal Hour', got %q", resp.Breakdown[0].Type)
	}
	if resp.Breakdown[1].Type != "Flash Sale" {
		t.Errorf("breakdown[1]: want 'Flash Sale', got %q", resp.Breakdown[1].Type)
	}
}

func TestCalculatePrice_FlashSaleAtStart(t *testing.T) {
	skipIfNoDB(t)

	// Buat flash sale: 10:00-14:00, 25000/jam
	fsID := uuid.NewString()
	actor := "test"
	fs := models.StoreFlashSale{
		ID:             fsID,
		StoreID:        testStoreID,
		RoomTemplateID: testRoomTplID,
		Name:           "TEST Flash Sale Start",
		PricePerHour:   25000,
		DateFrom:       time.Date(2026, 1, 1, 0, 0, 0, 0, time.Local),
		DateTo:         time.Date(2026, 12, 31, 0, 0, 0, 0, time.Local),
		TimeFrom:       "10:00",
		TimeTo:         "14:00",
		IsActive:       true,
		CreatedBy:      &actor,
	}
	config.DB.Create(&fs)
	defer config.DB.Unscoped().Delete(&models.StoreFlashSale{}, "id = ?", fsID)

	// Booking Senin 10:00-17:00:
	// • FS     10:00-14:00 → 4 jam × 25000 = 100000
	// • Normal 14:00-16:00 → 2 jam          = 100000  (EdgeCase2h two_x_1h)
	// • HH     16:00-17:00 → 1 jam × 30000  =  30000
	// Total                                  = 230000
	resp, err := CalculatePrice(dto.CalculatePriceRequest{
		StoreID:        testStoreID,
		RoomTemplateID: testRoomTplID,
		BookingDate:    mondayDate,
		StartTime:      "10:00",
		EndTime:        "17:00",
	})
	if err != nil {
		t.Fatalf("CalculatePrice error: %v", err)
	}
	if resp.BasePrice != 230000 {
		t.Errorf("base_price: want 230000, got %.0f", resp.BasePrice)
	}
	if !resp.HasFlashSale {
		t.Error("has_flash_sale harus true")
	}
	// FS di awal → breakdown: [Flash Sale, Normal Hour, Happy Hour]
	if len(resp.Breakdown) != 3 {
		t.Fatalf("want 3 breakdown items, got %d: %+v", len(resp.Breakdown), resp.Breakdown)
	}
	if resp.Breakdown[0].Type != "Flash Sale" {
		t.Errorf("breakdown[0]: want 'Flash Sale', got %q", resp.Breakdown[0].Type)
	}
}

func TestCalculatePrice_PricingNotFound(t *testing.T) {
	skipIfNoDB(t)

	// Store ID yang tidak ada → error
	_, err := CalculatePrice(dto.CalculatePriceRequest{
		StoreID:        "store-tidak-ada-xxxx",
		RoomTemplateID: testRoomTplID,
		BookingDate:    saturdayDate,
		StartTime:      "10:00",
		EndTime:        "12:00",
	})
	if err == nil {
		t.Error("harus return error untuk store yang tidak ada")
	}
}

// ── isWeekdayForPricing ───────────────────────────────────────

func TestIsWeekdayForPricing_Senin(t *testing.T) {
	skipIfNoDB(t)
	date := time.Date(2026, 5, 18, 0, 0, 0, 0, time.Local) // Senin
	if !isWeekdayForPricing(date, testStoreID) {
		t.Error("2026-05-18 (Senin) harus return true")
	}
}

func TestIsWeekdayForPricing_Sabtu(t *testing.T) {
	skipIfNoDB(t)
	date := time.Date(2026, 5, 16, 0, 0, 0, 0, time.Local) // Sabtu
	if isWeekdayForPricing(date, testStoreID) {
		t.Error("2026-05-16 (Sabtu) harus return false")
	}
}

func TestIsWeekdayForPricing_GlobalHoliday(t *testing.T) {
	skipIfNoDB(t)

	// Buat global holiday pada Senin di far future agar tidak konflik
	// 2099-06-15 = Senin (Selasa 2099-06-16, dst.)
	holidayDate := time.Date(2099, 6, 15, 0, 0, 0, 0, time.Local)
	actor := "test"

	// Bersihkan dulu kalau ada sisa dari run sebelumnya
	config.DB.Unscoped().Where("DATE(date) = ?", "2099-06-15").Delete(&models.GlobalHolidaySchedule{})

	h := models.GlobalHolidaySchedule{
		Date:      holidayDate,
		Name:      "TEST Global Holiday",
		OpenTime:  "10:00",
		CloseTime: "22:00",
		CreatedBy: &actor,
	}
	if err := config.DB.Create(&h).Error; err != nil {
		t.Fatalf("gagal buat global holiday fixture: %v", err)
	}
	defer config.DB.Unscoped().Delete(&h)

	// Meskipun hari itu Senin (weekday), global holiday membuatnya false
	if isWeekdayForPricing(holidayDate, testStoreID) {
		t.Error("hari dengan global holiday harus return false")
	}
}

func TestIsWeekdayForPricing_StoreHoliday(t *testing.T) {
	skipIfNoDB(t)

	// Buat store holiday pada Senin
	holidayDate := time.Date(2026, 5, 18, 0, 0, 0, 0, time.Local)
	actor := "test"
	sh := models.StoreHolidaySchedule{
		StoreID:   testStoreID,
		Date:      holidayDate,
		OpenTime:  "10:00",
		CloseTime: "22:00",
		CreatedBy: &actor,
	}
	if err := config.DB.Create(&sh).Error; err != nil {
		t.Fatalf("gagal buat store holiday fixture: %v", err)
	}
	defer config.DB.Unscoped().Delete(&sh)

	// Senin tapi ada store holiday → false
	if isWeekdayForPricing(holidayDate, testStoreID) {
		t.Error("Senin dengan store holiday harus return false")
	}
}
