![Logo](admin/eebus.png)
# ioBroker.eebus

[![NPM version](https://img.shields.io/npm/v/iobroker.eebus.svg)](https://www.npmjs.com/package/iobroker.eebus)
[![Downloads](https://img.shields.io/npm/dm/iobroker.eebus.svg)](https://www.npmjs.com/package/iobroker.eebus)
![Number of Installations](https://iobroker.live/badges/eebus-installed.svg)
![Current version in stable repository](https://iobroker.live/badges/eebus-stable.svg)

[![NPM](https://nodei.co/npm/iobroker.eebus.png?downloads=true)](https://nodei.co/npm/iobroker.eebus/)

**Tests:** ![Test and Release](https://github.com/darkv0id/ioBroker.eebus/workflows/Test%20and%20Release/badge.svg)

## ⚠️ EARLY DEVELOPMENT WARNING

**This adapter is in very early development stage and should be considered EXPERIMENTAL.**

- ❌ **Not production-ready** - expect bugs and breaking changes
- ❌ **Limited testing** - tested in controlled environments only, no real-world device testing yet
- ❌ **Missing functionality** - many EEBus features not yet implemented
- ❌ **AI-generated code** - large portions of the codebase were generated using AI without thorough code review
- ❌ **No device compatibility confirmed** - no devices have been confirmed to work with this implementation yet
- ❌ **Breaking changes likely** - API and functionality may change significantly between versions

**USE AT YOUR OWN RISK. This adapter may not work with your devices and could potentially cause issues.**

If you encounter problems, please report them via [GitHub Issues](https://github.com/darkv0id/ioBroker.eebus/issues).

---

## EEBus adapter for ioBroker

This adapter aims to connect and monitor EEBus-enabled devices (heat pumps, wallboxes, battery systems, solar inverters, etc.) in your ioBroker smart home.

### What is EEBus?

[EEBus](https://www.eebus.org/) is an open communication standard for smart energy management in buildings. It enables devices from different manufacturers to communicate using the SHIP (Smart Home IP) and SPINE (Smart Premises Interoperable Neutral-message Exchange) protocols.

Potentially compatible device types include:
- Heat pumps
- Wallboxes/EV chargers
- Battery storage systems
- Solar inverters
- Smart home energy managers

**Note:** Device compatibility is not guaranteed. This adapter implements the EEBus specification but has not been tested with real devices yet.

### Implemented Features

- ✅ Automatic device discovery via mDNS
- ✅ Manual device registration for cross-network scenarios
- ✅ TLS-encrypted SHIP protocol communication
- ✅ Certificate generation with Subject Key Identifier (SKI)
- ✅ Basic SPINE protocol implementation
- ✅ Use case support:
  - MGCP (Monitoring of Grid Connection Point)
  - MPC (Monitoring of Power Consumption)
- ✅ Multi-platform binaries (Linux, Windows, macOS)

### Missing / Planned Features

- ❌ Device control functionality (read-only currently)
- ❌ Advanced use cases (EV charging control, etc.)
- ❌ Comprehensive error handling and recovery
- ❌ Real-world device testing and compatibility verification
- ❌ User-friendly configuration interface
- ❌ Documentation for all supported data points
- ❌ IPv6 support
- ❌ Production-grade reliability and stability

---

## Installation

### Requirements

- **ioBroker** v4.0 or higher
- **Node.js** v20 or higher
- **Network:** mDNS/DNS-SD support (Avahi on Linux)
- **Supported Platforms:** Linux (x64, ARM64), Windows (x64), macOS (Intel, Apple Silicon)

### Install from npm

```bash
npm install iobroker.eebus
```

Or install from the ioBroker admin interface.

**Note:** Pre-built binaries are included, so no Go toolchain is required.

---

## Configuration

### Basic Setup

1. Install and create an adapter instance
2. Start the instance - it will:
   - Generate TLS certificates automatically
   - Start the EEBus bridge process
   - Begin mDNS device discovery on all network interfaces
3. Check adapter logs for discovered devices
4. Enable EEBus on your device (consult device manual)

### Device Discovery

#### Automatic (mDNS)

Devices on the same network should be discovered automatically:

```
eebus.0 | info | Device discovered: <Device Name> (SKI: <identifier>)
```

#### Manual Registration

For cross-network scenarios or if mDNS doesn't work:

1. Get your device's SKI (Subject Key Identifier) from device settings
2. Register manually (implementation in progress - check current code for API)

---

## State Structure

**Note:** State structure is preliminary and subject to change.

Planned structure:
```
eebus.0
├── devices
│   └── <SKI>
│       ├── info
│       │   ├── name
│       │   ├── brand
│       │   ├── model
│       │   ├── connected
│       ├── power
│       │   ├── active (W)
│       │   ├── reactive (VAr)
│       ├── energy
│       │   ├── consumed (Wh)
│       │   ├── produced (Wh)
│       ├── voltage, current, frequency...
```

Actual implementation may differ. Check adapter states in ioBroker admin.

---

## Troubleshooting

### No Devices Discovered

1. **Verify EEBus is enabled** on your device
2. **Check network connectivity** - device must be reachable
3. **Verify mDNS service** is running (Avahi on Linux)
4. **Check adapter logs** for errors
5. **Try different network configuration** - some routers may block mDNS

### Bridge Process Issues

1. **Check binary permissions:**
   ```bash
   chmod +x node_modules/iobroker.eebus/bin/eebus-bridge*
   ```
2. **Review adapter logs** for error messages
3. **Report issues** on GitHub with full logs

### General Issues

- **This is experimental software** - many things may not work
- Check GitHub Issues for known problems
- Provide detailed logs when reporting issues
- Consider waiting for stable release if you need reliability

---

## Development

### Architecture

**Hybrid Node.js + Go implementation:**

- **Node.js** (`main.js`, `lib/`): ioBroker integration, state management
- **Go** (`bridge/`): EEBus protocol implementation using eebus-go library
- **Communication:** stdio JSON protocol between Node.js and Go

### Building

```bash
# Install dependencies
npm install

# Build Go bridge for all platforms
npm run build

# Run tests
npm test
```

### Testing

**Test Coverage:**
- 76 Node.js tests (mocked, no real devices)
- 42 Go tests (unit tests only)
- **No integration tests with real devices**

The tests verify code structure and basic functionality but do not guarantee device compatibility.

### Project Structure

```
iobroker.eebus/
├── main.js                 # Adapter entry point
├── lib/                    # Node.js implementation
│   ├── eebusBridge.js     # Bridge process manager
│   ├── skiValidator.js    # SKI validation
│   ├── stateManager.js    # State management (incomplete)
│   └── *.test.js          # Tests (mocked)
├── bridge/                 # Go implementation
│   ├── cmd/eebus-bridge/  # Main binary
│   ├── internal/
│   │   ├── bridge/        # Protocol handler
│   │   └── eebus/         # EEBus service
│   └── build.sh           # Build script
└── bin/                    # Pre-built binaries
```

---

## Known Limitations

- **Device compatibility unknown** - not tested with real hardware
- **Limited protocol coverage** - only basic EEBus features implemented
- **No device control** - monitoring only, no write operations
- **Minimal error handling** - may crash on unexpected responses
- **AI-generated code** - quality and correctness not fully verified
- **Documentation incomplete** - many features undocumented
- **Cross-network setup complex** - requires advanced networking knowledge
- **No production deployment tested** - stability unknown

---

## Technical Details

### Specifications

- **SHIP 1.0.1** (Smart Home IP) - Transport layer with TLS
- **SPINE 1.3.0** (Smart Premises Interoperable Neutral-message Exchange) - Data model
- **Implemented Use Cases:**
  - MGCP 1.0 (Monitoring of Grid Connection Point) - partial
  - MPC 1.0 (Monitoring of Power Consumption) - partial

### Dependencies

**Node.js:**
- `@iobroker/adapter-core` - Adapter framework

**Go (statically compiled into binary):**
- `github.com/enbility/eebus-go` v0.7.0
- `github.com/enbility/ship-go` v0.6.0
- `github.com/enbility/spine-go` v0.7.0

---

## Contributing

Contributions are welcome! However, please note:

- This is an early-stage project with AI-generated code
- Code review and refactoring needed in many areas
- Testing with real devices urgently needed
- Documentation improvements appreciated

Please open an issue before starting major work to discuss the approach.

---

## Links

- **GitHub Repository:** https://github.com/darkv0id/ioBroker.eebus
- **Issue Tracker:** https://github.com/darkv0id/ioBroker.eebus/issues
- **EEBus Initiative:** https://www.eebus.org/
- **EEBus Specifications:** https://www.eebus.org/specifications/
- **ioBroker Forum:** https://forum.iobroker.net/

---

## Changelog

### 0.0.1 (2026-03-31)
* (Seemonster) Initial experimental release
* Basic EEBus protocol implementation
* Device discovery via mDNS
* MGCP/MPC use case support (partial)
* Multi-platform support
* ⚠️ Not tested with real devices
* ⚠️ AI-generated code, needs review
* ⚠️ Many features incomplete

---

## License

MIT License

Copyright (c) 2026 Seemonster <darkv0id@darkv0id.de>

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.

---

## Disclaimer

This project is not affiliated with or endorsed by the EEBus Initiative e.V. EEBus® is a registered trademark of EEBus Initiative e.V.

This is an experimental implementation of the EEBus specification for educational and development purposes. Device compatibility is not guaranteed. Use at your own risk.

**This adapter contains AI-generated code that has not been thoroughly reviewed or tested. It may contain bugs, security issues, or incorrect implementations of the EEBus specification.**
