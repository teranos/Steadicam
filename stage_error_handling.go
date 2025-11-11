package steadicam

import (
	"fmt"
	"sync/atomic"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// Update intercepts model updates to keep stage director in sync
// Stanley's steadicam operator - smooth, continuous state capture
func (w stageModelWrapper) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Panic recovery for fail-fast error handling
	// TODO: Add test with model that panics during Update() - see issue #50
	// https://github.com/sbvh-nl/qntx/issues/50
	defer func() {
		if r := recover(); r != nil {
			if w.director != nil {
				w.director.handleModelPanic(r, msg)
			}
		}
	}()

	newModel, cmd := w.REPLModel.Update(msg)

	// Validate model state for fail-fast detection
	if newModel == nil {
		if w.director != nil {
			w.director.handleInvalidModelState("Update returned nil model", msg)
		}
		return w, cmd
	}

	// Send the updated model with non-blocking delivery and overflow protection
	if w.director != nil && w.director.modelChan != nil {
		if replModel, ok := newModel.(REPLModel); ok {
			// Generate sequence number atomically
			seq := atomic.AddInt64(&w.director.updateSeq, 1)

			update := modelUpdate{
				model:     replModel,
				sequence:  seq,
				timestamp: time.Now(),
			}

			// Non-blocking send with buffer overflow protection
			select {
			case w.director.modelChan <- update:
				atomic.AddInt64(&w.director.updatesSent, 1)
			default:
				// Buffer full - handle overflow gracefully
				atomic.AddInt64(&w.director.bufferOverflows, 1)
				atomic.AddInt64(&w.director.droppedUpdates, 1)
			}
		}
	}

	// Return new wrapper with updated model instead of stale wrapper
	// This ensures BubbleTea program uses the latest model state
	if replModel, ok := newModel.(REPLModel); ok {
		return stageModelWrapper{
			REPLModel: replModel,
			director:  w.director,
		}, cmd
	}

	// If type assertion fails, handle gracefully by returning old wrapper
	// This should not happen in normal operation but provides fail-safe behavior
	if w.director != nil {
		w.director.handleInvalidModelState("Update returned non-REPLModel", msg)
	}
	return w, cmd
}

// GetSynchronizationStats returns detailed synchronization metrics
func (d *StageDirector) GetSynchronizationStats() map[string]int64 {
	return map[string]int64{
		"updates_generated":   atomic.LoadInt64(&d.updateSeq),
		"updates_sent":        atomic.LoadInt64(&d.updatesSent),
		"updates_processed":   atomic.LoadInt64(&d.updatesProcessed),
		"buffer_overflows":    atomic.LoadInt64(&d.bufferOverflows),
		"sequence_gaps":       atomic.LoadInt64(&d.sequenceGaps),
		"duplicate_updates":   atomic.LoadInt64(&d.duplicateUpdates),
		"updates_dropped":     atomic.LoadInt64(&d.droppedUpdates),
		"buffer_length":       int64(len(d.modelChan)),
		"buffer_capacity":     int64(cap(d.modelChan)),
	}
}

// HasDroppedUpdates returns true if any updates have been dropped
func (d *StageDirector) HasDroppedUpdates() bool {
	return atomic.LoadInt64(&d.droppedUpdates) > 0 ||
		atomic.LoadInt64(&d.bufferOverflows) > 0 ||
		atomic.LoadInt64(&d.sequenceGaps) > 0
}

// GetBufferUtilization returns current buffer usage as a percentage
func (d *StageDirector) GetBufferUtilization() float64 {
	if cap(d.modelChan) == 0 {
		return 0.0
	}
	return float64(len(d.modelChan)) / float64(cap(d.modelChan)) * 100.0
}

