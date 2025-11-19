package main

import (
	"edge-metrics-server/database"
	"edge-metrics-server/handlers"
	"log"
	"os"

	"github.com/gin-gonic/gin"
)

func main() {
	// Get configuration from environment
	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}

	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "./config.db"
	}

	// Initialize database
	if err := database.InitDB(dbPath); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer database.CloseDB()

	// Setup Gin router
	gin.SetMode(gin.ReleaseMode)
	router := gin.Default()

	// Routes
	router.GET("/config/:device_id", handlers.GetConfig)
	router.PUT("/config/:device_id", handlers.UpdateConfig)
	router.GET("/health", handlers.Health)

	// Start server
	log.Printf("Starting CONFIG SERVER on port %s", port)
	if err := router.Run("0.0.0.0:" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
