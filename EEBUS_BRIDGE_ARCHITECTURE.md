# EEBus Bridge Architecture

## Overview

This document describes the architecture for bridging between the ioBroker adapter (Node.js) and the EEBus protocol implementation (Go via eebus-go library).

## Integration Strategy

After researching available options, we will use the **Go Binary with stdio JSON communication** approach:

### Why This Approach?

1. **Separation of Concerns**: EEBus protocol complexity isolated in Go binary
2. **Leverage Existing Libraries**: Use mature eebus-go implementation
3. **Simple IPC**: JSON over stdio is straightforward and well-supported
4. **Process Isolation**: Go process crashes don't crash the adapter
5. **Easy Testing**: Both components can be tested independently

### Alternative Approaches Considered

- **Native TypeScript/JavaScript Implementation**: Too complex, protocol maintenance burden
- **Node.js Native Addons**: Platform-specific builds, compilation complexity
- **HTTP/REST API**: Additional networking complexity, overkill for local communication
- **gRPC**: More overhead than needed for this use case

## Architecture Diagram

```
┌─────────────────────────────────────────────────────────────┐
│                    ioBroker.eebus Adapter                   │
│                         (Node.js)                           │
│                                                             │
│  ┌──────────────┐      ┌──────────────┐    ┌────────────┐ │
│  │   Adapter    │      │  EEBus       │    │   State    │ │
│  │   Main       │◄────►│  Bridge      │◄──►│  Manager   │ │
│  │   (main.js)  │      │  (JS)        │    │  (JS)      │ │
│  └──────────────┘      └──────┬───────┘    └────────────┘ │
│                               │                            │
│                               │ JSON over stdio            │
│                               │                            │
└───────────────────────────────┼────────────────────────────┘
                                │
                                │ spawn/stdio
                                │
┌───────────────────────────────┼────────────────────────────┐
│                               ▼                            │
│                    ┌──────────────────┐                    │
│                    │  EEBus Bridge    │                    │
│                    │  Process (Go)    │                    │
│                    └────────┬─────────┘                    │
│                             │                              │
│           ┌─────────────────┼─────────────────┐            │
│           │                 │                 │            │
│  ┌────────▼────────┐ ┌──────▼──────┐ ┌───────▼────────┐  │
│  │   eebus-go      │ │   Device    │ │   Use Case     │  │
│  │   Service       │ │   Manager   │ │   Handlers     │  │
│  │   (SHIP/SPINE)  │ │   (mDNS)    │ │   (MGCP/MPC)   │  │
│  └─────────────────┘ └─────────────┘ └────────────────┘  │
│                                                            │
│                  Go Binary (eebus-bridge)                  │
└────────────────────────────┬───────────────────────────────┘
                             │
                             │ SHIP/SPINE Protocol
                             │ (TCP/TLS)
                             │
                   ┌─────────┴──────────┐
                   │                    │
            ┌──────▼──────┐      ┌──────▼──────┐
            │  Heat Pump  │      │  Other      │
            │  (EEBus)    │      │  EEBus      │
            │             │      │  Devices    │
            └─────────────┘      └─────────────┘
```

## Communication Protocol

### Message Format

All messages between Node.js and Go are JSON objects sent over stdio:

```json
{
  "id": "unique-message-id",
  "type": "command|response|event",
  "action": "discover|connect|subscribe|getData",
  "payload": { ... },
  "timestamp": "2026-03-29T12:00:00Z"
}
```

### Message Types

1. **Command** (Node.js → Go): Request an action
2. **Response** (Go → Node.js): Reply to a command
3. **Event** (Go → Node.js): Unsolicited notification

### Message Flow Examples

#### Device Discovery

```
Node.js → Go:
{
  "id": "1",
  "type": "command",
  "action": "startDiscovery",
  "payload": {}
}

Go → Node.js:
{
  "id": "1",
  "type": "response",
  "action": "startDiscovery",
  "payload": {"status": "started"}
}

Go → Node.js (events):
{
  "type": "event",
  "action": "deviceDiscovered",
  "payload": {
    "ski": "d89c67...",
    "name": "Heatpump-Living",
    "type": "HeatPump",
    "address": "192.168.1.100:4712"
  }
}
```

#### Subscribe to Measurements

```
Node.js → Go:
{
  "id": "2",
  "type": "command",
  "action": "subscribeMeasurements",
  "payload": {
    "ski": "d89c67..."
  }
}

Go → Node.js:
{
  "id": "2",
  "type": "response",
  "action": "subscribeMeasurements",
  "payload": {"status": "subscribed"}
}

Go → Node.js (periodic events):
{
  "type": "event",
  "action": "measurementUpdate",
  "payload": {
    "ski": "d89c67...",
    "measurements": {
      "power": {"active": 3500, "unit": "W"},
      "energy": {"consumed": 15234, "unit": "Wh"},
      "voltage": {"L1": 230, "L2": 231, "L3": 229},
      "current": {"L1": 5.1, "L2": 5.0, "L3": 5.2}
    }
  }
}
```

