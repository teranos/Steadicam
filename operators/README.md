# Steadicam Operators

This directory contains specialized camera operators for the steadicam visual testing system. Each operator focuses on a specific type of visual capture and recording.

## Operators

### `tea_operator.go`
BubbleTea interaction operator for capturing terminal interface state changes during automated testing. Provides smooth integration with BubbleTea applications.


## Architecture

Each operator follows the steadicam pattern:
- **Focused responsibility** - Single capture method (screenshots, recordings, etc.)
- **Stage integration** - Works with the stage director system
- **Professional quality** - Production-ready visual output
- **Consistent interface** - Similar API patterns across operators

## Usage

Operators are typically used through the main stage director system rather than directly. See the parent directory documentation for examples of automated visual testing workflows.

## Adding New Operators

When adding new operators:
1. Follow the naming pattern `{tool}_operator.go`
2. Integrate with the stage director system
3. Provide comprehensive error handling
4. Include usage examples and documentation
5. Maintain the professional steadicam aesthetic