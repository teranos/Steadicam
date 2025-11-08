package steadicam

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
)

// MockREPL provides a minimal test implementation of REPLModel
type MockREPL struct {
	input      string
	mode       string
	output     string
	conditions map[string]bool
}

func NewMockREPL() *MockREPL {
	return &MockREPL{
		mode:       "initial",
		conditions: make(map[string]bool),
	}
}

// tea.Model interface
func (m *MockREPL) Init() tea.Cmd { return nil }

func (m *MockREPL) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyRunes:
			m.input += string(msg.Runes)
			m.mode = "typing"
		case tea.KeyEnter:
			m.output = "processed: " + m.input
			m.mode = "result"
			m.input = ""
		case tea.KeyEsc:
			m.input = ""
			m.mode = "initial"
		}
	}
	return m, nil
}

func (m *MockREPL) View() string {
	if m.output != "" {
		return "Mode: " + m.mode + "\nOutput: " + m.output + "\nInput: " + m.input + "\n> "
	}
	return "Mode: " + m.mode + "\nInput: " + m.input + "\n> "
}

// REPLModel interface
func (m *MockREPL) CurrentInput() string { return m.input }
func (m *MockREPL) CurrentMode() string  { return m.mode }
func (m *MockREPL) CheckCondition(condition string) bool {
	switch condition {
	case "has_output":
		return m.output != ""
	case "empty_input":
		return m.input == ""
	case "has_input":
		return m.input != ""
	default:
		return m.conditions[condition]
	}
}

func (m *MockREPL) SetCondition(condition string, value bool) {
	m.conditions[condition] = value
}

func TestInteractiveTestDirector_BasicFlow(t *testing.T) {
	model := NewMockREPL()

	result := NewInteractiveTestDirector(t, model).
		WithTimeout(3 * time.Second).
		Start().
		AssertMode("initial").
		Type("hello").
		AssertMode("typing").
		AssertViewContains("Input: hello").
		PressEnter().
		WaitForMode("result").
		AssertViewContains("processed: hello").
		Stop()

	assert.True(t, result.Success, "Basic flow should succeed")
	assert.Greater(t, len(result.Interactions), 5, "Should record interactions")
}

func TestInteractiveTestDirector_Configuration(t *testing.T) {
	model := NewMockREPL()

	config := DirectorConfig{
		Timeout:      1 * time.Second,
		TypingSpeed:  10 * time.Millisecond,
		CaptureViews: true,
		MaxRetries:   2,
	}

	result := NewInteractiveTestDirectorWithConfig(t, model, config).
		Start().
		Type("test").
		AssertMode("typing").
		Stop()

	assert.True(t, result.Success)
	assert.Greater(t, len(result.Snapshots), 0, "Should capture views when enabled")
}

func TestInteractiveTestDirector_WaitConditions(t *testing.T) {
	model := NewMockREPL()

	// Test waiting for custom conditions
	go func() {
		time.Sleep(100 * time.Millisecond)
		model.SetCondition("custom_ready", true)
	}()

	result := NewInteractiveTestDirector(t, model).
		WithTimeout(2 * time.Second).
		Start().
		WaitForCondition("custom_ready").
		CheckCondition("custom_ready").
		Stop()

	assert.True(t, result.Success, "Should wait for custom conditions")
}

func TestInteractiveTestDirector_Assertions(t *testing.T) {
	model := NewMockREPL()

	result := NewInteractiveTestDirector(t, model).
		WithTimeout(2 * time.Second).
		Start().
		Type("test input").
		AssertInputEquals("test input").
		AssertViewContains("test input").
		CheckCondition("has_input").
		Stop()

	assert.True(t, result.Success, "All assertions should pass")
}

func TestInteractiveTestDirector_ErrorHandling(t *testing.T) {
	model := NewMockREPL()

	result := NewInteractiveTestDirector(t, model).
		WithTimeout(100 * time.Millisecond). // Very short timeout
		Start().
		WaitForText("text that will never appear"). // This should timeout
		Stop()

	assert.False(t, result.Success, "Should fail when timeout occurs")
	assert.Contains(t, result.ErrorMessage, "timeout", "Error should mention timeout")
}

func TestInteractiveTestDirector_KeyPresses(t *testing.T) {
	model := NewMockREPL()

	result := NewInteractiveTestDirector(t, model).
		WithTimeout(2 * time.Second).
		Start().
		Type("test").
		PressEscape(). // Should clear input
		AssertMode("initial").
		CheckCondition("empty_input").
		Stop()

	assert.True(t, result.Success, "Key press handling should work")
}

func TestDirectorConfig_Defaults(t *testing.T) {
	config := DefaultDirectorConfig()

	assert.Equal(t, 30*time.Second, config.Timeout)
	assert.Equal(t, 0*time.Millisecond, config.TypingSpeed)
	assert.True(t, config.CaptureViews)
	assert.Equal(t, 3, config.MaxRetries)
}

func TestOperator_Basic(t *testing.T) {
	model := NewMockREPL()

	result := NewOperator(t, model, "tmp/test-screenshots").
		WithTimeout(2 * time.Second).
		Start().
		CaptureTrackingShot("initial").
		TypeWithTrackingShot("hello", "typed").
		Stop()

	assert.True(t, result.Success, "Operator should work with screenshots")
}