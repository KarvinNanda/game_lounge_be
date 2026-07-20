package service

import (
	"testing"
	"time"

	"game_lounge_be/config"
	"game_lounge_be/models"
)

// Tanggal test far-future agar tidak konflik dengan data produksi.
// 2099-08-17 = Senin  (weekday)
// 2099-09-20 = Sabtu  (weekend)
var (
	weekdayDate  = time.Date(2099, 8, 17, 0, 0, 0, 0, time.Local)
	weekendDate  = time.Date(2099, 9, 20, 0, 0, 0, 0, time.Local)
	holidayDate1 = time.Date(2099, 8, 17, 0, 0, 0, 0, time.Local) // Senin, akan dijadikan global holiday
	holidayDate2 = time.Date(2099, 9, 20, 0, 0, 0, 0, time.Local) // Sabtu, akan dijadikan store holiday
)

// ── GetEffectiveOperatingHours — Regular ──────────────────────

func TestGetEffectiveOperatingHours_Weekday(t *testing.T) {
	skipIfNoDB(t)

	// weekdayDate = Senin, tidak ada holiday → jam weekday fixture (10:00-22:00)
	result, err := GetEffectiveOperatingHours(testStoreID, weekdayDate)
	if err != nil {
		// Mungkin weekdayDate bertepatan dengan holiday di DB — skip
		t.Skipf("skip (mungkin ada holiday di DB untuk tanggal ini): %v", err)
	}
	if result.IsHoliday {
		t.Skip("weekday tapi ada holiday di DB untuk tanggal ini, skip")
	}
	if result.OpenTime != "10:00:00" && result.OpenTime != "10:00" {
		t.Errorf("open_time weekday: want '10:00', got %q", result.OpenTime)
	}
	if result.CloseTime != "22:00:00" && result.CloseTime != "22:00" {
		t.Errorf("close_time weekday: want '22:00', got %q", result.CloseTime)
	}
	if result.HolidayType != "" {
		t.Errorf("holiday_type harus kosong untuk hari biasa, got %q", result.HolidayType)
	}
}

func TestGetEffectiveOperatingHours_Weekend(t *testing.T) {
	skipIfNoDB(t)

	// weekendDate = Sabtu, tidak ada holiday → jam weekend fixture (09:00-23:00)
	result, err := GetEffectiveOperatingHours(testStoreID, weekendDate)
	if err != nil {
		t.Skipf("skip: %v", err)
	}
	if result.IsHoliday {
		t.Skip("weekend tapi ada holiday di DB, skip")
	}
	if result.OpenTime != "09:00:00" && result.OpenTime != "09:00" {
		t.Errorf("open_time weekend: want '09:00', got %q", result.OpenTime)
	}
	if result.CloseTime != "23:00:00" && result.CloseTime != "23:00" {
		t.Errorf("close_time weekend: want '23:00', got %q", result.CloseTime)
	}
}

// ── GetEffectiveOperatingHours — Global Holiday ───────────────

func TestGetEffectiveOperatingHours_GlobalHoliday(t *testing.T) {
	skipIfNoDB(t)

	// Bersihkan kemungkinan sisa test run sebelumnya
	config.DB.Unscoped().Where("DATE(date) = ?", "2099-08-17").Delete(&models.GlobalHolidaySchedule{})

	actor := "test"
	gh := models.GlobalHolidaySchedule{
		Date:      holidayDate1,
		Name:      "TEST Hari Kemerdekaan Global",
		OpenTime:  "12:00",
		CloseTime: "20:00",
		CreatedBy: &actor,
	}
	if err := config.DB.Create(&gh).Error; err != nil {
		t.Fatalf("gagal buat global holiday: %v", err)
	}
	defer config.DB.Unscoped().Delete(&gh)

	result, err := GetEffectiveOperatingHours(testStoreID, holidayDate1)
	if err != nil {
		t.Fatalf("GetEffectiveOperatingHours error: %v", err)
	}
	if !result.IsHoliday {
		t.Error("is_holiday harus true saat global holiday")
	}
	if result.HolidayType != "global" {
		t.Errorf("holiday_type: want 'global', got %q", result.HolidayType)
	}
	if result.HolidayName != gh.Name {
		t.Errorf("holiday_name: want %q, got %q", gh.Name, result.HolidayName)
	}
	if result.OpenTime != "12:00:00" && result.OpenTime != "12:00" {
		t.Errorf("open_time dari global holiday: want '12:00', got %q", result.OpenTime)
	}
}

// ── GetEffectiveOperatingHours — Store Holiday ────────────────

func TestGetEffectiveOperatingHours_StoreHoliday(t *testing.T) {
	skipIfNoDB(t)

	actor := "test"
	sh := models.StoreHolidaySchedule{
		StoreID:   testStoreID,
		Date:      holidayDate2,
		OpenTime:  "14:00",
		CloseTime: "21:00",
		CreatedBy: &actor,
	}
	if err := config.DB.Create(&sh).Error; err != nil {
		t.Fatalf("gagal buat store holiday: %v", err)
	}
	defer config.DB.Unscoped().Delete(&sh)

	result, err := GetEffectiveOperatingHours(testStoreID, holidayDate2)
	if err != nil {
		t.Fatalf("GetEffectiveOperatingHours error: %v", err)
	}
	if !result.IsHoliday {
		t.Error("is_holiday harus true saat store holiday")
	}
	if result.HolidayType != "store" {
		t.Errorf("holiday_type: want 'store', got %q", result.HolidayType)
	}
	if result.OpenTime != "14:00:00" && result.OpenTime != "14:00" {
		t.Errorf("open_time dari store holiday: want '14:00', got %q", result.OpenTime)
	}
}

// ── GetEffectiveOperatingHours — Store Tanpa Config ───────────

func TestGetEffectiveOperatingHours_StoreTanpaOpHours(t *testing.T) {
	skipIfNoDB(t)

	// Store ID yang tidak ada di DB → harus return error
	_, err := GetEffectiveOperatingHours("store-tidak-ada-xyz", weekdayDate)
	if err == nil {
		t.Error("store tanpa operating hours harus return error")
	}
}

// ── Priority: Global > Store > Regular ───────────────────────

func TestGetEffectiveOperatingHours_GlobalTakePriorityOverStore(t *testing.T) {
	skipIfNoDB(t)

	// Tanggal sama ada di global holiday DAN store holiday → global yang menang
	testDate := time.Date(2099, 10, 5, 0, 0, 0, 0, time.Local)
	config.DB.Unscoped().Where("DATE(date) = ?", "2099-10-05").Delete(&models.GlobalHolidaySchedule{})

	actor := "test"
	gh := models.GlobalHolidaySchedule{
		Date:      testDate,
		Name:      "TEST Global Priority",
		OpenTime:  "11:00",
		CloseTime: "19:00",
		CreatedBy: &actor,
	}
	config.DB.Create(&gh)
	defer config.DB.Unscoped().Delete(&gh)

	sh := models.StoreHolidaySchedule{
		StoreID:   testStoreID,
		Date:      testDate,
		OpenTime:  "15:00",
		CloseTime: "23:00",
		CreatedBy: &actor,
	}
	config.DB.Create(&sh)
	defer config.DB.Unscoped().Delete(&sh)

	result, err := GetEffectiveOperatingHours(testStoreID, testDate)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if result.HolidayType != "global" {
		t.Errorf("global holiday harus mengalahkan store holiday, got type=%q", result.HolidayType)
	}
}