// ResetMetrics resets all performance counters (useful for testing)
func (d *StageDirector) ResetMetrics() {
	atomic.StoreInt64(&d.updateSeq, 0)
	atomic.StoreInt64(&d.lastProcessedSeq, 0)
	atomic.StoreInt64(&d.droppedUpdates, 0)
	atomic.StoreInt64(&d.updatesSent, 0)
	atomic.StoreInt64(&d.updatesProcessed, 0)
	atomic.StoreInt64(&d.bufferOverflows, 0)
	atomic.StoreInt64(&d.sequenceGaps, 0)
	atomic.StoreInt64(&d.duplicateUpdates, 0)
}

// truncateString helper for logging long strings
func (d *StageDirector) truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// handleModelPanic implements fail-fast error handling for model panics
// TODO: Add comprehensive test coverage for this path - see issue #50
// https://github.com/sbvh-nl/qntx/issues/50
func (d *StageDirector) handleModelPanic(panicValue interface{}, msg tea.Msg) {
	d.t.Logf("🚨 FAIL-FAST: Model panic detected: %v", panicValue)

	// Capture visual error state before failing
	d.captureErrorSnapshot("model_panic", fmt.Sprintf("Panic: %v", panicValue))

	// Create and record the trip to ensure proper test failure and reporting
	panicTrip := newStageTrip("MODEL_PANIC", fmt.Sprintf("Model panic during Update: %v", panicValue), map[string]interface{}{
		"panic_value": panicValue,
		"tea_msg":     fmt.Sprintf("%T: %+v", msg, msg),
		"model_type":  fmt.Sprintf("%T", d.model),
		"timestamp":   time.Now(),
	})
	d.recordTrip(panicTrip)

	// Cancel context to stop all operations immediately
	if d.cancel != nil {
		d.cancel()
	}

	// Log fail-fast behavior
	d.t.Logf("🛑 FAIL-FAST: Stage director stopped due to model panic")
}

// handleInvalidModelState implements fail-fast error handling for invalid model states
func (d *StageDirector) handleInvalidModelState(reason string, msg tea.Msg) {
	d.t.Logf("🚨 FAIL-FAST: Invalid model state detected: %s", reason)

	// Capture visual error state
	d.captureErrorSnapshot("invalid_model_state", reason)

	// Create and record the trip to ensure proper test failure and reporting
	invalidStateTrip := newStageTrip("INVALID_MODEL_STATE", reason, map[string]interface{}{
		"tea_msg":    fmt.Sprintf("%T: %+v", msg, msg),
		"model_type": fmt.Sprintf("%T", d.model),
		"timestamp":  time.Now(),
	})
	d.recordTrip(invalidStateTrip)

	// Cancel context for fail-fast behavior
	if d.cancel != nil {
		d.cancel()
	}

	d.t.Logf("🛑 FAIL-FAST: Stage director stopped due to invalid model state")
}

// captureErrorSnapshot captures a visual snapshot of error states for debugging
func (d *StageDirector) captureErrorSnapshot(errorType, errorMessage string) {
	// Safely get current view even if model is in error state
	var currentView string
	func() {
		defer func() {
			if r := recover(); r != nil {
				currentView = fmt.Sprintf("ERROR: Could not get view due to panic: %v", r)
			}
		}()
		currentView = d.getCurrentView()
	}()

	// Safely get current input even if model is in error state
	var currentInput string
	func() {
		defer func() {
			if r := recover(); r != nil {
				currentInput = fmt.Sprintf("ERROR: Could not get input due to panic: %v", r)
			}
		}()
		currentInput = d.getCurrentInput()
	}()

	// Create error snapshot with error details embedded in View
	errorSnapshot := StageSnapshot{
		Timestamp: time.Now(),
		View:      fmt.Sprintf("ERROR STATE (%s)\n%s\n\nLast View:\n%s", errorType, errorMessage, currentView),
		Mode:      fmt.Sprintf("error_%s", errorType),
		Input:     currentInput,
	}

	d.snapshots = append(d.snapshots, errorSnapshot)
	d.t.Logf("📸 ERROR SNAPSHOT: Captured visual state for %s", errorType)
}