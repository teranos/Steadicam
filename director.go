// Package steadicam provides automated testing for BubbleTea REPL and CLI applications.
//
// Steadicam enables smooth, cinematic testing of terminal user interfaces with
// precise timing and visual capture capabilities. Inspired by Stanley Kubrick's
// revolutionary steadicam cinematography, it provides fluid UI interaction testing.
//
// Basic usage:
//
//	model := NewYourREPLModel()
//	adapter := NewREPLAdapter(model)
//
//	result := steadicam.NewInteractiveTestDirector(t, adapter).
//		WithTimeout(5 * time.Second).
//		Start().
//		Type("hello world").
//		PressEnter().
//		WaitForMode("results").
//		AssertViewContains("success").
//		Stop()
//
//	assert.True(t, result.Success)
//
// For visual testing with screenshots:
//
//	steadicam.NewOperator(t, adapter, "screenshots/").
//		Start().
//		CaptureTrackingShot("initial").
//		TypeWithTrackingShot("query", "typed").
//		Stop()
package steadicam

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// REPLModel defines the interface for testable REPL/CLI applications.
//
// Any BubbleTea REPL or CLI application can implement this interface to enable
// rich automated testing with steadicam. The interface extends tea.Model with
// testing-specific methods that allow the test framework to inspect and validate
// the application's state.
//
// Required methods:
//   - CurrentInput() returns the current user input text
//   - CurrentMode() returns the current application mode as a string
//   - CheckCondition() enables custom wait conditions and assertions
//
// Example implementation:
//
//	func (m MyREPL) CurrentInput() string { return m.input }
//	func (m MyREPL) CurrentMode() string { return m.mode.String() }
//	func (m MyREPL) CheckCondition(condition string) bool {
//		switch condition {
//		case "has_results": return len(m.results) > 0
//		default: return false
//		}
//	}
type REPLModel interface {
	tea.Model
	// CurrentInput returns the current user input text
	CurrentInput() string
	// CurrentMode returns the current application mode as a string
	CurrentMode() string
	// CheckCondition allows custom wait conditions and assertions
	CheckCondition(condition string) bool
}

// InteractiveTestDirector orchestrates automated testing of BubbleTea applications.
//
// The director provides a fluent API for simulating user interactions, waiting for
// conditions, and asserting application state. It runs applications headlessly
// without terminal output, making it suitable for CI/CD environments.
//
// The director captures detailed interaction logs and view snapshots for debugging
// failed tests. Errors are collected and returned in the final TestResult rather
// than immediately failing the test.
//
// Example usage:
//
//	director := NewInteractiveTestDirector(t, model).
//		WithTimeout(10 * time.Second).
//		Start()
//
//	result := director.
//		Type("search query").
//		PressEnter().
//		WaitForMode("results").
//		AssertViewContains("Found 5 items").
//		Stop()
//
//	if !result.Success {
//		t.Fatalf("Test failed: %s", result.ErrorMessage)
//	}
type InteractiveTestDirector struct {
	t           *testing.T // Testing context for proper error handling
	model       REPLModel
	program     *tea.Program
	ctx         context.Context
	cancel      context.CancelFunc

	// Interaction tracking
	interactions []InteractionStep
	snapshots    []ViewSnapshot

	// Error tracking
	lastError    error
	failed       bool

	// Synchronization
	updateMu     sync.RWMutex
	waiting      map[string]chan struct{}

	// Model synchronization - non-blocking channel for model updates
	modelChan    chan REPLModel
	latestModel  REPLModel
	modelMu      sync.RWMutex

	// Configuration
	config  DirectorConfig
	started bool
}

// testModelWrapper wraps the original model to sync updates with the test director
// Stanley's camera crew - captures every scene transition with precision
type testModelWrapper struct {
	REPLModel
	director *InteractiveTestDirector
}

// Update intercepts model updates to keep test director in sync
// Stanley's steadicam operator - smooth, continuous state capture
func (w testModelWrapper) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	newModel, cmd := w.REPLModel.Update(msg)

	// Send the updated model to the test director via non-blocking channel
	if w.director != nil && w.director.modelChan != nil {
		if replModel, ok := newModel.(REPLModel); ok {
			select {
			case w.director.modelChan <- replModel:
				// Successfully sent update
			default:
				// Channel full, drop update (latest will be available)
			}
		}
	}

	// Return a new wrapped model to maintain the chain
	return testModelWrapper{
		REPLModel: newModel.(REPLModel),
		director: w.director,
	}, cmd
}

