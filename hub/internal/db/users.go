package db

import (
	"database/sql"
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

type User struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	Role      string `json:"role"`
	Active    bool   `json:"active"`
	CreatedAt string `json:"created_at"`
}

// CreateUser crea un nuevo usuario en la base de datos
func CreateUser(name, role, pin string) (int64, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(pin), bcrypt.DefaultCost)
	if err != nil {
		return 0, fmt.Errorf("error hashing pin: %w", err)
	}

	result, err := DB.Exec(`
		INSERT INTO users (name, role, pin_hash, active) 
		VALUES (?, ?, ?, 1)`,
		name, role, string(hash))
	
	if err != nil {
		return 0, err
	}
	
	return result.LastInsertId()
}

// GetUsers obtiene todos los usuarios (excepto el PIN hash por seguridad)
func GetUsers() ([]User, error) {
	rows, err := DB.Query(`SELECT id, name, role, active, created_at FROM users`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var u User
		var activeInt int
		if err := rows.Scan(&u.ID, &u.Name, &u.Role, &activeInt, &u.CreatedAt); err != nil {
			return nil, err
		}
		u.Active = activeInt == 1
		users = append(users, u)
	}
	
	if err := rows.Err(); err != nil {
		return nil, err
	}
	
	// Para devolver array vacío y no null en JSON
	if users == nil {
		users = []User{}
	}
	
	return users, nil
}

// UpdateUser actualiza los detalles de un usuario
func UpdateUser(id int, name, role string, active bool, pin string) error {
	activeInt := 0
	if active {
		activeInt = 1
	}
	
	if pin != "" {
		hash, err := bcrypt.GenerateFromPassword([]byte(pin), bcrypt.DefaultCost)
		if err != nil {
			return fmt.Errorf("error hashing pin: %w", err)
		}
		result, err := DB.Exec(`
			UPDATE users 
			SET name = ?, role = ?, active = ?, pin_hash = ?
			WHERE id = ?`,
			name, role, activeInt, string(hash), id)
		if err != nil {
			return err
		}
		rowsAffected, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if rowsAffected == 0 {
			return fmt.Errorf("user not found")
		}
		return nil
	}

	result, err := DB.Exec(`
		UPDATE users 
		SET name = ?, role = ?, active = ?
		WHERE id = ?`,
		name, role, activeInt, id)
		
	if err != nil {
		return err
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return fmt.Errorf("user not found")
	}
	
	return nil
}

// DeleteUser elimina un usuario
func DeleteUser(id int) error {
	result, err := DB.Exec(`DELETE FROM users WHERE id = ?`, id)
	if err != nil {
		return err
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return fmt.Errorf("user not found")
	}
	
	return nil
}

// UpdateUserPIN cambia el PIN de un usuario
func UpdateUserPIN(id int, newPin string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(newPin), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("error hashing pin: %w", err)
	}

	_, err = DB.Exec(`UPDATE users SET pin_hash = ? WHERE id = ?`, string(hash), id)
	return err
}

// VerifyUserPIN verifica si el PIN proporcionado es correcto para un usuario
func VerifyUserPIN(id int, pin string) (bool, error) {
	var hash string
	err := DB.QueryRow("SELECT pin_hash FROM users WHERE id = ?", id).Scan(&hash)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, err
	}

	err = bcrypt.CompareHashAndPassword([]byte(hash), []byte(pin))
	if err != nil {
		return false, nil // Hash mismatch
	}

	return true, nil
}

// CountUsers cuenta el número total de usuarios registrados en la base de datos
func CountUsers() (int, error) {
	var count int
	err := DB.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
	return count, err
}

// GetUserByName obtiene un usuario y su hash de contraseña por su nombre
func GetUserByName(name string) (*User, string, error) {
	var u User
	var hash string
	var activeInt int
	
	err := DB.QueryRow(`SELECT id, name, role, active, created_at, pin_hash FROM users WHERE name = ?`, name).
		Scan(&u.ID, &u.Name, &u.Role, &activeInt, &u.CreatedAt, &hash)
		
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, "", fmt.Errorf("user not found")
		}
		return nil, "", err
	}
	
	u.Active = activeInt == 1
	return &u, hash, nil
}
