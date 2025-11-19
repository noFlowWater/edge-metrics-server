package database

import (
	"database/sql"
	"log"

	_ "github.com/mattn/go-sqlite3"
)

var DB *sql.DB

// InitDB initializes the SQLite database connection
func InitDB(dbPath string) error {
	var err error
	DB, err = sql.Open("sqlite3", dbPath)
	if err != nil {
		return err
	}

	// Test connection
	if err = DB.Ping(); err != nil {
		return err
	}

	// Create tables
	if err = createTables(); err != nil {
		return err
	}

	// Insert sample data if empty
	if err = insertSampleData(); err != nil {
		return err
	}

	log.Println("Database initialized successfully")
	return nil
}

// createTables creates the necessary database tables
func createTables() error {
	query := `
	CREATE TABLE IF NOT EXISTS devices (
		device_id TEXT PRIMARY KEY,
		device_type TEXT NOT NULL,
		interval INTEGER DEFAULT 1,
		port INTEGER DEFAULT 9100,
		reload_port INTEGER DEFAULT 9101,
		enabled_metrics TEXT,
		jetson_config TEXT,
		shelly_config TEXT,
		ina260_config TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	`

	_, err := DB.Exec(query)
	return err
}

// insertSampleData inserts sample device configurations if the table is empty
func insertSampleData() error {
	// Check if table is empty
	var count int
	err := DB.QueryRow("SELECT COUNT(*) FROM devices").Scan(&count)
	if err != nil {
		return err
	}

	if count > 0 {
		return nil // Data already exists
	}

	// Insert sample devices
	samples := []struct {
		deviceID       string
		deviceType     string
		interval       int
		port           int
		reloadPort     int
		enabledMetrics string
		jetsonConfig   string
		shellyConfig   string
		ina260Config   string
	}{
		{
			deviceID:       "edge-01",
			deviceType:     "jetson_orin",
			interval:       1,
			port:           9100,
			reloadPort:     9101,
			enabledMetrics: `["jetson_power_vdd_gpu_soc_watts","jetson_temp_cpu_celsius","jetson_ram_used_percent"]`,
			jetsonConfig:   `{"use_tegrastats":true}`,
		},
		{
			deviceID:       "edge-02",
			deviceType:     "jetson_xavier",
			interval:       2,
			port:           9100,
			reloadPort:     9101,
			enabledMetrics: "",
		},
		{
			deviceID:   "rpi-sensor-01",
			deviceType: "raspberry_pi",
			interval:   5,
			port:       9100,
			reloadPort: 9101,
		},
		{
			deviceID:     "shelly-plug-01",
			deviceType:   "shelly",
			interval:     10,
			port:         9100,
			reloadPort:   9101,
			shellyConfig: `{"host":"192.168.1.100","switch_id":0}`,
		},
	}

	stmt, err := DB.Prepare(`
		INSERT INTO devices (device_id, device_type, interval, port, reload_port, enabled_metrics, jetson_config, shelly_config, ina260_config)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, s := range samples {
		_, err = stmt.Exec(s.deviceID, s.deviceType, s.interval, s.port, s.reloadPort, s.enabledMetrics, s.jetsonConfig, s.shellyConfig, s.ina260Config)
		if err != nil {
			return err
		}
	}

	log.Println("Sample data inserted successfully")
	return nil
}

// CloseDB closes the database connection
func CloseDB() {
	if DB != nil {
		DB.Close()
	}
}
