package main

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/teranos/steadicam"
)

func TestSimpleREPL_BasicInteraction(t *testing.T) {
	// Create a new REPL instance
	model := NewSimpleREPL()

	// Test with steadicam Director
	result := steadicam.NewInteractiveTestDirector(t, model).
		WithTimeout(5 * time.Second).
		Start().
		AssertMode("input").                    // Should start in input mode
		Type("world").                          // Type some text
		AssertViewContains("Input: world").     // Verify input appears
		AssertViewContains("repl>").           // Verify prompt exists
		PressEnter().                          // Submit the input
		WaitForMode("result").                 // Wait for mode change
		AssertViewContains("Hello, world").    // Verify output appears
		AssertViewContains("You entered: world"). // Verify echo works
		Stop()

	assert.True(t, result.Success, "Basic interaction test should succeed")
	assert.Greater(t, len(result.Interactions), 5, "Should record multiple interactions")
}

func TestSimpleREPL_ClearInput(t *testing.T) {
	model := NewSimpleREPL()

	result := steadicam.NewInteractiveTestDirector(t, model).
		WithTimeout(3 * time.Second).
		Start().
		Type("hello").                         // Type some text
		AssertViewContains("Input: hello").    // Verify input
		PressEscape().                         // Clear with Escape
		AssertMode("input").                   // Should return to input mode
		CheckCondition("empty_input").         // Input should be cleared
		Stop()

	assert.True(t, result.Success, "Clear input test should succeed")
}

func TestSimpleREPL_VisualTesting(t *testing.T) {
	model := NewSimpleREPL()

	// Use steadicam Operator for visual testing
	result := steadicam.NewOperator(t, model, "tmp/screenshots/simple-repl").
		WithTimeout(5 * time.Second).
		Start().
		CaptureTrackingShot("initial").                    // Capture starting state
		TypeWithTrackingShot("steadicam", "typed_input").  // Type with capture
		CaptureTrackingShot("before_submit").              // Capture before submission
		PressEnterWithTrackingShot("submitted").           // Submit with capture
		CaptureTrackingShot("final_result").               // Capture final state
		Stop()

	assert.True(t, result.Success, "Visual testing should succeed")
}

func TestSimpleREPL_Conditions(t *testing.T) {
	model := NewSimpleREPL()

	result := steadicam.NewInteractiveTestDirector(t, model).
		WithTimeout(3 * time.Second).
		Start().
		CheckCondition("input_mode").          // Should start in input mode
		CheckCondition("empty_input").         // Should start with empty input
		Type("test").                          // Add some input
		CheckCondition("has_input").           // Should now have input
		PressEnter().                          // Submit
		WaitForMode("result").                 // Wait for result mode
		CheckCondition("result_mode").         // Should be in result mode
		CheckCondition("has_output").          // Should have output
		Stop()

	assert.True(t, result.Success, "Conditions test should succeed")
}

func TestSimpleREPL_Performance(t *testing.T) {
	model := NewSimpleREPL()

	start := time.Now()

	result := steadicam.NewInteractiveTestDirector(t, model).
		WithTimeout(1 * time.Second).
		Start().
		Type("performance test").
		PressEnter().
		WaitForMode("result").
		Stop()

	duration := time.Since(start)

	assert.True(t, result.Success, "Performance test should succeed")
	assert.Less(t, duration, 500*time.Millisecond, "Test should complete quickly")
}