package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

// DB es la instancia global de conexión de base de datos.
var DB *sql.DB

// InitDB inicializa la base de datos SQLite en la ruta proporcionada.
func InitDB(dbPath string) error {
	// Asegurar que el directorio de la base de datos existe
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("error al crear directorio para la DB: %w", err)
	}

	var err error
	// El driver 'sqlite' de modernc.org se registra como "sqlite"
	DB, err = sql.Open("sqlite", dbPath)
	if err != nil {
		return fmt.Errorf("error al abrir base de datos SQLite: %w", err)
	}

	if err = DB.Ping(); err != nil {
		return fmt.Errorf("error de conexión con SQLite: %w", err)
	}

	// Ejecutar migraciones iniciales
	if err = runMigrations(); err != nil {
		return fmt.Errorf("error al ejecutar migraciones: %w", err)
	}

	return nil
}

// runMigrations crea las tablas necesarias si no existen.
func runMigrations() error {
	// Tabla de configuración general
	querySettings := `
	CREATE TABLE IF NOT EXISTS settings (
		key TEXT PRIMARY KEY,
		value TEXT NOT NULL
	);`

	// Tabla de monitoreo de dispositivos IoT (TablePads y otros periféricos)
	queryDevices := `
	CREATE TABLE IF NOT EXISTS devices (
		mac_address TEXT PRIMARY KEY,
		device_type TEXT NOT NULL DEFAULT 'table_pad',
		table_number TEXT,
		location_name TEXT,
		ip_address TEXT NOT NULL,
		public_key TEXT,
		battery_level INTEGER DEFAULT 100,
		wifi_signal INTEGER DEFAULT 0,
		status TEXT DEFAULT 'unprovisioned',
		last_seen TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);`

	// Tabla de log de eventos locales (para modo offline)
	queryLogs := `
	CREATE TABLE IF NOT EXISTS event_logs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		event_type TEXT NOT NULL,
		payload TEXT NOT NULL,
		synced INTEGER DEFAULT 0,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);`

	// Tabla de usuarios (Staff/Meseros)
	queryUsers := `
	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		role TEXT NOT NULL,
		pin_hash TEXT NOT NULL,
		active INTEGER DEFAULT 1,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);`

	// Tabla de tokens de arranque (bootstrap) para aprovisionamiento de dispositivos
	queryBootstrapTokens := `
	CREATE TABLE IF NOT EXISTS bootstrap_tokens (
		table_id TEXT PRIMARY KEY,
		token TEXT NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);`

	// Tabla de redes Wi-Fi guardadas para aprovisionamiento
	queryWifiNetworks := `
	CREATE TABLE IF NOT EXISTS wifi_networks (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		ssid TEXT NOT NULL UNIQUE,
		password TEXT NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);`

	statements := []string{querySettings, queryDevices, queryLogs, queryUsers, queryBootstrapTokens, queryWifiNetworks}

	for _, stmt := range statements {
		_, err := DB.Exec(stmt)
		if err != nil {
			return err
		}
	}

	// Ejecutar migraciones incrementales para agregar nuevas columnas a tablas existentes
	// Se ignoran errores en caso de que las columnas ya existan.
	_, _ = DB.Exec("ALTER TABLE devices ADD COLUMN device_type TEXT NOT NULL DEFAULT 'table_pad'")
	_, _ = DB.Exec("ALTER TABLE devices ADD COLUMN location_name TEXT")
	_, _ = DB.Exec("ALTER TABLE devices ADD COLUMN public_key TEXT")
	_, _ = DB.Exec("ALTER TABLE devices ADD COLUMN created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP")

	return nil
}

// GetSetting obtiene un valor de configuración por su clave.
func GetSetting(key string) (string, error) {
	var value string
	err := DB.QueryRow("SELECT value FROM settings WHERE key = ?", key).Scan(&value)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return value, nil
}

// SaveSetting guarda o actualiza una clave de configuración.
func SaveSetting(key, value string) error {
	_, err := DB.Exec(`
		INSERT INTO settings (key, value) 
		VALUES (?, ?) 
		ON CONFLICT(key) DO UPDATE SET value = excluded.value`, 
		key, value)
	return err
}

// UpdateDeviceStatus actualiza el estado y telemetría de un terminal TablePad o dispositivo IoT.
// Inserta el dispositivo si es nuevo (ej: telemetría antes de aprobación) o actualiza campos no estructurales si ya existe.
func UpdateDeviceStatus(mac, ip string, battery, wifi int, status string) error {
	_, err := DB.Exec(`
		INSERT INTO devices (mac_address, ip_address, battery_level, wifi_signal, status, last_seen)
		VALUES (?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(mac_address) DO UPDATE SET
			ip_address = excluded.ip_address,
			battery_level = excluded.battery_level,
			wifi_signal = excluded.wifi_signal,
			status = excluded.status,
			last_seen = CURRENT_TIMESTAMP`,
		mac, ip, battery, wifi, status)
	return err
}

// SetDeviceOffline marca un dispositivo como desconectado sin alterar su telemetría
func SetDeviceOffline(mac string) error {
	_, err := DB.Exec(`
		UPDATE devices 
		SET status = 'offline', last_seen = CURRENT_TIMESTAMP
		WHERE mac_address = ?`, mac)
	return err
}
