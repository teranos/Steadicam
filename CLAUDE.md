# Steadicam Package - Visual Test Inspection & QCQA

## Purpose

The steadicam package provides **visual test inspection and Quality Control/Quality Assurance (QCQA)** for the qntx REPL and terminal applications. Its primary mission is to capture and document actual user interactions with PTY. It is written in Go and is open-source.

## Core Principles

### 1. Real Testing Only
- **Never mock, never fake, always real REPL test**
- Launch actual binary processes using PTY
- Send genuine keystrokes to real running applications
- Capture authentic terminal output on every keystroke and every change in output

### 2. Authentic Visual Representation
- Convert ANSI escape sequences to rich HTML with proper colors and formatting
- Preserve terminal aesthetics and real-time interaction patterns
- Generate comprehensive visual reports that accurately represent user experience

### 3. Thorough Quality Assurance
- Character-by-character typing to demonstrate live behavior
- Capture every stage of user interaction for complete QCQA coverage
- Document real timing and performance characteristics down to the milisecond 

## Technical Guidelines

### Test Execution
- Use PTY for real terminal interaction
- Launch actual binaries
- Send individual keystrokes with realistic timing delays
- Debounce limit consideration: ~51ms

### ANSI Output Handling
- Capture raw ANSI output for each interaction step
- Convert ANSI for visual report generation
- Preserve color codes, cursor movements, and terminal formatting
- Generate progressive screenshots showing character-by-character changes

### Report Generation
- Create HTML reports 
- Show complete interaction flow from initial state to final results
- Include timing information and search performance metrics
- Integrate with central test dashboard for QCQA oversight

## Testing Patterns

### Visual Quality Assurance
- Verify live search results appear correctly
- Confirm search result counts update in real-time
- Document search performance timing
- Validate terminal color and formatting preservation

## Quality Standards

- Tests must demonstrate actual user experience, not simulated behavior
- Visual reports must accurately represent terminal appearance
- All interactions must be captured for complete quality documentation
- Long test duration is acceptable to ensure thorough coverage
- Real-time behavior must be visually documented

✅ **Always use these approaches:**
- Real binary execution
- Authentic PTY interaction
- Complete interaction capture
- Rich visual report generation
- Comprehensive QCQA documentation
