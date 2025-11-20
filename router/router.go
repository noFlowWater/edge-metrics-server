package router

import (
	"edge-metrics-server/handlers"

	"github.com/gin-gonic/gin"
)

// SetupRoutes configures all API routes
func SetupRoutes(r *gin.Engine) {
	// Config routes
	r.GET("/config/:device_id", handlers.GetConfig)
	r.POST("/config/:device_id", handlers.CreateConfig)
	r.PUT("/config/:device_id", handlers.UpdateConfig)
	r.DELETE("/config/:device_id", handlers.DeleteConfig)

	// Device routes
	r.GET("/devices", handlers.ListDevices)
	r.GET("/devices/:device_id/status", handlers.GetDeviceStatus)

	// Health route
	r.GET("/health", handlers.Health)
}
