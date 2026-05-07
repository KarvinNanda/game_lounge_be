package config

import (
    _ "fmt"
    "log"
    "os"

    "github.com/joho/godotenv"
)

func LoadEnv() {
    // Mencoba memuat file .env di root directory
    err := godotenv.Load("../.env")
    if err != nil {
        log.Println("Warning: .env file not found, using system environment variables")
    }
}

func GetEnv(key, fallback string) string {
    value := os.Getenv(key)
    if value != "" {
        return value
    }
    return fallback
}