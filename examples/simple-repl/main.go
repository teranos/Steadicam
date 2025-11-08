// Package main provides a simple REPL example for demonstrating Steadicam testing.
//
// This example shows a minimal BubbleTea REPL application that implements
// the steadicam.REPLModel interface, making it testable with the steadicam framework.
//
// Run the example:
//   go run main.go
//
// Run the tests:
//   go test -v
package main

import (
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// SimpleREPL is a basic REPL that echoes user input with a greeting.
type SimpleREPL struct {
	input  string
	output string
	mode   string
}

// NewSimpleREPL creates a new instance of the simple REPL.
func NewSimpleREPL() SimpleREPL {
	return SimpleREPL{
		mode: "input",
	}
}

// Init implements tea.Model interface.
func (m SimpleREPL) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model interface.
func (m SimpleREPL) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC:
			return m, tea.Quit
		case tea.KeyEnter:
			if m.input != "" {
				m.output = fmt.Sprintf("Hello, %s! You entered: %s", m.input, m.input)
				m.input = ""
				m.mode = "result"
			}
		case tea.KeyEsc:
			m.input = ""
			m.output = ""
			m.mode = "input"
		case tea.KeyRunes:
			m.input += string(msg.Runes)
		case tea.KeyBackspace:
			if len(m.input) > 0 {
				m.input = m.input[:len(m.input)-1]
			}
		}
	}
	return m, nil
}

// View implements tea.Model interface.
func (m SimpleREPL) View() string {
	var b strings.Builder

	b.WriteString("🎬 Simple REPL Example\n")
	b.WriteString("======================\n\n")

	if m.output != "" {
		b.WriteString(fmt.Sprintf("Output: %s\n\n", m.output))
	}

	b.WriteString(fmt.Sprintf("Mode: %s\n", m.mode))
	b.WriteString(fmt.Sprintf("Input: %s\n", m.input))
	b.WriteString("repl> ")

	b.WriteString("\n\n(Press Enter to submit, Esc to clear, Ctrl+C to quit)")

	return b.String()
}

// CurrentInput implements steadicam.REPLModel interface.
func (m SimpleREPL) CurrentInput() string {
	return m.input
}

// CurrentMode implements steadicam.REPLModel interface.
func (m SimpleREPL) CurrentMode() string {
	return m.mode
}

// CheckCondition implements steadicam.REPLModel interface.
func (m SimpleREPL) CheckCondition(condition string) bool {
	switch condition {
	case "has_output":
		return m.output != ""
	case "empty_input":
		return m.input == ""
	case "has_input":
		return m.input != ""
	case "input_mode":
		return m.mode == "input"
	case "result_mode":
		return m.mode == "result"
	default:
		return false
	}
}

func main() {
	model := NewSimpleREPL()

	program := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := program.Run(); err != nil {
		fmt.Printf("Error running program: %v\n", err)
		os.Exit(1)
	}
}