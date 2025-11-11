package steadicam

import (
	"context"
	"sync"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// BenchmarkModel implements REPLModel for benchmarking
type BenchmarkModel struct {
	input string
	mode  string
	mu    sync.RWMutex
}

func (m *BenchmarkModel) Init() tea.Cmd { return nil }

func (m *BenchmarkModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok {
		m.mu.Lock()
		m.input += key.String()
		m.mu.Unlock()
	}
	return m, nil
}

func (m *BenchmarkModel) View() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return "Benchmark: " + m.input
}

func (m *BenchmarkModel) CurrentInput() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.input
}

func (m *BenchmarkModel) CurrentMode() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.mode
}

func (m *BenchmarkModel) CheckCondition(condition string) bool {
	return condition == "benchmark"
}

// BenchmarkModelUpdateProcessing measures the performance of model update processing
// Tests the core synchronization path including atomic operations and channel sends
func BenchmarkModelUpdateProcessing(b *testing.B) {
	model := &BenchmarkModel{mode: "benchmark"}
	director := NewStageDirectorWithConfig(
		&testing.T{}, // We need a *testing.T for constructor
		model,
		StageConfig{
			Timeout:      time.Second,
			CaptureViews: false, // Disable for performance
		},
	)

	// Start the director to initialize synchronization
	director.Start()
	defer director.Stop()

	// Wait for initialization
	time.Sleep(10 * time.Millisecond)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// Simulate a keystroke update
		director.sendMessage(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
	}
}

// BenchmarkHighVolumeUpdates tests performance under high update volume
// Simulates rapid user input or automated test scenarios
func BenchmarkHighVolumeUpdates(b *testing.B) {
	model := &BenchmarkModel{mode: "benchmark"}
	director := NewStageDirectorWithConfig(
		&testing.T{},
		model,
		StageConfig{
			Timeout:      time.Second,
			CaptureViews: false,
		},
	)

	director.Start()
	defer director.Stop()

	// Wait for initialization
	time.Sleep(10 * time.Millisecond)

	b.ResetTimer()
	b.ReportAllocs()

	// Send updates in batches to test buffer utilization
	batchSize := 10
	for i := 0; i < b.N; i += batchSize {
		for j := 0; j < batchSize && i+j < b.N; j++ {
			director.sendMessage(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})
		}
		// Brief pause to simulate realistic input patterns
		time.Sleep(time.Microsecond)
	}
}

// BenchmarkBufferOverflowScenario tests behavior when channel buffer is full
// Important for understanding graceful degradation characteristics
func BenchmarkBufferOverflowScenario(b *testing.B) {
	model := &BenchmarkModel{mode: "benchmark"}

	// Use smaller buffer for this test to trigger overflow conditions
	director := &StageDirector{
		t:                &testing.T{},
		model:           model,
		ctx:             context.Background(),
		cancel:          func() {},
		interactions:    make([]StageAction, 0),
		snapshots:       make([]StageSnapshot, 0),
		waiting:         make(map[string]chan struct{}),
		config:          StageConfig{Timeout: time.Second, CaptureViews: false},
		modelChan:       make(chan modelUpdate, 5), // Small buffer for overflow testing
		latestModel:     model,
		updateSeq:       0,
		lastProcessedSeq: 0,
		droppedUpdates:   0,
		updatesSent:      0,
		updatesProcessed: 0,
		bufferOverflows:  0,
	}

	// Start sync goroutine manually for this test
	go director.syncModelUpdates()
	defer director.cancel()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// Create wrapper to test overflow behavior
		wrapper := stageModelWrapper{
			REPLModel: model,
			director:  director,
		}

		// Send rapid updates to trigger buffer overflow
		wrapper.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("o")})
	}

	// Report overflow statistics
	stats := director.GetSynchronizationStats()
	b.ReportMetric(float64(stats["buffer_overflows"]), "overflows")
	b.ReportMetric(float64(stats["updates_dropped"]), "dropped")
}

// BenchmarkConcurrentAccess measures performance under concurrent read/write access
// Tests the RWMutex performance for model state access
func BenchmarkConcurrentAccess(b *testing.B) {
	model := &BenchmarkModel{mode: "benchmark"}
	director := NewStageDirectorWithConfig(
		&testing.T{},
		model,
		StageConfig{
			Timeout:      time.Second,
			CaptureViews: false,
		},
	)

	director.Start()
	defer director.Stop()

	// Wait for initialization
	time.Sleep(10 * time.Millisecond)

	b.ResetTimer()
	b.ReportAllocs()

	// Run concurrent readers and writers
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			// Mix of read and write operations
			switch b.N % 4 {
			case 0, 1, 2:
				// 75% read operations
				_ = director.getCurrentView()
				_ = director.getCurrentInput()
				_ = director.getCurrentMode()
			case 3:
				// 25% write operations (updates)
				director.sendMessage(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("c")})
			}
		}
	})
}

// BenchmarkMemoryAllocation measures memory allocation patterns
// Important for understanding GC pressure in long-running tests
func BenchmarkMemoryAllocation(b *testing.B) {
	model := &BenchmarkModel{mode: "benchmark"}
	director := NewStageDirectorWithConfig(
		&testing.T{},
		model,
		StageConfig{
			Timeout:      time.Second,
			CaptureViews: false,
		},
	)

	director.Start()
	defer director.Stop()

	// Wait for initialization
	time.Sleep(10 * time.Millisecond)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// Test various operations that allocate memory
		director.sendMessage(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("m")})

		// Occasional state queries to test read allocations
		if i%10 == 0 {
			_ = director.getCurrentView()
			_ = director.GetLatestSnapshot()
		}
	}
}

// BenchmarkSequenceTracking measures the overhead of sequence number tracking
// Tests atomic operations performance under load
func BenchmarkSequenceTracking(b *testing.B) {
	model := &BenchmarkModel{mode: "benchmark"}
	director := NewStageDirectorWithConfig(
		&testing.T{},
		model,
		StageConfig{
			Timeout:      time.Second,
			CaptureViews: false,
		},
	)

	director.Start()
	defer director.Stop()

	// Wait for initialization
	time.Sleep(10 * time.Millisecond)

	b.ResetTimer()
	b.ReportAllocs()

	// Measure atomic sequence operations
	for i := 0; i < b.N; i++ {
		wrapper := stageModelWrapper{
			REPLModel: model,
			director:  director,
		}

		// This will trigger atomic sequence increment and tracking
		wrapper.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")})
	}

	// Report sequence statistics
	stats := director.GetSynchronizationStats()
	b.ReportMetric(float64(stats["updates_generated"]), "sequences")
	b.ReportMetric(float64(stats["updates_sent"]), "sent")
	b.ReportMetric(float64(stats["updates_processed"]), "processed")
}

// BenchmarkContextCancellation measures cleanup performance
// Important for understanding test teardown costs
func BenchmarkContextCancellation(b *testing.B) {
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		model := &BenchmarkModel{mode: "benchmark"}
		director := NewStageDirectorWithConfig(
			&testing.T{},
			model,
			StageConfig{
				Timeout:      100 * time.Millisecond,
				CaptureViews: false,
			},
		)

		director.Start()

		// Send a few updates
		director.sendMessage(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("c")})

		// Measure cleanup time
		director.Stop()
	}
}