# Simple REPL Example

This example demonstrates a minimal BubbleTea REPL application that implements the `steadicam.REPLModel` interface for automated testing.

## Running the Example

```bash
# Run the interactive REPL
go run main.go

# Run the automated tests
go test -v

# Run tests with visual capture
go test -v -run TestSimpleREPL_VisualTesting
```

## Features Demonstrated

- **Basic REPL interaction** - Type input, press Enter to submit
- **Mode tracking** - Switches between "input" and "result" modes
- **Custom conditions** - Implements `CheckCondition` for test validation
- **Visual testing** - Screenshots captured during test execution

## Test Coverage

- `TestSimpleREPL_BasicInteraction` - Complete user flow with assertions
- `TestSimpleREPL_ClearInput` - Escape key behavior
- `TestSimpleREPL_VisualTesting` - Screenshot generation with Operator
- `TestSimpleREPL_Conditions` - Custom condition checking
- `TestSimpleREPL_Performance` - Response time validation

## Key Implementation Details

### REPLModel Interface

```go
func (m SimpleREPL) CurrentInput() string { return m.input }
func (m SimpleREPL) CurrentMode() string { return m.mode }
func (m SimpleREPL) CheckCondition(condition string) bool {
    switch condition {
    case "has_output": return m.output != ""
    case "empty_input": return m.input == ""
    // ... more conditions
    }
}
```

### Steadicam Test Pattern

```go
result := steadicam.NewInteractiveTestDirector(t, model).
    WithTimeout(5 * time.Second).
    Start().
    Type("hello").
    PressEnter().
    WaitForMode("result").
    AssertViewContains("Hello, hello").
    Stop()
```

This example serves as a template for implementing steadicam testing in your own BubbleTea applications.