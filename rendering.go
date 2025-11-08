package steadicam

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"regexp"
	"strings"

	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
)

// Config defines the visual parameters for smooth UI tracking shots
// Kubrick's meticulous approach to camera movement and composition
type Config struct {
	Width       int        // Terminal width in characters
	Height      int        // Terminal height in characters
	FontSize    int        // Font size in pixels
	Background  color.RGBA // Background color
	Foreground  color.RGBA // Default text color
	OutputDir   string     // Directory to save film frames
}

// RenderingStage renders terminal output to image buffers with fluid motion tracking
// Like Kubrick's revolutionary steadicam work in The Shining's hotel corridors
type RenderingStage struct {
	config     Config
	buffer     [][]rune       // Character buffer
	colorMap   [][]color.RGBA // Color information per character
	charWidth  int            // Character width in pixels
	charHeight int            // Character height in pixels
	font       font.Face      // Font for rendering
}

// NewRenderingStage creates a camera rig that can capture smooth UI tracking shots
func NewRenderingStage(config Config) *RenderingStage {
	// Ensure output directory exists
	if config.OutputDir != "" {
		os.MkdirAll(config.OutputDir, 0755)
	}

	return &RenderingStage{
		config:     config,
		buffer:     make([][]rune, config.Height),
		colorMap:   make([][]color.RGBA, config.Height),
		charWidth:  8,  // Basic font character width
		charHeight: 16, // Basic font character height
		font:       basicfont.Face7x13, // Use basic font for now
	}
}

// RenderText processes terminal output and updates the virtual buffer
func (rs *RenderingStage) RenderText(terminalOutput string) {
	lines := strings.Split(terminalOutput, "\n")

	// Initialize or clear buffer (reuse existing allocations)
	for i := range rs.buffer {
		// Initialize buffer if needed
		if rs.buffer[i] == nil {
			rs.buffer[i] = make([]rune, rs.config.Width)
		} else {
			// Clear existing buffer
			for j := range rs.buffer[i] {
				rs.buffer[i][j] = ' '
			}
		}

		// Initialize color map if needed
		if rs.colorMap[i] == nil {
			rs.colorMap[i] = make([]color.RGBA, rs.config.Width)
		} else {
			// Clear existing colors
			for j := range rs.colorMap[i] {
				rs.colorMap[i][j] = rs.config.Foreground
			}
		}
	}

	// Process each line of terminal output
	for lineIdx, line := range lines {
		if lineIdx >= rs.config.Height {
			break
		}

		// Parse ANSI escape sequences and render text
		rs.renderLine(lineIdx, line)
	}
}

// renderLine processes a single line with ANSI escape sequences
func (rs *RenderingStage) renderLine(lineIdx int, line string) {
	// Strip ANSI escape sequences for now (can be enhanced to parse colors)
	cleanLine := rs.stripANSI(line)

	for charIdx, char := range []rune(cleanLine) {
		if charIdx >= rs.config.Width {
			break
		}

		rs.buffer[lineIdx][charIdx] = char
		rs.colorMap[lineIdx][charIdx] = rs.config.Foreground
	}
}

// stripANSI removes ANSI escape sequences
func (rs *RenderingStage) stripANSI(text string) string {
	// Remove common ANSI escape sequences
	ansiRegex := regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)
	return ansiRegex.ReplaceAllString(text, "")
}

// CaptureFrame renders the current buffer to a PNG image
// Smooth, continuous tracking shot like Kubrick's flowing camera movements
func (rs *RenderingStage) CaptureFrame(filename string) error {
	width := rs.config.Width * rs.charWidth
	height := rs.config.Height * rs.charHeight

	img := image.NewRGBA(image.Rect(0, 0, width, height))

	// Fill background
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, rs.config.Background)
		}
	}

	// Draw text
	drawer := &font.Drawer{
		Dst:  img,
		Src:  image.NewUniform(rs.config.Foreground),
		Face: rs.font,
	}

	for lineIdx, line := range rs.buffer {
		if line == nil {
			continue
		}

		for charIdx, char := range line {
			if char == ' ' || char == 0 {
				continue
			}

			x := charIdx * rs.charWidth
			y := (lineIdx + 1) * rs.charHeight

			drawer.Dot = fixed.Point26_6{
				X: fixed.Int26_6(x << 6),
				Y: fixed.Int26_6(y << 6),
			}

			drawer.DrawString(string(char))
		}
	}

	// Save to file
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	return png.Encode(file, img)
}