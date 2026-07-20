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
	testCustomerID string
)

func TestMain(m *testing.M) {
	dbAvailable = connectTestDB()
	if dbAvailable {
		setupVoucherFixtures()
	}
	code := m.Run()
	if dbAvailable {
		teardownVoucherFixtures()
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

// setupVoucherFixtures membuat customer test untuk skenario "sudah pernah pakai".
func setupVoucherFixtures() {
	testCustomerID = uuid.NewString()
	wa := "08999000001"
	config.DB.Create(&models.Customer{
		ID:       testCustomerID,
		Name:     "TEST Customer Voucher",
		Whatsapp: wa,
		Type:     "member",
		Status:   "active",
	})
}

func teardownVoucherFixtures() {
	config.DB.Unscoped().Delete(&models.Customer{}, "id = ?", testCustomerID)
}

// ── Helper: buat voucher test dan kembalikan cleanup func ─────

func createTestVoucher(code, discountType string, discountValue float64, opts ...func(*models.Voucher)) (*models.Voucher, func()) {
	actor := "test"
	v := &models.Voucher{
		ID:             uuid.NewString(),
		Name:           "TEST Voucher " + code,
		Code:           code,
		Type:           "booking",
		DiscountType:   discountType,
		DiscountValue:  discountValue,
		StartDate:      time.Now().AddDate(0, 0, -1), // kemarin (sudah aktif)
		IsAllStores:    true,
		IsAllRoomTypes: true,
		IsActive:       true,
		CreatedBy:      &actor,
	}
	for _, opt := range opts {
		opt(v)
	}
	config.DB.Create(v)
	return v, func() {
		config.DB.Unscoped().Where("voucher_id = ?", v.ID).Delete(&models.VoucherUsage{})
		config.DB.Unscoped().Where("voucher_id = ?", v.ID).Delete(&models.VoucherRoomTemplate{})
		config.DB.Unscoped().Delete(&models.Voucher{}, "id = ?", v.ID)
	}
}

func withExpired() func(*models.Voucher) {
	return func(v *models.Voucher) {
		past := time.Now().AddDate(0, 0, -2)
		v.EndDate = &past
	}
}

func withInactive() func(*models.Voucher) {
	return func(v *models.Voucher) {
		v.IsActive = false
	}
}

func withMinPurchase(min float64) func(*models.Voucher) {
	return func(v *models.Voucher) {
		v.MinPurchase = &min
	}
}

func withRoomTypeRestriction(rtID uint) func(*models.Voucher) {
	return func(v *models.Voucher) {
		v.IsAllRoomTypes = false
	}
}
