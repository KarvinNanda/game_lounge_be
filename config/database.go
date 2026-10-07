package config

import (
	"fmt"
	"log"
	"os"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

func InitDB() {
	dsn := fmt.Sprintf(
		"%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		GetEnv("DB_USER", "root"),
		GetEnv("DB_PASSWORD", ""),
		GetEnv("DB_HOST", "127.0.0.1"),
		GetEnv("DB_PORT", "3306"),
		GetEnv("DB_NAME", "game_lounge_db"),
	)

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: newDBLogger(),
	})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	DB = db
	log.Println("Database connected successfully")
}

// newDBLogger: log level Info mencetak SEMUA SQL beserta nilainya (password hash,
// token reset, data customer) ke stdout → ikut masuk log hosting. Default Warn,
// dan nilai parameter tidak pernah dicetak. DB_LOG_LEVEL=info untuk debug lokal.
func newDBLogger() logger.Interface {
	level := logger.Warn
	if GetEnv("DB_LOG_LEVEL", "") == "info" {
		level = logger.Info
	}
	return logger.New(log.New(os.Stdout, "\r\n", log.LstdFlags), logger.Config{
		SlowThreshold:             200 * time.Millisecond,
		LogLevel:                  level,
		IgnoreRecordNotFoundError: true,
		ParameterizedQueries:      true,
		Colorful:                  false,
	})
}
