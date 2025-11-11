# 🎬 Steadicam Tutorial - Cinematic Testing for BubbleTea

Welcome to Steadicam, a testing framework inspired by Stanley Kubrick's revolutionary cinematography. Just as Kubrick's steadicam captured smooth, precise movements through complex scenes, Steadicam captures fluid interactions in your terminal applications.

## Quick Start: Your First Test

Let's create a simple REPL application and test it with Steadicam:

### Step 1: Implement REPLModel Interface

```go
package main

import (
    "strings"
    tea "github.com/charmbracelet/bubbletea"
    "github.com/sbvh/qntx/cmd/repl/bubble/steadicam"
)

type SimpleREPL struct {
    input string
    mode  string
    output string
}

// Implement steadicam.REPLModel interface
func (m SimpleREPL) CurrentInput() string { return m.input }
func (m SimpleREPL) CurrentMode() string { return m.mode }
func (m SimpleREPL) CheckCondition(condition string) bool {
    switch condition {
    case "has_output": return m.output != ""
    case "empty_input": return m.input == ""
    default: return false
    }
}

// Standard BubbleTea methods
func (m SimpleREPL) Init() tea.Cmd { return nil }
func (m SimpleREPL) View() string {
    return fmt.Sprintf("Mode: %s\nInput: %s\nOutput: %s\n> ", m.mode, m.input, m.output)
}
func (m SimpleREPL) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        switch msg.Type {
        case tea.KeyRunes:
            m.input += string(msg.Runes)
        case tea.KeyEnter:
            m.output = "Hello " + m.input
            m.input = ""
        }
    }
    return m, nil
}
```

### Step 2: Write Your First Steadicam Test

```go
func TestSimpleREPL(t *testing.T) {
    // Create your REPL model
    model := SimpleREPL{mode: "input"}

    // Start the test with Steadicam's smooth direction
    result := steadicam.NewInteractiveTestDirector(t, model).
        WithTimeout(5 * time.Second).
        Start().
        Type("world").                           // Smooth typing
        AssertViewContains("world").            // Validate state
        PressEnter().                           // Execute
        AssertViewContains("Hello world").      // Check results
        Stop()

    // Verify the cinematic sequence completed successfully
    assert.True(t, result.Success, "The test should complete without errors")
    assert.Greater(t, len(result.Interactions), 3, "Should capture multiple interactions")
}
```

## Core Concepts

### The Director's Vision

Like Kubrick planning each shot, `InteractiveTestDirector` orchestrates your test sequence:

```go
director := steadicam.NewInteractiveTestDirector(t, model).
    WithTimeout(10 * time.Second).              // Set scene timing
    WithViewCapture(true).                      // Enable visual snapshots
    Start()                                     // Begin filming
```

### Smooth Interactions

Steadicam provides fluid input simulation:

```go
director.
    Type("search query").                       // Character-by-character typing
    PressEnter().                              // Key press simulation
    PressTab().                                // Navigation
    PressArrowDown().                          // List movement
    Wait(100 * time.Millisecond)               // Pause for timing
```

### Smart Waiting (The Art of Patience)

Just as Kubrick waited for the perfect moment, Steadicam waits for your application:

```go
director.
    Type("query").
    WaitForMode("results").                    // Wait for mode change
    WaitForText("Found 5 items").             // Wait for specific text
    WaitForSearchResults()                     // Wait for search completion
```

### Precise Assertions

Validate your application's state with surgical precision:

```go
director.
    AssertMode("search").                      // Verify current mode
    AssertViewContains("Welcome").             // Check rendered content
    AssertInputEquals("hello").                // Validate input state
    AssertNoSearchResults()                    // Ensure empty results
```

## Visual Testing: The Operator's Touch

For visual testing, use the `Operator` - Steadicam's master cinematographer:

```go
func TestWithVisualCapture(t *testing.T) {
    model := NewMyREPLModel()

    steadicam.NewOperator(t, model, "screenshots/").
        Start().
        CaptureTrackingShot("initial").         // Capture opening frame
        TypeWithTrackingShot("hello", "typing"). // Type with visual tracking
        PressEnterWithTrackingShot("executed"). // Execute with capture
        WaitForTextWithTrackingShot("result", "completed"). // Wait and capture
        Stop()
}
```

This creates a series of PNG screenshots documenting your application's visual journey.

## Configuration: Perfecting the Shot

Customize Steadicam's behavior like adjusting camera settings:

```go
config := steadicam.DirectorConfig{
    Timeout:      5 * time.Second,             // Shorter timeouts for fast tests
    TypingSpeed:  0,                           // No delay for speed
    CaptureViews: false,                       // Disable snapshots for performance
    MaxRetries:   1,                           // No retries for deterministic tests
}

director := steadicam.NewInteractiveTestDirectorWithConfig(t, model, config)
```

## Error Handling: When Things Don't Go to Plan

