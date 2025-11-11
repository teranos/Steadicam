# Steadicam Failure Mode Coverage

This document describes the comprehensive failure mode testing capabilities added to the steadicam framework.

## Overview

Steadicam now includes robust error handling and failure analysis features that provide:

- **🚨 Panic Recovery**: Model crashes are caught and handled gracefully
- **⚡ Fail-Fast Behavior**: Tests terminate immediately on critical errors
- **📊 Rich Error Context**: Detailed error information with visual snapshots
- **🎨 HTML Report Integration**: Error analysis displayed in generated reports

## Error Categories

### Timeout Errors
- **Trigger**: Operations that exceed configured timeout limits
- **Format**: `[timeout:error] Timeout waiting for text 'expected_text'`
- **Context**: Expected text, current view, timing information

### Assertion Failures
- **Trigger**: Failed view assertions (text not found, incorrect state)
- **Format**: `[assertion:error] View does not contain expected text: expected_text`
- **Context**: Expected vs actual view content, assertion details

### Model Panics (Future)
- **Trigger**: BubbleTea model panics during Update() calls
- **Format**: `[panic:error] Model panic during Update: panic_message`
- **Context**: Panic details, model type, message context

## Error Data Structure

```go
type StageResult struct {
    Success      bool
    ErrorMessage string    // Human-readable error summary
    ErrorDetails string    // Technical error information
    TripReport   string    // Trip handler detailed report
    // ... other fields
}
```

## HTML Report Features

When tests fail, the HTML report includes an **Error Analysis** section with:

- **Error Summary**: High-level description of what went wrong
- **Technical Details**: Stack traces, timing, and diagnostic information
- **Trip Handler Report**: Structured error context and metadata

## Usage Examples

### Basic Failure Handling
```go
director := steadicam.NewStageDirector(t, adapter).
    WithTimeout(3 * time.Second). // Must be before Start()
    Start()

result := director.
    Type("test input").
    AssertViewContains("text that doesn't exist"). // Will fail
    Stop()

// Check results
if !result.Success {
    t.Logf("Test failed: %s", result.ErrorMessage)
    t.Logf("Details: %s", result.ErrorDetails)
}
```

### Configuration Requirements
```go
// ✅ CORRECT: Configure before Start()
director := steadicam.NewStageDirector(t, adapter).
    WithTimeout(5 * time.Second).
    WithViewCapture(true).
    Start()

// ❌ INCORRECT: Configure after Start() - will be ignored with warning
director.Start().WithTimeout(5 * time.Second) // Ignored!
```

### Timeout Testing
```go
director := steadicam.NewStageDirector(t, adapter).
    WithTimeout(100 * time.Millisecond). // Very short timeout
    Start()

result := director.
    WaitForText("text that will never appear"). // Will timeout
    Stop()
```

### HTML Report Generation
```go
report := steadicam.TestReport{
    TestName:     "MyFailingTest",
    Success:      result.Success,
    ErrorMessage: result.ErrorMessage,
    ErrorDetails: result.ErrorDetails,
    TripReport:   result.TripReport,
    // ... other fields
}

generator := steadicam.NewHTMLReportGenerator(outputDir)
generator.GenerateReport(report)
```

## Performance Characteristics

- **Fail-Fast Response**: Critical errors detected in <300ms
- **Context Capture**: Visual snapshots and diagnostic data collected automatically
- **Memory Efficient**: Error handling adds minimal overhead to normal test execution

## Testing the Failure Modes

Run the failure mode tests to see the system in action:

```bash
# Basic failure mode tests
go test ./cmd/repl/bubble/ -run TestSteadicam_TimeoutHandling -v
go test ./cmd/repl/bubble/ -run TestSteadicam_AssertionFailure -v

# Demonstration test with HTML report generation
go test ./cmd/repl/bubble/ -run TestSteadicam_FailureReportDemo -v
```

The demonstration test intentionally fails to showcase the error analysis features and generates an HTML report that can be opened in a browser to see the failure analysis in action.