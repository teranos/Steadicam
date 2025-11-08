package steadicam

import (
	_ "embed"
	"encoding/base64"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

//go:embed html_templates/dashboard.html
var dashboardTemplate string

//go:embed html_templates/test_report.html
var testReportTemplate string

// TestReport represents a complete test execution report
type TestReport struct {
	TestName     string              `json:"test_name"`
	Timestamp    string              `json:"timestamp"`
	Duration     time.Duration       `json:"duration"`
	Success      bool                `json:"success"`
	Screenshots  []ScreenshotEntry   `json:"screenshots"`
	Interactions []InteractionRecord `json:"interactions"`
	Metadata     map[string]string   `json:"metadata"`
}

// ScreenshotEntry represents a single screenshot with context
type ScreenshotEntry struct {
	Label       string       `json:"label"`
	Filename    string       `json:"filename"`
	Timestamp   time.Time    `json:"timestamp"`
	Step        int          `json:"step"`
	Description string       `json:"description"`
	DataURL     template.URL `json:"data_url"` // Base64 encoded data URL for embedding
}

// InteractionRecord represents a user interaction during testing
type InteractionRecord struct {
	Type      string                 `json:"type"`
	Timestamp time.Time              `json:"timestamp"`
	Details   map[string]interface{} `json:"details"`
}

// HTMLReportGenerator creates visual test reports
type HTMLReportGenerator struct {
	outputDir     string
	templateCache map[string]*template.Template
}

// DashboardEntry represents a single test report for the dashboard
type DashboardEntry struct {
	TestName      string    `json:"test_name"`
	Timestamp     string    `json:"timestamp"`
	Success       bool      `json:"success"`
	ScreenshotCount int     `json:"screenshot_count"`
	Duration      string    `json:"duration"`
	ReportPath    string    `json:"report_path"`
	RelativePath  string    `json:"relative_path"`
	CreatedAt     time.Time `json:"created_at"`
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

// generateMainReport creates the primary test result HTML
func (g *HTMLReportGenerator) generateMainReport(report TestReport) error {
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

	tmpl := template.Must(template.New("main").Parse(testReportTemplate))
	g.templateCache["main"] = tmpl
	return tmpl
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

// GenerateDashboard creates a central dashboard HTML file for all test reports
func GenerateDashboard(baseDir string) error {
	// Scan for all test report directories
	entries, err := scanTestReports(baseDir)
	if err != nil {
		return fmt.Errorf("failed to scan test reports: %w", err)
	}

	// Generate dashboard HTML
	dashboardPath := filepath.Join(baseDir, "index.html")
	file, err := os.Create(dashboardPath)
	if err != nil {
		return fmt.Errorf("failed to create dashboard file: %w", err)
	}
	defer file.Close()

	tmpl := getDashboardTemplate()

	dashboardData := struct {
		Reports []DashboardEntry
		GeneratedAt time.Time
	}{
		Reports: entries,
		GeneratedAt: time.Now(),
	}

	if err := tmpl.Execute(file, dashboardData); err != nil {
		return fmt.Errorf("failed to execute dashboard template: %w", err)
	}

	fmt.Printf("📊 Dashboard generated: %s\n", dashboardPath)
	fmt.Printf("🔗 Found %d test reports\n", len(entries))

	return nil
}

// scanTestReports finds all test reports in the base directory
func scanTestReports(baseDir string) ([]DashboardEntry, error) {
	var entries []DashboardEntry

	// Walk through all subdirectories
	err := filepath.Walk(baseDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Look for index.html files in timestamped directories
		if info.Name() == "index.html" && path != filepath.Join(baseDir, "index.html") {
			// Extract directory info
			dir := filepath.Dir(path)
			timestamp := filepath.Base(dir)

			// Validate timestamp format (20060102_150405)
			if _, err := time.Parse("20060102_150405", timestamp); err == nil {
				// Extract test type from parent directory
				testType := filepath.Base(filepath.Dir(dir))

				// Try to extract basic report info
				entry := DashboardEntry{
					TestName:     testType,
					Timestamp:    timestamp,
					ReportPath:   path,
					RelativePath: getRelativePath(baseDir, path),
					CreatedAt:    info.ModTime(),
				}

				// Try to extract more info by parsing the HTML (basic extraction)
				if reportInfo, err := extractReportInfo(path); err == nil {
					entry.Success = reportInfo.Success
					entry.ScreenshotCount = reportInfo.ScreenshotCount
					entry.Duration = reportInfo.Duration
					entry.TestName = reportInfo.TestName
				}

				entries = append(entries, entry)
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// Sort by creation time (newest first)
	for i := 0; i < len(entries)-1; i++ {
		for j := i + 1; j < len(entries); j++ {
			if entries[j].CreatedAt.After(entries[i].CreatedAt) {
				entries[i], entries[j] = entries[j], entries[i]
			}
		}
	}

	return entries, nil
}

// extractReportInfo extracts basic info from an existing HTML report
func extractReportInfo(htmlPath string) (*DashboardEntry, error) {
	content, err := os.ReadFile(htmlPath)
	if err != nil {
		return nil, err
	}

	html := string(content)
	entry := &DashboardEntry{}

	// Extract test name (simple regex-based extraction)
	if matches := regexp.MustCompile(`<title>(.+?) - qntx Test Report</title>`).FindStringSubmatch(html); len(matches) > 1 {
		entry.TestName = matches[1]
	}

	// Extract success status
	entry.Success = strings.Contains(html, ">PASSED<")

	// Extract screenshot count
	if matches := regexp.MustCompile(`<strong>Screenshots:</strong> (\d+) captured`).FindStringSubmatch(html); len(matches) > 1 {
		if count, err := parseInt(matches[1]); err == nil {
			entry.ScreenshotCount = count
		}
	}

	// Extract duration
	if matches := regexp.MustCompile(`<strong>Duration:</strong> (.+?)</p>`).FindStringSubmatch(html); len(matches) > 1 {
		entry.Duration = matches[1]
	}

	return entry, nil
}

// getRelativePath returns a relative path from base to target
func getRelativePath(base, target string) string {
	rel, err := filepath.Rel(base, target)
	if err != nil {
		return target
	}
	return rel
}

// parseInt safely converts string to int
func parseInt(s string) (int, error) {
	var result int
	for _, r := range s {
		if r < '0' || r > '9' {
			return 0, fmt.Errorf("invalid integer: %s", s)
		}
		result = result*10 + int(r-'0')
	}
	return result, nil
}

// getDashboardTemplate returns the HTML template for the central test dashboard
func getDashboardTemplate() *template.Template {
	return template.Must(template.New("dashboard").Parse(dashboardTemplate))
}