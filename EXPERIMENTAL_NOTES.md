# ⚠️ EXPERIMENTAL TERMINAL RECORDING SYSTEM

## Status: NEEDS MAJOR REVIEW AND REFINEMENT

This directory contains experimental work on terminal stream recording and playback for REPL visual testing. **This code is not production-ready** and requires significant development before reliable use.

## What Works

### ✅ Basic Foundation
- **Real PTY interaction capture**: Uses `github.com/creack/pty` for authentic terminal communication
- **Timing preservation**: Millisecond-accurate event capture during typing
- **HTML playback interface**: Professional-looking player with xterm.js
- **Continuous stream recording**: 90+ events captured per typing session

### ✅ Visual Interface
- Film strip timeline with auto-play
- Variable speed controls (0.25x to 8x)
- Keyboard navigation (Space, R, Arrow keys)
- Progress tracking and seeking

## What Needs Major Work

### 🚨 Critical Issues

#### **ANSI Parsing is Broken**
- Terminal state reconstruction is rudimentary at best
- Complex cursor positioning commands not handled properly
- Screen clearing and buffer management needs complete rewrite
- Color codes preserved but positioning is wrong

#### **Terminal Buffer Management**
- Simple 80x24 character grid implementation is insufficient
- No proper handling of terminal scrolling or viewport
- Cursor positioning logic is naive and incorrect
- Text wrapping and overflow not implemented

#### **State Reconstruction Logic**
- Current ANSI parser is a basic regex-based hack
- Needs proper state machine for terminal control sequences
- Missing support for most VT100/ANSI escape sequences
- Terminal mode switching not handled

### 🔧 Technical Debt

#### **Stream Capture Accuracy**
- Byte-by-byte capture may miss atomic terminal updates
- Buffer flushing timing (25ms) is arbitrary
- No validation that captured stream matches actual terminal state
- Race conditions possible between keypress and output capture

#### **Performance and Memory**
- All terminal states kept in memory during playback
- No streaming or chunked loading for long sessions
- JavaScript player loads entire event stream upfront
- No optimization for large capture sessions

## Files Overview

```
actual_repl_test.go           # Frame-based discrete snapshots (works better)
stream_repl_test.go          # Continuous stream capture (experimental)
terminal-state-player.js     # BROKEN: State reconstruction player
terminal-timeline.js         # Basic timeline player for frames
stream-player.js            # Raw ANSI dumping (incorrect approach)
```

## Recommended Next Steps

### 1. **Choose One Approach**
- Frame-based snapshots (`actual_repl_test.go`) are more reliable
- Continuous stream needs complete ANSI parsing rewrite
- Don't try to maintain both approaches simultaneously

### 2. **If Continuing Stream Approach**
- Replace simple regex parsing with proper terminal emulator library
- Consider using existing ANSI parsing libraries
- Implement proper VT100/xterm terminal state machine
- Add comprehensive test suite for ANSI sequence handling

### 3. **Validation Strategy**
- Create reference terminal recordings with known tools
- Compare output against actual terminal screenshots
- Test with various terminal applications beyond REPL
- Validate timing accuracy and synchronization

## Usage Warning

**DO NOT USE IN PRODUCTION** without thorough review and testing. The current terminal state reconstruction produces incorrect output that doesn't match actual terminal behavior.

The visual interface looks professional but the underlying terminal emulation is fundamentally flawed.

## Contact

If working on this code, consult with original implementer about design decisions and known limitations.

---
*Last updated: 2025-11-09*
*Commit: b2338f4 - Add experimental terminal stream recording system*