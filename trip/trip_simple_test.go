package trip

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestTrip_Core tests core Trip functionality
func TestTrip_Core(t *testing.T) {
	context := Context{
		"component": "steadicam",
		"operation": "screenshot",
	}

	trip := NewTrip("screenshot", "Failed to capture", context)

	// Basic properties
	assert.Equal(t, "screenshot", trip.Type)
	assert.Equal(t, "Failed to capture", trip.Message)
	assert.Equal(t, context, trip.Context)
	assert.Equal(t, Error, trip.Severity)
	assert.WithinDuration(t, time.Now(), trip.Timestamp, time.Second)

	// Error interface
	assert.Contains(t, trip.Error(), "Failed to capture")
	assert.Contains(t, trip.Error(), "screenshot")
	assert.Contains(t, trip.Error(), "error")
}

// TestTrip_Severities tests different severity levels
func TestTrip_Severities(t *testing.T) {
	stumble := NewStumble("timing", "Slight delay", nil)
	error_ := NewTrip("validation", "Invalid input", nil)
	fall := NewFall("system", "Critical failure", nil)

	// Severity values
	assert.Equal(t, Stumble, stumble.Severity)
	assert.Equal(t, Error, error_.Severity)
	assert.Equal(t, Fall, fall.Severity)

	// Recovery capabilities
	assert.True(t, stumble.CanRecover())
	assert.False(t, error_.CanRecover())
	assert.False(t, fall.CanRecover())

	// Fall detection
	assert.False(t, stumble.IsFall())
	assert.False(t, error_.IsFall())
	assert.True(t, fall.IsFall())
}

// TestTrip_Methods tests trip methods
func TestTrip_Methods(t *testing.T) {
	trip := NewTrip("test", "Test message", Context{"key": "value"})

	// WithAttempt
	trip.WithAttempt(3)
	assert.Equal(t, 3, trip.Attempt)

	// WithSeverity
	trip.WithSeverity(Fall)
	assert.Equal(t, Fall, trip.Severity)

	// GetContext
	val, exists := trip.GetContext("key")
	assert.True(t, exists)
	assert.Equal(t, "value", val)

	_, exists = trip.GetContext("missing")
	assert.False(t, exists)

	// DetailedString
	detailed := trip.DetailedString()
	assert.Contains(t, detailed, "Test message")
	assert.Contains(t, detailed, "key: value")
}

// TestHandler_Basic tests basic Handler functionality
func TestHandler_Basic(t *testing.T) {
	handler := NewHandler("test_component", DefaultPolicy())

	// Should continue initially
	assert.True(t, handler.ShouldContinue())

	// Record stumble - should still continue
	stumble := NewStumble("minor", "Minor issue", nil)
	handler.Record(stumble)
	assert.True(t, handler.ShouldContinue())

	// Record fall - should stop
	fall := NewFall("critical", "Critical error", nil)
	handler.Record(fall)
	assert.False(t, handler.ShouldContinue())
}

// TestPolicy_Default tests default policy
func TestPolicy_Default(t *testing.T) {
	policy := DefaultPolicy()

	assert.True(t, policy.StopOnFall)
	assert.Equal(t, 10, policy.MaxStumbles)
	assert.Contains(t, policy.RecoverableTypes, "visual")
	assert.Contains(t, policy.RecoverableTypes, "interaction")
	assert.Contains(t, policy.RecoverableTypes, "timing")

	// Check retry policies exist
	assert.NotNil(t, policy.RetryPolicy["visual"])
	assert.NotNil(t, policy.RetryPolicy["interaction"])
	assert.NotNil(t, policy.RetryPolicy["timing"])
}

// TestSeverity_String tests severity string representation
func TestSeverity_String(t *testing.T) {
	assert.Equal(t, "stumble", Stumble.String())
	assert.Equal(t, "error", Error.String())
	assert.Equal(t, "fall", Fall.String())
}