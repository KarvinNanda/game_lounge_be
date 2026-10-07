// Command migrate membuat / memperbarui schema database dari struct di package models.
//
// Jalankan dari root project:
//
//	go run ./cmd/migrate
//
// Opsional — buat role super admin + 1 akun staff (idempotent, skip jika username sudah ada):
//
//	SEED_ADMIN_USERNAME=admin SEED_ADMIN_EMAIL=admin@example.com SEED_ADMIN_PASSWORD=... go run ./cmd/migrate
package main

import (
	"log"
	"os"

	"game_lounge_be/config"
	"game_lounge_be/models"
	"game_lounge_be/utils"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"gorm.io/gorm"
)

func main() {
	// Cari .env dari root project maupun dari dalam cmd/ (sama seperti main server).
	for _, p := range []string{".env", "../.env", "../../.env"} {
		if godotenv.Load(p) == nil {
			break
		}
	}

	config.InitDB()
	db := config.DB

	// FK constraint dari GORM dimatikan: relasi seperti StorePricing.HappyHourSchedules
	// (foreignKey:StoreID;references:StoreID) akan menghasilkan FK yang salah arah
	// (store_happy_hour_schedules.store_id -> store_pricings.store_id).
	db.Config.DisableForeignKeyConstraintWhenMigrating = true

	// RoomTemplateFacility harus dimigrasi sebelum RoomTemplate, supaya tabel
	// room_template_facilities punya kolom id (bukan join table default GORM).
	err := db.AutoMigrate(
		&models.Role{},
		&models.RolePermission{},
		&models.Staff{},
		&models.StaffStore{},
		&models.PasswordResetToken{},

		&models.Store{},
		&models.StoreOperatingHour{},
		&models.StoreHolidaySchedule{},
		&models.GlobalHolidaySchedule{},
		&models.StoreRoom{},
		&models.StoreEventPrice{},

		&models.FacilityCategory{},
		&models.Facility{},
		&models.RoomTemplateFacility{},
		&models.RoomTemplate{},

		&models.StorePricing{},
		&models.StoreHappyHourSchedule{},
		&models.StoreHappyHourPrice{},
		&models.StorePackagePrice{},
		&models.StoreFlashSale{},

		&models.Customer{},
		&models.CustomerFavoriteRoomType{},
		&models.CustomerPasswordReset{},

		&models.Booking{},
		&models.BookingHold{},
		&models.BookingSequence{},
		&models.EventBooking{},

		&models.PlayCreditsPackage{},
		&models.PlayCreditsPackageStore{},
		&models.CustomerPlayCredit{},
		&models.PlayCreditsPurchaseIntent{},

		&models.Voucher{},
		&models.VoucherStore{},
		&models.VoucherRoomTemplate{},
		&models.VoucherUsage{},

		&models.FnbCategory{},
		&models.FnbItem{},
		&models.FnbOrder{},
		&models.FnbOrderItem{},

		&models.NotificationTemplate{},
		&models.Banner{},
	)
	if err != nil {
		log.Fatalf("AutoMigrate gagal: %v", err)
	}

	// UNIQUE (voucher_id, customer_id): voucher hanya boleh dipakai 1x per customer.
	// Tidak bisa lewat tag GORM tanpa mengubah model, jadi dibuat manual.
	if !db.Migrator().HasIndex(&models.VoucherUsage{}, "uq_voucher_usages_voucher_customer") {
		if err := db.Exec("CREATE UNIQUE INDEX uq_voucher_usages_voucher_customer ON voucher_usages (voucher_id, customer_id)").Error; err != nil {
			log.Fatalf("Gagal membuat unique index voucher_usages: %v", err)
		}
	}

	// Single-row counter untuk booking code.
	if err := db.Exec("INSERT IGNORE INTO booking_sequences (id, last_sequence, updated_at) VALUES (1, 0, NOW())").Error; err != nil {
		log.Fatalf("Gagal seed booking_sequences: %v", err)
	}

	seedSuperAdmin(db)

	log.Println("Migrasi selesai")
}

// seedSuperAdmin membuat role is_system + 1 staff jika SEED_ADMIN_* diset.
// Password hanya dibaca dari environment, tidak pernah di-hardcode.
func seedSuperAdmin(db *gorm.DB) {
	username := os.Getenv("SEED_ADMIN_USERNAME")
	email := os.Getenv("SEED_ADMIN_EMAIL")
	password := os.Getenv("SEED_ADMIN_PASSWORD")
	if username == "" || email == "" || password == "" {
		log.Println("SEED_ADMIN_* tidak lengkap, skip seed super admin")
		return
	}

	var count int64
	db.Model(&models.Staff{}).Where("username = ?", username).Count(&count)
	if count > 0 {
		log.Printf("Staff %q sudah ada, skip seed super admin", username)
		return
	}

	actor := "seed"
	var role models.Role
	if err := db.Where("is_system = ? AND deleted_at IS NULL", true).First(&role).Error; err != nil {
		role = models.Role{Name: "Super Admin", IsSystem: true, CreatedBy: &actor}
		if err := db.Create(&role).Error; err != nil {
			log.Fatalf("Gagal membuat role super admin: %v", err)
		}
	}

	hash, err := utils.HashPassword(password)
	if err != nil {
		log.Fatalf("Gagal hash password: %v", err)
	}

	staff := models.Staff{
		ID:           uuid.NewString(),
		RoleID:       role.ID,
		Username:     username,
		Email:        email,
		PasswordHash: hash,
		IsAllStores:  true,
		CreatedBy:    &actor,
	}
	if err := db.Create(&staff).Error; err != nil {
		log.Fatalf("Gagal membuat staff super admin: %v", err)
	}
	log.Printf("Super admin %q dibuat (role_id=%d)", username, role.ID)
}
