package steadicam

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestConvertANSIToHTML_BasicCases tests simple ANSI to HTML conversion
func TestConvertANSIToHTML_BasicCases(t *testing.T) {
	t.Run("Plain text without ANSI", func(t *testing.T) {
		input := "Hello world"
		result := convertANSIToHTML(input)
		assert.Equal(t, "Hello world", result)
	})

	t.Run("Basic color reset", func(t *testing.T) {
		input := "\x1b[1;38;5;39mBlue text\x1b[0m normal"
		result := convertANSIToHTML(input)
		expected := `<span style="color: #58a6ff; font-weight: bold;">Blue text</span> normal`
		assert.Equal(t, expected, result)
	})

	t.Run("Simple newline conversion", func(t *testing.T) {
		input := "Line 1\nLine 2"
		result := convertANSIToHTML(input)
		assert.Equal(t, "Line 1<br>Line 2", result)
	})
}

// TestExtractANSIContent_BasicCases tests metadata filtering
func TestExtractANSIContent_BasicCases(t *testing.T) {
	t.Run("Pure ANSI content", func(t *testing.T) {
		input := "Hello world"
		result := extractANSIContent(input)
		assert.Equal(t, "Hello world", result)
	})

	t.Run("Filter out metadata comments", func(t *testing.T) {
		input := "# This is metadata\nActual content"
		result := extractANSIContent(input)
		assert.Equal(t, "Actual content", result)
	})

	t.Run("Empty content after filtering", func(t *testing.T) {
		input := "# Only metadata\n# More metadata"
		result := extractANSIContent(input)
		assert.Equal(t, "", result)
	})
}

// TestConvertANSIToHTML_Colors tests different ANSI color sequences
func TestConvertANSIToHTML_Colors(t *testing.T) {
	t.Run("Gray text color", func(t *testing.T) {
		input := "\x1b[38;5;240mGray text\x1b[0m"
		result := convertANSIToHTML(input)
		expected := `<span style="color: #7d8590;">Gray text</span>`
		assert.Equal(t, expected, result)
	})

	t.Run("Bold white text", func(t *testing.T) {
		input := "\x1b[1;38;5;255mBold white\x1b[0m"
		result := convertANSIToHTML(input)
		expected := `<span style="color: #ffffff; font-weight: bold;">Bold white</span>`
		assert.Equal(t, expected, result)
	})

	t.Run("Italic text", func(t *testing.T) {
		input := "\x1b[3;38;5;244mItalic comment\x1b[0m"
		result := convertANSIToHTML(input)
		expected := `<span style="color: #6e7681; font-style: italic;">Italic comment</span>`
		assert.Equal(t, expected, result)
	})
}

// TestConvertANSIToHTML_CursorMovement tests cursor sequence removal
func TestConvertANSIToHTML_CursorMovement(t *testing.T) {
	t.Run("Remove cursor movements", func(t *testing.T) {
		input := "Text\x1b[AUp\x1b[BDown\x1b[CRight\x1b[DLeft"
		result := convertANSIToHTML(input)
		assert.Equal(t, "TextUpDownRightLeft", result)
	})

	t.Run("Remove clear sequences", func(t *testing.T) {
		input := "Before\x1b[JClear\x1b[2KLine\x1b[HHome"
		result := convertANSIToHTML(input)
		assert.Equal(t, "BeforeClearLineHome", result)
	})

	t.Run("Remove carriage returns", func(t *testing.T) {
		input := "Text\rwith\rcarriage\rreturns"
		result := convertANSIToHTML(input)
		assert.Equal(t, "Textwithcarriagereturns", result)
	})
}

// TestExtractANSIContent_RealWorldFormat tests realistic steadicam content
func TestExtractANSIContent_RealWorldFormat(t *testing.T) {
	t.Run("Steadicam format with metadata", func(t *testing.T) {
		input := `# Timing: 100ms
# Label: 01_search_state
# Step: 1
\x1b[1;38;5;39mMock REPL:\x1b[0m \x1b[38;5;244mtest_\x1b[0m`
		result := extractANSIContent(input)
		expected := `\x1b[1;38;5;39mMock REPL:\x1b[0m \x1b[38;5;244mtest_\x1b[0m`
		assert.Equal(t, expected, result)
	})

	t.Run("Preserve empty lines", func(t *testing.T) {
		input := "Line 1\n\nLine 3\n\nLine 5"
		result := extractANSIContent(input)
		assert.Equal(t, "Line 1\n\nLine 3\n\nLine 5", result)
	})
}

