package tool

import (
	"fmt"
	"image"
	"image/png"
	"os"

	"github.com/kbinani/screenshot"
)

// CaptureScreen captures the primary display and saves it as a PNG file.
func CaptureScreen(filename string) error {
	n := screenshot.NumActiveDisplays()
	if n == 0 {
		return fmt.Errorf("no active displays found")
	}

	bounds := screenshot.GetDisplayBounds(0)
	img, err := screenshot.CaptureRect(bounds)
	if err != nil {
		return fmt.Errorf("failed to capture screen: %w", err)
	}

	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	err = png.Encode(file, img)
	if err != nil {
		return fmt.Errorf("failed to encode image: %w", err)
	}

	return nil
}

func GetDiffBounds(before, after *image.RGBA) image.Rectangle {
	minX, minY := after.Bounds().Max.X, after.Bounds().Max.Y
	maxX, maxY := 0, 0
	changed := false

	// Optimization: Scan every 4th pixel to save CPU
	for y := after.Bounds().Min.Y; y < after.Bounds().Max.Y; y += 4 {
		for x := after.Bounds().Min.X; x < after.Bounds().Max.X; x += 4 {
			off := after.PixOffset(x, y)
			// Compare RGB values (ignore Alpha for speed)
			if after.Pix[off] != before.Pix[off] ||
				after.Pix[off+1] != before.Pix[off+1] ||
				after.Pix[off+2] != before.Pix[off+2] {

				if x < minX {
					minX = x
				}
				if x > maxX {
					maxX = x
				}
				if y < minY {
					minY = y
				}
				if y > maxY {
					maxY = y
				}
				changed = true
			}
		}
	}

	if !changed {
		return image.ZR
	} // No change detected

	// Add 20px padding so the AI sees the edges of the panel
	return image.Rect(minX-20, minY-20, maxX+20, maxY+20).Canon().Intersect(after.Bounds())
}
