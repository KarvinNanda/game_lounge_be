package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"game_lounge_be/config"
	"game_lounge_be/jobs"
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
	// HS256 dengan secret pendek bisa di-brute-force offline dari 1 token yang bocor.
	if len(os.Getenv("JWT_SECRET")) < 32 {
		log.Fatal("JWT_SECRET minimal 32 karakter (contoh: openssl rand -hex 32) — server dihentikan")
	}

	// Fail-closed: di production, config yang kosong membuat CORS/CSRF/Xendit terbuka.
	if errs := utils.ProductionConfigErrors(); len(errs) > 0 {
		for _, e := range errs {
			log.Println("Config error:", e)
		}
		log.Fatal("Config production tidak aman — server dihentikan")
	}

	config.InitDB()
	utils.InitUploadDir()
	utils.InitAssetsDir()

	r := routes.SetupRouter()
	jobs.StartCleanup(time.Minute)

	// Cek PORT terlebih dahulu (standar Coolify/Railway), lalu APP_PORT, lalu default 8080
	port := config.GetEnv("PORT", config.GetEnv("APP_PORT", "8080"))
	// Timeout eksplisit: r.Run() memakai http.Server tanpa timeout, sehingga
	// client lambat (slowloris) bisa menahan koneksi tanpa batas.
	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
	}
	log.Printf("Server running on port %s", port)
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
