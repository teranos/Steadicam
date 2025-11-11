package steadicam

import (
	_ "embed"
	"encoding/base64"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode"
)

//go:embed html_templates/dashboard.html
var dashboardTemplate string

//go:embed html_templates/main_report.html
var testReportTemplate string

//go:embed html_templates/terminal-timeline.js
var terminalTimelineJS string

//go:embed html_templates/stream_report.html
var streamReportTemplate string

//go:embed html_templates/stream-player.js
var streamPlayerJS string

//go:embed html_templates/terminal-state-player.js
var terminalStatePlayerJS string

// TestReport represents a complete test execution report
type TestReport struct {
	TestName     string              `json:"test_name"`
	Timestamp    string              `json:"timestamp"`
	Duration     time.Duration       `json:"duration"`
	Success      bool                `json:"success"`
	ErrorMessage string              `json:"error_message,omitempty"`
	ErrorDetails string              `json:"error_details,omitempty"`
	TripReport   string              `json:"trip_report,omitempty"`
	Screenshots  []ScreenshotEntry   `json:"screenshots"`
	Interactions []InteractionRecord `json:"interactions"`
	SourceLines  []SourceLine        `json:"source_lines"`
	SourceFile   string              `json:"source_file"`
	Metadata     map[string]string   `json:"metadata"`
}

// ScreenshotEntry represents a single screenshot with context
type ScreenshotEntry struct {
	Label       string        `json:"label"`
	Filename    string        `json:"filename"`
	Timestamp   time.Time     `json:"timestamp"`
	Step        int           `json:"step"`
	Description string        `json:"description"`
	DataURL     template.URL  `json:"data_url"`  // Base64 encoded data URL for embedding (images)
	HTMLContent template.HTML `json:"html_content"` // Rendered HTML content (ANSI)
	IsANSI      bool          `json:"is_ansi"`   // True if this is ANSI content, false for images
}

// InteractionRecord represents a user interaction during testing
type InteractionRecord struct {
	Type      string                 `json:"type"`
	Timestamp time.Time              `json:"timestamp"`
	Details   map[string]interface{} `json:"details"`
}

// SourceLine represents a line of source code
type SourceLine struct {
	Number      int    `json:"line_number"`
	Content     string `json:"content"`
	IsExecuting bool   `json:"is_executing"`
	StepIndex   int    `json:"step_index"`
}

// HTMLReportGenerator creates visual test reports
type HTMLReportGenerator struct {
	outputDir     string
	templateCache map[string]*template.Template
}


// NewHTMLReportGenerator creates a new report generator
func NewHTMLReportGenerator(outputDir string) *HTMLReportGenerator {
	return &HTMLReportGenerator{
		outputDir:     outputDir,
		templateCache: make(map[string]*template.Template),
	}
}

