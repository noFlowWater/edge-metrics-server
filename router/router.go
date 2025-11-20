package router

import (
	"edge-metrics-server/handlers"

	"github.com/gin-gonic/gin"
)

// SetupRoutes configures all API routes
func SetupRoutes(r *gin.Engine) {
	// Config routes
	r.GET("/config", handlers.ListConfigs)
	r.GET("/config/:device_id", handlers.GetConfig)
	r.POST("/config/:device_id", handlers.CreateConfig)
	r.PUT("/config/:device_id", handlers.UpdateConfig)
	r.PATCH("/config/:device_id", handlers.PatchConfig)
	r.DELETE("/config/:device_id", handlers.DeleteConfig)

	// Device routes
	r.GET("/devices", handlers.ListDevices)
	r.POST("/devices/reload", handlers.ReloadAllDevices)
	r.GET("/devices/:device_id/status", handlers.GetDeviceStatus)
	r.POST("/devices/:device_id/reload", handlers.ReloadDevice)

	// Metrics routes
	r.GET("/metrics/summary", handlers.GetMetricsSummary)

	// Health route
	r.GET("/health", handlers.Health)
}
