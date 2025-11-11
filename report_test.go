package steadicam

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTestReport_Creation tests basic test report creation
func TestTestReport_Creation(t *testing.T) {
	report := TestReport{
		TestName:  "TestExample",
		Timestamp: "20241109_153045",
		Duration:  250 * time.Millisecond,
		Success:   true,
		Metadata: map[string]string{
			"framework": "bubbletea",
			"type":      "interactive",
		},
	}

	assert.Equal(t, "TestExample", report.TestName)
	assert.Equal(t, "20241109_153045", report.Timestamp)
	assert.Equal(t, 250*time.Millisecond, report.Duration)
	assert.True(t, report.Success)
	assert.Equal(t, "bubbletea", report.Metadata["framework"])
}

// TestScreenshotEntry_Properties tests screenshot entry structure
func TestScreenshotEntry_Properties(t *testing.T) {
	timestamp := time.Now()
	entry := ScreenshotEntry{
		Label:       "initial_state",
		Filename:    "screenshot_001.png",
		Timestamp:   timestamp,
		Step:        0,
		Description: "Initial REPL state",
	}

	assert.Equal(t, "initial_state", entry.Label)
	assert.Equal(t, "screenshot_001.png", entry.Filename)
	assert.Equal(t, timestamp, entry.Timestamp)
	assert.Equal(t, 0, entry.Step)
	assert.Equal(t, "Initial REPL state", entry.Description)
}

// TestHTMLReportGenerator_Creation tests report generator creation
func TestHTMLReportGenerator_Creation(t *testing.T) {
	tempDir := t.TempDir()
	generator := NewHTMLReportGenerator(tempDir)

	// Generator should be created successfully
	assert.NotNil(t, generator)
}

// TestHTMLReportGenerator_GenerateReport tests HTML report generation
func TestHTMLReportGenerator_GenerateReport(t *testing.T) {
	tempDir := t.TempDir()
	generator := NewHTMLReportGenerator(tempDir)

	report := TestReport{
		TestName:  "TestHTMLGeneration",
		Timestamp: "20241109_153045",
		Duration:  150 * time.Millisecond,
		Success:   true,
		Screenshots: []ScreenshotEntry{
			{
				Label:       "step_1",
				Filename:    "step_1.png",
				Timestamp:   time.Now(),
				Step:        1,
				Description: "First step",
			},
		},
		Interactions: []InteractionRecord{
			{
				Type:      "type",
				Timestamp: time.Now(),
				Details:   map[string]interface{}{"text": "hello"},
			},
		},
		Metadata: map[string]string{
			"framework": "steadicam",
			"version":   "1.0",
		},
	}

	err := generator.GenerateReport(report)
	assert.NoError(t, err)

	// Check that HTML file was created
	htmlPath := filepath.Join(tempDir, "index.html")
	assert.FileExists(t, htmlPath)

	// Read and verify HTML content
	content, err := os.ReadFile(htmlPath)
	require.NoError(t, err)

	htmlContent := string(content)
	assert.Contains(t, htmlContent, "TestHTMLGeneration")
	assert.Contains(t, htmlContent, "step_1")
	assert.Contains(t, htmlContent, "150") // Duration in some form
	assert.Contains(t, htmlContent, "<!DOCTYPE html>") // Valid HTML
	assert.Contains(t, htmlContent, "</html>") // Complete HTML
}

// TestHTMLReportGenerator_ErrorHandling tests error handling in report generation
func TestHTMLReportGenerator_ErrorHandling(t *testing.T) {
	// Use invalid directory (read-only root)
	generator := NewHTMLReportGenerator("/root/invalid")

	report := TestReport{
		TestName:  "TestError",
		Timestamp: "20241109_153045",
		Success:   false,
	}

	err := generator.GenerateReport(report)
	// Should handle error gracefully
	assert.Error(t, err)
}

// TestGenerateDashboard tests dashboard generation functionality
func TestGenerateDashboard(t *testing.T) {
	tempDir := t.TempDir()

	// Create mock test directories with proper structure: testType/timestamp/index.html
	testConfigs := []struct {
		testType  string
		timestamp string
	}{
		{"interactive_test", "20240101_120000"},
		{"visual_test", "20240101_130000"},
		{"performance_test", "20240101_140000"},
	}

	for _, config := range testConfigs {
		// Create test type directory
		testDir := filepath.Join(tempDir, config.testType)
		err := os.MkdirAll(testDir, 0755)
		require.NoError(t, err)

		// Create timestamped subdirectory
		timestampDir := filepath.Join(testDir, config.timestamp)
		err = os.MkdirAll(timestampDir, 0755)
		require.NoError(t, err)

		// Create a mock index.html in the timestamped directory
		indexPath := filepath.Join(timestampDir, "index.html")
		mockHTML := `<html><head><title>` + config.testType + `</title></head><body>Test Report for ` + config.testType + `</body></html>`
		err = os.WriteFile(indexPath, []byte(mockHTML), 0644)
		require.NoError(t, err)
	}

	// Generate dashboard
	err := GenerateDashboard(tempDir)
	assert.NoError(t, err)

	// Check that dashboard was created
	dashboardPath := filepath.Join(tempDir, "index.html")
	assert.FileExists(t, dashboardPath)

	// Verify dashboard content
	content, err := os.ReadFile(dashboardPath)
	require.NoError(t, err)

	dashboardHTML := string(content)
	assert.Contains(t, dashboardHTML, "interactive_test")
	assert.Contains(t, dashboardHTML, "visual_test")
	assert.Contains(t, dashboardHTML, "performance_test")
	assert.Contains(t, dashboardHTML, "qntx Test Dashboard")
}

