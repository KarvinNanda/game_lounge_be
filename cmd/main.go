package main

import (
	"log"
	"os"

	"game_lounge_be/config"
	"game_lounge_be/routes"
	"game_lounge_be/utils"

	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load("../.env"); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	// Fail-fast: server tidak boleh jalan tanpa JWT secret.
	// Secret kosong = semua token auth bisa dipalsukan penyerang.
	if os.Getenv("JWT_SECRET") == "" {
		log.Fatal("JWT_SECRET wajib diset di environment — server dihentikan")
	}

	config.InitDB()
	utils.InitUploadDir()
	utils.InitAssetsDir()

	r := routes.SetupRouter()

	// Cek PORT terlebih dahulu (standar Coolify/Railway), lalu APP_PORT, lalu default 8080
	port := config.GetEnv("PORT", config.GetEnv("APP_PORT", "8080"))
	log.Printf("Server running on port %s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
