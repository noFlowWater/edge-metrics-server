package models

// DeviceConfig represents the configuration for a device
type DeviceConfig struct {
	DeviceID       string                 `json:"-"`
	DeviceType     string                 `json:"device_type"`
	Interval       int                    `json:"interval,omitempty"`
	Port           int                    `json:"port,omitempty"`
	ReloadPort     int                    `json:"reload_port,omitempty"`
	EnabledMetrics []string               `json:"enabled_metrics,omitempty"`
	ExtraConfig    map[string]interface{} `json:"-"` // Device-specific config (shelly, jetson, ina260, etc.)
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error    string `json:"error"`
	DeviceID string `json:"device_id,omitempty"`
	Message  string `json:"message,omitempty"`
}

// UpdateResponse represents a successful update response
type UpdateResponse struct {
	Status   string `json:"status"`
	DeviceID string `json:"device_id"`
}

// HealthResponse represents a health check response
type HealthResponse struct {
	Status  string `json:"status"`
	Service string `json:"service"`
	Version string `json:"version"`
}
