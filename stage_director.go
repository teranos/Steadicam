package steadicam

// This file contains only the essential type definitions and interfaces.
// The implementation has been split across multiple files for better maintainability:
//
// - stage_director_methods.go: Core StageDirector lifecycle and configuration methods
// - stage_error_handling.go: Error handling, panic recovery, and metrics
// - stage_synchronization.go: Model synchronization and concurrent access
// - stage_interactions.go: User interaction simulation methods
// - stage_types.go: Type definitions and data structures