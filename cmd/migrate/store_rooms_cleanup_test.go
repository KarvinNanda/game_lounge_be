package main

import (
	"fmt"
	"os"
	"testing"
	"time"

	"game_lounge_be/models"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func testDB(t *testing.T) *gorm.DB {
	t.Helper()
	for _, p := range []string{"../../.env", "../.env"} {
		_ = godotenv.Load(p)
	}
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		os.Getenv("DB_USER"), os.Getenv("DB_PASSWORD"), os.Getenv("DB_HOST"), os.Getenv("DB_PORT"), os.Getenv("DB_NAME"))
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil || db.Exec("SELECT 1").Error != nil {
		t.Skip("skip: koneksi DB tidak tersedia")
	}
	return db
}

func TestSoftDeleteRoomsOfDeletedStores(t *testing.T) {
	db := testDB(t)
	deleted := time.Now().Add(-time.Hour)
	gone := models.Store{ID: uuid.NewString(), Name: "TEST gone", Address: "x", Status: "inactive", DeletedAt: &deleted}
	live := models.Store{ID: uuid.NewString(), Name: "TEST live", Address: "x", Status: "active"}
	db.Create(&gone)
	db.Create(&live)
	t.Cleanup(func() {
		db.Unscoped().Where("store_id IN ?", []string{gone.ID, live.ID}).Delete(&models.StoreRoom{})
		db.Unscoped().Delete(&models.Store{}, "id IN ?", []string{gone.ID, live.ID})
	})
	for _, sid := range []string{gone.ID, live.ID} {
		db.Create(&models.StoreRoom{ID: uuid.NewString(), StoreID: sid, RoomTemplateID: 8, UnitNumber: 1, Name: "TEST 1", IsActive: true})
	}

	softDeleteRoomsOfDeletedStores(db)
	softDeleteRoomsOfDeletedStores(db) // idempotent

	var r models.StoreRoom
	db.Where("store_id = ?", gone.ID).First(&r)
	if r.DeletedAt == nil || r.IsActive {
		t.Errorf("room of a deleted store must be soft-deleted and inactive: %+v", r)
	}
	var lr models.StoreRoom // fresh struct: First on a struct with an ID filters by that ID
	db.Where("store_id = ?", live.ID).First(&lr)
	if lr.DeletedAt != nil || !lr.IsActive {
		t.Errorf("room of a live store must not change: %+v", lr)
	}
}
