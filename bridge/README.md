# EEBus Bridge (Go)

This is the Go binary that implements the EEBus protocol communication for the ioBroker.eebus adapter.

## Architecture

```
bridge/
├── cmd/
│   └── eebus-bridge/      # Main entry point
│       └── main.go
├── internal/
│   ├── bridge/            # Bridge core logic
│   │   ├── bridge.go      # Main bridge implementation
│   │   └── handlers.go    # Command handlers
│   └── eebus/             # EEBus protocol integration
│       ├── service.go     # EEBus service wrapper
│       ├── discovery.go   # Device discovery (mDNS)
│       └── devices.go     # Device management
└── pkg/
    └── protocol/          # Message protocol definitions
        └── messages.go    # JSON message structures
```

## Communication Protocol

The bridge communicates with the Node.js adapter via stdio using newline-delimited JSON messages.

### Message Format

```json
{
  "id": "unique-id",
  "type": "command|response|event",
  "action": "action-name",
  "payload": {},
  "timestamp": "2026-03-29T12:00:00Z"
}
```

## Building

```bash
# Build for current platform
go build -o ../bin/eebus-bridge ./cmd/eebus-bridge

# Build for all platforms
./build.sh
```

## Testing

```bash
# Run tests
go test ./...

# Run with verbose output
go test -v ./...
```

## Dependencies

- `github.com/enbility/eebus-go` - EEBus protocol implementation
- `github.com/enbility/ship-go` - SHIP transport protocol
- `github.com/enbility/spine-go` - SPINE data model

## Development

The bridge can be tested independently by running it directly:

```bash
go run ./cmd/eebus-bridge
```

It will read JSON commands from stdin and write responses/events to stdout.
