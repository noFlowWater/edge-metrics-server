package handlers

import (
	"edge-metrics-server/models"
	"fmt"
	"net/http"
	"time"
)

// CheckDeviceHealth checks device health and returns detailed status
func CheckDeviceHealth(device models.DeviceConfig) models.DeviceStatus {
	status := models.DeviceStatus{
		DeviceID:   device.DeviceID,
		DeviceType: device.DeviceType,
		IPAddress:  device.IPAddress,
		Port:       device.Port,
		ReloadPort: device.ReloadPort,
	}

	// Check if IP address is available
	if device.IPAddress == "" || device.IPAddress == "unknown" {
		status.Status = "unknown"
		if device.IPAddress == "" {
			status.Error = "No IP address registered"
		}
		return status
	}

	// Perform health check
	healthURL := fmt.Sprintf("http://%s:%d/health", device.IPAddress, device.ReloadPort)
	client := &http.Client{Timeout: 2 * time.Second}

	resp, err := client.Get(healthURL)
	if err != nil {
		status.Status = "unreachable"
		status.Error = err.Error()
		return status
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		status.Status = "healthy"
		status.LastSeen = time.Now().Format(time.RFC3339)
	} else {
		status.Status = "unhealthy"
		status.Error = fmt.Sprintf("HTTP %d", resp.StatusCode)
	}

	return status
}

// IsDeviceHealthy returns true if device is healthy
func IsDeviceHealthy(device models.DeviceConfig) bool {
	return CheckDeviceHealth(device).Status == "healthy"
}
