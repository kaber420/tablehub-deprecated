package db

import (
	"database/sql"
	"time"
)

type WifiNetwork struct {
	ID        int       `json:"id"`
	SSID      string    `json:"ssid"`
	Password  string    `json:"password,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

func ListWifiNetworks() ([]WifiNetwork, error) {
	rows, err := DB.Query(`SELECT id, ssid, created_at FROM wifi_networks ORDER BY ssid ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := []WifiNetwork{}
	for rows.Next() {
		var n WifiNetwork
		if err := rows.Scan(&n.ID, &n.SSID, &n.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, n)
	}
	return list, nil
}

func SaveWifiNetwork(ssid, password string) error {
	_, err := DB.Exec(`
		INSERT INTO wifi_networks (ssid, password)
		VALUES (?, ?)
		ON CONFLICT(ssid) DO UPDATE SET password = excluded.password`,
		ssid, password)
	return err
}

func GetWifiNetworkByID(id int) (*WifiNetwork, error) {
	var n WifiNetwork
	err := DB.QueryRow(`SELECT id, ssid, password, created_at FROM wifi_networks WHERE id = ?`, id).
		Scan(&n.ID, &n.SSID, &n.Password, &n.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &n, nil
}

func DeleteWifiNetwork(id int) error {
	_, err := DB.Exec(`DELETE FROM wifi_networks WHERE id = ?`, id)
	return err
}
