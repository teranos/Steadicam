// Package steadicam provides automated stage testing for BubbleTea REPL and CLI applications.
//
// Steadicam enables smooth, cinematic stage testing of terminal user interfaces with
// precise timing and visual capture capabilities. Inspired by Stanley Kubrick's
// revolutionary steadicam cinematography, it provides fluid UI interaction staging.
//
// Basic usage:
//
//	model := NewYourREPLModel()
//	adapter := NewREPLAdapter(model)
//
//	result := steadicam.NewStageDirector(t, adapter).
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
// For visual staging with screenshots:
//
//	steadicam.NewOperator(t, adapter, "screenshots/").
//		Start().
//		CaptureTrackingShot("initial").
//		TypeWithTrackingShot("query", "typed").
//		Stop()
package steadicam

import (
	"context"
	"sync"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/sbvh/qntx/cmd/repl/bubble/steadicam/trip"
)

// modelUpdate represents a timestamped model state change with sequence tracking
// Used by the syncModelUpdates() method for ordered model synchronization
type modelUpdate struct {
	model     REPLModel // The updated model state
	sequence  int64     // Unique sequence number for ordering
	timestamp time.Time // When the update was generated
}

// Closeable defines the interface for models that need resource cleanup
// Models implementing this interface will have their Close() method called
// automatically when the stage director stops
type Closeable interface {
	Close() error
}

// REPLModel defines the interface for stageable REPL/CLI applications.
//
// Any BubbleTea REPL or CLI application can implement this interface to enable
// rich automated staging with steadicam. The interface extends tea.Model with
// staging-specific methods that allow the stage framework to inspect and validate
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

// StageDirector orchestrates automated staging of BubbleTea applications.
//
// The stage director provides a fluent API for simulating user interactions, waiting for
// conditions, and asserting application state. It runs applications headlessly
// without terminal output, making it suitable for CI/CD environments.
//
// The director captures detailed interaction logs and view snapshots for debugging
// failed stages. Errors are collected and returned in the final StageResult rather
// than immediately failing the stage.
//
// Example usage:
//
//	director := NewStageDirector(t, model).
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
//		t.Fatalf("Stage failed: %s", result.ErrorMessage)
//	}
type StageDirector struct {
	t           *testing.T // Testing context for proper error handling
	model       REPLModel
	program     *tea.Program
	ctx         context.Context
	cancel      context.CancelFunc

	// Interaction tracking
	interactions []StageAction
	snapshots    []StageSnapshot

	// Error tracking with trip package
	tripHandler  *trip.Handler
	lastTrip     *trip.Trip
	failed       bool

	// Synchronization
	updateMu     sync.RWMutex
	waiting      map[string]chan struct{}

	// Model synchronization with atomic sequence tracking
	modelChan         chan modelUpdate
	latestModel       REPLModel
	modelMu           sync.RWMutex
	updateSeq         int64 // atomic counter for update ordering
	lastProcessedSeq  int64 // atomic counter for processed updates
	droppedUpdates    int64 // atomic counter for diagnostic purposes

	// Enhanced metrics for performance monitoring
	updatesSent       int64 // atomic counter for successfully sent updates
	updatesProcessed  int64 // atomic counter for processed updates
	bufferOverflows   int64 // atomic counter for buffer overflow events
	sequenceGaps      int64 // atomic counter for sequence gaps detected
	duplicateUpdates  int64 // atomic counter for duplicate/out-of-order updates

	// Configuration
	config  StageConfig
	started bool
}


// stageModelWrapper wraps the original model to sync updates with the stage director
// Stanley's camera crew - captures every scene transition with precision
type stageModelWrapper struct {
	REPLModel
	director *StageDirector
}

// StageAction records a single interaction with the REPL during staging
type StageAction struct {
	Timestamp time.Time
	Type      string      // "keypress", "wait", "assertion", "screenshot"
	Details   interface{} // Specific interaction details
	Result    interface{} // Result of the interaction
}

// StageSnapshot captures the complete state of the application at a specific moment.
//
// Snapshots are automatically captured during stage execution and can be used
// for debugging failed stages or understanding application state transitions.
type StageSnapshot struct {
	Timestamp time.Time // When the snapshot was captured
	View      string    // The rendered view content
	Mode      string    // Application mode at capture time
	Input     string    // User input at capture time
}

