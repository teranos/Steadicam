# Steadicam Synchronization

## Overview

The StageDirector uses a simple, embedded synchronization approach to track model updates from the BubbleTea program.

## Architecture

**Implementation**: `syncModelUpdates()` method in `stage_director_methods.go`

**Key Features**:
- Sequential model update processing with order detection
- Duplicate detection using atomic sequence tracking
- Non-blocking channel sends with overflow protection
- Integration with StageDirector's atomic counters for metrics

## How It Works

### 1. Initialization
```go
// Started automatically in constructor
go director.syncModelUpdates()
```

### 2. Model Updates
When the BubbleTea program calls `Update()` on the wrapped model:
1. **Sequence Generation**: Atomic sequence number assigned
2. **Non-blocking Send**: Update sent to director's channel with overflow protection
3. **Background Processing**: `syncModelUpdates()` processes updates sequentially
4. **State Sync**: Latest model state updated with atomic operations

### 3. State Access
```go
// Thread-safe access to current model state
view := director.getCurrentView()
input := director.getCurrentInput()
mode := director.getCurrentMode()
```

## Error Handling

- **Buffer Overflow**: Gracefully handled with metrics tracking
- **Sequence Gaps**: Detected and counted for diagnostics
- **Duplicate Updates**: Filtered out automatically
- **Context Cancellation**: Clean shutdown when context expires

## Metrics

Available through `GetSynchronizationStats()`:
- `updates_generated`: Total updates created
- `updates_sent`: Updates sent to processing
- `updates_processed`: Updates handled by background goroutine
- `buffer_overflows`: Times channel was full
- `sequence_gaps`: Detected ordering issues
- `duplicate_updates`: Filtered duplicate updates
- `updates_dropped`: Total dropped (sum of overflows)

## Configuration

Synchronization behavior is configured through `StageConfig`:

```go
config := StageConfig{
    Timeout: 5 * time.Second,  // Operation timeout
    CaptureViews: true,        // Enable view snapshots
    // Channel buffer size is fixed at 100 updates
}
```

## Thread Safety

All operations are thread-safe through:
- **Atomic counters** for metrics and sequence tracking
- **RWMutex** for model state access in director methods
- **Buffered channels** for update queuing (100 capacity)
- **Context cancellation** for clean shutdown

## Performance

**Benchmark Results (Apple M1)**:

| Scenario | Throughput | Latency | Memory/Op | Allocs/Op | Notes |
|----------|------------|---------|-----------|-----------|-------|
| **Basic Updates** | 457 ops/sec | 2.2ms | 8.6KB | 65 | Normal operation pattern |
| **High Volume** | 65,191 ops/sec | 15.3μs | 9.5KB | 65 | Batch processing mode |
| **Concurrent Access** | 6.9M ops/sec | 320ns | 0B | 0 | Read-heavy workload (75% reads) |
| **Buffer Overflow** | 23,368 ops/sec | 42.8μs | 504KB | 5 | Graceful degradation under load |
| **Memory Allocation** | 1,134 ops/sec | 884μs | 8.9KB | 65 | With periodic snapshot capture |
| **Context Cleanup** | 10 ops/sec | 101ms | 27KB | 201 | Full lifecycle cost |

**Key Performance Characteristics**:
- **Throughput**: Handles typical UI update rates (65K+ ops/sec in batch mode)
- **Latency**: Sub-millisecond for concurrent reads (320ns), low latency for updates (15-42μs)
- **Memory**: Minimal overhead with constant allocation patterns
- **Reliability**: Graceful buffer overflow handling with metrics tracking
- **Scalability**: Excellent concurrent read performance, bounded write performance

## Performance Testing

The synchronization path includes comprehensive benchmarks in `sync_bench_test.go`:

- **BenchmarkModelUpdateProcessing**: Tests basic update handling performance
- **BenchmarkHighVolumeUpdates**: Validates behavior under rapid input simulation
- **BenchmarkConcurrentAccess**: Tests RWMutex performance with mixed read/write workloads
- **BenchmarkBufferOverflowScenario**: Validates graceful degradation when buffer fills
- **BenchmarkMemoryAllocation**: Measures GC pressure from update processing
- **BenchmarkSequenceTracking**: Tests atomic sequence operation overhead
- **BenchmarkContextCancellation**: Measures cleanup and teardown costs

Run benchmarks with: `go test -bench=Benchmark -benchmem`

## Best Practices

1. **Monitor Dropped Updates**: Check `HasDroppedUpdates()` after tests
2. **Use Appropriate Timeouts**: Allow enough time for model state changes
3. **Event-Driven Waits**: Prefer `WaitForMode()` over polling loops
4. **Resource Cleanup**: Always call `Stop()` to clean up resources
5. **Performance Monitoring**: Use benchmarks to validate performance under load

## Example Usage

```go
director := steadicam.NewStageDirector(t, adapter).
    WithTimeout(5 * time.Second).
    WithViewCapture(true).
    Start()

result := director.
    Type("hello").
    WaitForMode("results").
    AssertViewContains("success").
    Stop()

// Check for any synchronization issues
if result.HasDroppedUpdates() {
    t.Log("Warning: Some updates were dropped during test")
}
```

This simple, embedded approach provides reliable synchronization without the complexity of separate components or sophisticated queuing mechanisms.