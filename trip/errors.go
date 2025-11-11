// Package trip provides error handling for steadicam testing operations.
//
// The trip package uses stumbling metaphors for test error handling - when tests
// encounter issues, they "trip up" or "stumble", then need to recover gracefully.
package trip

import (
	"fmt"
	"strings"
	"time"
)

// Trip represents an error during test execution with rich context.
//
// Trips categorize different types of failures that can occur during
// automated testing, providing structured context for debugging without
// immediately failing tests.
//
// Error types:
//   - "interaction": User input simulation, timing, or coordination issues
//   - "assertion": Test validation, expectation failures, or verification issues
//   - "visual": Screenshot capture, rendering, or display failures
//   - "system": Infrastructure, initialization, or framework-level issues
//
// Example usage:
//
//	err := NewTrip("assertion", "Expected text not found in view",
//	    Context{"expected": "hello", "actual_view": "goodbye world"})
//
//	if err.CanRecover() {
//	    // Continue testing despite this stumble
//	}
type Trip struct {
	Type      string    // Error category for systematic handling
	Message   string    // Human-readable description
	Context   Context   // Additional debugging information
	Timestamp time.Time // When the error occurred
	Attempt   int       // Which attempt/retry this was
	Severity  Severity  // How serious this error is
}

// Context provides structured debugging information for trips.
//
// Context captures the state of the testing session when an error occurs,
// including UI state, interaction history, and system information.
type Context map[string]interface{}

// Severity indicates how serious a trip is and how it should be handled.
type Severity int

const (
	// Stumble indicates a minor issue that doesn't affect test validity.
	// Examples: Screenshot capture failed, minor timing variations
	Stumble Severity = iota

	// Error indicates a significant issue that may affect test results.
	// Examples: Assertion failures, unexpected state transitions
	Error

	// Fall indicates a serious issue that invalidates test results.
	// Examples: Application crashes, initialization failures
	Fall
)

func (s Severity) String() string {
	switch s {
	case Stumble:
		return "stumble"
	case Error:
		return "error"
	case Fall:
		return "fall"
	default:
		return "unknown"
	}
}

// NewTrip creates a new trip with the current timestamp.
func NewTrip(errorType, message string, context Context) *Trip {
	return &Trip{
		Type:      errorType,
		Message:   message,
		Context:   context,
		Timestamp: time.Now(),
		Severity:  Error, // Default severity
	}
}

// NewStumble creates a new trip with Stumble severity.
func NewStumble(errorType, message string, context Context) *Trip {
	return &Trip{
		Type:      errorType,
		Message:   message,
		Context:   context,
		Timestamp: time.Now(),
		Severity:  Stumble,
	}
}

// NewFall creates a new trip with Fall severity.
func NewFall(errorType, message string, context Context) *Trip {
	return &Trip{
		Type:      errorType,
		Message:   message,
		Context:   context,
		Timestamp: time.Now(),
		Severity:  Fall,
	}
}

// WithAttempt sets the attempt number for this error.
func (t *Trip) WithAttempt(attemptNumber int) *Trip {
	t.Attempt = attemptNumber
	return t
}

// WithSeverity sets the severity level for this error.
func (t *Trip) WithSeverity(severity Severity) *Trip {
	t.Severity = severity
	return t
}

// Error implements the error interface.
func (t *Trip) Error() string {
	return fmt.Sprintf("[%s:%s] %s", t.Type, t.Severity, t.Message)
}

// CanRecover returns true if testing can continue despite this error.
func (t *Trip) CanRecover() bool {
	return t.Severity == Stumble
}

// IsFall returns true if this error should immediately stop testing.
func (t *Trip) IsFall() bool {
	return t.Severity == Fall
}

// GetContext returns a specific context value if it exists.
func (t *Trip) GetContext(key string) (interface{}, bool) {
	if t.Context == nil {
		return nil, false
	}
	val, exists := t.Context[key]
	return val, exists
}

// DetailedString returns a comprehensive error description with context.
func (t *Trip) DetailedString() string {
	var details strings.Builder

	details.WriteString(fmt.Sprintf("[%s:%s] %s", t.Type, t.Severity, t.Message))
	details.WriteString(fmt.Sprintf("\n  Time: %s", t.Timestamp.Format("15:04:05.000")))

	if t.Attempt > 0 {
		details.WriteString(fmt.Sprintf("\n  Attempt: %d", t.Attempt))
	}

	if len(t.Context) > 0 {
		details.WriteString("\n  Context:")
		for key, value := range t.Context {
			details.WriteString(fmt.Sprintf("\n    %s: %v", key, value))
		}
	}

	return details.String()
}

