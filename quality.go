package steadicam

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
)

// ScriptSupervisor ensures visual consistency between takes using familiar QA patterns
// Leon Vitali's obsessive attention to detail in Kubrick productions
type ScriptSupervisor struct {
	baselineDir string
	currentDir  string
	tolerance   float64 // Percentage difference tolerance
}

// NewScriptSupervisor creates a new visual regression validator
func NewScriptSupervisor(baselineDir, currentDir string) *ScriptSupervisor {
	return &ScriptSupervisor{
		baselineDir: baselineDir,
		currentDir:  currentDir,
		tolerance:   0.05, // 5% difference tolerance
	}
}

// ValidateConsistency compares current tracking shot with baseline
func (ss *ScriptSupervisor) ValidateConsistency(testName string) error {
	baselinePath := fmt.Sprintf("%s/%s.png", ss.baselineDir, testName)
	currentPath := fmt.Sprintf("%s/%s.png", ss.currentDir, testName)

	// Load images
	baseline, err := ss.loadImage(baselinePath)
	if err != nil {
		return fmt.Errorf("failed to load baseline: %w", err)
	}

	current, err := ss.loadImage(currentPath)
	if err != nil {
		return fmt.Errorf("failed to load current: %w", err)
	}

	// Calculate difference
	difference := ss.calculateDifference(baseline, current)

	// Generate diff image if significant difference
	if difference > ss.tolerance {
		diffPath := fmt.Sprintf("%s/%s_diff.png", ss.currentDir, testName)
		if err := ss.generateDiffImage(baseline, current, diffPath); err != nil {
			// Log error but don't fail the main validation
			fmt.Printf("Warning: failed to generate diff image: %v\n", err)
		}

		return fmt.Errorf("visual regression detected: %.2f%% difference (tolerance: %.2f%%)",
			difference*100, ss.tolerance*100)
	}

	return nil
}

// loadImage loads an image from file
func (ss *ScriptSupervisor) loadImage(path string) (image.Image, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	return img, err
}

// calculateDifference calculates the percentage difference between two images
func (ss *ScriptSupervisor) calculateDifference(img1, img2 image.Image) float64 {
	bounds1 := img1.Bounds()
	bounds2 := img2.Bounds()

	// Images must have same dimensions
	if bounds1 != bounds2 {
		return 1.0 // 100% different
	}

	totalPixels := (bounds1.Max.X - bounds1.Min.X) * (bounds1.Max.Y - bounds1.Min.Y)
	differentPixels := 0

	for y := bounds1.Min.Y; y < bounds1.Max.Y; y++ {
		for x := bounds1.Min.X; x < bounds1.Max.X; x++ {
			if img1.At(x, y) != img2.At(x, y) {
				differentPixels++
			}
		}
	}

	return float64(differentPixels) / float64(totalPixels)
}

// generateDiffImage creates a visual diff highlighting differences
func (ss *ScriptSupervisor) generateDiffImage(baseline, current image.Image, outputPath string) error {
	bounds := baseline.Bounds()
	diff := image.NewRGBA(bounds)

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			baseColor := baseline.At(x, y)
			currColor := current.At(x, y)

			if baseColor != currColor {
				// Highlight differences in red
				diff.Set(x, y, color.RGBA{255, 0, 0, 255})
			} else {
				// Keep original color but dimmed
				r, g, b, a := baseColor.RGBA()
				diff.Set(x, y, color.RGBA{
					uint8(r >> 9), // Dim by dividing by 2 (shift right)
					uint8(g >> 9),
					uint8(b >> 9),
					uint8(a >> 8),
				})
			}
		}
	}

	// Save diff image
	file, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer file.Close()

	return png.Encode(file, diff)
}

// SetBaseline saves the current tracking shot as a baseline for future comparisons
func (ss *ScriptSupervisor) SetBaseline(testName, trackingShotPath string) error {
	baselinePath := fmt.Sprintf("%s/%s.png", ss.baselineDir, testName)

	// Ensure baseline directory exists
	if err := os.MkdirAll(ss.baselineDir, 0755); err != nil {
		return fmt.Errorf("failed to create baseline directory: %w", err)
	}

	// Copy tracking shot to baseline
	input, err := os.Open(trackingShotPath)
	if err != nil {
		return err
	}
	defer input.Close()

	output, err := os.Create(baselinePath)
	if err != nil {
		return err
	}
	defer output.Close()

	_, err = output.ReadFrom(input)
	return err
}