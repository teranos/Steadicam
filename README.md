# 🎬 Steadicam - BubbleTea Visual QA Framework

**Smooth, cinematic testing for your terminal applications.**

Steadicam provides automated testing for [BubbleTea](https://github.com/charmbracelet/bubbletea) applications with visual tracking capabilities. Inspired by Stanley Kubrick's revolutionary steadicam work, it captures fluid UI interactions with precision timing.

## Quick Start

```bash
go get github.com/your-org/steadicam
```

## Basic Example

```go
func TestREPLSearch(t *testing.T) {
    // Create your BubbleTea model
    model := NewYourREPLModel()

    // Create adapter for steadicam
    adapter := NewREPLAdapter(model)

    // Test with smooth interactions
    result := steadicam.NewInteractiveTestDirector(t, adapter).
        WithTimeout(5 * time.Second).
        Start().
        Type("search query").              // Type naturally
        PressTab().                        // Navigate
        WaitForMode("results").           // Wait for state change
        AssertViewContains("Found").      // Verify results
        Stop()

    assert.True(t, result.Success)
}
```

## Visual Testing with Screenshots

```go
func TestWithScreenshots(t *testing.T) {
    model := NewYourREPLModel()
    adapter := NewREPLAdapter(model)

    steadicam.NewOperator(t, adapter, "screenshots/").
        Start().
        CaptureTrackingShot("initial").
        TypeWithTrackingShot("hello", "typed_hello").
        PressEnterWithTrackingShot("executed").
        Stop()
}
```

## Required Interface

Your REPL model needs to implement:

```go
type REPLModel interface {
    tea.Model
    CurrentInput() string
    CurrentMode() string
    CheckCondition(condition string) bool
}
```

## Features

- 🎯 **Precise interactions** - Type, key presses, navigation
- ⏱️ **Smart waiting** - Wait for conditions, modes, text
- 🔍 **Rich assertions** - View contents, modes, states
- 📸 **Visual testing** - Automated screenshots and tracking shots
- 🎬 **Smooth operator** - Fluid test execution like Kubrick's cinematography

## Documentation

See [examples/](examples/) for complete working examples.

## License

MIT
