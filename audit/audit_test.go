package audit

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

func connect(t *testing.T) {
	t.Helper()
	for _, p := range []string{"../.env", "../../.env"} {
		_ = godotenv.Load(p)
	}
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		os.Getenv("DB_USER"), os.Getenv("DB_PASSWORD"), os.Getenv("DB_HOST"), os.Getenv("DB_PORT"), os.Getenv("DB_NAME"))
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil || db.Exec("SELECT 1").Error != nil {
		t.Skip("skip: koneksi DB tidak tersedia")
	}
	config.DB = db
}

func str(s string) *string { return &s }

func TestFill_ResolvesStaffCustomersAndKeepsUsernames(t *testing.T) {
	connect(t)
	deleted := time.Now()
	cust := models.Customer{ID: uuid.NewString(), Name: "TEST Ani", Whatsapp: "0894" + uuid.NewString()[:8], DeletedAt: &deleted}
	staff := models.Staff{ID: uuid.NewString(), RoleID: 6, Username: "test.audit." + uuid.NewString()[:6], Email: uuid.NewString()[:8] + "@example.invalid", PasswordHash: "x"}
	if err := config.DB.Create(&cust).Error; err != nil {
		t.Fatal(err)
	}
	if err := config.DB.Create(&staff).Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		config.DB.Unscoped().Delete(&models.Customer{}, "id = ?", cust.ID)
		config.DB.Unscoped().Delete(&models.Staff{}, "id = ?", staff.ID)
	})
	unknown := uuid.NewString()

	bookings := []models.Booking{
		{CreatedBy: str(cust.ID), UpdatedBy: str(staff.ID)}, // customer (soft-deleted) and staff ids
		{CreatedBy: str("admin"), UpdatedBy: nil},           // username kept, nil stays nil
		{CreatedBy: str(unknown)},                           // unknown id shown as is
	}
	if err := FillSlice(bookings); err != nil {
		t.Fatal(err)
	}
	check := func(i int, gotC, gotU *string, wantC, wantU any) {
		t.Helper()
		val := func(p *string) any {
			if p == nil {
				return nil
			}
			return *p
		}
		if val(gotC) != wantC || val(gotU) != wantU {
			t.Errorf("row %d: created_by_name=%v updated_by_name=%v, want %v / %v", i, val(gotC), val(gotU), wantC, wantU)
		}
	}
	check(0, bookings[0].CreatedByName, bookings[0].UpdatedByName, "TEST Ani", staff.Username)
	check(1, bookings[1].CreatedByName, bookings[1].UpdatedByName, "admin", nil)
	check(2, bookings[2].CreatedByName, bookings[2].UpdatedByName, unknown, nil)
}

// Every audited model must expose the same two JSON fields once filled.
func TestFill_AllAuditedModelsShareTheContract(t *testing.T) {
	connect(t)
	items := []Audited{&models.Booking{}, &models.EventBooking{}, &models.Voucher{}, &models.Role{},
		&models.Staff{}, &models.Customer{}, &models.StoreFlashSale{}, &models.PlayCreditsPackage{}}
	for _, it := range items {
		c, u, names := it.AuditRefs()
		*c, *u = str("seed"), str("admin")
		if err := Fill(it); err != nil {
			t.Fatal(err)
		}
		if names.CreatedByName == nil || *names.CreatedByName != "seed" || *names.UpdatedByName != "admin" {
			t.Errorf("%T: names not filled: %+v", it, names)
		}
	}
}
