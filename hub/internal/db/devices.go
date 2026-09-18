package db

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
)

// Device representa un dispositivo IoT (TablePad u otros) en el sistema.
type Device struct {
	MACAddress   string    `json:"mac_address"`
	DeviceType   string    `json:"device_type"`
	TableNumber  *string   `json:"table_number"`  // Puntero para soportar null en JSON
	LocationName *string   `json:"location_name"` // Puntero para soportar null en JSON
	IPAddress    string    `json:"ip_address"`
	PublicKey    *string   `json:"public_key"`    // Puntero para soportar null en JSON
	BatteryLevel int       `json:"battery_level"`
	WiFiSignal   int       `json:"wifi_signal"`
	Status       string    `json:"status"`
	LastSeen     time.Time `json:"last_seen"`
	CreatedAt    time.Time `json:"created_at"`
}

// GetDeviceByMAC obtiene un dispositivo por su dirección MAC.
func GetDeviceByMAC(mac string) (*Device, error) {
	row := DB.QueryRow(`
		SELECT mac_address, device_type, table_number, location_name, ip_address, public_key, battery_level, wifi_signal, status, last_seen, created_at
		FROM devices
		WHERE mac_address = ?`, mac)

	var d Device
	var tableNum, locName, pubKey sql.NullString

	err := row.Scan(
		&d.MACAddress,
		&d.DeviceType,
		&tableNum,
		&locName,
		&d.IPAddress,
		&pubKey,
		&d.BatteryLevel,
		&d.WiFiSignal,
		&d.Status,
		&d.LastSeen,
		&d.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if tableNum.Valid {
		d.TableNumber = &tableNum.String
	}
	if locName.Valid {
		d.LocationName = &locName.String
	}
	if pubKey.Valid {
		d.PublicKey = &pubKey.String
	}

	return &d, nil
}

// ListDevices lista todos los dispositivos registrados en orden de creación descendente, con filtro opcional de estado.
func ListDevices(status string) ([]Device, error) {
	query := `
		SELECT mac_address, device_type, table_number, location_name, ip_address, public_key, battery_level, wifi_signal, status, last_seen, created_at
		FROM devices`
	var args []interface{}
	
	if status != "" && status != "all" {
		query += ` WHERE status = ?`
		args = append(args, status)
	}
	
	query += ` ORDER BY created_at DESC`
	
	rows, err := DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := []Device{}
	for rows.Next() {
		var d Device
		var tableNum, locName, pubKey sql.NullString

		err := rows.Scan(
			&d.MACAddress,
			&d.DeviceType,
			&tableNum,
			&locName,
			&d.IPAddress,
			&pubKey,
			&d.BatteryLevel,
			&d.WiFiSignal,
			&d.Status,
			&d.LastSeen,
			&d.CreatedAt,
		)
		if err != nil {
			return nil, err
		}

		if tableNum.Valid {
			d.TableNumber = &tableNum.String
		}
		if locName.Valid {
			d.LocationName = &locName.String
		}
		if pubKey.Valid {
			d.PublicKey = &pubKey.String
		}

		list = append(list, d)
	}

	return list, nil
}

// ApproveDevice aprueba un dispositivo, le asigna rol, mesa opcional y ubicación, y guarda la clave pública.
func ApproveDevice(mac string, deviceType string, tableNumber *string, locationName *string, publicKey *string) error {
	existing, err := GetDeviceByMAC(mac)
	if err != nil {
		return err
	}

	var tNum, lName, pKey interface{}
	if tableNumber != nil && *tableNumber != "" {
		tNum = *tableNumber
	}
	if locationName != nil && *locationName != "" {
		lName = *locationName
	}
	if publicKey != nil && *publicKey != "" {
		pKey = *publicKey
	}

	if existing == nil {
		// Si no existía por telemetría previa, lo creamos de cero
		_, err = DB.Exec(`
			INSERT INTO devices (mac_address, device_type, table_number, location_name, ip_address, public_key, status, last_seen)
			VALUES (?, ?, ?, ?, '0.0.0.0', ?, 'active', CURRENT_TIMESTAMP)`,
			mac, deviceType, tNum, lName, pKey)
	} else {
		// Si ya existía, actualizamos sus campos estructurales y activamos su estado
		if publicKey == nil && existing.PublicKey != nil {
			pKey = *existing.PublicKey
		}
		_, err = DB.Exec(`
			UPDATE devices
			SET device_type = ?, table_number = ?, location_name = ?, public_key = ?, status = 'active', last_seen = CURRENT_TIMESTAMP
			WHERE mac_address = ?`,
			deviceType, tNum, lName, pKey, mac)
	}
	return err
}

// BlockDevice cambia el estado de un dispositivo a bloqueado para ignorar sus tramas.
func BlockDevice(mac string) error {
	_, err := DB.Exec("UPDATE devices SET status = 'blocked' WHERE mac_address = ?", mac)
	return err
}

// SaveBootstrapToken guarda o actualiza un token de bootstrap asociado a un table_id.
func SaveBootstrapToken(tableID, token string) error {
	_, err := DB.Exec(`
		INSERT INTO bootstrap_tokens (table_id, token)
		VALUES (?, ?)
		ON CONFLICT(table_id) DO UPDATE SET token = excluded.token`,
		tableID, token)
	return err
}

// GetBootstrapToken obtiene el token de bootstrap para un table_id.
func GetBootstrapToken(tableID string) (string, error) {
	var token string
	err := DB.QueryRow("SELECT token FROM bootstrap_tokens WHERE table_id = ?", tableID).Scan(&token)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return token, nil
}

// ValidateAndConsumeBootstrapToken valida un token de bootstrap.
func ValidateAndConsumeBootstrapToken(tableID, token string) (bool, error) {
	if tableID == "" || token == "" {
		return false, nil
	}
	storedToken, err := GetBootstrapToken(tableID)
	if err != nil {
		return false, err
	}
	if storedToken != "" && storedToken == token {
		return true, nil
	}
	return false, nil
}

// CreatePendingDevice crea o asegura la existencia de un registro de mesa/dispositivo pendiente con su bootstrap_token.
func CreatePendingDevice(tableNumber, locationName, deviceType string) (string, error) {
	if deviceType == "" {
		deviceType = "table_pad"
	}
	
	token, err := GetBootstrapToken(tableNumber)
	if err != nil {
		return "", err
	}
	if token == "" {
		token = uuid.New().String()
		if err := SaveBootstrapToken(tableNumber, token); err != nil {
			return "", err
		}
	}

	var existingMAC string
	err = DB.QueryRow(`SELECT mac_address FROM devices WHERE table_number = ?`, tableNumber).Scan(&existingMAC)
	if err == sql.ErrNoRows {
		macPlaceholder := "PENDING-" + tableNumber
		var locName interface{}
		if locationName != "" {
			locName = locationName
		}
		_, err = DB.Exec(`
			INSERT INTO devices (mac_address, device_type, table_number, location_name, ip_address, battery_level, wifi_signal, status, last_seen)
			VALUES (?, ?, ?, ?, '0.0.0.0', -1, 0, 'unprovisioned', CURRENT_TIMESTAMP)`,
			macPlaceholder, deviceType, tableNumber, locName)
		if err != nil {
			return "", err
		}
	} else if err != nil {
		return "", err
	}

	return token, nil
}

// DeletePendingDevice elimina dispositivos pendientes placeholder asociados a un table_number.
func DeletePendingDevice(tableNumber string) error {
	_, err := DB.Exec("DELETE FROM devices WHERE table_number = ? AND mac_address LIKE 'PENDING-%'", tableNumber)
	return err
}

// DeleteDevice elimina un dispositivo del sistema.
func DeleteDevice(mac string) error {
	_, err := DB.Exec("DELETE FROM devices WHERE mac_address = ?", mac)
	return err
}

// GetDeviceSummary retorna contadores por estado de los dispositivos.
func GetDeviceSummary() (map[string]int, error) {
	rows, err := DB.Query(`SELECT status, COUNT(*) FROM devices GROUP BY status`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	summary := map[string]int{
		"all": 0,
		"active": 0,
		"unprovisioned": 0,
		"blocked": 0,
	}

	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return nil, err
		}
		summary[status] = count
		summary["all"] += count
	}

	return summary, nil
}

// GetActiveDeviceByTable devuelve la información del dispositivo activo asociado a una mesa específica.
func GetActiveDeviceByTable(tableNumber string) (*Device, error) {
	row := DB.QueryRow(`
		SELECT mac_address, device_type, table_number, location_name, ip_address, public_key, battery_level, wifi_signal, status, last_seen, created_at
		FROM devices
		WHERE table_number = ? AND status = 'active' LIMIT 1`, tableNumber)

	var d Device
	var tableNum, locName, pubKey sql.NullString

	err := row.Scan(
		&d.MACAddress,
		&d.DeviceType,
		&tableNum,
		&locName,
		&d.IPAddress,
		&pubKey,
		&d.BatteryLevel,
		&d.WiFiSignal,
		&d.Status,
		&d.LastSeen,
		&d.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if tableNum.Valid {
		d.TableNumber = &tableNum.String
	}
	if locName.Valid {
		d.LocationName = &locName.String
	}
	if pubKey.Valid {
		d.PublicKey = &pubKey.String
	}

	return &d, nil
}

