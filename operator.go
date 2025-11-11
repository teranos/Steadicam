package steadicam

import (
	"fmt"
	"image/color"
	"testing"
	"time"

	"github.com/sbvh/qntx/cmd/repl/bubble/steadicam/trip"
)

// Operator extends StageDirector with smooth visual tracking capabilities
// Kubrick's steadicam operator capturing fluid UI movement without jarring cuts
type Operator struct {
	*StageDirector
	renderingStage *RenderingStage
	frameCount     int
	filmDir        string
}

// NewOperator creates a test director that can capture smooth visual tracking shots
// Stanley's trusted camera operator for fluid UI cinematography
func NewOperator(t *testing.T, model REPLModel, outputDir string) *Operator {
	config := Config{
		Width:      80,
		Height:     24,
		FontSize:   12,
		Background: color.RGBA{0, 0, 0, 255},       // Black background
		Foreground: color.RGBA{255, 255, 255, 255}, // White text
		OutputDir:  outputDir,
	}

	baseDirector := NewStageDirector(t, model)

	return &Operator{
		StageDirector: baseDirector,
		renderingStage: NewRenderingStage(config),
		frameCount:     0,
		filmDir:        outputDir,
	}
}

// WithConfig allows customizing the steadicam shot appearance
func (op *Operator) WithConfig(config Config) *Operator {
	op.renderingStage = NewRenderingStage(config)
	op.filmDir = config.OutputDir
	return op
}

// WithTimeout wraps the base WithTimeout method to return *Operator
func (op *Operator) WithTimeout(timeout time.Duration) *Operator {
	op.StageDirector.WithTimeout(timeout)
	return op
}

// Start wraps the base Start method to return *Operator
func (op *Operator) Start() *Operator {
	op.StageDirector.Start()
	return op
}

// WaitForSearchResults wraps the base method to return *Operator
func (op *Operator) WaitForSearchResults() *Operator {
	op.StageDirector.WaitForSearchResults()
	return op
}

// WaitForText wraps the base method to return *Operator
func (op *Operator) WaitForText(text string) *Operator {
	op.StageDirector.WaitForText(text)
	return op
}

// Stop wraps the base method to return stage result
func (op *Operator) Stop() *StageResult {
	return op.StageDirector.Stop()
}

// CaptureTrackingShot captures the current visual state as a smooth film frame
// Kubrick's signature fluid camera movement captured digitally
func (op *Operator) CaptureTrackingShot(label string) *Operator {
	// Get current view from test director
	// We'll need to expose getCurrentView() method in the test director
	// For now, let's use the View() method from the underlying model
	currentView := "[view capture not yet implemented]"

	// Render to steadicam rig
	op.renderingStage.RenderText(currentView)

	// Generate filename with timestamp and counter for uniqueness
	timestamp := time.Now().Format("20060102_150405")
	filename := fmt.Sprintf("%s/frame_%s_%03d_%s.png",
		op.filmDir, timestamp, op.frameCount, label)

	// Save frame
	if err := op.renderingStage.CaptureFrame(filename); err != nil {
		// Camera jam is fatal - stop everything
		cameraTrip := trip.NewFall("visual", fmt.Sprintf("Camera seizure during frame capture: %v", err),
			trip.Context{"filename": filename, "frame_count": op.frameCount, "original_error": err.Error()})
		op.StageDirector.recordTrip(cameraTrip)
		return op
	}

	op.frameCount++
	// Record the tracking shot interaction
	// Note: recordInteraction is not exported, so we'll skip this for now
	// op.InteractiveTestDirector.recordInteraction("tracking_shot", ...)

	return op
}

// Enhanced fluent methods with automatic tracking shots
func (op *Operator) TypeWithTrackingShot(text string, label string) *Operator {
	op.Type(text)
	op.CaptureTrackingShot(label)
	return op
}

func (op *Operator) PressEnterWithTrackingShot(label string) *Operator {
	op.PressEnter()
	op.CaptureTrackingShot(label)
	return op
}

func (op *Operator) PressTabWithTrackingShot(label string) *Operator {
	op.PressTab()
	op.CaptureTrackingShot(label)
	return op
}

func (op *Operator) WaitForTextWithTrackingShot(text string, label string) *Operator {
	op.WaitForText(text)
	op.CaptureTrackingShot(label)
	return op
}

func (op *Operator) WaitForModeWithTrackingShot(mode string, label string) *Operator {
	op.WaitForMode(mode)
	op.CaptureTrackingShot(label)
	return op
}