// InteractionStep records a single interaction with the REPL
type InteractionStep struct {
	Timestamp time.Time
	Type      string      // "keypress", "wait", "assertion", "screenshot"
	Details   interface{} // Specific interaction details
	Result    interface{} // Result of the interaction
}

// ViewSnapshot captures the complete state of the application at a specific moment.
//
// Snapshots are automatically captured during test execution and can be used
// for debugging failed tests or understanding application state transitions.
type ViewSnapshot struct {
	Timestamp time.Time // When the snapshot was captured
	View      string    // The rendered view content
	Mode      string    // Application mode at capture time
	Input     string    // User input at capture time
}

// TestResult contains the complete results of an interactive test session.
//
// The result includes all interactions performed, snapshots captured, timing
// information, and any errors encountered. Success indicates whether all
// operations completed without errors.
//
// Example usage:
//
//	result := director.Stop()
//	if !result.Success {
//		t.Logf("Test failed after %v", result.Duration)
//		t.Logf("Error: %s", result.ErrorMessage)
//		for _, snapshot := range result.Snapshots {
//			t.Logf("View at %v: %s", snapshot.Timestamp, snapshot.View)
//		}
//	}
type TestResult struct {
	Interactions []InteractionStep // All interactions performed
	Snapshots    []ViewSnapshot    // View snapshots captured
	Success      bool              // Whether test completed without errors
	Duration     time.Duration     // Total test execution time
	ErrorMessage string            // Human-readable error description
	Error        error             // Structured error for programmatic handling
}

// TestError represents a structured test failure with context.
//
// TestError provides detailed information about what went wrong, including
// the error type, descriptive message, and contextual information that
// can help with debugging.
//
// Error types:
//   - "timeout": Operations that exceeded configured timeouts
//   - "assertion": Failed assertions about application state
//   - "initialization": Problems starting or setting up the test
type TestError struct {
	Type    string                 // Error category
	Message string                 // Human-readable description
	Context map[string]interface{} // Additional debugging context
}

func (e *TestError) Error() string {
	return fmt.Sprintf("%s: %s", e.Type, e.Message)
}

// newTestError creates a new structured test error
func newTestError(errorType, message string, context map[string]interface{}) *TestError {
	return &TestError{
		Type:    errorType,
		Message: message,
		Context: context,
	}
}

// DirectorConfig configures the behavior of the InteractiveTestDirector.
//
// The configuration allows customization of timeouts, interaction timing,
// and debugging features to suit different testing scenarios.
//
// Example usage:
//
//	config := steadicam.DirectorConfig{
//		Timeout:      5 * time.Second,  // Shorter timeout for fast tests
//		TypingSpeed:  0,                // No delay for speed
//		CaptureViews: false,            // Disable snapshots for performance
//		MaxRetries:   1,                // No retries
//	}
//
//	director := NewInteractiveTestDirectorWithConfig(t, model, config)
type DirectorConfig struct {
	// Timeout for operations like waiting for conditions
	Timeout time.Duration
	// TypingSpeed controls delay between keystrokes (0 = no delay)
	TypingSpeed time.Duration
	// CaptureViews enables/disables automatic view snapshots
	CaptureViews bool
	// MaxRetries for transient operations (future use)
	MaxRetries int
	// AutoReportErrors controls whether errors are automatically reported to t.Error()
	// Set to false when testing error conditions to avoid test framework failures
	AutoReportErrors bool
}

// DefaultDirectorConfig returns a DirectorConfig with sensible defaults.
//
// The default configuration provides:
//   - 30 second timeout for operations
//   - 10ms typing delay for realistic input simulation
//   - View snapshot capture enabled
//   - 3 retries for transient failures
func DefaultDirectorConfig() DirectorConfig {
	return DirectorConfig{
		Timeout:          30 * time.Second,
		TypingSpeed:      10 * time.Millisecond,
		CaptureViews:     true,
		MaxRetries:       3,
		AutoReportErrors: true, // Default to existing behavior
	}
}

// recordError records an error and marks the test as failed
func (d *InteractiveTestDirector) recordError(err error) {
	d.lastError = err
	d.failed = true
	if d.t != nil && d.config.AutoReportErrors {
		d.t.Helper()
		d.t.Error(err) // Report to testing framework but don't stop execution
	}
}