## Component Responsibilities

### Node.js Side

#### `lib/eebusBridge.js`
- Spawn and manage Go process lifecycle
- Handle stdio communication
- Message queuing and sequencing
- Reconnection logic on process failure
- JSON parsing and validation

#### `lib/stateManager.js`
- Create and update ioBroker objects/states
- Map EEBus data to ioBroker state tree
- Handle data type conversions
- Manage state subscriptions

#### `main.js` (Adapter)
- Adapter lifecycle management
- Configuration handling
- Coordinate between bridge and state manager
- Error handling and logging
- User notifications

### Go Side

#### `cmd/eebus-bridge/main.go`
- Entry point for Go binary
- stdio handling
- JSON message parsing
- Message routing

#### `internal/bridge/bridge.go`
- Core bridge logic
- Message handling
- Command execution
- Event emission

#### `internal/eebus/service.go`
- eebus-go service initialization
- Certificate management
- SHIP connection handling
- Device discovery (mDNS)

#### `internal/eebus/devices.go`
- Device tracking
- Connection state management
- Capability querying

#### `internal/eebus/usecases.go`
- Use case implementations (MGCP, MPC)
- SPINE feature subscriptions
- Measurement data extraction
- Data transformation

## Implementation Phases

### Phase 1: Basic Infrastructure (Current)
- ✓ Project setup
- ✓ Architecture design
- ⧗ Skeleton code for Node.js bridge
- ⧗ Skeleton code for Go bridge
- ⧗ Basic stdio communication test

### Phase 2: Go Binary Development
- Go project structure setup
- eebus-go integration
- Basic device discovery
- stdio message handling
- Build and distribution strategy

### Phase 3: Node.js Integration
- Process lifecycle management
- Message protocol implementation
- Error handling and recovery
- State manager implementation

### Phase 4: Device Support
- Heat pump device support
- MGCP use case implementation
- Measurement data handling
- Real-time updates

### Phase 5: Production Readiness
- Comprehensive testing
- Performance optimization
- Documentation
- Release preparation

## Error Handling Strategy

### Go Process Failures
- Automatic restart with exponential backoff
- State recovery after reconnection
- Maximum retry limit with user notification
- Graceful degradation

### Communication Errors
- Message timeout handling
- Invalid JSON recovery
- Queue overflow protection
- Duplicate message detection

### EEBus Protocol Errors
- Device connection failures
- Certificate validation errors
- Protocol version mismatches
- Use case not supported

## Security Considerations

### Certificate Management
- Certificates stored in adapter data directory
- Private keys with restricted permissions
- Automatic certificate generation on first run
- Certificate renewal support

### Process Isolation
- Go process runs with minimal privileges
- No network access except local devices
- Sandboxed data directory

### Input Validation
- All JSON messages validated
- SKI format verification
- IP address whitelisting (optional)
- Rate limiting for commands

## Performance Considerations

### Resource Usage
- Expected Go process memory: 20-50 MB
- Message processing latency: <10ms
- Device update frequency: 1-5 seconds
- Maximum concurrent devices: 10-20

### Optimization Strategies
- Message batching for bulk updates
- Connection pooling for multiple devices
- Lazy loading of device details
- Caching of device capabilities

## Testing Strategy

### Unit Tests
- Node.js bridge message handling
- Go bridge command processing
- State manager transformations

### Integration Tests
- End-to-end message flow
- Process lifecycle
- Error recovery

### System Tests
- Real device integration
- Performance under load
- Long-running stability

## Distribution Strategy

### Go Binary
- Pre-built binaries for common platforms (Linux x64/ARM, Windows, macOS)
- Stored in `bin/` directory
- Platform detection in Node.js
- Build script for all platforms

### npm Package
- Go binaries included in package
- Automatic platform selection
- Fallback to Go compilation if binary not available
- Build instructions for unsupported platforms

## Future Enhancements

### Advanced Features
- Multiple use case support
- Device control capabilities
- Historical data export
- Device grouping

### Performance Improvements
- Binary protocol (MessagePack/Protocol Buffers)
- WebSocket communication
- Multi-process support for many devices

### Developer Experience
- Simulator/emulator for testing
- Debug mode with protocol logging
- Admin UI for protocol inspection

## References

- [eebus-go GitHub](https://github.com/enbility/eebus-go)
- [EEBus Specification](https://www.eebus.org/)
- [Node.js child_process](https://nodejs.org/api/child_process.html)
- [ioBroker Adapter Development](https://github.com/ioBroker/ioBroker/wiki/Adapter-Development-Documentation)

---

**Document Version:** 1.0
**Last Updated:** 2026-03-29
**Status:** Design Phase
