package repository

import (
	"database/sql"
	"edge-metrics-server/database"
	"edge-metrics-server/models"
	"encoding/json"
	"time"
)

// GetByDeviceID retrieves a device configuration by device ID
func GetByDeviceID(deviceID string) (*models.DeviceConfig, error) {
	query := `
		SELECT device_id, device_type, interval, port, reload_port,
		       enabled_metrics, jetson_config, shelly_config, ina260_config
		FROM devices
		WHERE device_id = ?
	`

	var config models.DeviceConfig
	var enabledMetrics, jetsonConfig, shellyConfig, ina260Config sql.NullString

	err := database.DB.QueryRow(query, deviceID).Scan(
		&config.DeviceID,
		&config.DeviceType,
		&config.Interval,
		&config.Port,
		&config.ReloadPort,
		&enabledMetrics,
		&jetsonConfig,
		&shellyConfig,
		&ina260Config,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // Device not found
		}
		return nil, err
	}

	// Parse JSON fields
	if enabledMetrics.Valid && enabledMetrics.String != "" {
		if err := json.Unmarshal([]byte(enabledMetrics.String), &config.EnabledMetrics); err != nil {
			return nil, err
		}
	}

	if jetsonConfig.Valid && jetsonConfig.String != "" {
		config.Jetson = &models.JetsonConfig{}
		if err := json.Unmarshal([]byte(jetsonConfig.String), config.Jetson); err != nil {
			return nil, err
		}
	}

	if shellyConfig.Valid && shellyConfig.String != "" {
		config.Shelly = &models.ShellyConfig{}
		if err := json.Unmarshal([]byte(shellyConfig.String), config.Shelly); err != nil {
			return nil, err
		}
	}

	if ina260Config.Valid && ina260Config.String != "" {
		config.INA260 = &models.INA260Config{}
		if err := json.Unmarshal([]byte(ina260Config.String), config.INA260); err != nil {
			return nil, err
		}
	}

	return &config, nil
}

// Update updates an existing device configuration
func Update(deviceID string, config *models.DeviceConfig) error {
	// Check if device exists
	existing, err := GetByDeviceID(deviceID)
	if err != nil {
		return err
	}
	if existing == nil {
		return sql.ErrNoRows // Device not found
	}

	// Convert slices and structs to JSON
	var enabledMetrics, jetsonConfig, shellyConfig, ina260Config sql.NullString

	if len(config.EnabledMetrics) > 0 {
		data, err := json.Marshal(config.EnabledMetrics)
		if err != nil {
			return err
		}
		enabledMetrics = sql.NullString{String: string(data), Valid: true}
	}

	if config.Jetson != nil {
		data, err := json.Marshal(config.Jetson)
		if err != nil {
			return err
		}
		jetsonConfig = sql.NullString{String: string(data), Valid: true}
	}

	if config.Shelly != nil {
		data, err := json.Marshal(config.Shelly)
		if err != nil {
			return err
		}
		shellyConfig = sql.NullString{String: string(data), Valid: true}
	}

	if config.INA260 != nil {
		data, err := json.Marshal(config.INA260)
		if err != nil {
			return err
		}
		ina260Config = sql.NullString{String: string(data), Valid: true}
	}

	query := `
		UPDATE devices
		SET device_type = ?, interval = ?, port = ?, reload_port = ?,
		    enabled_metrics = ?, jetson_config = ?, shelly_config = ?, ina260_config = ?,
		    updated_at = ?
		WHERE device_id = ?
	`

	_, err = database.DB.Exec(query,
		config.DeviceType,
		config.Interval,
		config.Port,
		config.ReloadPort,
		enabledMetrics,
		jetsonConfig,
		shellyConfig,
		ina260Config,
		time.Now(),
		deviceID,
	)

	return err
}

// Create creates a new device configuration
func Create(config *models.DeviceConfig) error {
	// Convert slices and structs to JSON
	var enabledMetrics, jetsonConfig, shellyConfig, ina260Config sql.NullString

	if len(config.EnabledMetrics) > 0 {
		data, err := json.Marshal(config.EnabledMetrics)
		if err != nil {
			return err
		}
		enabledMetrics = sql.NullString{String: string(data), Valid: true}
	}

	if config.Jetson != nil {
		data, err := json.Marshal(config.Jetson)
		if err != nil {
			return err
		}
		jetsonConfig = sql.NullString{String: string(data), Valid: true}
	}

	if config.Shelly != nil {
		data, err := json.Marshal(config.Shelly)
		if err != nil {
			return err
		}
		shellyConfig = sql.NullString{String: string(data), Valid: true}
	}

	if config.INA260 != nil {
		data, err := json.Marshal(config.INA260)
		if err != nil {
			return err
		}
		ina260Config = sql.NullString{String: string(data), Valid: true}
	}

	query := `
		INSERT INTO devices (device_id, device_type, interval, port, reload_port,
		                    enabled_metrics, jetson_config, shelly_config, ina260_config)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := database.DB.Exec(query,
		config.DeviceID,
		config.DeviceType,
		config.Interval,
		config.Port,
		config.ReloadPort,
		enabledMetrics,
		jetsonConfig,
		shellyConfig,
		ina260Config,
	)

	return err
}

// Exists checks if a device exists
func Exists(deviceID string) (bool, error) {
	var count int
	err := database.DB.QueryRow("SELECT COUNT(*) FROM devices WHERE device_id = ?", deviceID).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}