// HasFailed returns true if the test has encountered any errors
func (d *InteractiveTestDirector) HasFailed() bool {
	return d.failed
}

// GetError returns the last error encountered
func (d *InteractiveTestDirector) GetError() error {
	return d.lastError
}

// NewInteractiveTestDirector creates a new InteractiveTestDirector with default configuration.
//
// This is the main entry point for automated testing of BubbleTea applications.
// The director runs the application headlessly and provides a fluent API for
// simulating user interactions and asserting application state.
//
// Parameters:
//   - t: The testing.T instance for error reporting
//   - model: A REPLModel implementation of your BubbleTea application
//
// Returns a director ready to be configured and started. You must call Start()
// before performing interactions and Stop() to get results.
//
// Example:
//
//	director := NewInteractiveTestDirector(t, myModel)
//	result := director.Start().Type("hello").Stop()
func NewInteractiveTestDirector(t *testing.T, model REPLModel) *InteractiveTestDirector {
	return NewInteractiveTestDirectorWithConfig(t, model, DefaultDirectorConfig())
}

// NewInteractiveTestDirectorWithConfig creates a new InteractiveTestDirector with custom configuration.
//
// Use this constructor when you need to customize timeouts, typing speed, or other
// behavior. For most cases, NewInteractiveTestDirector with defaults is sufficient.
//
// Parameters:
//   - t: The testing.T instance for error reporting
//   - model: A REPLModel implementation of your BubbleTea application
//   - config: Custom configuration for director behavior
//
// Example:
//
//	config := DirectorConfig{
//		Timeout: 5 * time.Second,
//		TypingSpeed: 0, // No typing delay
//	}
//	director := NewInteractiveTestDirectorWithConfig(t, model, config)
func NewInteractiveTestDirectorWithConfig(t *testing.T, model REPLModel, config DirectorConfig) *InteractiveTestDirector {
	ctx, cancel := context.WithTimeout(context.Background(), config.Timeout)

	director := &InteractiveTestDirector{
		t:            t,
		model:        model,
		ctx:          ctx,
		cancel:       cancel,
		interactions: make([]InteractionStep, 0),
		snapshots:    make([]ViewSnapshot, 0),
		waiting:      make(map[string]chan struct{}),
		config:       config,
		started:      false,
		modelChan:    make(chan REPLModel, 10), // Buffered channel for model updates
		latestModel:  model,                    // Initialize with starting model
	}

	// Start model synchronization goroutine
	go director.syncModelUpdates()

	return director
}

// syncModelUpdates handles model updates from the bubbletea program
func (d *InteractiveTestDirector) syncModelUpdates() {
	for {
		select {
		case <-d.ctx.Done():
			return
		case newModel := <-d.modelChan:
			d.modelMu.Lock()
			d.latestModel = newModel
			d.modelMu.Unlock()
		}
	}
}

// WithTimeout configures the default timeout for wait operations.
//
// This timeout applies to WaitForMode, WaitForText, WaitForSearchResults,
// and other wait operations. The timeout can be changed at any time before
// or after starting the test session.
//
// Example:
//
//	director.WithTimeout(5 * time.Second).Start()
func (d *InteractiveTestDirector) WithTimeout(timeout time.Duration) *InteractiveTestDirector {
	d.config.Timeout = timeout
	d.ctx, d.cancel = context.WithTimeout(context.Background(), timeout)
	return d
}

// WithViewCapture enables or disables automatic view snapshot capture.
//
// When enabled (default), the director captures view snapshots after each
// interaction for debugging purposes. Disable to improve performance when
// snapshots are not needed.
//
// Example:
//
//	director.WithViewCapture(false).Start()  // Disable for performance
func (d *InteractiveTestDirector) WithViewCapture(enabled bool) *InteractiveTestDirector {
	d.config.CaptureViews = enabled
	return d
}

