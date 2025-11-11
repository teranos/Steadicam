package steadicam

import (
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/sbvh/qntx/cmd/repl/bubble/steadicam/trip"
)

// Type simulates typing the given text character by character on the stage.
//
// Each character is sent as a separate key event with a configurable delay
// between keystrokes (see StageConfig.TypingSpeed). This provides realistic
// input simulation that matches human typing patterns.
//
// Example:
//
//	director.Type("hello world")  // Types each character with delay
//
// The text is typed exactly as provided, including spaces and special characters.
// Use key-specific methods (PressEnter, PressTab, etc.) for control keys.
func (d *StageDirector) Type(text string) *StageDirector {
	d.updateMu.Lock()
	defer d.updateMu.Unlock()

	for _, char := range text {
		msg := tea.KeyMsg{
			Type:  tea.KeyRunes,
			Runes: []rune{char},
		}

		d.sendMessage(msg)
		d.recordStageAction("type", string(char))
		if d.config.TypingSpeed > 0 {
			time.Sleep(d.config.TypingSpeed) // Configurable typing speed
		}
	}

	return d
}

// PressEnter simulates pressing the Enter key.
//
// This sends a KeyEnter event to the application, typically used to
// confirm input, execute commands, or navigate to the next field.
func (d *StageDirector) PressEnter() *StageDirector {
	d.sendMessage(tea.KeyMsg{Type: tea.KeyEnter})
	d.recordStageAction("keypress", "enter")
	return d
}

// PressTab simulates pressing the Tab key.
//
// Commonly used for autocomplete, field navigation, or triggering
// application-specific tab behavior.
func (d *StageDirector) PressTab() *StageDirector {
	d.sendMessage(tea.KeyMsg{Type: tea.KeyTab})
	d.recordStageAction("keypress", "tab")
	return d
}

// PressArrowDown simulates pressing the down arrow key.
//
// Used for navigating lists, menus, or moving the cursor down in
// multi-line input fields.
func (d *StageDirector) PressArrowDown() *StageDirector {
	d.sendMessage(tea.KeyMsg{Type: tea.KeyDown})
	d.recordStageAction("keypress", "down")
	return d
}

// PressArrowUp simulates pressing the up arrow key.
//
// Used for navigating lists, menus, or moving the cursor up in
// multi-line input fields.
func (d *StageDirector) PressArrowUp() *StageDirector {
	d.sendMessage(tea.KeyMsg{Type: tea.KeyUp})
	d.recordStageAction("keypress", "up")
	return d
}

// PressEscape simulates pressing the Escape key.
//
// Typically used to cancel operations, close dialogs, or return
// to a previous state in the application.
func (d *StageDirector) PressEscape() *StageDirector {
	d.sendMessage(tea.KeyMsg{Type: tea.KeyEsc})
	d.recordStageAction("keypress", "escape")
	return d
}

// PressBackspace simulates pressing the Backspace key.
//
// Used to delete characters from the input buffer, useful for
// correcting typos or clearing unwanted input.
func (d *StageDirector) PressBackspace() *StageDirector {
	d.sendMessage(tea.KeyMsg{Type: tea.KeyBackspace})
	d.recordStageAction("keypress", "backspace")
	return d
}

// ClearInput clears the current input by pressing backspace repeatedly.
//
// This is a convenience method that clears the entire input buffer
// by simulating multiple backspace presses.
func (d *StageDirector) ClearInput() *StageDirector {
	// Get current input length to know how many backspaces to send
	currentInput := d.getCurrentInput()
	for range currentInput {
		d.PressBackspace()
		// Small delay between backspaces for realistic input clearing
		if d.config.TypingSpeed > 0 {
			d.Wait(d.config.TypingSpeed / 4) // Faster than typing speed
		}
	}
	return d
}

// Wait pauses stage execution for the specified duration.
//
// Use this method when you need to wait for a specific amount of time,
// for example to allow animations to complete or to simulate user thinking time.
//
// For waiting on application state changes, prefer WaitForMode, WaitForText,
// or other condition-based wait methods.
//
// Example:
//
//	director.Type("hello").Wait(100*time.Millisecond).PressEnter()
func (d *StageDirector) Wait(duration time.Duration) *StageDirector {
	time.Sleep(duration)
	d.recordStageAction("wait", duration)
	d.captureSnapshot("wait")
	return d
}

