package handlers

import (
	"database/sql"
	"edge-metrics-server/models"
	"edge-metrics-server/repository"
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

	if config.Jetson != nil {
		response["jetson"] = config.Jetson
	}

	if config.Shelly != nil {
		response["shelly"] = config.Shelly
	}

	if config.INA260 != nil {
		response["ina260"] = config.INA260
	}

	c.JSON(http.StatusOK, response)
}

// UpdateConfig handles PUT /config/:device_id
func UpdateConfig(c *gin.Context) {
	deviceID := c.Param("device_id")
	log.Printf("Update request for device: %s", deviceID)

	var config models.DeviceConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		log.Printf("Invalid JSON for %s: %v", deviceID, err)
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "Invalid request body",
			Message: err.Error(),
		})
		return
	}

	// Ensure required field
	if config.DeviceType == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "Missing required field",
			Message: "device_type is required",
		})
		return
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