// Start begins the interactive test session.
//
// This method initializes the BubbleTea program in headless mode and waits
// for it to be ready for interaction. You must call Start() before performing
// any interactions with the application.
//
// Returns the director for method chaining. If initialization fails, the
// error will be available in the final TestResult.
//
// Example:
//
//	result := director.Start().Type("hello").Stop()
func (d *InteractiveTestDirector) Start() *InteractiveTestDirector {
	if d.started {
		d.t.Logf("[TRACE] Start: Already started, returning existing instance")
		return d
	}

	d.t.Logf("[TRACE] Start: Creating headless bubbletea program...")

	// Create headless program (no terminal output) for testing
	// Wrap the model to keep test director in sync with updates
	wrappedModel := testModelWrapper{
		REPLModel: d.model,
		director: d,
	}

	d.program = tea.NewProgram(
		wrappedModel,
		tea.WithContext(d.ctx),
		tea.WithoutRenderer(),    // Key: no terminal output for automated testing
		tea.WithInput(nil),      // No input needed for testing
		tea.WithOutput(os.Stderr), // Send any output to stderr for debugging
	)

	d.t.Logf("[TRACE] Start: Starting program in background goroutine...")

	// Start program in background
	go func() {
		d.t.Logf("[TRACE] Start: About to call program.Run()...")
		_, err := d.program.Run()
		d.t.Logf("[TRACE] Start: program.Run() returned with error=%v", err)
	}()

	d.t.Logf("[TRACE] Start: Waiting for program to be ready...")

	// Wait for program to be ready by checking for non-empty view
	if err := d.waitForProgramReady(); err != nil {
		d.recordError(newTestError("initialization", fmt.Sprintf("Program failed to initialize: %v", err), map[string]interface{}{"original_error": err}))
		return d
	}

	d.t.Logf("[TRACE] Start: Program ready, capturing initial snapshot...")
	d.captureSnapshot("initial")
	d.started = true
	d.t.Logf("[TRACE] Start: Start completed successfully")

	return d
}

// Stop ends the test session and returns complete results.
//
// This method cleans up the BubbleTea program and returns a TestResult
// containing all interactions, snapshots, timing information, and any
// errors that occurred during the test.
//
// You must call Stop() to get test results. The TestResult.Success field
// indicates whether the test completed without errors.
//
// Example:
//
//	result := director.Start().Type("hello").Stop()
//	if !result.Success {
//		t.Fatalf("Test failed: %s", result.ErrorMessage)
//	}
func (d *InteractiveTestDirector) Stop() *TestResult {
	if !d.started {
		return &TestResult{
			Success:      false,
			ErrorMessage: "Test director was never started",
		}
	}

	startTime := time.Now()
	if len(d.interactions) > 0 {
		startTime = d.interactions[0].Timestamp
	}

	d.cancel()
	if d.program != nil {
		d.program.Quit()
	}

	success := !d.failed
	errorMessage := ""
	if d.lastError != nil {
		errorMessage = d.lastError.Error()
	}

	return &TestResult{
		Interactions: d.interactions,
		Snapshots:    d.snapshots,
		Success:      success,
		Duration:     time.Since(startTime),
		ErrorMessage: errorMessage,
		Error:        d.lastError,
	}
}


// WaitForMode waits until the REPL enters the specified mode
func (d *InteractiveTestDirector) WaitForMode(expectedMode string) *InteractiveTestDirector {
	timeout := time.After(d.config.Timeout)
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			d.recordError(newTestError("timeout", fmt.Sprintf("Timeout waiting for mode %v (current mode: %v)", expectedMode, d.getCurrentMode()), map[string]interface{}{"expected": expectedMode, "actual": d.getCurrentMode()}))
			return d
		case <-ticker.C:
			if d.getCurrentMode() == expectedMode {
				d.recordInteraction("wait_condition", fmt.Sprintf("mode=%v", expectedMode))
				return d
			}
		}
	}
}

// WaitForSearchResults waits until search results are available
func (d *InteractiveTestDirector) WaitForSearchResults() *InteractiveTestDirector {
	timeout := time.After(d.config.Timeout)
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			d.recordError(newTestError("timeout", "Timeout waiting for search results", nil))
			return d
		case <-ticker.C:
			if d.latestModel.CheckCondition("search_results") {
				d.recordInteraction("wait_condition", "search_results")
				return d
			}
		}
	}
}

// WaitForText waits until the specified text appears in the view
func (d *InteractiveTestDirector) WaitForText(text string) *InteractiveTestDirector {
	timeout := time.After(d.config.Timeout)
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			currentView := d.getCurrentView()
			d.recordError(newTestError("timeout", fmt.Sprintf("Timeout waiting for text '%s'", text), map[string]interface{}{"expected_text": text, "current_view": currentView}))
			return d
		case <-ticker.C:
			if strings.Contains(d.getCurrentView(), text) {
				d.recordInteraction("wait_condition", fmt.Sprintf("text=%s", text))
				return d
			}
		}
	}
}