// AssertViewContains verifies that the current view contains the specified text
func (d *StageDirector) AssertViewContains(text string) *StageDirector {
	view := d.getCurrentView()
	if !strings.Contains(view, text) {
		trip := newStageTrip("assertion", "View does not contain expected text: "+text, map[string]interface{}{"expected": text, "actual_view": view})
		d.recordTrip(trip)
		return d
	}
	d.recordStageAction("assertion", "contains="+text)
	return d
}

// AssertMode verifies that the REPL is in the expected mode
func (d *StageDirector) AssertMode(expectedMode string) *StageDirector {
	actualMode := d.getCurrentMode()
	if actualMode != expectedMode {
		trip := newStageTrip("assertion", "Expected mode "+expectedMode+", got "+actualMode, map[string]interface{}{"expected": expectedMode, "actual": actualMode})
		d.recordTrip(trip)
		return d
	}
	d.recordStageAction("assertion", "mode="+expectedMode)
	return d
}

// AssertInputEquals verifies that the current input matches the expected value
func (d *StageDirector) AssertInputEquals(expected string) *StageDirector {
	actual := d.getCurrentInput()
	if actual != expected {
		trip := newStageTrip("assertion", "Expected input '"+expected+"', got '"+actual+"'", map[string]interface{}{"expected": expected, "actual": actual})
		d.recordTrip(trip)
		return d
	}
	d.recordStageAction("assertion", "input="+expected)
	return d
}

// AssertNoSearchResults verifies that no search results are currently displayed
func (d *StageDirector) AssertNoSearchResults() *StageDirector {
	if d.latestModel.CheckCondition("search_results") {
		trip := newStageTrip("assertion", "Expected no search results, but found some", nil)
		d.recordTrip(trip)
		return d
	}
	d.recordStageAction("assertion", "no_search_results")
	return d
}

// sendMessage sends a message to the bubbletea program
func (d *StageDirector) sendMessage(msg tea.Msg) {
	if d.program != nil {
		currentView := d.getCurrentView()
		d.t.Logf("[TRACE] sendMessage: About to send message type=%T", msg)
		d.t.Logf("[TRACE] sendMessage: Current view length=%d, first_50_chars=%q",
			len(currentView), d.truncateString(currentView, 50))

		sendStart := time.Now()
		d.program.Send(msg)
		d.t.Logf("[TRACE] sendMessage: Message sent in %v, waiting for view change...", time.Since(sendStart))

		// Wait for the UI to update by checking for view changes
		d.waitForViewChange(currentView)
		d.captureSnapshot("interaction")

		finalView := d.getCurrentView()
		d.t.Logf("[TRACE] sendMessage: Complete. Final view length=%d, changed=%t",
			len(finalView), finalView != currentView)
	}
}

// recordStageAction logs an interaction step
func (d *StageDirector) recordStageAction(actionType string, details interface{}) {
	d.interactions = append(d.interactions, StageAction{
		Timestamp: time.Now(),
		Type:      actionType,
		Details:   details,
	})
}

// captureSnapshot captures the current state of the REPL
func (d *StageDirector) captureSnapshot(reason string) {
	if !d.config.CaptureViews {
		return
	}

	snapshot := StageSnapshot{
		Timestamp: time.Now(),
		View:      d.getCurrentView(),
		Mode:      d.getCurrentMode(),
		Input:     d.getCurrentInput(),
	}

	d.snapshots = append(d.snapshots, snapshot)
}

// recordTrip records a trip using the trip handler and marks stage as failed if needed
func (d *StageDirector) recordTrip(trip *trip.Trip) {
	d.tripHandler.Record(trip)
	d.lastTrip = trip

	// Only mark as failed for non-recoverable trips
	if !trip.CanRecover() {
		d.failed = true
	}

	if d.t != nil {
		d.t.Helper()
		if trip.IsFall() {
			d.t.Error(trip) // Report critical trips to testing framework
		} else {
			d.t.Log(trip.DetailedString()) // Log other trips for debugging
		}
	}
}

// recordError maintains compatibility with existing error handling
func (d *StageDirector) recordError(err error) {
	// Convert generic error to trip
	trip := trip.NewTrip("system", err.Error(), nil)
	d.recordTrip(trip)
}

// HasFailed returns true if the stage has encountered any errors
func (d *StageDirector) HasFailed() bool {
	return d.failed || !d.tripHandler.ShouldContinue()
}

// GetError returns the last error encountered (for compatibility)
func (d *StageDirector) GetError() error {
	if d.lastTrip != nil {
		return d.lastTrip
	}
	return nil
}

// GetTripHandler returns the trip handler for detailed error analysis
func (d *StageDirector) GetTripHandler() *trip.Handler {
	return d.tripHandler
}