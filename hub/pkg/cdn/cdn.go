package cdn

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
)

// ProcessDynamicImage toma una imagen original desde la carpeta assets del proyecto,
// la recorta a targetW x targetH y la guarda como .bin en la caché.
func ProcessDynamicImage(hash string, targetW, targetH int) (string, error) {
	dataDir := "./data"
	cdnAssetsDir := filepath.Join(dataDir, "cdn", "assets")
	os.MkdirAll(cdnAssetsDir, 0755)

	outName := fmt.Sprintf("%s_%dx%d", hash, targetW, targetH)
	binPath := filepath.Join(cdnAssetsDir, outName+".bin")

	// Si ya existe en caché, la devolvemos
	if _, err := os.Stat(binPath); err == nil {
		return binPath, nil
	}

	log.Printf("[CDN] Generando nueva imagen dinámica: %s (%dx%d)", hash, targetW, targetH)

	// Buscar original en ../assets
	origPngPath := filepath.Join("../assets", hash+".png")
	if _, err := os.Stat(origPngPath); err != nil {
		return "", fmt.Errorf("imagen original no encontrada: %s", origPngPath)
	}

	tempDir := filepath.Join(dataDir, "tmp")
	os.MkdirAll(tempDir, 0755)

	tempPyPath := filepath.Join(tempDir, "convert_dynamic_"+hash+".py")
	tempPngPath := filepath.Join(tempDir, outName+".png") // Usamos el nombre final para que LVGLImage use este nombre

	// Script de Python para Crop-to-fill
	pyScript := fmt.Sprintf(`from PIL import Image
import sys

try:
    target_w = %d
    target_h = %d
    img = Image.open(sys.argv[1]).convert("RGBA")
    
    img_ratio = img.width / img.height
    target_ratio = target_w / target_h
    
    if img_ratio > target_ratio:
        new_w = int(img.height * target_ratio)
        offset = (img.width - new_w) // 2
        img = img.crop((offset, 0, offset + new_w, img.height))
    else:
        new_h = int(img.width / target_ratio)
        offset = (img.height - new_h) // 2
        img = img.crop((0, offset, img.width, offset + new_h))
        
    img = img.resize((target_w, target_h), Image.Resampling.LANCZOS)
    img.save(sys.argv[2], "PNG")
except Exception as e:
    print("Error:", e)
    sys.exit(1)
`, targetW, targetH)

	os.WriteFile(tempPyPath, []byte(pyScript), 0644)
	defer os.Remove(tempPyPath)

	pythonBin := "/home/kaber420/Documentos/proyectos/tablehub2/firmware/venv/bin/python3"
	lvglScript := "/home/kaber420/Documentos/proyectos/tablehub2/firmware/.pio/libdeps/esp32/lvgl/scripts/LVGLImage.py"

	// Conversión Crop to PNG
	cmdPng := exec.Command(pythonBin, tempPyPath, origPngPath, tempPngPath)
	if out, err := cmdPng.CombinedOutput(); err != nil {
		return "", fmt.Errorf("error convirtiendo a PNG recortado: %v | Salida: %s", err, string(out))
	}
	defer os.Remove(tempPngPath)

	// Conversión a BIN
	cmdBin := exec.Command(pythonBin, lvglScript, "--ofmt", "BIN", "--cf", "RGB565", tempPngPath, "--name", outName, "-o", cdnAssetsDir)
	if out, err := cmdBin.CombinedOutput(); err != nil {
		return "", fmt.Errorf("error convirtiendo a BIN: %v | Salida: %s", err, string(out))
	}

	log.Printf("[CDN] Generación exitosa: %s", binPath)
	return binPath, nil
}

// HandleDynamicCDNImage sirve como endpoint HTTP para el CDN dinámico.
func HandleDynamicCDNImage(w http.ResponseWriter, r *http.Request) {
	hash := r.URL.Query().Get("hash")
	widthStr := r.URL.Query().Get("w")
	heightStr := r.URL.Query().Get("h")

	if hash == "" || widthStr == "" || heightStr == "" {
		http.Error(w, "Missing hash, w or h parameter", http.StatusBadRequest)
		return
	}

	targetW, errW := strconv.Atoi(widthStr)
	targetH, errH := strconv.Atoi(heightStr)

	if errW != nil || errH != nil {
		http.Error(w, "Invalid w or h parameter", http.StatusBadRequest)
		return
	}

	binPath, err := ProcessDynamicImage(hash, targetW, targetH)
	if err != nil {
		log.Printf("[CDN API Error] %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	http.ServeFile(w, r, binPath)
}
