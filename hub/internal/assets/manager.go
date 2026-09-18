package assets

import (
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

const AssetsDir = "data/assets/processed"

// EnsureDirExists checks if the directory exists and creates it if not.
func EnsureDirExists(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return os.MkdirAll(path, os.ModePerm)
	}
	return nil
}

// DownloadAndConvertImage downloads an image from a URL, converts it to RGB565 and saves it to disk.
func DownloadAndConvertImage(url string, itemId string) error {
	err := EnsureDirExists(AssetsDir)
	if err != nil {
		return fmt.Errorf("failed to create assets directory: %w", err)
	}

	// 1. Download image
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to download image: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status code downloading image: %d", resp.StatusCode)
	}

	// 2. Decode image
	img, _, err := image.Decode(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to decode image: %w", err)
	}

	// 3. Convert to RGB565 Little Endian and save to file
	filename := fmt.Sprintf("item_%s_rgb565.bin", itemId)
	filePath := filepath.Join(AssetsDir, filename)
	
	outFile, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer outFile.Close()

	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// RGB565 Little Endian
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			c := img.At(x, y)
			r, g, b, _ := c.RGBA() // 0-65535 range
			
			// Scale to 5-6-5 bits
			r5 := uint16(r >> 11)
			g6 := uint16(g >> 10)
			b5 := uint16(b >> 11)
			
			rgb565 := (r5 << 11) | (g6 << 5) | b5
			
			// Write little endian
			err = binary.Write(outFile, binary.LittleEndian, rgb565)
			if err != nil {
				return fmt.Errorf("failed to write pixel data: %w", err)
			}
		}
	}

	return nil
}
