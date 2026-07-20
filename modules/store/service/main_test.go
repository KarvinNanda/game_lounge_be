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

var (
	dbAvailable    bool
	testStoreID    string
)

func TestMain(m *testing.M) {
	dbAvailable = connectTestDB()
	if dbAvailable {
		setupStoreFixtures()
	}
	code := m.Run()
	if dbAvailable {
		teardownStoreFixtures()
	}
	os.Exit(code)
}

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

// setupStoreFixtures buat store test + operating hours (weekday & weekend).
func setupStoreFixtures() {
	testStoreID = uuid.NewString()
	actor := "test"
	config.DB.Create(&models.Store{
		ID:      testStoreID,
		Name:    "TEST Store Service",
		Address: "Test Address",
		Status:  "active",
	})

	// Operating hours
	config.DB.Create(&models.StoreOperatingHour{
		StoreID:   testStoreID,
		DayType:   "weekday",
		OpenTime:  "10:00",
		CloseTime: "22:00",
		IsActive:  true,
		CreatedBy: &actor,
	})
	config.DB.Create(&models.StoreOperatingHour{
		StoreID:   testStoreID,
		DayType:   "weekend",
		OpenTime:  "09:00",
		CloseTime: "23:00",
		IsActive:  true,
		CreatedBy: &actor,
	})
}

func teardownStoreFixtures() {
	config.DB.Unscoped().Where("store_id = ?", testStoreID).Delete(&models.StoreOperatingHour{})
	config.DB.Unscoped().Where("store_id = ?", testStoreID).Delete(&models.StoreHolidaySchedule{})
	config.DB.Unscoped().Where("store_id = ?", testStoreID).Delete(&models.Store{})
	// Bersihkan global holiday test jika ada sisa
	for _, d := range []string{"2099-08-17", "2099-09-20"} {
		config.DB.Unscoped().Where("DATE(date) = ? AND name LIKE 'TEST%'", d).
			Delete(&models.GlobalHolidaySchedule{})
	}
	_ = time.Now() // suppress unused import
}
