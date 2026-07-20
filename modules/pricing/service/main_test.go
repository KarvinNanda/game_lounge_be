package service

import (
	"fmt"
	"os"
	"testing"
	"time"

	"game_lounge_be/config"
	"game_lounge_be/models"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// ── Shared state ──────────────────────────────────────────────

var (
	dbAvailable    bool
	testStoreID    string
	testRoomTplID  uint
)

// ── TestMain ──────────────────────────────────────────────────

func TestMain(m *testing.M) {
	dbAvailable = connectTestDB()
	if dbAvailable {
		setupPricingFixtures()
	}
	code := m.Run()
	if dbAvailable {
		teardownPricingFixtures()
	}
	os.Exit(code)
}

// connectTestDB mencoba koneksi ke DB dari .env (walk-up dari package dir).
// Mengembalikan false jika tidak bisa konek — integration test akan di-skip.
func connectTestDB() bool {
	for _, p := range []string{".env", "../.env", "../../.env", "../../../.env"} {
		_ = godotenv.Load(p)
	}
	getenv := func(key, fb string) string {
		if v := os.Getenv(key); v != "" {
			return v
		}
		return fb
	}
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		getenv("DB_USER", "root"),
		getenv("DB_PASSWORD", ""),
		getenv("DB_HOST", "127.0.0.1"),
		getenv("DB_PORT", "3306"),
		getenv("DB_NAME", "game_lounge_db"),
	)
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return false
	}
	config.DB = db
	return true
}

func skipIfNoDB(t *testing.T) {
	t.Helper()
	if !dbAvailable {
		t.Skip("skip: koneksi DB tidak tersedia")
	}
}

// ── Fixtures ──────────────────────────────────────────────────

// setupPricingFixtures membuat data test: store, room template, pricing config,
// package prices (1h & 3h), happy hour schedule (16:00-20:00), happy hour price.
func setupPricingFixtures() {
	actor := "test_setup"
	testStoreID = uuid.NewString()

	// Store
	config.DB.Create(&models.Store{
		ID:      testStoreID,
		Name:    "TEST Integration Pricing Store",
		Address: "Integration Test",
		Status:  "active",
	})

	// Room template (auto-increment ID)
	rt := models.RoomTemplate{
		Name:        "TEST Integration Room",
		CapacityMax: 4,
	}
	config.DB.Create(&rt)
	testRoomTplID = rt.ID

	// Pricing config
	config.DB.Create(&models.StorePricing{
		StoreID:            testStoreID,
		IsHappyHourEnabled: true,
		IsMixedTimeEnabled: true,
		EdgeCase2h:         "two_x_1h",
		EdgeCase4h:         "3h_plus_1h",
		CreatedBy:          &actor,
	})

	// Package prices: 1h & 3h
	config.DB.Create(&models.StorePackagePrice{
		StoreID:        testStoreID,
		RoomTemplateID: testRoomTplID,
		DurationHours:  1,
		Price:          50000,
		CreatedBy:      &actor,
	})
	config.DB.Create(&models.StorePackagePrice{
		StoreID:        testStoreID,
		RoomTemplateID: testRoomTplID,
		DurationHours:  3,
		Price:          120000,
		CreatedBy:      &actor,
	})

	// Happy hour schedule 16:00–20:00
	config.DB.Create(&models.StoreHappyHourSchedule{
		StoreID:   testStoreID,
		StartTime: "16:00",
		EndTime:   "20:00",
		CreatedBy: &actor,
	})

	// Happy hour price: 30000/jam untuk room template ini
	config.DB.Create(&models.StoreHappyHourPrice{
		StoreID:        testStoreID,
		RoomTemplateID: testRoomTplID,
		PricePerHour:   30000,
		CreatedBy:      &actor,
	})
}

// teardownPricingFixtures hapus semua data test yang dibuat di setupPricingFixtures.
func teardownPricingFixtures() {
	now := time.Now()
	config.DB.Unscoped().Where("store_id = ?", testStoreID).Delete(&models.StoreHappyHourPrice{})
	config.DB.Unscoped().Where("store_id = ?", testStoreID).Delete(&models.StoreHappyHourSchedule{})
	config.DB.Unscoped().Where("store_id = ?", testStoreID).Delete(&models.StorePackagePrice{})
	config.DB.Unscoped().Where("store_id = ?", testStoreID).Delete(&models.StorePricing{})
	config.DB.Unscoped().Where("store_id = ?", testStoreID).Delete(&models.StoreFlashSale{})
	config.DB.Unscoped().Where("store_id = ?", testStoreID).Delete(&models.Store{})
	if testRoomTplID > 0 {
		config.DB.Unscoped().Delete(&models.RoomTemplate{}, testRoomTplID)
	}
	// Bersihkan global holiday test jika ada sisa
	config.DB.Unscoped().Where("DATE(date) = ? AND name LIKE 'TEST%'", "2099-06-15").
		Delete(&models.GlobalHolidaySchedule{})
	_ = now
}