// AssertViewContains verifies that the current view contains the specified text
func (d *InteractiveTestDirector) AssertViewContains(text string) *InteractiveTestDirector {
	view := d.getCurrentView()
	if !strings.Contains(view, text) {
		d.recordError(newTestError("assertion", fmt.Sprintf("View does not contain expected text: %s", text), map[string]interface{}{"expected": text, "actual_view": view}))
		return d
	}
	d.recordInteraction("assertion", fmt.Sprintf("contains=%s", text))
	return d
}

// AssertMode verifies that the REPL is in the expected mode
func (d *InteractiveTestDirector) AssertMode(expectedMode string) *InteractiveTestDirector {
	actualMode := d.getCurrentMode()
	if actualMode != expectedMode {
		d.recordError(newTestError("assertion", fmt.Sprintf("Expected mode %v, got %v", expectedMode, actualMode), map[string]interface{}{"expected": expectedMode, "actual": actualMode}))
		return d
	}
	d.recordInteraction("assertion", fmt.Sprintf("mode=%v", expectedMode))
	return d
}

// AssertInputEquals verifies that the current input matches the expected value
func (d *InteractiveTestDirector) AssertInputEquals(expected string) *InteractiveTestDirector {
	actual := d.getCurrentInput()
	if actual != expected {
		d.recordError(newTestError("assertion", fmt.Sprintf("Expected input '%s', got '%s'", expected, actual), map[string]interface{}{"expected": expected, "actual": actual}))
		return d
	}
	d.recordInteraction("assertion", fmt.Sprintf("input=%s", expected))
	return d
}

// AssertNoSearchResults verifies that no search results are currently displayed
func (d *InteractiveTestDirector) AssertNoSearchResults() *InteractiveTestDirector {
	if d.latestModel.CheckCondition("search_results") {
		d.recordError(newTestError("assertion", "Expected no search results, but found some", nil))
		return d
	}
	d.recordInteraction("assertion", "no_search_results")
	return d
}

