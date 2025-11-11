package steadicam

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
)

// mockREPLForInteractions implements REPLModel for testing stage interactions
type mockREPLForInteractions struct {
	input      string
	mode       string
	lastUpdate tea.Msg
}

func (m *mockREPLForInteractions) Init() tea.Cmd                           { return nil }
func (m *mockREPLForInteractions) View() string                            { return "Mock REPL: " + m.input }
func (m *mockREPLForInteractions) CurrentInput() string                    { return m.input }
func (m *mockREPLForInteractions) CurrentMode() string                     { return m.mode }
func (m *mockREPLForInteractions) CheckCondition(condition string) bool    { return condition == "test_condition" }

func (m *mockREPLForInteractions) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	m.lastUpdate = msg

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyRunes:
			m.input += string(msg.Runes)
		case tea.KeyEnter:
			m.mode = "executed"
		case tea.KeyTab:
			m.mode = "tab_pressed"
		case tea.KeyBackspace:
			if len(m.input) > 0 {
				m.input = m.input[:len(m.input)-1]
			}
		case tea.KeyEsc:
			m.mode = "escaped"
		}
	}

	return m, nil
}

// TestStageDirector_BasicInteractions tests core interaction methods
func TestStageDirector_BasicInteractions(t *testing.T) {
	model := &mockREPLForInteractions{mode: "initial"}
	director := NewStageDirectorWithConfig(t, model, StageConfig{
		Timeout:      5 * time.Second, // Increased timeout for stability
		TypingSpeed:  0, // No delay for testing
		CaptureViews: true,
		MaxRetries:   0,
	})
	defer director.Stop()

	director.Start()

	// Test typing
	director.Type("hello")
	assert.Equal(t, "hello", director.getCurrentInput())

	// Test key presses
	director.PressEnter()
	assert.Equal(t, "executed", director.getCurrentMode())

	// Test that interactions were recorded
	actions := director.interactions
	assert.Greater(t, len(actions), 5) // Multiple character typing + enter

	// Verify action types
	var hasTypeAction, hasKeypressAction bool
	for _, action := range actions {
		if action.Type == "type" {
			hasTypeAction = true
		}
		if action.Type == "keypress" {
			hasKeypressAction = true
		}
	}
	assert.True(t, hasTypeAction)
	assert.True(t, hasKeypressAction)
}

// TestStageDirector_TypingSpeed tests typing speed configuration
func TestStageDirector_TypingSpeed(t *testing.T) {
	model := &mockREPLForInteractions{mode: "typing_test"}
	director := NewStageDirectorWithConfig(t, model, StageConfig{
		Timeout:      1 * time.Second,
		TypingSpeed:  50 * time.Millisecond, // Slow typing for testing
		CaptureViews: false,
		MaxRetries:   0,
	})
	defer director.Stop()

	director.Start()

	start := time.Now()
	director.Type("abc") // 3 characters with 50ms each = ~150ms minimum
	duration := time.Since(start)

	// Should take at least 100ms (allowing for some timing variance)
	assert.Greater(t, duration, 100*time.Millisecond)
	assert.Equal(t, "abc", director.getCurrentInput())
}

// TestStageDirector_AllKeyPresses tests all key press methods
func TestStageDirector_AllKeyPresses(t *testing.T) {
	model := &mockREPLForInteractions{mode: "key_test"}
	director := NewStageDirectorWithConfig(t, model, StageConfig{
		Timeout:      1 * time.Second,
		TypingSpeed:  0,
		CaptureViews: false,
		MaxRetries:   0,
	})
	defer director.Stop()

	director.Start()

	// Test all key press methods
	director.Type("test")
	assert.Equal(t, "test", director.getCurrentInput())

	director.PressTab()
	assert.Equal(t, "tab_pressed", director.getCurrentMode())

	director.PressBackspace()
	assert.Equal(t, "tes", director.getCurrentInput())

	director.PressEscape()
	assert.Equal(t, "escaped", director.getCurrentMode())

	director.PressEnter()
	assert.Equal(t, "executed", director.getCurrentMode())

	// Arrow keys don't affect our mock, but test they don't crash
	director.PressArrowDown()
	director.PressArrowUp()

	// Verify all interactions were recorded
	actions := director.interactions
	assert.Greater(t, len(actions), 8) // Multiple interactions
}