Steadicam collects errors gracefully instead of crashing immediately:

```go
result := director.
    Type("bad input").
    WaitForText("impossible text").            // This will fail
    Stop()

if !result.Success {
    t.Logf("Test failed: %s", result.ErrorMessage)
    t.Logf("Error type: %T", result.Error)

    // Examine what happened
    for _, snapshot := range result.Snapshots {
        t.Logf("View at %v:\n%s", snapshot.Timestamp, snapshot.View)
    }
}
```

## Advanced Patterns

### Performance Testing

```go
func TestPerformance(t *testing.T) {
    start := time.Now()

    result := director.
        Type("large query").
        WaitForSearchResults().
        Stop()

    assert.Less(t, time.Since(start), 200*time.Millisecond)
    assert.True(t, result.Success)
}
```

### Debounced Input Testing

```go
func TestDebouncing(t *testing.T) {
    director.
        Type("R").                             // Start typing
        Wait(25 * time.Millisecond).          // Half debounce delay
        CheckCondition("no_search").           // Shouldn't search yet
        Wait(60 * time.Millisecond).          // Complete debounce
        WaitForSearchResults()                 // Now should search
}
```

### Complex Workflows

```go
func TestCompleteWorkflow(t *testing.T) {
    result := director.Start().
        // Login sequence
        Type("username").PressTab().
        Type("password").PressEnter().
        WaitForMode("main").

        // Search sequence
        Type("search term").
        WaitForSearchResults().
        PressArrowDown().                      // Navigate results
        PressEnter().                          // Select
        WaitForMode("detail").

        // Validate final state
        AssertViewContains("Details for").
        Stop()

    assert.True(t, result.Success)
}
```

## Best Practices

### 1. Model Your Application State

Implement `CheckCondition` thoughtfully:

```go
func (m MyREPL) CheckCondition(condition string) bool {
    switch condition {
    case "search_results": return len(m.results) > 0
    case "loading": return m.isLoading
    case "error_shown": return m.errorMsg != ""
    case "has_selection": return m.selectedIndex >= 0
    default: return false
    }
}
```

### 2. Use Descriptive Test Names

```go
func TestSearchFlow_FindsResultsAndNavigates(t *testing.T)
func TestErrorHandling_RecoversFromInvalidInput(t *testing.T)
func TestPerformance_SearchCompletesUnder200ms(t *testing.T)
```

### 3. Group Related Tests

```go
func TestSearchFeatures(t *testing.T) {
    tests := []struct {
        name  string
        query string
        expect string
    }{
        {"exact match", "Rob Pike", "Found 1"},
        {"partial match", "Rob", "Found 2"},
        {"no match", "xyz123", "No results"},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := director.Start().
                Type(tt.query).
                WaitForSearchResults().
                AssertViewContains(tt.expect).
                Stop()
            assert.True(t, result.Success)
        })
    }
}
```

### 4. Clean Up Resources

```go
func TestWithCleanup(t *testing.T) {
    model := NewMyREPLModel()
    defer model.Close()  // Clean up any resources

    result := steadicam.NewInteractiveTestDirector(t, model).
        // ... test sequence
        Stop()

    assert.True(t, result.Success)
}
```

## Troubleshooting

### Common Issues

**Timeout waiting for text:**
- Check that the expected text actually appears in your UI
- Increase timeout with `WithTimeout()`
- Use `CheckCondition` for custom wait logic

**Mode assertions failing:**
- Verify your `CurrentMode()` implementation returns expected strings
- Check mode transitions in your application logic

**Performance issues:**
- Disable view capture with `WithViewCapture(false)`
- Set `TypingSpeed: 0` in config for faster input

### Debugging Tips

```go
// Enable verbose logging
result := director.Start().
    // ... interactions
    Stop()

// Examine the test journey
t.Logf("Test completed in %v", result.Duration)
for i, interaction := range result.Interactions {
    t.Logf("Step %d: %s at %v", i, interaction.Type, interaction.Timestamp)
}
```

## Integration with Existing Test Suites

Steadicam works seamlessly with your existing Go test infrastructure:

```go
// TestMain for global setup
func TestMain(m *testing.M) {
    setupDatabase()
    code := m.Run()
    teardownDatabase()
    os.Exit(code)
}

// Standard Go testing patterns
func TestREPLFeatures(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping interactive tests in short mode")
    }

    // ... steadicam tests
}
```

## Next Steps

- Explore the operator package for advanced visual testing
- Set up visual regression testing with baselines
- Integrate Steadicam tests into your CI/CD pipeline
- Contribute to the Steadicam project on GitHub

Welcome to the world of cinematic testing. Like Kubrick's steadicam revolutionized filmmaking, Steadicam brings precision and artistry to terminal application testing.

*"The perfect steadicam shot is not just about smooth movement—it's about capturing the essence of the moment with unshakeable precision."* - Stanley Kubrick (paraphrased)