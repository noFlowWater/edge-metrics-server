package models

// DeviceConfig represents the configuration for a device
type DeviceConfig struct {
	DeviceID       string            `json:"device_id,omitempty"`
	DeviceType     string            `json:"device_type"`
	Interval       int               `json:"interval,omitempty"`
	Port           int               `json:"port,omitempty"`
	ReloadPort     int               `json:"reload_port,omitempty"`
	EnabledMetrics []string          `json:"enabled_metrics,omitempty"`
	Jetson         *JetsonConfig     `json:"jetson,omitempty"`
	Shelly         *ShellyConfig     `json:"shelly,omitempty"`
	INA260         *INA260Config     `json:"ina260,omitempty"`
}

// JetsonConfig represents Jetson-specific configuration
type JetsonConfig struct {
	UseTegrastats bool `json:"use_tegrastats,omitempty"`
}

// ShellyConfig represents Shelly-specific configuration
type ShellyConfig struct {
	Host     string `json:"host,omitempty"`
	SwitchID int    `json:"switch_id,omitempty"`
}

// INA260Config represents INA260 power sensor configuration
type INA260Config struct {
	I2CAddress string `json:"i2c_address,omitempty"`
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
