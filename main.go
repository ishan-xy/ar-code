package main

import (
	"backend/db"
	"backend/handlers"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	// Load .env file
	if err := godotenv.Load(); err != nil {
		panic("Error loading .env file")
	}

	// Get database URL from environment variables
	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		panic("DB_URL is not set in the environment")
	}

	// Connect to MongoDB
	db.Connect(dbURL)

	// Ensure upload directory exists
	if err := os.MkdirAll("./uploaded_models", os.ModePerm); err != nil {
		panic("Failed to create base upload directory")
	}

	// Initialize router
	r := gin.Default()
	r.POST("/upload", handlers.UploadModelHandler)
	r.GET("/models", handlers.GetModelsHandler)

	// Run the server
	r.Run(":8080")
}
