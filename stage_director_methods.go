package steadicam

import (
	"context"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// syncModelUpdates processes model updates in order with duplicate detection
// Kubrick's perfectionist approach - every frame must be in sequence
func (d *StageDirector) syncModelUpdates() {
	defer func() {
		if r := recover(); r != nil {
			d.t.Logf("🚨 Model sync goroutine panicked: %v", r)
		}
	}()

	for {
		select {
		case update := <-d.modelChan:
			currentSeq := atomic.LoadInt64(&d.lastProcessedSeq)

			// Detect sequence gaps and out-of-order updates
			if update.sequence <= currentSeq {
				atomic.AddInt64(&d.duplicateUpdates, 1)
				continue // Skip duplicate or out-of-order update
			}

			if update.sequence > currentSeq+1 {
				atomic.AddInt64(&d.sequenceGaps, 1)
			}

			// Update model state safely with read lock
			d.modelMu.Lock()
			d.latestModel = update.model
			atomic.StoreInt64(&d.lastProcessedSeq, update.sequence)
			atomic.AddInt64(&d.updatesProcessed, 1)
			d.modelMu.Unlock()

		case <-d.ctx.Done():
			return
		}
	}
}

// WithTimeout sets a custom timeout for operations.
// IMPORTANT: Must be called before Start() - timeout cannot be changed after startup
// to avoid context management issues with running goroutines.
func (d *StageDirector) WithTimeout(timeout time.Duration) *StageDirector {
	// Prevent timeout changes after startup to avoid context lifecycle bugs
	if d.started {
		d.t.Logf("⚠️ Cannot change timeout after director has started - ignoring WithTimeout(%v)", timeout)
		return d
	}

	// Cancel existing context and create new one with new timeout
	if d.cancel != nil {
		d.cancel()
	}
	d.ctx, d.cancel = context.WithTimeout(context.Background(), timeout)
	d.config.Timeout = timeout
	return d
}

// WithViewCapture enables or disables automatic view snapshots.
// IMPORTANT: Must be called before Start() for consistent behavior.
func (d *StageDirector) WithViewCapture(enabled bool) *StageDirector {
	if d.started {
		d.t.Logf("⚠️ Cannot change view capture after director has started - ignoring WithViewCapture(%v)", enabled)
		return d
	}
	d.config.CaptureViews = enabled
	return d
}

// Start initializes the stage director and begins interaction recording
func (d *StageDirector) Start() *StageDirector {
	if d.started {
		d.t.Logf("⚠️ StageDirector already started")
		return d
	}

	d.t.Logf("[TRACE] Start: Creating headless bubbletea program...")

	// Wrap the model to capture state changes
	wrappedModel := stageModelWrapper{
		REPLModel: d.model,
		director:  d,
	}

	// Create a headless program (no terminal output) with test-friendly options
	d.program = tea.NewProgram(wrappedModel,
		tea.WithoutRenderer(),    // No terminal rendering
		tea.WithInput(nil),       // No input reader (prevents TTY access)
		tea.WithOutput(nil),      // No output writer
	)

	d.t.Logf("[TRACE] Start: Starting program in background goroutine...")

	// Run the program in a separate goroutine
	go func() {
		defer func() {
			if r := recover(); r != nil {
				d.t.Logf("🚨 Program goroutine panicked: %v", r)
			}
		}()

		d.t.Logf("[TRACE] Start: About to call program.Run()...")
		_, err := d.program.Run()
		if err != nil {
			d.t.Logf("[TRACE] Start: program.Run() returned with error=%v", err)
		}
	}()

	d.t.Logf("[TRACE] Start: Waiting for program to be ready...")
	if err := d.waitForProgramReady(); err != nil {
		d.recordTrip(newStageTrip("STARTUP_FAILED", err.Error(), map[string]interface{}{
			"error": err.Error(),
		}))
		return d
	}

	d.t.Logf("[TRACE] Start: Program ready, capturing initial snapshot...")
	d.started = true

	// Capture initial state with panic protection
	if d.config.CaptureViews {
		// Safely capture initial view
		var initialView string
		func() {
			defer func() {
				if r := recover(); r != nil {
					initialView = fmt.Sprintf("ERROR: Could not get initial view due to panic: %v", r)
				}
			}()
			initialView = d.getCurrentView()
		}()

		// Safely capture initial mode
		var initialMode string
		func() {
			defer func() {
				if r := recover(); r != nil {
					initialMode = fmt.Sprintf("error_initial_capture")
				}
			}()
			initialMode = d.getCurrentMode()
		}()

		// Safely capture initial input
		var initialInput string
		func() {
			defer func() {
				if r := recover(); r != nil {
					initialInput = fmt.Sprintf("ERROR: Could not get initial input due to panic: %v", r)
				}
			}()
			initialInput = d.getCurrentInput()
		}()

		d.snapshots = append(d.snapshots, StageSnapshot{
			Timestamp: time.Now(),
			View:      initialView,
			Mode:      initialMode,
			Input:     initialInput,
		})
	}

	d.t.Logf("[TRACE] Start: Start completed successfully")
	return d
}

// Stop ends the interaction session and returns comprehensive results
func (d *StageDirector) Stop() *StageResult {
	startTime := time.Now()

	// Ensure we capture final state before stopping with panic protection
	if d.config.CaptureViews && d.started {
		// Safely capture final view
		var finalView string
		func() {
			defer func() {
				if r := recover(); r != nil {
					finalView = fmt.Sprintf("ERROR: Could not get final view due to panic: %v", r)
				}
			}()
			finalView = d.getCurrentView()
		}()

		// Safely capture final mode
		var finalMode string
		func() {
			defer func() {
				if r := recover(); r != nil {
					finalMode = fmt.Sprintf("error_final_capture")
				}
			}()
			finalMode = d.getCurrentMode()
		}()

		// Safely capture final input
		var finalInput string
		func() {
			defer func() {
				if r := recover(); r != nil {
					finalInput = fmt.Sprintf("ERROR: Could not get final input due to panic: %v", r)
				}
			}()
			finalInput = d.getCurrentInput()
		}()

		d.snapshots = append(d.snapshots, StageSnapshot{
			Timestamp: time.Now(),
			View:      finalView,
			Mode:      finalMode,
			Input:     finalInput,
		})
	}

	// Cancel context and stop program
	if d.cancel != nil {
		d.cancel()
	}

	if d.program != nil {
		d.program.Quit()
	}

	duration := time.Since(startTime)
	success := !d.failed && (d.lastTrip == nil)

	// Prepare error details for comprehensive reporting
	var errorDetails strings.Builder
	var tripReport string

	if d.lastTrip != nil {
		if report := d.tripHandler.DetailedReport(); report != "" {
			tripReport = report
		}

		errorDetails.WriteString(fmt.Sprintf("Trip Type: %s\n", d.lastTrip.Type))
		errorDetails.WriteString(fmt.Sprintf("Error: %s\n", d.lastTrip.Message))
		errorDetails.WriteString(fmt.Sprintf("Timestamp: %s\n", d.lastTrip.Timestamp.Format(time.RFC3339)))

		if len(d.lastTrip.Context) > 0 {
			errorDetails.WriteString("Context:\n")
			for key, value := range d.lastTrip.Context {
				errorDetails.WriteString(fmt.Sprintf("  %s: %v\n", key, value))
			}
		}

		// Add synchronization stats if there are issues
		if d.HasDroppedUpdates() {
			errorDetails.WriteString("\nSynchronization Issues:\n")
			stats := d.GetSynchronizationStats()
			for key, value := range stats {
				if strings.Contains(key, "dropped") || strings.Contains(key, "overflow") || strings.Contains(key, "gap") {
					if value > 0 {
						errorDetails.WriteString(fmt.Sprintf("  %s: %d\n", key, value))
					}
				}
			}
		}
	}

	return &StageResult{
		Actions:      d.interactions,
		Snapshots:    d.snapshots,
		Success:      success,
		Duration:     duration,
		ErrorMessage: d.getErrorMessage(),
		Error:        d.getError(),
		ErrorDetails: errorDetails.String(),
		TripReport:   tripReport,
	}
}

// WaitForMode waits for the application to enter a specific mode
func (d *StageDirector) WaitForMode(expectedMode string) *StageDirector {
	if d.failed {
		return d
	}

	timeout := time.NewTimer(d.config.Timeout)
	defer timeout.Stop()

	for {
		select {
		case <-timeout.C:
			trip := newStageTrip("WAIT_MODE_TIMEOUT", fmt.Sprintf("Timeout waiting for mode '%s'", expectedMode), map[string]interface{}{
				"expected_mode": expectedMode,
				"current_mode":  d.getCurrentMode(),
			})
			d.recordTrip(trip)
			return d
		case <-d.ctx.Done():
			return d
		default:
			if d.getCurrentMode() == expectedMode {
				return d
			}
			time.Sleep(10 * time.Millisecond)
		}
	}
}

// WaitForSearchResults waits for search results to appear and stabilize
func (d *StageDirector) WaitForSearchResults() *StageDirector {
	if d.failed {
		return d
	}

	timeout := time.NewTimer(d.config.Timeout)
	defer timeout.Stop()

	for {
		select {
		case <-timeout.C:
			trip := newStageTrip("WAIT_RESULTS_TIMEOUT", "Timeout waiting for search results", map[string]interface{}{
				"current_view": d.truncateString(d.getCurrentView(), 200),
			})
			d.recordTrip(trip)
			return d
		case <-d.ctx.Done():
			return d
		default:
			view := d.getCurrentView()
			if strings.Contains(view, "Live Results:") || strings.Contains(view, "Found") {
				time.Sleep(50 * time.Millisecond) // Brief stabilization
				return d
			}
			time.Sleep(10 * time.Millisecond)
		}
	}
}

// WaitForText waits for specific text to appear in the current view
func (d *StageDirector) WaitForText(text string) *StageDirector {
	if d.failed {
		return d
	}

	timeout := time.NewTimer(d.config.Timeout)
	defer timeout.Stop()

	for {
		select {
		case <-timeout.C:
			trip := newStageTrip("WAIT_TEXT_TIMEOUT", fmt.Sprintf("Timeout waiting for text '%s'", text), map[string]interface{}{
				"expected_text": text,
				"current_view":  d.getCurrentView(),
			})
			d.recordTrip(trip)
			return d
		case <-d.ctx.Done():
			return d
		default:
			if strings.Contains(d.getCurrentView(), text) {
				return d
			}
			time.Sleep(10 * time.Millisecond)
		}
	}
}

// waitForProgramReady waits for the BubbleTea program to be ready
func (d *StageDirector) waitForProgramReady() error {
	d.t.Logf("[TRACE] waitForProgramReady: Starting wait with timeout=%v", d.config.Timeout)

	timeout := time.NewTimer(d.config.Timeout)
	defer timeout.Stop()

	for i := 0; i < 50; i++ { // Try up to 50 times
		select {
		case <-timeout.C:
			return fmt.Errorf("timeout waiting for program to be ready")
		case <-d.ctx.Done():
			return fmt.Errorf("context cancelled while waiting for program")
		default:
			// Check if we have a non-empty view (indicates program is ready)
			if view := d.getCurrentView(); len(view) > 0 {
				d.t.Logf("[TRACE] waitForProgramReady: SUCCESS after %d checks - view length=%d", i+1, len(view))
				return nil
			}
			time.Sleep(20 * time.Millisecond)
		}
	}

	d.t.Logf("[TRACE] waitForProgramReady: TIMEOUT after 50 checks")
	return fmt.Errorf("program never became ready")
}

// waitForViewChange waits for the view to change from a previous state
func (d *StageDirector) waitForViewChange(previousView string) {
	timeout := d.config.Timeout
	if timeout > time.Second {
		timeout = time.Second // Cap at 1 second for view changes
	}

	d.t.Logf("[TRACE] waitForViewChange: Starting wait, timeout=%v, prev_view_len=%d", timeout, len(previousView))

	start := time.Now()
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	checkCount := 0
	for {
		select {
		case <-timer.C:
			d.t.Logf("[TRACE] waitForViewChange: TIMEOUT after %v (%d checks)", time.Since(start), checkCount)
			return
		case <-d.ctx.Done():
			d.t.Logf("[TRACE] waitForViewChange: Context cancelled")
			return
		default:
			checkCount++
			currentView := d.getCurrentView()
			if currentView != previousView {
				elapsed := time.Since(start)
				d.t.Logf("[TRACE] waitForViewChange: SUCCESS after %v (%d checks), new_view_len=%d", elapsed, checkCount, len(currentView))
				return
			}
			time.Sleep(2 * time.Millisecond) // Short sleep between checks
		}
	}
}

// getCurrentView safely retrieves the current view content
func (d *StageDirector) getCurrentView() string {
	d.modelMu.RLock()
	model := d.latestModel
	d.modelMu.RUnlock()

	if model != nil {
		return model.View()
	}
	return ""
}

// getCurrentState returns the current application state
func (d *StageDirector) getCurrentState() interface{} {
	d.modelMu.RLock()
	defer d.modelMu.RUnlock()
	return d.latestModel
}

// getCurrentInput safely retrieves the current input
func (d *StageDirector) getCurrentInput() string {
	d.modelMu.RLock()
	model := d.latestModel
	d.modelMu.RUnlock()

	if model != nil {
		return model.CurrentInput()
	}
	return ""
}

// getCurrentMode safely retrieves the current mode
func (d *StageDirector) getCurrentMode() string {
	d.modelMu.RLock()
	model := d.latestModel
	d.modelMu.RUnlock()

	if model != nil {
		return model.CurrentMode()
	}
	return ""
}

// GetLatestSnapshot returns the most recent view snapshot
func (d *StageDirector) GetLatestSnapshot() StageSnapshot {
	if len(d.snapshots) == 0 {
		return StageSnapshot{}
	}
	return d.snapshots[len(d.snapshots)-1]
}

// GetStageActionCount returns the current number of recorded interactions
func (d *StageDirector) GetStageActionCount() int {
	return len(d.interactions)
}

// notifyStateChange handles state transition notifications
func (d *StageDirector) notifyStateChange(prevModel, newModel REPLModel) {
	// State change notification - currently used for debugging
	// Currently disabled to prevent race conditions in tests
	// The polling approach in waitForViewChange handles state detection reliably
	_ = prevModel
	_ = newModel
}

// getErrorMessage returns a human-readable error message
func (d *StageDirector) getErrorMessage() string {
	if d.lastTrip != nil {
		return fmt.Sprintf("[%s] %s", strings.ToLower(d.lastTrip.Type), d.lastTrip.Message)
	}
	return ""
}

// getError returns the structured error
func (d *StageDirector) getError() error {
	if d.lastTrip != nil {
		return fmt.Errorf("[%s] %s", d.lastTrip.Type, d.lastTrip.Message)
	}
	return nil
}

