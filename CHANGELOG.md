# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.1.0] - 2024-11-08

### Added
- Initial release of Steadicam testing framework
- `InteractiveTestDirector` for orchestrating automated BubbleTea tests
- `Operator` for visual testing with screenshot capabilities
- `REPLModel` interface for testable BubbleTea applications
- Comprehensive interaction methods: Type, PressEnter, PressTab, navigation keys
- Smart waiting capabilities: WaitForMode, WaitForText, WaitForCondition
- Rich assertions: AssertViewContains, AssertMode, AssertInputEquals
- Visual testing with PNG screenshot generation
- HTML report generation for test runs
- Configurable timeouts, typing speeds, and capture options
- Complete documentation with README and TUTORIAL
- Working example application with test suite
- Unit tests for core framework functionality

### Features
- 🎯 Precise UI interactions with configurable timing
- ⏱️ Smart condition-based waiting with timeout handling
- 🔍 Rich assertions for view contents and application state
- 📸 Visual regression testing with automated screenshots
- 🎬 Kubrick-inspired API design for memorable testing experience
- 📊 HTML reports with interaction timelines and visual captures
- 🔧 Flexible configuration for different testing scenarios

[0.1.0]: https://github.com/teranos/steadicam/releases/tag/v0.1.0