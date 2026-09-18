package web

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/tablehub/hub/internal/db"
	"golang.org/x/crypto/bcrypt"
)

var (
	isSetupComplete bool
	jwtSecret       []byte
)

// InitAuth lee o genera el secreto JWT y verifica si el setup ya se realizó.
func InitAuth() {
	count, _ := db.CountUsers()
	if count > 0 {
		isSetupComplete = true
	} else {
		isSetupComplete = false
		log.Println("Hub en Modo Setup: No se han configurado usuarios administradores.")
	}

	// Cargar o generar JWT Secret
	secretPath := "./data/hub_secret.key"
	secret, err := os.ReadFile(secretPath)
	if err == nil && len(secret) > 0 {
		jwtSecret = secret
	} else {
		log.Println("Generando nuevo secreto JWT...")
		os.MkdirAll(filepath.Dir(secretPath), 0755)
		newSecret := make([]byte, 32)
		_, err := rand.Read(newSecret)
		if err != nil {
			log.Fatalf("Error crítico al generar secreto aleatorio: %v", err)
		}
		// Guardar con permisos 0600 (solo lectura/escritura para el dueño)
		err = os.WriteFile(secretPath, newSecret, 0600)
		if err != nil {
			log.Fatalf("Error guardando secreto JWT: %v", err)
		}
		jwtSecret = newSecret
	}
}

// GenerateToken crea un nuevo JWT para un usuario.
func GenerateToken(role string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"role": role,
		"exp":  time.Now().Add(24 * time.Hour).Unix(),
		"iat":  time.Now().Unix(),
	})
	return token.SignedString(jwtSecret)
}

// VerifyToken verifica la validez del token y devuelve error si es inválido o está expirado.
func VerifyToken(tokenString string) error {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Validar que el método de firma sea el esperado
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("método de firma inesperado")
		}
		return jwtSecret, nil
	})

	if err != nil {
		return err
	}

	if !token.Valid {
		return errors.New("token inválido")
	}

	return nil
}

// AuthMiddleware intercepta las rutas protegidas para validar la cookie HttpOnly.
func AuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("token")
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		if err := VerifyToken(cookie.Value); err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Token es válido, pasar a la siguiente función
		next.ServeHTTP(w, r)
	}
}

// IsAuthenticated verifica si la petición actual tiene un token válido
func IsAuthenticated(r *http.Request) bool {
	cookie, err := r.Cookie("token")
	if err != nil {
		return false
	}
	return VerifyToken(cookie.Value) == nil
}

// HandleAuthCheck informa al cliente del estado actual (setup_required o authenticated)
func HandleAuthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if !isSetupComplete {
		json.NewEncoder(w).Encode(map[string]string{"status": "setup_required"})
		return
	}

	cookie, err := r.Cookie("token")
	if err == nil && VerifyToken(cookie.Value) == nil {
		json.NewEncoder(w).Encode(map[string]string{"status": "authenticated"})
		return
	}

	// Si no está autenticado, pero el setup está completo:
	http.Error(w, "Unauthorized", http.StatusUnauthorized)
}

// HandleSetup guarda la primera contraseña y desactiva el endpoint en memoria.
func HandleSetup(w http.ResponseWriter, r *http.Request) {
	if isSetupComplete {
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}

	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	if len(req.Password) < 8 {
		http.Error(w, "La contraseña debe tener al menos 8 caracteres", http.StatusBadRequest)
		return
	}
	if req.Username == "" {
		http.Error(w, "El nombre de usuario es obligatorio", http.StatusBadRequest)
		return
	}

	_, err := db.CreateUser(req.Username, "admin", req.Password)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Bloquear el endpoint de forma irreversible hasta el próximo reinicio
	isSetupComplete = true
	log.Println("Setup inicial completado. Usuario administrador configurado.")

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// HandleLogin verifica las credenciales y devuelve una cookie HttpOnly.
func HandleLogin(w http.ResponseWriter, r *http.Request) {
	if !isSetupComplete {
		http.Error(w, "Forbidden: Setup required", http.StatusForbidden)
		return
	}

	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	user, hash, err := db.GetUserByName(req.Username)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if !user.Active {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Comparar hash
	err = bcrypt.CompareHashAndPassword([]byte(hash), []byte(req.Password))
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Generar Token
	tokenString, err := GenerateToken(user.Role)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Enviar cookie segura
	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    tokenString,
		Expires:  time.Now().Add(24 * time.Hour),
		HttpOnly: true,
		Secure:   false, // PONER A TRUE SI SE USA HTTPS EN PRODUCCIÓN
		SameSite: http.SameSiteStrictMode,
		Path:     "/",
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// HandleLogout destruye la cookie de sesión.
func HandleLogout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    "",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   false, // PONER A TRUE SI SE USA HTTPS
		SameSite: http.SameSiteStrictMode,
		Path:     "/",
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