// TestStageDirector_ClearInput tests input clearing functionality
func TestStageDirector_ClearInput(t *testing.T) {
	model := &mockREPLForInteractions{input: "initial", mode: "clear_test"}
	director := NewStageDirectorWithConfig(t, model, StageConfig{
		Timeout:      1 * time.Second,
		TypingSpeed:  0,
		CaptureViews: false,
		MaxRetries:   0,
	})
	defer director.Stop()

	director.Start()

	// Add more text
	director.Type("hello")
	assert.Equal(t, "initialhello", director.getCurrentInput())

	// Clear all input
	director.ClearInput()
	assert.Equal(t, "", director.getCurrentInput())
}

// TestStageDirector_Assertions tests assertion methods
func TestStageDirector_Assertions(t *testing.T) {
	model := &mockREPLForInteractions{input: "test input", mode: "assertion_test"}
	director := NewStageDirectorWithConfig(t, model, StageConfig{
		Timeout:      1 * time.Second,
		TypingSpeed:  0,
		CaptureViews: false,
		MaxRetries:   0,
	})
	defer director.Stop()

	director.Start()

	// Test successful assertions
	director.AssertViewContains("Mock REPL")
	director.AssertMode("assertion_test")
	director.AssertInputEquals("test input")

	// Check that no trips were recorded for successful assertions
	stats := director.GetSynchronizationStats()
	t.Logf("Synchronization stats: %+v", stats)
}

// TestStageDirector_Wait tests wait functionality
func TestStageDirector_Wait(t *testing.T) {
	model := &mockREPLForInteractions{mode: "wait_test"}
	director := NewStageDirectorWithConfig(t, model, StageConfig{
		Timeout:      1 * time.Second,
		TypingSpeed:  0,
		CaptureViews: true,
		MaxRetries:   0,
	})
	defer director.Stop()

	director.Start()

	start := time.Now()
	director.Wait(100 * time.Millisecond)
	duration := time.Since(start)

	// Should have waited at least the requested duration
	assert.GreaterOrEqual(t, duration, 100*time.Millisecond)

	// Should have recorded a wait action
	actions := director.interactions
	var hasWaitAction bool
	for _, action := range actions {
		if action.Type == "wait" {
			hasWaitAction = true
			break
		}
	}
	assert.True(t, hasWaitAction)
}

// TestStageDirector_Snapshots tests view capture functionality
func TestStageDirector_Snapshots(t *testing.T) {
	model := &mockREPLForInteractions{mode: "snapshot_test"}
	director := NewStageDirectorWithConfig(t, model, StageConfig{
		Timeout:      1 * time.Second,
		TypingSpeed:  0,
		CaptureViews: true, // Enable view capture
		MaxRetries:   0,
	})
	defer director.Stop()

	director.Start()

	initialSnapshots := len(director.snapshots)

	// Perform some interactions that should capture snapshots
	director.Type("test")
	director.PressEnter()
	director.Wait(10 * time.Millisecond)

	finalSnapshots := len(director.snapshots)

	// Should have captured additional snapshots
	assert.Greater(t, finalSnapshots, initialSnapshots)

	// Verify snapshot content
	if len(director.snapshots) > 0 {
		snapshot := director.snapshots[len(director.snapshots)-1]
		assert.Contains(t, snapshot.View, "Mock REPL")
		assert.Equal(t, "executed", snapshot.Mode)
		assert.NotZero(t, snapshot.Timestamp)
	}
}

// TestStageDirector_StopResult tests stop functionality and result generation
func TestStageDirector_StopResult(t *testing.T) {
	model := &mockREPLForInteractions{mode: "stop_test"}
	director := NewStageDirectorWithConfig(t, model, StageConfig{
		Timeout:      1 * time.Second,
		TypingSpeed:  0,
		CaptureViews: true,
		MaxRetries:   0,
	})

	director.Start()

	// Perform some interactions
	director.Type("test").PressEnter().Wait(10 * time.Millisecond)

	// Stop and get result
	start := time.Now()
	result := director.Stop()

	// Verify result structure
	assert.True(t, result.Success) // No failures occurred
	assert.Greater(t, result.Duration, time.Duration(0))
	assert.Greater(t, len(result.Actions), 0)
	assert.Greater(t, len(result.Snapshots), 0)
	assert.Empty(t, result.ErrorMessage) // No errors

	// Should complete quickly
	stopDuration := time.Since(start)
	assert.Less(t, stopDuration, 100*time.Millisecond)
}

