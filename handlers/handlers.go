package handlers

import (
	"database/sql"
	"edge-metrics-server/models"
	"edge-metrics-server/repository"
	"encoding/json"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

// GetConfig handles GET /config/:device_id
func GetConfig(c *gin.Context) {
	deviceID := c.Param("device_id")
	log.Printf("Config request for device: %s", deviceID)

	config, err := repository.GetByDeviceID(deviceID)
	if err != nil {
		log.Printf("Error fetching config for %s: %v", deviceID, err)
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "Internal server error",
			Message: "Failed to fetch device configuration",
		})
		return
	}

	if config == nil {
		log.Printf("Device not found: %s", deviceID)
		c.JSON(http.StatusNotFound, models.ErrorResponse{
			Error:    "Device not found",
			DeviceID: deviceID,
			Message:  "No configuration available for this device",
		})
		return
	}

	log.Printf("Returning config for %s: %s", deviceID, config.DeviceType)

	// Build response without device_id field (as per API spec)
	response := gin.H{
		"device_type": config.DeviceType,
		"interval":    config.Interval,
		"port":        config.Port,
		"reload_port": config.ReloadPort,
	}

	if len(config.EnabledMetrics) > 0 {
		response["enabled_metrics"] = config.EnabledMetrics
	}

	// Spread extra_config into response (e.g., "shelly": {...}, "jetson": {...})
	for key, value := range config.ExtraConfig {
		response[key] = value
	}

	c.JSON(http.StatusOK, response)
}

// UpdateConfig handles PUT /config/:device_id
func UpdateConfig(c *gin.Context) {
	deviceID := c.Param("device_id")
	log.Printf("Update request for device: %s", deviceID)

	// Parse raw JSON to extract extra config fields
	var rawData map[string]interface{}
	if err := c.ShouldBindJSON(&rawData); err != nil {
		log.Printf("Invalid JSON for %s: %v", deviceID, err)
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "Invalid request body",
			Message: err.Error(),
		})
		return
	}

	// Build DeviceConfig from raw data
	config := models.DeviceConfig{}

	if deviceType, ok := rawData["device_type"].(string); ok {
		config.DeviceType = deviceType
	} else {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "Missing required field",
			Message: "device_type is required",
		})
		return
	}

	if interval, ok := rawData["interval"].(float64); ok {
		config.Interval = int(interval)
	}
	if port, ok := rawData["port"].(float64); ok {
		config.Port = int(port)
	}
	if reloadPort, ok := rawData["reload_port"].(float64); ok {
		config.ReloadPort = int(reloadPort)
	}

	// Parse enabled_metrics
	if metrics, ok := rawData["enabled_metrics"].([]interface{}); ok {
		for _, m := range metrics {
			if s, ok := m.(string); ok {
				config.EnabledMetrics = append(config.EnabledMetrics, s)
			}
		}
	}

	// Extract extra config (any keys that are not standard fields)
	standardFields := map[string]bool{
		"device_type":     true,
		"interval":        true,
		"port":            true,
		"reload_port":     true,
		"enabled_metrics": true,
	}

	config.ExtraConfig = make(map[string]interface{})
	for key, value := range rawData {
		if !standardFields[key] {
			config.ExtraConfig[key] = value
		}
	}

	// Set defaults if not provided
	if config.Interval == 0 {
		config.Interval = 1
	}
	if config.Port == 0 {
		config.Port = 9100
	}
	if config.ReloadPort == 0 {
		config.ReloadPort = 9101
	}

	err := repository.Update(deviceID, &config)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Printf("Device not found for update: %s", deviceID)
			c.JSON(http.StatusNotFound, models.ErrorResponse{
				Error:    "Device not found",
				DeviceID: deviceID,
			})
			return
		}
		log.Printf("Error updating config for %s: %v", deviceID, err)
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "Internal server error",
			Message: "Failed to update device configuration",
		})
		return
	}

	log.Printf("Updated config for device: %s", deviceID)
	c.JSON(http.StatusOK, models.UpdateResponse{
		Status:   "updated",
		DeviceID: deviceID,
	})
}

// Health handles GET /health
func Health(c *gin.Context) {
	c.JSON(http.StatusOK, models.HealthResponse{
		Status:  "healthy",
		Service: "config-server",
		Version: "1.0.0",
	})
}

// Helper to convert interface to JSON string (for logging/debugging)
func toJSON(v interface{}) string {
	b, _ := json.Marshal(v)
	return string(b)
}
