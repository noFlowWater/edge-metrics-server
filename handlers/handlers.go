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

// isLocalhost checks if the IP is a localhost address
func isLocalhost(ip string) bool {
	return ip == "127.0.0.1" || ip == "::1"
}

// resolveIPAddress determines the IP address to save
// Preserves existing real IP when request comes from localhost
func resolveIPAddress(clientIP string, existingIP string) string {
	if isLocalhost(clientIP) && existingIP != "" && !isLocalhost(existingIP) {
		return existingIP
	}
	return clientIP
}

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
	if config.Port == 0 {
		config.Port = 9100
	}
	if config.ReloadPort == 0 {
		config.ReloadPort = 9101
	}

	// Save client IP address (preserve existing if updating from localhost)
	existingIP := ""
	if existing, _ := repository.GetByDeviceID(deviceID); existing != nil {
		existingIP = existing.IPAddress
	}
	config.IPAddress = resolveIPAddress(c.ClientIP(), existingIP)

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
		reloadTriggered = TriggerDeviceReloadWithLogging(deviceID, config)
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
	if config.Port == 0 {
		config.Port = 9100
	}
	if config.ReloadPort == 0 {
		config.ReloadPort = 9101
	}

	// Save client IP address
	clientIP := c.ClientIP()
	config.IPAddress = clientIP

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
		status := CheckDeviceHealth(device)

		if status.Status == "healthy" {
			healthy++
		} else {
			unhealthy++
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

	status := CheckDeviceHealth(*device)
	c.JSON(http.StatusOK, status)
}

// ReloadDevice handles POST /devices/:device_id/reload
func ReloadDevice(c *gin.Context) {
	deviceID := c.Param("device_id")
	log.Printf("Reload request for device: %s", deviceID)

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

	// Trigger reload
	success, errMsg := TriggerDeviceReload(device)
	if !success {
		log.Printf("Failed to trigger reload for %s: %s", deviceID, errMsg)
		statusCode := http.StatusServiceUnavailable
		if errMsg == "No IP address" {
			statusCode = http.StatusBadRequest
		} else if errMsg[:4] == "HTTP" {
			statusCode = http.StatusBadGateway
		}
		c.JSON(statusCode, gin.H{
			"status":    "failed",
			"device_id": deviceID,
			"error":     errMsg,
		})
		return
	}

	log.Printf("Reload triggered for device: %s", deviceID)
	c.JSON(http.StatusOK, gin.H{
		"status":    "reloaded",
		"device_id": deviceID,
	})
}

// ReloadAllDevices handles POST /devices/reload
func ReloadAllDevices(c *gin.Context) {
	log.Printf("Reload all devices request")

	devices, err := repository.GetAll()
	if err != nil {
		log.Printf("Error fetching devices: %v", err)
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "Internal server error",
			Message: "Failed to fetch devices",
		})
		return
	}

	results := make([]gin.H, 0)
	success := 0
	failed := 0

	for _, device := range devices {
		result := gin.H{
			"device_id": device.DeviceID,
		}

		reloadSuccess, errMsg := TriggerDeviceReload(device)
		if reloadSuccess {
			result["status"] = "reloaded"
			success++
		} else {
			if errMsg == "No IP address" {
				result["status"] = "skipped"
			} else {
				result["status"] = "failed"
			}
			result["error"] = errMsg
			failed++
		}

		results = append(results, result)
	}

	log.Printf("Reload all: %d success, %d failed", success, failed)
	c.JSON(http.StatusOK, gin.H{
		"results": results,
		"total":   len(devices),
		"success": success,
		"failed":  failed,
	})
}

// ListConfigs handles GET /config
func ListConfigs(c *gin.Context) {
	log.Printf("List all configs request")

	devices, err := repository.GetAll()
	if err != nil {
		log.Printf("Error fetching configs: %v", err)
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "Internal server error",
			Message: "Failed to fetch configurations",
		})
		return
	}

	configs := make([]gin.H, 0)
	for _, device := range devices {
		config := gin.H{
			"device_id":   device.DeviceID,
			"device_type": device.DeviceType,
			"port":        device.Port,
			"reload_port": device.ReloadPort,
		}

		if len(device.EnabledMetrics) > 0 {
			config["enabled_metrics"] = device.EnabledMetrics
		}

		// Spread extra_config
		for key, value := range device.ExtraConfig {
			config[key] = value
		}

		configs = append(configs, config)
	}

	c.JSON(http.StatusOK, gin.H{
		"configs": configs,
		"total":   len(configs),
	})
}