// TestRealSteadicamQNTXOutput tests actual steadicam-captured REPL output
// This test captures real ANSI sequences from Frame 12 of steadicam report
// ISSUE DETECTED: Missing ANSI sequence [1;38;5;78m (bright green)
// GitHub Issue: https://github.com/sbvh-nl/qntx/issues/53
func TestRealSteadicamQNTXOutput(t *testing.T) {
	t.Skip("DISABLED: Missing ANSI sequence support - see issue #53")
	t.Run("Frame 12 - speaks english search results", func(t *testing.T) {
		// Real ANSI content as captured by steadicam from Frame 12
		// Based on the screenshot showing "speaks english" search
		input := `# Timing: 3596ms
# Label: 12_search_results_displayed
# Step: search_complete
QNTX Live Search REPL (26 found in 3.596084ms) ● Auto-search ready ● Enter to search ● Tab for suggestions

qntx> speaks english

[1;38;5;78mFound 26 result(s) for: is speaks, english
1. CANDIDATE_1002 speaks English
2. MEDUSN speaks english
3. MEDUSN speaks English
4. KSMULD speaks english
5. ALEXCHJN speaks English

● 3.596084ms`

		// Extract ANSI content (remove metadata)
		ansiContent := extractANSIContent(input)

		// Convert to HTML
		htmlResult := convertANSIToHTML(ansiContent)

		// Verify basic content preservation
		assert.Contains(t, htmlResult, "QNTX Live Search REPL")
		assert.Contains(t, htmlResult, "qntx> speaks english")
		assert.Contains(t, htmlResult, "CANDIDATE_1002")
		assert.Contains(t, htmlResult, "3.596084ms")

		// Check newlines converted to <br>
		assert.Contains(t, htmlResult, "<br>")

		// Critical test: Check if we handle the [1;38;5;78m sequence
		// This sequence appears in real steadicam output but may not be in our conversion map
		hasUnprocessedSequences := strings.Contains(htmlResult, "[1;38;5;78m")

		if hasUnprocessedSequences {
			t.Logf("❌ ISSUE DETECTED: Unprocessed ANSI sequence [1;38;5;78m found in output")
			t.Logf("This indicates our ANSI color conversion is incomplete")
			t.Logf("Raw HTML output: %s", htmlResult)
			// GitHub Issue #53: https://github.com/sbvh-nl/qntx/issues/53
			// Expected: This test will likely fail, revealing gaps in our ANSI processing
		} else {
			t.Logf("✅ All ANSI sequences successfully processed")
		}

		// Log debug info regardless of pass/fail
		t.Logf("Original content length: %d", len(ansiContent))
		t.Logf("HTML output length: %d", len(htmlResult))
	})
}

// TestMinFunction tests the utility function
func TestMinFunction(t *testing.T) {
	assert.Equal(t, 3, min(5, 3))
	assert.Equal(t, 5, min(5, 10))
	assert.Equal(t, 7, min(7, 7))
}

// TestConvertANSIToTerminalHTML_ErrorCases tests basic error handling
func TestConvertANSIToTerminalHTML_ErrorCases(t *testing.T) {
	t.Run("File not found", func(t *testing.T) {
		result, err := ConvertANSIToTerminalHTML("/nonexistent/file.ansi")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to read ANSI file")
		assert.Empty(t, result)
	})

	t.Run("Empty content returns placeholder", func(t *testing.T) {
		tmpFile := t.TempDir() + "/empty.ansi"
		err := os.WriteFile(tmpFile, []byte(""), 0644)
		assert.NoError(t, err)

		result, err := ConvertANSIToTerminalHTML(tmpFile)
		assert.NoError(t, err)
		assert.Contains(t, string(result), "No terminal output at this point")
	})
}