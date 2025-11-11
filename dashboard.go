package steadicam

import (
	"encoding/json"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// DashboardEntry represents a single test report for the dashboard
type DashboardEntry struct {
	TestName        string    `json:"test_name"`
	Timestamp       string    `json:"timestamp"`
	Success         bool      `json:"success"`
	ScreenshotCount int       `json:"screenshot_count"`
	Duration        string    `json:"duration"`
	ReportPath      string    `json:"report_path"`
	RelativePath    string    `json:"relative_path"`
	CreatedAt       time.Time `json:"created_at"`
}

// TestMetadata represents the structured data embedded in test reports
type TestMetadata struct {
	TestName   string `json:"testName"`
	Duration   string `json:"duration"`
	FrameCount int    `json:"frameCount"`
	Timestamp  string `json:"timestamp"`
	Success    bool   `json:"success"`
	ReportType string `json:"reportType"`
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
		Reports     []DashboardEntry
		GeneratedAt time.Time
	}{
		Reports:     entries,
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

				// Try to extract more info by parsing the HTML (structured data approach)
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

// extractReportInfo extracts basic info from an HTML report using structured JSON metadata
func extractReportInfo(htmlPath string) (*DashboardEntry, error) {
	content, err := os.ReadFile(htmlPath)
	if err != nil {
		return nil, err
	}

	// Try to extract structured JSON metadata first
	if entry, err := extractFromJSON(string(content)); err == nil {
		return entry, nil
	}

	// For legacy reports without JSON metadata, return empty entry
	// This eliminates regex-based HTML parsing entirely
	return &DashboardEntry{}, nil
}

// extractFromJSON extracts test metadata from embedded JSON
func extractFromJSON(htmlContent string) (*DashboardEntry, error) {
	// Find the JSON metadata script block
	start := strings.Index(htmlContent, `<script type="application/json" id="test-metadata">`)
	if start == -1 {
		return nil, fmt.Errorf("no JSON metadata found")
	}

	// Find the JSON opening brace
	jsonStart := strings.Index(htmlContent[start:], "{")
	if jsonStart == -1 {
		return nil, fmt.Errorf("no JSON opening brace found in metadata")
	}
	start = jsonStart + start

	// Find the script closing tag
	scriptEnd := strings.Index(htmlContent[start:], "</script>")
	if scriptEnd == -1 {
		return nil, fmt.Errorf("no script closing tag found")
	}
	end := scriptEnd + start

	if end <= start {
		return nil, fmt.Errorf("malformed JSON metadata")
	}

	jsonStr := strings.TrimSpace(htmlContent[start:end])

	var metadata TestMetadata
	if err := json.Unmarshal([]byte(jsonStr), &metadata); err != nil {
		return nil, fmt.Errorf("failed to parse JSON metadata: %w", err)
	}

	return &DashboardEntry{
		TestName:        metadata.TestName,
		Duration:        metadata.Duration,
		Success:         metadata.Success,
		ScreenshotCount: metadata.FrameCount,
	}, nil
}

// getRelativePath returns a relative path from base to target
func getRelativePath(base, target string) string {
	rel, err := filepath.Rel(base, target)
	if err != nil {
		return target
	}
	return rel
}


// getDashboardTemplate returns the HTML template for the central test dashboard
func getDashboardTemplate() *template.Template {
	return template.Must(template.New("dashboard").Parse(dashboardTemplate))
}