package kubernetes

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"edge-metrics-server/models"
)

// SyncResult represents the result of a sync operation
type SyncResult struct {
	DeviceID string `json:"device_id"`
	Service  string `json:"service,omitempty"`
	Status   string `json:"status"` // created, updated, failed
	Error    string `json:"error,omitempty"`
}

// SyncResponse represents the response from a sync operation
type SyncResponse struct {
	Status       string       `json:"status"`
	Created      []SyncResult `json:"created"`
	Updated      []SyncResult `json:"updated"`
	Deleted      []SyncResult `json:"deleted"`
	Failed       []SyncResult `json:"failed"`
	TotalHealthy int          `json:"total_healthy"`
}

// SyncDevices synchronizes healthy devices to Kubernetes
func SyncDevices(namespace, serverURL string) (*SyncResponse, error) {
	if !IsInitialized() {
		return nil, fmt.Errorf("kubernetes client not initialized")
	}

	// Get healthy devices from the API
	devices, err := getHealthyDevices(serverURL)
	if err != nil {
		return nil, fmt.Errorf("failed to get healthy devices: %w", err)
	}

	response := &SyncResponse{
		Status:       "synced",
		Created:      []SyncResult{},
		Updated:      []SyncResult{},
		Deleted:      []SyncResult{},
		Failed:       []SyncResult{},
		TotalHealthy: len(devices),
	}

	// Track existing services in K8s
	existingServices, err := ListEdgeServices(namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to list existing services: %w", err)
	}
	existingMap := make(map[string]bool)
	for _, svc := range existingServices {
		existingMap[svc] = true
	}

	// Create or update services for healthy devices
	for _, device := range devices {
		if device.IPAddress == "" {
			response.Failed = append(response.Failed, SyncResult{
				DeviceID: device.DeviceID,
				Status:   "failed",
				Error:    "no IP address registered",
			})
			continue
		}

		serviceName := fmt.Sprintf("edge-device-%s", device.DeviceID)
		wasExisting := existingMap[serviceName]
		delete(existingMap, serviceName) // Mark as processed

		// Create/Update Service
		err := CreateOrUpdateService(namespace, device.DeviceID, device.DeviceType, device.Port)
		if err != nil {
			response.Failed = append(response.Failed, SyncResult{
				DeviceID: device.DeviceID,
				Service:  serviceName,
				Status:   "failed",
				Error:    fmt.Sprintf("service: %v", err),
			})
			continue
		}

		// Create/Update Endpoints
		err = CreateOrUpdateEndpoints(namespace, device.DeviceID, device.IPAddress, device.Port)
		if err != nil {
			response.Failed = append(response.Failed, SyncResult{
				DeviceID: device.DeviceID,
				Service:  serviceName,
				Status:   "failed",
				Error:    fmt.Sprintf("endpoints: %v", err),
			})
			continue
		}

		result := SyncResult{
			DeviceID: device.DeviceID,
			Service:  serviceName,
		}

		if wasExisting {
			result.Status = "updated"
			response.Updated = append(response.Updated, result)
		} else {
			result.Status = "created"
			response.Created = append(response.Created, result)
		}
	}

	// Delete services that are no longer healthy
	for serviceName := range existingMap {
		// Extract device_id from service name (edge-device-{device_id})
		deviceID := serviceName[len("edge-device-"):]

		err := DeleteService(namespace, deviceID)
		if err != nil {
			response.Failed = append(response.Failed, SyncResult{
				DeviceID: deviceID,
				Service:  serviceName,
				Status:   "failed",
				Error:    fmt.Sprintf("delete service: %v", err),
			})
			continue
		}

		err = DeleteEndpoints(namespace, deviceID)
		if err != nil {
			response.Failed = append(response.Failed, SyncResult{
				DeviceID: deviceID,
				Service:  serviceName,
				Status:   "failed",
				Error:    fmt.Sprintf("delete endpoints: %v", err),
			})
			continue
		}

		response.Deleted = append(response.Deleted, SyncResult{
			DeviceID: deviceID,
			Service:  serviceName,
			Status:   "deleted",
		})
	}

	return response, nil
}

// getHealthyDevices fetches healthy devices from the server API
func getHealthyDevices(serverURL string) ([]models.DeviceStatus, error) {
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	url := fmt.Sprintf("%s/devices", serverURL)
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	var listResponse models.DevicesListResponse
	if err := json.NewDecoder(resp.Body).Decode(&listResponse); err != nil {
		return nil, err
	}

	// Filter only healthy devices
	var healthyDevices []models.DeviceStatus
	for _, device := range listResponse.Devices {
		if device.Status == "healthy" {
			healthyDevices = append(healthyDevices, device)
		}
	}

	return healthyDevices, nil
}

// CleanupAllResources removes all edge-device-* resources from a namespace
func CleanupAllResources(namespace string) ([]string, []string, error) {
	if !IsInitialized() {
		return nil, nil, fmt.Errorf("kubernetes client not initialized")
	}

	// List all edge services
	services, err := ListEdgeServices(namespace)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list services: %w", err)
	}

	// List all edge endpoints
	endpoints, err := ListEdgeEndpoints(namespace)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list endpoints: %w", err)
	}

	// Delete all services
	for _, serviceName := range services {
		deviceID := serviceName[len("edge-device-"):]
		if err := DeleteService(namespace, deviceID); err != nil {
			return nil, nil, fmt.Errorf("failed to delete service %s: %w", serviceName, err)
		}
	}

	// Delete all endpoints
	for _, endpointName := range endpoints {
		deviceID := endpointName[len("edge-device-"):]
		if err := DeleteEndpoints(namespace, deviceID); err != nil {
			return nil, nil, fmt.Errorf("failed to delete endpoints %s: %w", endpointName, err)
		}
	}

	return services, endpoints, nil
}
