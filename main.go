package main

import (
	"edge-metrics-server/database"
	"edge-metrics-server/kubernetes"
	"edge-metrics-server/router"
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

	// Initialize Kubernetes client (optional, will fail gracefully if not in k8s)
	if err := kubernetes.InitClient(); err != nil {
		log.Printf("Kubernetes client not initialized: %v (Kubernetes features disabled)", err)
	} else {
		log.Printf("Kubernetes client initialized successfully")
	}

	// Setup Gin router
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	// Setup routes
	router.SetupRoutes(r)

	// Start server
	log.Printf("Starting CONFIG SERVER on port %s", port)
	if err := r.Run("0.0.0.0:" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
