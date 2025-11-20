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
		       enabled_metrics, extra_config, ip_address
		FROM devices
		WHERE device_id = ?
	`

	var config models.DeviceConfig
	var enabledMetrics, extraConfig, ipAddress sql.NullString

	err := database.DB.QueryRow(query, deviceID).Scan(
		&config.DeviceID,
		&config.DeviceType,
		&config.Interval,
		&config.Port,
		&config.ReloadPort,
		&enabledMetrics,
		&extraConfig,
		&ipAddress,
	)

	if ipAddress.Valid {
		config.IPAddress = ipAddress.String
	}

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

	if extraConfig.Valid && extraConfig.String != "" {
		config.ExtraConfig = make(map[string]interface{})
		if err := json.Unmarshal([]byte(extraConfig.String), &config.ExtraConfig); err != nil {
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

	// Convert slices and maps to JSON
	var enabledMetrics, extraConfig sql.NullString

	if len(config.EnabledMetrics) > 0 {
		data, err := json.Marshal(config.EnabledMetrics)
		if err != nil {
			return err
		}
		enabledMetrics = sql.NullString{String: string(data), Valid: true}
	}

	if len(config.ExtraConfig) > 0 {
		data, err := json.Marshal(config.ExtraConfig)
		if err != nil {
			return err
		}
		extraConfig = sql.NullString{String: string(data), Valid: true}
	}

	query := `
		UPDATE devices
		SET device_type = ?, interval = ?, port = ?, reload_port = ?,
		    enabled_metrics = ?, extra_config = ?, ip_address = ?, updated_at = ?
		WHERE device_id = ?
	`

	_, err = database.DB.Exec(query,
		config.DeviceType,
		config.Interval,
		config.Port,
		config.ReloadPort,
		enabledMetrics,
		extraConfig,
		config.IPAddress,
		time.Now(),
		deviceID,
	)

	return err
}

// Create creates a new device configuration
func Create(config *models.DeviceConfig) error {
	// Convert slices and maps to JSON
	var enabledMetrics, extraConfig sql.NullString

	if len(config.EnabledMetrics) > 0 {
		data, err := json.Marshal(config.EnabledMetrics)
		if err != nil {
			return err
		}
		enabledMetrics = sql.NullString{String: string(data), Valid: true}
	}

	if len(config.ExtraConfig) > 0 {
		data, err := json.Marshal(config.ExtraConfig)
		if err != nil {
			return err
		}
		extraConfig = sql.NullString{String: string(data), Valid: true}
	}

	query := `
		INSERT INTO devices (device_id, device_type, interval, port, reload_port,
		                    enabled_metrics, extra_config, ip_address)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := database.DB.Exec(query,
		config.DeviceID,
		config.DeviceType,
		config.Interval,
		config.Port,
		config.ReloadPort,
		enabledMetrics,
		extraConfig,
		config.IPAddress,
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

// Upsert creates or updates a device configuration
func Upsert(deviceID string, config *models.DeviceConfig) (bool, error) {
	exists, err := Exists(deviceID)
	if err != nil {
		return false, err
	}

	if exists {
		err = Update(deviceID, config)
		return false, err // false = updated
	}

	config.DeviceID = deviceID
	err = Create(config)
	return true, err // true = created
}

// Delete deletes a device configuration
func Delete(deviceID string) error {
	result, err := database.DB.Exec("DELETE FROM devices WHERE device_id = ?", deviceID)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return sql.ErrNoRows // Device not found
	}

	return nil
}

// GetAll retrieves all device configurations
func GetAll() ([]models.DeviceConfig, error) {
	query := `
		SELECT device_id, device_type, interval, port, reload_port,
		       enabled_metrics, extra_config, ip_address
		FROM devices
		ORDER BY device_id
	`

	rows, err := database.DB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var devices []models.DeviceConfig
	for rows.Next() {
		var config models.DeviceConfig
		var enabledMetrics, extraConfig, ipAddress sql.NullString

		err := rows.Scan(
			&config.DeviceID,
			&config.DeviceType,
			&config.Interval,
			&config.Port,
			&config.ReloadPort,
			&enabledMetrics,
			&extraConfig,
			&ipAddress,
		)
		if err != nil {
			return nil, err
		}

		if ipAddress.Valid {
			config.IPAddress = ipAddress.String
		}

		if enabledMetrics.Valid && enabledMetrics.String != "" {
			json.Unmarshal([]byte(enabledMetrics.String), &config.EnabledMetrics)
		}

		if extraConfig.Valid && extraConfig.String != "" {
			config.ExtraConfig = make(map[string]interface{})
			json.Unmarshal([]byte(extraConfig.String), &config.ExtraConfig)
		}

		devices = append(devices, config)
	}

	return devices, rows.Err()
}
