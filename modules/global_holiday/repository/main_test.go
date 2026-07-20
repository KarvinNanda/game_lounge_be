package repository

import (
	"fmt"
	"os"
	"testing"

	"game_lounge_be/config"

	"github.com/joho/godotenv"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var dbAvailable bool

func TestMain(m *testing.M) {
	dbAvailable = connectTestDB()
	os.Exit(m.Run())
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
