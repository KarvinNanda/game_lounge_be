package main

import (
	"log"
	"game_lounge_be/config"
	"game_lounge_be/routes"
	"game_lounge_be/utils"

	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load("../.env"); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	config.InitDB()
	utils.InitUploadDir()
	utils.InitAssetsDir()

	r := routes.SetupRouter()

	port := config.GetEnv("APP_PORT", "8080")
	log.Printf("Server running on port %s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