// TestInteractionRecord_Structure tests interaction record structure
func TestInteractionRecord_Structure(t *testing.T) {
	timestamp := time.Now()
	record := InteractionRecord{
		Type:      "keypress",
		Timestamp: timestamp,
		Details: map[string]interface{}{
			"key":      "Enter",
			"duration": "50ms",
		},
	}

	assert.Equal(t, "keypress", record.Type)
	assert.Equal(t, timestamp, record.Timestamp)
	assert.Equal(t, "Enter", record.Details["key"])
	assert.Equal(t, "50ms", record.Details["duration"])
}

// TestSourceLine_Structure tests source line structure
func TestSourceLine_Structure(t *testing.T) {
	sourceLine := SourceLine{
		Number:      42,
		Content:     "func TestExample(t *testing.T) {",
		IsExecuting: true,
		StepIndex:   1,
	}

	assert.Equal(t, 42, sourceLine.Number)
	assert.Equal(t, "func TestExample(t *testing.T) {", sourceLine.Content)
	assert.True(t, sourceLine.IsExecuting)
	assert.Equal(t, 1, sourceLine.StepIndex)
}

// TestHTMLTemplateEmbedding tests that embedded templates are available
func TestHTMLTemplateEmbedding(t *testing.T) {
	// Verify templates are embedded
	assert.NotEmpty(t, dashboardTemplate)
	assert.NotEmpty(t, testReportTemplate)
	assert.NotEmpty(t, terminalTimelineJS)

	// Verify templates contain expected content
	assert.Contains(t, dashboardTemplate, "<!DOCTYPE html>")
	assert.Contains(t, testReportTemplate, "<!DOCTYPE html>")
	assert.Contains(t, terminalTimelineJS, "function")
}

// TestReportGeneration_Integration tests end-to-end report generation
func TestReportGeneration_Integration(t *testing.T) {
	tempDir := t.TempDir()

	// Create a realistic test report
	report := TestReport{
		TestName:  "IntegrationTest",
		Timestamp: time.Now().Format("20060102_150405"),
		Duration:  2*time.Second + 500*time.Millisecond,
		Success:   true,
		Screenshots: []ScreenshotEntry{
			{
				Label:       "initial",
				Filename:    "001_initial.png",
				Timestamp:   time.Now().Add(-2 * time.Second),
				Step:        0,
				Description: "Initial state",
			},
			{
				Label:       "typing",
				Filename:    "002_typing.png",
				Timestamp:   time.Now().Add(-1 * time.Second),
				Step:        1,
				Description: "After typing input",
			},
			{
				Label:       "result",
				Filename:    "003_result.png",
				Timestamp:   time.Now(),
				Step:        2,
				Description: "Final result",
			},
		},
		Interactions: []InteractionRecord{
			{
				Type:      "type",
				Timestamp: time.Now().Add(-1500 * time.Millisecond),
				Details:   map[string]interface{}{"text": "hello world"},
			},
			{
				Type:      "keypress",
				Timestamp: time.Now().Add(-500 * time.Millisecond),
				Details:   map[string]interface{}{"key": "Enter"},
			},
		},
		Metadata: map[string]string{
			"test_framework": "steadicam",
			"ui_framework":   "bubbletea",
			"duration_ms":    "2500",
			"screenshots":    "3",
		},
	}

	// Generate report
	generator := NewHTMLReportGenerator(tempDir)
	err := generator.GenerateReport(report)
	require.NoError(t, err)

	// Verify report file exists and has expected content
	reportPath := filepath.Join(tempDir, "index.html")
	assert.FileExists(t, reportPath)

	content, err := os.ReadFile(reportPath)
	require.NoError(t, err)

	htmlContent := string(content)

	// Verify key content is present
	assert.Contains(t, htmlContent, "IntegrationTest")
	assert.Contains(t, htmlContent, "2.5s") // Duration formatting
	assert.Contains(t, htmlContent, "Success")
	assert.Contains(t, htmlContent, "initial")
	assert.Contains(t, htmlContent, "typing")
	assert.Contains(t, htmlContent, "result")
	assert.Contains(t, htmlContent, "hello world")
	assert.Contains(t, htmlContent, "bubbletea")
	assert.Contains(t, htmlContent, "steadicam")
}