// StageResult contains the complete results of an interactive stage session.
//
// The result includes all interactions performed, snapshots captured, timing
// information, and any errors encountered. Success indicates whether all
// operations completed without errors.
//
// Example usage:
//
//	result := director.Stop()
//	if !result.Success {
//		t.Logf("Stage failed after %v", result.Duration)
//		t.Logf("Error: %s", result.ErrorMessage)
//		for _, snapshot := range result.Snapshots {
//			t.Logf("View at %v: %s", snapshot.Timestamp, snapshot.View)
//		}
//	}
type StageResult struct {
	Actions      []StageAction   // All interactions performed
	Snapshots    []StageSnapshot // View snapshots captured
	Success      bool            // Whether stage completed without errors
	Duration     time.Duration   // Total stage execution time
	ErrorMessage string          // Human-readable error description
	Error        error           // Structured error for programmatic handling
	ErrorDetails string          // Detailed technical error information for debugging
	TripReport   string          // Detailed trip handling report
}

// newStageTrip creates a new trip for stage errors
func newStageTrip(errorType, message string, context map[string]interface{}) *trip.Trip {
	tripContext := make(trip.Context)
	for k, v := range context {
		tripContext[k] = v
	}
	return trip.NewTrip(errorType, message, tripContext)
}

// StageConfig configures the behavior of the StageDirector.
//
// The configuration allows customization of timeouts, interaction timing,
// and debugging features to suit different staging scenarios.
//
// Example usage:
//
//	config := steadicam.StageConfig{
//		Timeout:      5 * time.Second,  // Shorter timeout for fast stages
//		TypingSpeed:  0,                // No delay for speed
//		CaptureViews: false,            // Disable snapshots for performance
//		MaxRetries:   1,                // No retries
//	}
//
//	director := NewStageDirectorWithConfig(t, model, config)
type StageConfig struct {
	// Timeout for operations like waiting for conditions
	Timeout time.Duration
	// TypingSpeed controls delay between keystrokes (0 = no delay)
	TypingSpeed time.Duration
	// CaptureViews enables/disables automatic view snapshots
	CaptureViews bool
	// MaxRetries for transient operations (future use)
	MaxRetries int
}

// DefaultStageConfig returns a StageConfig with sensible defaults.
//
// The default configuration provides:
//   - 30 second timeout for operations
//   - 10ms typing delay for realistic input simulation
//   - View snapshot capture enabled
//   - 3 retries for transient failures
func DefaultStageConfig() StageConfig {
	return StageConfig{
		Timeout:      30 * time.Second,
		TypingSpeed:  10 * time.Millisecond,
		CaptureViews: true,
		MaxRetries:   3,
	}
}

// NewStageDirector creates a new StageDirector with default configuration.
//
// This is the main entry point for automated staging of BubbleTea applications.
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
//	director := NewStageDirector(t, myModel)
//	result := director.Start().Type("hello").Stop()
func NewStageDirector(t *testing.T, model REPLModel) *StageDirector {
	return NewStageDirectorWithConfig(t, model, DefaultStageConfig())
}

// NewStageDirectorWithConfig creates a new StageDirector with custom configuration.
//
// Use this constructor when you need to customize timeouts, typing speed, or other
// behavior. For most cases, NewStageDirector with defaults is sufficient.
//
// Parameters:
//   - t: The testing.T instance for error reporting
//   - model: A REPLModel implementation of your BubbleTea application
//   - config: Custom configuration for director behavior
//
// Example:
//
//	config := StageConfig{
//		Timeout: 5 * time.Second,
//		TypingSpeed: 0, // No typing delay
//	}
//	director := NewStageDirectorWithConfig(t, model, config)
func NewStageDirectorWithConfig(t *testing.T, model REPLModel, config StageConfig) *StageDirector {
	ctx, cancel := context.WithTimeout(context.Background(), config.Timeout)

	// Create trip handler for error management
	tripHandler := trip.NewHandler("stage_director", trip.DefaultPolicy())

	director := &StageDirector{
		t:            t,
		model:        model,
		ctx:          ctx,
		cancel:       cancel,
		interactions: make([]StageAction, 0),
		snapshots:    make([]StageSnapshot, 0),
		waiting:      make(map[string]chan struct{}),
		config:       config,
		started:      false,
		modelChan:         make(chan modelUpdate, 50), // Larger buffer with update metadata
		latestModel:       model,                       // Initialize with starting model
		updateSeq:         0,
		lastProcessedSeq:  0,
		droppedUpdates:    0,
		updatesSent:       0,
		updatesProcessed:  0,
		bufferOverflows:   0,
		sequenceGaps:      0,
		duplicateUpdates:  0,
		tripHandler:  tripHandler,
	}

	// Start model synchronization goroutine
	go director.syncModelUpdates()

	return director
}