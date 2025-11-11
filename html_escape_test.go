package steadicam

import (
	"html"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestHTMLEscapingCorrectness validates that HTML escaping doesn't double-escape
func TestHTMLEscapingCorrectness(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Basic HTML characters",
			input:    `<script>alert("xss")</script>`,
			expected: `&lt;script&gt;alert(&#34;xss&#34;)&lt;/script&gt;`,
		},
		{
			name:     "Ampersand escaping (critical test)",
			input:    `<tag attr="value">`,
			expected: `&lt;tag attr=&#34;value&#34;&gt;`,
		},
		{
			name:     "ANSI escape sequences with special chars",
			input:    "\x1b[31m<error>&msg\x1b[0m",
			expected: "\x1b[31m&lt;error&gt;&amp;msg\x1b[0m",
		},
		{
			name:     "Newlines converted to literals",
			input:    "line1\nline2\r\nline3",
			expected: `line1\nline2\r\nline3`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := escapeForHTML(tc.input)
			assert.Equal(t, tc.expected, result)

			// Critical: Ensure no double-escaping occurred
			assert.NotContains(t, result, "&amp;lt;", "Double-escaping detected: &amp;lt;")
			assert.NotContains(t, result, "&amp;gt;", "Double-escaping detected: &amp;gt;")
			assert.NotContains(t, result, "&amp;quot;", "Double-escaping detected: &amp;quot;")
		})
	}
}

// TestHTMLEscapingBugDemo demonstrates the original double-escaping bug
func TestHTMLEscapingBugDemo(t *testing.T) {
	input := `<script>`

	// Simulate the original buggy implementation (for documentation)
	buggyEscape := func(content string) string {
		// This is the WRONG order - demonstrates the bug
		content = strings.ReplaceAll(content, `<`, `&lt;`)  // "<" becomes "&lt;"
		content = strings.ReplaceAll(content, `>`, `&gt;`)  // ">" becomes "&gt;"
		content = strings.ReplaceAll(content, `&`, `&amp;`) // "&lt;" becomes "&amp;lt;" (WRONG!)
		return content
	}

	// Original buggy result would be double-escaped
	buggyResult := buggyEscape(input)
	assert.Equal(t, "&amp;lt;script&amp;gt;", buggyResult, "Bug demo: shows double-escaping")

	// Fixed implementation uses standard library (correct)
	correctResult := html.EscapeString(input)
	assert.Equal(t, "&lt;script&gt;", correctResult, "Fixed: no double-escaping")

	// Our function should match the standard library
	ourResult := escapeForHTML(input)
	// Remove newline conversions for this comparison
	ourResultClean := strings.ReplaceAll(strings.ReplaceAll(ourResult, `\n`, "\n"), `\r`, "\r")
	assert.Equal(t, correctResult, ourResultClean, "Our implementation should match html.EscapeString")
}