// sendMessage sends a message to the bubbletea program
func (d *InteractiveTestDirector) sendMessage(msg tea.Msg) {
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

// recordInteraction logs an interaction step
func (d *InteractiveTestDirector) recordInteraction(interactionType string, details interface{}) {
	d.interactions = append(d.interactions, InteractionStep{
		Timestamp: time.Now(),
		Type:      interactionType,
		Details:   details,
	})
}

// captureSnapshot captures the current state of the REPL
func (d *InteractiveTestDirector) captureSnapshot(reason string) {
	if !d.config.CaptureViews {
		return
	}

	snapshot := ViewSnapshot{
		Timestamp: time.Now(),
		View:      d.getCurrentView(),
		Mode:      d.getCurrentMode(),
		Input:     d.getCurrentInput(),
	}

	d.snapshots = append(d.snapshots, snapshot)
}

// getCurrentView returns the current rendered view
func (d *InteractiveTestDirector) getCurrentView() string {
	// Thread-safe access to the synchronized model
	d.modelMu.RLock()
	view := d.latestModel.View()
	d.modelMu.RUnlock()
	return view
}

// getCurrentState returns the current model state if available
func (d *InteractiveTestDirector) getCurrentState() interface{} {
	d.modelMu.RLock()
	defer d.modelMu.RUnlock()

	// For now, return nil since we don't have a TestableModel interface
	// This could be enhanced later with reflection or specific interfaces
	return nil
}

// getCurrentInput returns the current input string
func (d *InteractiveTestDirector) getCurrentInput() string {
	d.modelMu.RLock()
	defer d.modelMu.RUnlock()
	return d.latestModel.CurrentInput()
}

// getCurrentMode returns the current REPL mode as a string
func (d *InteractiveTestDirector) getCurrentMode() string {
	d.modelMu.RLock()
	defer d.modelMu.RUnlock()
	return d.latestModel.CurrentMode()
}

// GetLatestSnapshot returns the most recent view snapshot
func (d *InteractiveTestDirector) GetLatestSnapshot() ViewSnapshot {
	if len(d.snapshots) == 0 {
		return ViewSnapshot{}
	}
	return d.snapshots[len(d.snapshots)-1]
}

// waitForProgramReady waits for the bubbletea program to initialize
func (d *InteractiveTestDirector) waitForProgramReady() error {
	timeout := time.After(d.config.Timeout)
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	d.t.Logf("[TRACE] waitForProgramReady: Starting wait with timeout=%v", d.config.Timeout)
	checkCount := 0

	for {
		select {
		case <-timeout:
			d.t.Logf("[TRACE] waitForProgramReady: TIMEOUT after %d checks", checkCount)
			return fmt.Errorf("timeout waiting for program to initialize")
		case <-ticker.C:
			checkCount++
			// Check if program is ready by verifying non-empty view
			view := d.getCurrentView()
			if checkCount%100 == 0 { // Log every second
				d.t.Logf("[TRACE] waitForProgramReady: Check %d - view length=%d", checkCount, len(view))
			}
			if view != "" && len(view) > 10 { // Ensure meaningful content
				d.t.Logf("[TRACE] waitForProgramReady: SUCCESS after %d checks - view length=%d", checkCount, len(view))
				return nil
			}
		}
	}
}

// waitForViewChange waits for the UI view to change after sending a message
func (d *InteractiveTestDirector) waitForViewChange(previousView string) {
	deadline := time.Now().Add(500 * time.Millisecond) // Shorter timeout for UI updates
	ticker := time.NewTicker(2 * time.Millisecond)     // More frequent checks
	defer ticker.Stop()

	waitStart := time.Now()
	checkCount := 0
	d.t.Logf("[TRACE] waitForViewChange: Starting wait, timeout=500ms, prev_view_len=%d", len(previousView))

	for time.Now().Before(deadline) {
		select {
		case <-ticker.C:
			checkCount++
			currentView := d.getCurrentView()
			if currentView != previousView {
				elapsed := time.Since(waitStart)
				d.t.Logf("[TRACE] waitForViewChange: SUCCESS after %v (%d checks), new_view_len=%d",
					elapsed, checkCount, len(currentView))
				return
			}

			// Log every 50ms to track progress without spam
			if checkCount%25 == 0 { // 25 * 2ms = 50ms
				elapsed := time.Since(waitStart)
				d.t.Logf("[TRACE] waitForViewChange: Still waiting... %v elapsed (%d checks)", elapsed, checkCount)
			}
		}
	}

	// If we reach here, view didn't change within timeout
	elapsed := time.Since(waitStart)
	d.t.Logf("[TRACE] waitForViewChange: TIMEOUT after %v (%d checks), view unchanged", elapsed, checkCount)
	// This might be okay for some interactions, so we don't error
}

// WaitForCondition waits for a custom condition to be true on the model
func (d *InteractiveTestDirector) WaitForCondition(condition string) *InteractiveTestDirector {
	timeout := time.After(d.config.Timeout)
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	d.t.Logf("[TRACE] WaitForCondition: Waiting for condition '%s' with timeout=%v", condition, d.config.Timeout)

	for {
		select {
		case <-timeout:
			d.recordError(newTestError("timeout", fmt.Sprintf("Timeout waiting for condition: %s", condition), map[string]interface{}{"condition": condition, "timeout": d.config.Timeout}))
			return d
		case <-ticker.C:
			if d.latestModel.CheckCondition(condition) {
				d.t.Logf("[TRACE] WaitForCondition: Condition '%s' satisfied", condition)
				d.recordInteraction("wait_condition", condition)
				d.captureSnapshot("condition_met")
				return d
			}
		}
	}
}

// CheckCondition verifies that a condition is currently true on the model
func (d *InteractiveTestDirector) CheckCondition(condition string) *InteractiveTestDirector {
	if !d.latestModel.CheckCondition(condition) {
		d.recordError(newTestError("assertion", fmt.Sprintf("Condition check failed: %s", condition), map[string]interface{}{"condition": condition}))
		return d
	}

	d.t.Logf("[TRACE] CheckCondition: Condition '%s' is true", condition)
	d.recordInteraction("check_condition", condition)
	return d
}

// GetInteractionCount returns the number of recorded interactions
func (d *InteractiveTestDirector) GetInteractionCount() int {
	return len(d.interactions)
}

// truncateString helper for logging long strings
func (d *InteractiveTestDirector) truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}