// GenerateReport creates an HTML report from test results
func (g *HTMLReportGenerator) GenerateReport(report TestReport) error {
	// Ensure output directory exists
	if err := os.MkdirAll(g.outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Generate main report HTML
	if err := g.generateMainReport(report); err != nil {
		return fmt.Errorf("failed to generate main report: %w", err)
	}

	return nil
}

// GenerateReportAsync creates an HTML report asynchronously to avoid blocking tests
func (g *HTMLReportGenerator) GenerateReportAsync(report TestReport) <-chan error {
	errChan := make(chan error, 1)

	go func() {
		defer close(errChan)
		if err := g.GenerateReport(report); err != nil {
			errChan <- err
		}
	}()

	return errChan
}

// GenerateReportBackground creates HTML report in background with optional callback
func (g *HTMLReportGenerator) GenerateReportBackground(report TestReport, callback func(error)) {
	go func() {
		err := g.GenerateReport(report)
		if callback != nil {
			callback(err)
		}
	}()
}

// generateMainReport creates the primary test result HTML
func (g *HTMLReportGenerator) generateMainReport(report TestReport) error {
	// Create the JavaScript file
	jsPath := filepath.Join(g.outputDir, "terminal-timeline.js")
	if err := os.WriteFile(jsPath, []byte(terminalTimelineJS), 0644); err != nil {
		return fmt.Errorf("failed to write JavaScript file: %w", err)
	}

	// Create the HTML file
	tmpl := g.getMainTemplate()

	reportPath := filepath.Join(g.outputDir, "index.html")
	file, err := os.Create(reportPath)
	if err != nil {
		return err
	}
	defer file.Close()

	return tmpl.Execute(file, report)
}


// getMainTemplate returns the HTML template for main test reports
func (g *HTMLReportGenerator) getMainTemplate() *template.Template {
	if tmpl, exists := g.templateCache["main"]; exists {
		return tmpl
	}

	// Define custom template functions
	funcMap := template.FuncMap{
		"mul": func(a, b float64) float64 {
			return a * b
		},
		"add": func(a, b int) int {
			return a + b
		},
		"title": func(s string) string {
			if s == "" {
				return s
			}
			runes := []rune(s)
			runes[0] = unicode.ToUpper(runes[0])
			return string(runes)
		},
		"replace": strings.ReplaceAll,
		"extractTiming": extractFrameTiming,
	}

	tmpl := template.Must(template.New("main").Funcs(funcMap).Parse(testReportTemplate))
	g.templateCache["main"] = tmpl
	return tmpl
}


// extractFrameTiming extracts timing information from a label using structured parsing
func extractFrameTiming(label string) string {
	// Parse frame number from structured label format "N_description"
	if len(label) == 0 {
		return "0"
	}

	// Find the first underscore to separate frame number from description
	underscoreIndex := -1
	for i, r := range label {
		if r == '_' {
			underscoreIndex = i
			break
		}
	}

	if underscoreIndex == -1 {
		return "0" // No structured format found
	}

	// Extract the frame number part
	frameStr := label[:underscoreIndex]

	// Convert frame string to numeric timing (frame * 100ms)
	var frameNum int
	for _, r := range frameStr {
		if r < '0' || r > '9' {
			return "0" // Invalid frame number
		}
		frameNum = frameNum*10 + int(r-'0')
	}

	// Convert frame number to milliseconds (frame * 100ms)
	timing := frameNum * 100
	return fmt.Sprintf("%d", timing)
}

// convertImageToDataURL reads an image file and converts it to a base64 data URL
func convertImageToDataURL(imagePath string) (template.URL, error) {
	// Read the image file
	imageBytes, err := os.ReadFile(imagePath)
	if err != nil {
		return "", fmt.Errorf("failed to read image file: %w", err)
	}

	// Determine MIME type based on file extension
	ext := strings.ToLower(filepath.Ext(imagePath))
	var mimeType string
	switch ext {
	case ".png":
		mimeType = "image/png"
	case ".jpg", ".jpeg":
		mimeType = "image/jpeg"
	case ".gif":
		mimeType = "image/gif"
	case ".webp":
		mimeType = "image/webp"
	default:
		mimeType = "image/png" // Default to PNG
	}

	// Encode to base64
	base64Data := base64.StdEncoding.EncodeToString(imageBytes)

	// Create data URL
	dataURL := fmt.Sprintf("data:%s;base64,%s", mimeType, base64Data)

	return template.URL(dataURL), nil
}




// StreamReportGenerator creates stream-based visual test reports
type StreamReportGenerator struct {
	outputDir string
}

// NewStreamReportGenerator creates a new stream report generator
func NewStreamReportGenerator(outputDir string) *StreamReportGenerator {
	return &StreamReportGenerator{
		outputDir: outputDir,
	}
}

// GenerateStreamReport creates an HTML report for streaming terminal output
func (g *StreamReportGenerator) GenerateStreamReport(report TestReport) error {
	// Ensure output directory exists
	if err := os.MkdirAll(g.outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Create the terminal state player JavaScript file
	jsPath := filepath.Join(g.outputDir, "terminal-state-player.js")
	if err := os.WriteFile(jsPath, []byte(terminalStatePlayerJS), 0644); err != nil {
		return fmt.Errorf("failed to write terminal state player JavaScript: %w", err)
	}

	// Create the HTML file with template functions
	funcMap := template.FuncMap{
		"mul": func(a, b float64) float64 {
			return a * b
		},
		"add": func(a, b int) int {
			return a + b
		},
		"title": func(s string) string {
			if s == "" {
				return s
			}
			runes := []rune(s)
			runes[0] = unicode.ToUpper(runes[0])
			return string(runes)
		},
		"replace": strings.ReplaceAll,
	}

	tmpl := template.Must(template.New("stream").Funcs(funcMap).Parse(streamReportTemplate))

	reportPath := filepath.Join(g.outputDir, "index.html")
	file, err := os.Create(reportPath)
	if err != nil {
		return err
	}
	defer file.Close()

	return tmpl.Execute(file, report)
}