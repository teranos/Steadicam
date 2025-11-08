package steadicam

import (
	"time"
	tea "github.com/charmbracelet/bubbletea"
)

// Type simulates typing the given text character by character.
//
// Each character is sent as a separate key event with a configurable delay
// between keystrokes (see DirectorConfig.TypingSpeed). This provides realistic
// input simulation that matches human typing patterns.
//
// Example:
//
//	director.Type("hello world")  // Types each character with delay
//
// The text is typed exactly as provided, including spaces and special characters.
// Use key-specific methods (PressEnter, PressTab, etc.) for control keys.
func (d *InteractiveTestDirector) Type(text string) *InteractiveTestDirector {
	d.updateMu.Lock()
	defer d.updateMu.Unlock()

	for _, char := range text {
		msg := tea.KeyMsg{
			Type:  tea.KeyRunes,
			Runes: []rune{char},
		}

		d.sendMessage(msg)
		d.recordInteraction("type", string(char))
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
func (d *InteractiveTestDirector) PressEnter() *InteractiveTestDirector {
	d.sendMessage(tea.KeyMsg{Type: tea.KeyEnter})
	d.recordInteraction("keypress", "enter")
	return d
}

// PressTab simulates pressing the Tab key.
//
// Commonly used for autocomplete, field navigation, or triggering
// application-specific tab behavior.
func (d *InteractiveTestDirector) PressTab() *InteractiveTestDirector {
	d.sendMessage(tea.KeyMsg{Type: tea.KeyTab})
	d.recordInteraction("keypress", "tab")
	return d
}

// PressArrowDown simulates pressing the down arrow key.
//
// Used for navigating lists, menus, or moving the cursor down in
// multi-line input fields.
func (d *InteractiveTestDirector) PressArrowDown() *InteractiveTestDirector {
	d.sendMessage(tea.KeyMsg{Type: tea.KeyDown})
	d.recordInteraction("keypress", "down")
	return d
}

// PressArrowUp simulates pressing the up arrow key.
//
// Used for navigating lists, menus, or moving the cursor up in
// multi-line input fields.
func (d *InteractiveTestDirector) PressArrowUp() *InteractiveTestDirector {
	d.sendMessage(tea.KeyMsg{Type: tea.KeyUp})
	d.recordInteraction("keypress", "up")
	return d
}

// PressEscape simulates pressing the Escape key.
//
// Typically used to cancel operations, close dialogs, or return
// to a previous state in the application.
func (d *InteractiveTestDirector) PressEscape() *InteractiveTestDirector {
	d.sendMessage(tea.KeyMsg{Type: tea.KeyEsc})
	d.recordInteraction("keypress", "escape")
	return d
}

// Wait pauses test execution for the specified duration.
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
func (d *InteractiveTestDirector) Wait(duration time.Duration) *InteractiveTestDirector {
	time.Sleep(duration)
	d.recordInteraction("wait", duration)
	d.captureSnapshot("wait")
	return d
}