// PatchConfig handles PATCH /config/:device_id
func PatchConfig(c *gin.Context) {
	deviceID := c.Param("device_id")
	log.Printf("Patch request for device: %s", deviceID)

	// Check if device exists
	existing, err := repository.GetByDeviceID(deviceID)
	if err != nil {
		log.Printf("Error fetching device %s: %v", deviceID, err)
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "Internal server error",
			Message: "Failed to fetch device",
		})
		return
	}

	if existing == nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse{
			Error:    "Device not found",
			DeviceID: deviceID,
			Message:  "Use POST or PUT to create new device",
		})
		return
	}

	// Parse patch data
	var patchData map[string]interface{}
	if err := c.ShouldBindJSON(&patchData); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "Invalid request body",
			Message: err.Error(),
		})
		return
	}

	// Apply patches to existing config
	// null values will reset fields to defaults or remove them
	if val, exists := patchData["device_type"]; exists {
		if val == nil {
			existing.DeviceType = ""
		} else if s, ok := val.(string); ok {
			existing.DeviceType = s
		}
	}
	if val, exists := patchData["port"]; exists {
		if val == nil {
			existing.Port = 9100 // default
		} else if f, ok := val.(float64); ok {
			existing.Port = int(f)
		}
	}
	if val, exists := patchData["reload_port"]; exists {
		if val == nil {
			existing.ReloadPort = 9101 // default
		} else if f, ok := val.(float64); ok {
			existing.ReloadPort = int(f)
		}
	}
	if val, exists := patchData["enabled_metrics"]; exists {
		if val == nil {
			existing.EnabledMetrics = nil
		} else if metrics, ok := val.([]interface{}); ok {
			existing.EnabledMetrics = nil
			for _, m := range metrics {
				if s, ok := m.(string); ok {
					existing.EnabledMetrics = append(existing.EnabledMetrics, s)
				}
			}
		}
	}

	// Handle extra config patches
	standardFields := map[string]bool{
		"device_type":     true,
		"port":            true,
		"reload_port":     true,
		"enabled_metrics": true,
	}

	if existing.ExtraConfig == nil {
		existing.ExtraConfig = make(map[string]interface{})
	}

	for key, value := range patchData {
		if !standardFields[key] {
			if value == nil {
				// Remove the key if value is null
				delete(existing.ExtraConfig, key)
			} else {
				existing.ExtraConfig[key] = value
			}
		}
	}

	// Update IP address (preserve existing if updating from localhost)
	existing.IPAddress = resolveIPAddress(c.ClientIP(), existing.IPAddress)

	// Save updated config
	err = repository.Update(deviceID, existing)
	if err != nil {
		log.Printf("Error updating config for %s: %v", deviceID, err)
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "Internal server error",
			Message: "Failed to update device configuration",
		})
		return
	}

	// Trigger reload
	reloadTriggered := false
	if existing.IPAddress != "" {
		reloadTriggered = TriggerDeviceReloadWithLogging(deviceID, existing)
	}

	log.Printf("Patched config for device: %s", deviceID)
	c.JSON(http.StatusOK, gin.H{
		"status":           "patched",
		"device_id":        deviceID,
		"reload_triggered": reloadTriggered,
	})
}

// GetMetricsSummary handles GET /metrics/summary
func GetMetricsSummary(c *gin.Context) {
	log.Printf("Metrics summary request")

	devices, err := repository.GetAll()
	if err != nil {
		log.Printf("Error fetching devices: %v", err)
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "Internal server error",
			Message: "Failed to fetch devices",
		})
		return
	}

	// Count by device type
	typeCount := make(map[string]int)
	healthy := 0
	unhealthy := 0

	client := &http.Client{Timeout: 2 * time.Second}

	for _, device := range devices {
		typeCount[device.DeviceType]++

		// Check health
		if device.IPAddress == "" {
			unhealthy++
		} else {
			healthURL := fmt.Sprintf("http://%s:%d/health", device.IPAddress, device.ReloadPort)
			resp, err := client.Get(healthURL)
			if err != nil {
				unhealthy++
			} else {
				resp.Body.Close()
				if resp.StatusCode == http.StatusOK {
					healthy++
				} else {
					unhealthy++
				}
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"total":          len(devices),
		"healthy":        healthy,
		"unhealthy":      unhealthy,
		"by_device_type": typeCount,
	})
}

