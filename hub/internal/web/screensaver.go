package web

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/tablehub/hub/internal/db"
)

// ScreensaverSettingsPayload representa los ajustes de personalización del screensaver
type ScreensaverSettingsPayload struct {
	VenueName            string `json:"venue_name"`
	VenueSubtitle        string `json:"venue_subtitle"`
	ScreensaverMode      string `json:"screensaver_mode"`       // "text" o "image"
	ScreensaverImagePath string `json:"screensaver_image_path"` // ej. "/uploads/screensaver_logo.png"
}

// HandleScreensaverSettings maneja GET y POST en /api/settings/screensaver
func HandleScreensaverSettings(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		venueName, _ := db.GetSetting("venue_name")
		venueSubtitle, _ := db.GetSetting("venue_subtitle")
		screensaverMode, _ := db.GetSetting("screensaver_mode")
		imagePath, _ := db.GetSetting("screensaver_image_path")

		if screensaverMode == "" {
			screensaverMode = "text"
		}

		resp := ScreensaverSettingsPayload{
			VenueName:            venueName,
			VenueSubtitle:        venueSubtitle,
			ScreensaverMode:      screensaverMode,
			ScreensaverImagePath: imagePath,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
		return
	}

	if r.Method == http.MethodPost {
		var req ScreensaverSettingsPayload
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
			return
		}

		db.SaveSetting("venue_name", strings.TrimSpace(req.VenueName))
		db.SaveSetting("venue_subtitle", strings.TrimSpace(req.VenueSubtitle))
		if req.ScreensaverMode == "image" || req.ScreensaverMode == "text" {
			db.SaveSetting("screensaver_mode", req.ScreensaverMode)
		}
		if req.ScreensaverImagePath != "" {
			db.SaveSetting("screensaver_image_path", req.ScreensaverImagePath)
		}

		log.Printf("[Screensaver] Configuración actualizada: venue_name='%s', mode='%s'", req.VenueName, req.ScreensaverMode)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "success"})
		return
	}

	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

// HandleScreensaverImageUpload procesa la carga de la imagen del logotipo vía POST /api/settings/screensaver/image
func HandleScreensaverImageUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse multipart form (máx 10 MB)
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		http.Error(w, "Error parsing form data", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("image")
	if err != nil {
		file, header, err = r.FormFile("file")
		if err != nil {
			http.Error(w, "No image file provided in request", http.StatusBadRequest)
			return
		}
	}
	defer file.Close()

	// Crear directorio ./data/uploads si no existe
	uploadsDir := filepath.Join("data", "uploads")
	if err := os.MkdirAll(uploadsDir, 0755); err != nil {
		http.Error(w, "Failed to create uploads directory", http.StatusInternalServerError)
		return
	}

	// Obtener extensión original o por defecto .png
	ext := filepath.Ext(header.Filename)
	if ext == "" {
		ext = ".png"
	}

	filename := "screensaver_logo" + strings.ToLower(ext)
	targetPath := filepath.Join(uploadsDir, filename)

	dst, err := os.Create(targetPath)
	if err != nil {
		http.Error(w, "Failed to save image to disk", http.StatusInternalServerError)
		return
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		http.Error(w, "Failed to write image data", http.StatusInternalServerError)
		return
	}

	publicURL := "/uploads/" + filename
	db.SaveSetting("screensaver_image_path", publicURL)
	db.SaveSetting("screensaver_mode", "image")

	log.Printf("[Screensaver] Nueva imagen guardada en %s -> %s", targetPath, publicURL)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":    "success",
		"image_url": publicURL,
	})
}

// HandlePublicScreensaverConfig endpoint público /api/device/screensaver para que las terminales obtengan los ajustes de marca
func HandlePublicScreensaverConfig(w http.ResponseWriter, r *http.Request) {
	venueName, _ := db.GetSetting("venue_name")
	venueSubtitle, _ := db.GetSetting("venue_subtitle")
	screensaverMode, _ := db.GetSetting("screensaver_mode")
	imagePath, _ := db.GetSetting("screensaver_image_path")

	if screensaverMode == "" {
		screensaverMode = "text"
	}

	resp := ScreensaverSettingsPayload{
		VenueName:            venueName,
		VenueSubtitle:        venueSubtitle,
		ScreensaverMode:      screensaverMode,
		ScreensaverImagePath: imagePath,
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	json.NewEncoder(w).Encode(resp)
}
