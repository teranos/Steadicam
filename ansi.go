package steadicam

import (
	"fmt"
	"html/template"
	"os"
	"strings"
)

// convertANSIToTerminalHTML reads an ANSI file and prepares it for HTML terminal emulator
func ConvertANSIToTerminalHTML(ansiPath string) (template.HTML, error) {
	// Read the ANSI file
	ansiBytes, err := os.ReadFile(ansiPath)
	if err != nil {
		return "", fmt.Errorf("failed to read ANSI file: %w", err)
	}

	// Extract only the actual ANSI content, skip metadata headers
	ansiContent := extractANSIContent(string(ansiBytes))
	if ansiContent == "" {
		// Create a placeholder for empty content
		return template.HTML(`<div style="color: #666;">No terminal output at this point</div>`), nil
	}

	// Convert ANSI to HTML
	htmlContent := convertANSIToHTML(ansiContent)
	return template.HTML(htmlContent), nil
}

// convertANSIToHTML converts ANSI escape sequences to HTML with colors using a state machine
func convertANSIToHTML(ansiText string) string {
	var result strings.Builder
	i := 0

	for i < len(ansiText) {
		char := ansiText[i]

		// Skip carriage returns
		if char == '\r' {
			i++
			continue
		}

		// Convert newlines to HTML
		if char == '\n' {
			result.WriteString("<br>")
			i++
			continue
		}

		// Check for ANSI escape sequence
		if char == '\x1b' && i+1 < len(ansiText) && ansiText[i+1] == '[' {
			// Find the end of the ANSI sequence
			i += 2 // Skip \x1b[

			// Collect the sequence
			var seqBuilder strings.Builder
			for i < len(ansiText) {
				c := ansiText[i]
				if (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') {
					// Found the terminator
					seqBuilder.WriteByte(c)
					i++
					break
				}
				seqBuilder.WriteByte(c)
				i++
			}

			sequence := seqBuilder.String()

			// Convert known sequences to HTML
			if html := convertSequenceToHTML(sequence); html != "" {
				result.WriteString(html)
			}
			// Skip unknown sequences (cursor movements, clears, etc.)

		} else {
			// Regular character
			result.WriteByte(char)
			i++
		}
	}

	return result.String()
}

// convertSequenceToHTML converts a single ANSI sequence to HTML
func convertSequenceToHTML(sequence string) string {
	switch sequence {
	case "0m":
		return "</span>"
	case "1;38;5;39m":
		return `<span style="color: #58a6ff; font-weight: bold;">`
	case "38;5;240m":
		return `<span style="color: #7d8590;">`
	case "1;38;5;255m":
		return `<span style="color: #ffffff; font-weight: bold;">`
	case "38;5;246m":
		return `<span style="color: #8b949e;">`
	case "38;5;244m":
		return `<span style="color: #6e7681;">`
	case "1;38;5;255;48;5;240m":
		return `<span style="color: #ffffff; background: #7d8590; font-weight: bold;">`
	case "3;38;5;244m":
		return `<span style="color: #6e7681; font-style: italic;">`
	default:
		// Skip unknown sequences (cursor movements, clears, etc.)
		return ""
	}
}

// escapeForHTML escapes ANSI content for safe embedding in HTML attributes
// Only escapes dangerous HTML characters while preserving ANSI escape sequences
func escapeForHTML(content string) string {
	// Only escape the minimum necessary characters for HTML attribute safety
	// Do NOT use html.EscapeString() as it converts ANSI control chars to Unicode escapes
	content = strings.ReplaceAll(content, "&", "&amp;")   // Must be first
	content = strings.ReplaceAll(content, "\"", "&#34;")   // Escape quotes for attribute
	content = strings.ReplaceAll(content, "'", "&#39;")    // Escape single quotes
	content = strings.ReplaceAll(content, "<", "&lt;")     // Escape angle brackets
	content = strings.ReplaceAll(content, ">", "&gt;")
	// Convert newlines to literal \n for data attribute
	content = strings.ReplaceAll(content, "\n", `\n`)
	content = strings.ReplaceAll(content, "\r", `\r`)
	return content
}

// extractANSIContent filters out metadata headers and cleans problematic ANSI sequences
func extractANSIContent(content string) string {
	lines := strings.Split(content, "\n")
	var ansiLines []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Skip comment lines that start with # (timing metadata)
		if strings.HasPrefix(trimmed, "#") {
			continue
		}

		// Include all non-comment lines, including empty lines (they may be significant for terminal layout)
		ansiLines = append(ansiLines, line)
	}

	result := strings.Join(ansiLines, "\n")
	result = strings.TrimSpace(result) // Clean up leading/trailing whitespace

	// Don't clean ANSI sequences - let buildkite/terminal-to-html handle them properly

	// Debug log for empty content
	if result == "" {
		fmt.Printf("DEBUG: No ANSI content extracted from content starting with: %s\n", content[:min(100, len(content))])
	} else {
		fmt.Printf("DEBUG: Extracted %d bytes of ANSI content\n", len(result))
	}
	return result
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}