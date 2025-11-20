package handlers

import (
	"database/sql"
	"edge-metrics-server/models"
	"edge-metrics-server/repository"
	"fmt"
	"log"
	"net/http"
	"time"

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

	// Save client IP address
	config.IPAddress = c.ClientIP()

	created, err := repository.Upsert(deviceID, &config)
	if err != nil {
		log.Printf("Error upserting config for %s: %v", deviceID, err)
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "Internal server error",
			Message: "Failed to save device configuration",
		})
		return
	}

	status := "updated"
	if created {
		status = "registered"
		log.Printf("Registered new device: %s", deviceID)
	} else {
		log.Printf("Updated config for device: %s", deviceID)
	}

	// Trigger reload on exporter if IP is available
	reloadTriggered := false
	if config.IPAddress != "" {
		reloadURL := fmt.Sprintf("http://%s:%d/reload", config.IPAddress, config.ReloadPort)
		client := &http.Client{Timeout: 2 * time.Second}
		resp, err := client.Post(reloadURL, "application/json", nil)
		if err != nil {
			log.Printf("Failed to trigger reload for %s: %v", deviceID, err)
		} else {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				reloadTriggered = true
				log.Printf("Reload triggered for device: %s", deviceID)
			} else {
				log.Printf("Reload failed for %s: HTTP %d", deviceID, resp.StatusCode)
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"status":           status,
		"device_id":        deviceID,
		"reload_triggered": reloadTriggered,
	})
}

// CreateConfig handles POST /config/:device_id
func CreateConfig(c *gin.Context) {
	deviceID := c.Param("device_id")
	log.Printf("Create request for device: %s", deviceID)

	// Check if device already exists
	exists, err := repository.Exists(deviceID)
	if err != nil {
		log.Printf("Error checking device %s: %v", deviceID, err)
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "Internal server error",
			Message: "Failed to check device existence",
		})
		return
	}

	if exists {
		log.Printf("Device already exists: %s", deviceID)
		c.JSON(http.StatusConflict, models.ErrorResponse{
			Error:    "Device already exists",
			DeviceID: deviceID,
			Message:  "Use PUT to update existing device",
		})
		return
	}

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
	config := models.DeviceConfig{
		DeviceID: deviceID,
	}

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

	// Extract extra config
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

	// Set defaults
	if config.Interval == 0 {
		config.Interval = 1
	}
	if config.Port == 0 {
		config.Port = 9100
	}
	if config.ReloadPort == 0 {
		config.ReloadPort = 9101
	}

	// Save client IP address
	config.IPAddress = c.ClientIP()

	err = repository.Create(&config)
	if err != nil {
		log.Printf("Error creating config for %s: %v", deviceID, err)
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "Internal server error",
			Message: "Failed to create device configuration",
		})
		return
	}

	log.Printf("Created new device: %s", deviceID)
	c.JSON(http.StatusCreated, models.UpdateResponse{
		Status:   "created",
		DeviceID: deviceID,
	})
}

// DeleteConfig handles DELETE /config/:device_id
func DeleteConfig(c *gin.Context) {
	deviceID := c.Param("device_id")
	log.Printf("Delete request for device: %s", deviceID)

	err := repository.Delete(deviceID)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Printf("Device not found for delete: %s", deviceID)
			c.JSON(http.StatusNotFound, models.ErrorResponse{
				Error:    "Device not found",
				DeviceID: deviceID,
			})
			return
		}
		log.Printf("Error deleting config for %s: %v", deviceID, err)
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "Internal server error",
			Message: "Failed to delete device configuration",
		})
		return
	}

	log.Printf("Deleted device: %s", deviceID)
	c.JSON(http.StatusOK, models.UpdateResponse{
		Status:   "deleted",
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

// ListDevices handles GET /devices
func ListDevices(c *gin.Context) {
	log.Printf("List devices request")

	devices, err := repository.GetAll()
	if err != nil {
		log.Printf("Error fetching devices: %v", err)
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "Internal server error",
			Message: "Failed to fetch devices",
		})
		return
	}

	// Check health status of each device
	var deviceStatuses []models.DeviceStatus
	healthy := 0
	unhealthy := 0

	for _, device := range devices {
		status := models.DeviceStatus{
			DeviceID:   device.DeviceID,
			DeviceType: device.DeviceType,
			IPAddress:  device.IPAddress,
			Port:       device.Port,
			ReloadPort: device.ReloadPort,
		}

		if device.IPAddress == "" {
			status.Status = "unknown"
			status.Error = "No IP address registered"
			unhealthy++
		} else {
			// Check device health
			healthURL := fmt.Sprintf("http://%s:%d/health", device.IPAddress, device.ReloadPort)
			client := &http.Client{Timeout: 2 * time.Second}

			resp, err := client.Get(healthURL)
			if err != nil {
				status.Status = "unreachable"
				status.Error = err.Error()
				unhealthy++
			} else {
				resp.Body.Close()
				if resp.StatusCode == http.StatusOK {
					status.Status = "healthy"
					status.LastSeen = time.Now().Format(time.RFC3339)
					healthy++
				} else {
					status.Status = "unhealthy"
					status.Error = fmt.Sprintf("HTTP %d", resp.StatusCode)
					unhealthy++
				}
			}
		}

		deviceStatuses = append(deviceStatuses, status)
	}

	c.JSON(http.StatusOK, models.DevicesListResponse{
		Devices:   deviceStatuses,
		Total:     len(devices),
		Healthy:   healthy,
		Unhealthy: unhealthy,
	})
}

// GetDeviceStatus handles GET /devices/:device_id/status
func GetDeviceStatus(c *gin.Context) {
	deviceID := c.Param("device_id")
	log.Printf("Device status request for: %s", deviceID)

	device, err := repository.GetByDeviceID(deviceID)
	if err != nil {
		log.Printf("Error fetching device %s: %v", deviceID, err)
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "Internal server error",
			Message: "Failed to fetch device",
		})
		return
	}

	if device == nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse{
			Error:    "Device not found",
			DeviceID: deviceID,
		})
		return
	}

	status := models.DeviceStatus{
		DeviceID:   device.DeviceID,
		DeviceType: device.DeviceType,
		IPAddress:  device.IPAddress,
		Port:       device.Port,
		ReloadPort: device.ReloadPort,
	}

	if device.IPAddress == "" {
		status.Status = "unknown"
		status.Error = "No IP address registered"
	} else {
		// Check device health
		healthURL := fmt.Sprintf("http://%s:%d/health", device.IPAddress, device.ReloadPort)
		client := &http.Client{Timeout: 2 * time.Second}

		resp, err := client.Get(healthURL)
		if err != nil {
			status.Status = "unreachable"
			status.Error = err.Error()
		} else {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				status.Status = "healthy"
				status.LastSeen = time.Now().Format(time.RFC3339)
			} else {
				status.Status = "unhealthy"
				status.Error = fmt.Sprintf("HTTP %d", resp.StatusCode)
			}
		}
	}

	c.JSON(http.StatusOK, status)
}