// Handler manages error collection and reporting during testing.
//
// The handler provides component-specific error management that allows
// different types of failures to be handled appropriately. Visual errors
// don't stop test execution, while critical system errors do.
type Handler struct {
	component string   // Component name (e.g., "interaction", "assertion", "visual")
	trips     []*Trip  // Collected errors in chronological order
	stumbles  []*Trip  // Collected minor issues in chronological order
	policy    *Policy  // How to handle different error types
}

// Policy defines how different types and severities of errors should be handled.
type Policy struct {
	// StopOnFall determines if testing should stop immediately on fall errors
	StopOnFall bool

	// MaxStumbles sets a limit on accumulated stumbles before treating as trip
	MaxStumbles int

	// RecoverableTypes lists error types that are considered recoverable
	RecoverableTypes []string

	// RetryPolicy defines retry behavior for different error types
	RetryPolicy map[string]RetryConfig
}

// RetryConfig defines retry behavior for specific error types.
type RetryConfig struct {
	MaxRetries  int           // Maximum retry attempts
	Backoff     time.Duration // Delay between retries
	Exponential bool          // Whether to use exponential backoff
}

// DefaultPolicy returns a sensible default error handling policy.
func DefaultPolicy() *Policy {
	return &Policy{
		StopOnFall:       true,
		MaxStumbles:      10,
		RecoverableTypes: []string{"visual", "interaction", "timing"},
		RetryPolicy: map[string]RetryConfig{
			"visual":      {MaxRetries: 3, Backoff: 100 * time.Millisecond, Exponential: false},
			"interaction": {MaxRetries: 2, Backoff: 50 * time.Millisecond, Exponential: true},
			"timing":      {MaxRetries: 1, Backoff: 25 * time.Millisecond, Exponential: false},
		},
	}
}

// NewHandler creates a new error handler for a specific component.
func NewHandler(component string, policy *Policy) *Handler {
	if policy == nil {
		policy = DefaultPolicy()
	}

	return &Handler{
		component: component,
		trips:     make([]*Trip, 0),
		stumbles:  make([]*Trip, 0),
		policy:    policy,
	}
}

// Record adds an error to the handler's collection.
func (h *Handler) Record(trip *Trip) {
	if trip.Severity == Stumble {
		h.stumbles = append(h.stumbles, trip)
	} else {
		h.trips = append(h.trips, trip)
	}
}

// ShouldContinue determines if testing should continue based on current errors.
func (h *Handler) ShouldContinue() bool {
	// Stop on fall errors if policy requires it
	if h.policy.StopOnFall {
		for _, trip := range h.trips {
			if trip.IsFall() {
				return false
			}
		}
	}

	// Stop if too many stumbles have accumulated
	if h.policy.MaxStumbles > 0 && len(h.stumbles) > h.policy.MaxStumbles {
		return false
	}

	return true
}

// HasTrips returns true if any errors (non-stumbles) have been recorded.
func (h *Handler) HasTrips() bool {
	return len(h.trips) > 0
}

// HasStumbles returns true if any stumbles have been recorded.
func (h *Handler) HasStumbles() bool {
	return len(h.stumbles) > 0
}

// GetTrips returns all recorded errors.
func (h *Handler) GetTrips() []*Trip {
	return h.trips
}

// GetStumbles returns all recorded stumbles.
func (h *Handler) GetStumbles() []*Trip {
	return h.stumbles
}

// GetRetryConfig returns the retry configuration for a specific error type.
func (h *Handler) GetRetryConfig(errorType string) (RetryConfig, bool) {
	config, exists := h.policy.RetryPolicy[errorType]
	return config, exists
}

// CanRecover returns true if the given error type is considered recoverable.
func (h *Handler) CanRecover(errorType string) bool {
	for _, recoverableType := range h.policy.RecoverableTypes {
		if recoverableType == errorType {
			return true
		}
	}
	return false
}

// Summary provides a concise overview of all errors and stumbles.
func (h *Handler) Summary() string {
	if len(h.trips) == 0 && len(h.stumbles) == 0 {
		return fmt.Sprintf("[%s] No issues during testing", h.component)
	}

	return fmt.Sprintf("[%s] %d trips, %d stumbles",
		h.component, len(h.trips), len(h.stumbles))
}

// DetailedReport provides a comprehensive report of all issues.
func (h *Handler) DetailedReport() string {
	var report strings.Builder

	report.WriteString(fmt.Sprintf("=== %s Component Report ===\n", h.component))
	report.WriteString(h.Summary() + "\n")

	if len(h.trips) > 0 {
		report.WriteString("\nTrips:\n")
		for i, trip := range h.trips {
			report.WriteString(fmt.Sprintf("%d. %s\n", i+1, trip.DetailedString()))
		}
	}

	if len(h.stumbles) > 0 {
		report.WriteString("\nStumbles:\n")
		for i, stumble := range h.stumbles {
			report.WriteString(fmt.Sprintf("%d. %s\n", i+1, stumble.DetailedString()))
		}
	}

	return report.String()
}