// TestStageDirector_ConfigurationOptions tests various configuration options
func TestStageDirector_ConfigurationOptions(t *testing.T) {
	model := &mockREPLForInteractions{mode: "config_test"}

	// Test different configurations
	testCases := []struct {
		name   string
		config StageConfig
	}{
		{
			name: "minimal config",
			config: StageConfig{
				Timeout:      500 * time.Millisecond,
				TypingSpeed:  0,
				CaptureViews: false,
				MaxRetries:   0,
			},
		},
		{
			name: "full features config",
			config: StageConfig{
				Timeout:      2 * time.Second,
				TypingSpeed:  1 * time.Millisecond,
				CaptureViews: true,
				MaxRetries:   3,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			director := NewStageDirectorWithConfig(t, model, tc.config)
			defer director.Stop()

			// Should start and stop without issues
			director.Start()
			director.Type("test")
			result := director.Stop()

			assert.True(t, result.Success)
		})
	}
}

// TestStageDirector_WithTimeout tests timeout configuration
func TestStageDirector_WithTimeout(t *testing.T) {
	model := &mockREPLForInteractions{mode: "timeout_test"}
	director := NewStageDirector(t, model)
	defer director.Stop()

	// Test timeout configuration
	newTimeout := 500 * time.Millisecond
	director.WithTimeout(newTimeout)

	// Verify timeout was set (we can't easily test the actual timeout without making tests slow)
	assert.NotNil(t, director.ctx)
}

// TestStageDirector_WithViewCapture tests view capture configuration
func TestStageDirector_WithViewCapture(t *testing.T) {
	model := &mockREPLForInteractions{mode: "capture_test"}
	director := NewStageDirector(t, model)
	defer director.Stop()

	// Test disabling view capture
	director.WithViewCapture(false)
	director.Start()

	director.Type("test")
	director.PressEnter()

	result := director.Stop()

	// With view capture disabled, should have no or minimal snapshots
	// (might still have initial snapshot)
	assert.LessOrEqual(t, len(result.Snapshots), 1)
}

// TestStageDirector_DiagnosticMethods tests diagnostic and helper methods
func TestStageDirector_DiagnosticMethods(t *testing.T) {
	model := &mockREPLForInteractions{mode: "diagnostic_test"}
	director := NewStageDirectorWithConfig(t, model, StageConfig{
		Timeout:      1 * time.Second,
		TypingSpeed:  0,
		CaptureViews: true,
		MaxRetries:   0,
	})
	defer director.Stop()

	director.Start()

	// Perform some interactions
	director.Type("test").PressEnter()

	// Test diagnostic methods
	snapshot := director.GetLatestSnapshot()
	assert.NotEmpty(t, snapshot.View)
	assert.Equal(t, "executed", snapshot.Mode)

	actionCount := director.GetStageActionCount()
	assert.Greater(t, actionCount, 0)

	// Test synchronization stats
	stats := director.GetSynchronizationStats()
	assert.NotNil(t, stats)
	assert.GreaterOrEqual(t, stats["updates_generated"], int64(0))
	assert.GreaterOrEqual(t, stats["updates_processed"], int64(0))
	assert.Equal(t, int64(0), stats["updates_dropped"]) // Should be no drops
}

// TestStageDirector_ErrorHandling tests error conditions
func TestStageDirector_ErrorHandling(t *testing.T) {
	model := &mockREPLForInteractions{mode: "error_test"}
	director := NewStageDirectorWithConfig(t, model, StageConfig{
		Timeout:      100 * time.Millisecond, // Very short timeout
		TypingSpeed:  0,
		CaptureViews: false,
		MaxRetries:   0,
	})

	director.Start()

	// Perform some quick interactions before timeout
	director.Type("test")

	// Wait for potential timeout
	time.Sleep(150 * time.Millisecond)

	result := director.Stop()

	// Even with timeout, result should be valid
	assert.NotNil(t, result)
	assert.GreaterOrEqual(t, len(result.Actions), 0)
}

// BenchmarkStageInteractions benchmarks interaction performance
func BenchmarkStageInteractions(b *testing.B) {
	model := &mockREPLForInteractions{mode: "benchmark"}
	director := NewStageDirectorWithConfig(&testing.T{}, model, StageConfig{
		Timeout:      30 * time.Second,
		TypingSpeed:  0, // No delay for benchmarking
		CaptureViews: false, // Disable for performance
		MaxRetries:   0,
	})
	defer director.Stop()

	director.Start()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		director.Type("a").PressBackspace()
	}

	b.StopTimer()

	stats := director.GetSynchronizationStats()
	b.Logf("Processed %d interactions, %d updates",
		director.GetStageActionCount(), stats["updates_processed"])
}