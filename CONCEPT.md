# ioBroker.eebus - Project Concept

## Executive Summary

This document outlines the concept for creating an ioBroker adapter that integrates EEBus-enabled devices into the ioBroker smart home platform. The initial implementation will focus on reading power consumption data from heat pumps, with potential for future expansion to other EEBus use cases.

## 1. Technology Overview

### 1.1 EEBus Protocol

EEBus is a standardized communication protocol for energy management in buildings, designed for the Internet of Things (IoT). It enables interoperability between various energy-related devices.

**Key Components:**
- **SHIP (Smart Home IP Protocol)**: Transport layer protocol providing secure communication over IP networks, aligned with German BSI TR-03109 security standards
- **SPINE (Smart Premises Interoperable Neutral-message Exchange)**: Application layer data model defining device models, message content, and use cases
- **Use Cases**: Predefined interaction patterns for specific functionality (e.g., monitoring, control, optimization)

**Supported Devices:**
- Heat pumps
- EV charging stations (wallboxes)
- Solar inverters (PV systems)
- Battery storage systems
- Smart appliances (white goods)
- Energy management systems
- Smart meter gateways

### 1.2 ioBroker Platform

ioBroker is a popular open-source platform for home automation that integrates various smart home devices and services through adapters.

**Adapter Characteristics:**
- Written in JavaScript/TypeScript (Node.js)
- Independent processes communicating via centralized data storage
- Configuration via JSON UI or other admin frameworks
- State-based architecture with objects and states

## 2. Available EEBus Implementations

### 2.1 Existing Libraries

| Implementation | Language | License | Maturity | Repository |
|---------------|----------|---------|----------|------------|
| eebus-go | Go | Open Source | Production-ready | github.com/enbility/eebus-go |
| openeebus | C | Apache 2.0 | Development (2025) | github.com/NIBEGroup/openeebus |
| evcc EEBUS | Go | Open Source | Partial implementation | github.com/evcc-io/eebus |
| EEBUS.Net | C#/.NET | Open Source | Reference impl. | github.com/digitaltwinconsortium |
| jEEBus | Java | Open Source | Available | openmuc.org/eebus |

**Key Observation:** No native Node.js/TypeScript implementation currently exists.

### 2.2 Implementation Strategy Options

Given the absence of a Node.js/TypeScript library, there are three viable approaches:

**Option A: Native Node.js/TypeScript Implementation**
- Implement SHIP and SPINE protocols from scratch in TypeScript
- Pros: Full control, native integration, no FFI overhead
- Cons: Significant development effort, protocol complexity, maintenance burden

**Option B: Bridge to Go Implementation (eebus-go)**
- Create Node.js bindings to eebus-go using child processes or gRPC
- Pros: Leverage mature implementation, active community support
- Cons: Additional runtime dependency (Go binary), inter-process communication complexity

**Option C: Bridge to C Implementation (openeebus)**
- Create Node.js native addons using node-addon-api or node-ffi
- Pros: Performance, direct memory access
- Cons: New library (released 2025), compilation complexity, platform-specific builds

**Recommended Approach: Option B (eebus-go bridge)**
- Reasons: Production-ready implementation, active maintenance, examples available for heat pumps

## 3. EEBus Use Cases for Heat Pump Power Monitoring

### 3.1 Monitoring of Grid Connection Point (MGCP)

**Purpose:** Monitor electrical measurands at the grid connection point (where public grid connects to building)

**Capabilities:**
- Momentary power consumption/production
- Total feed-in energy
- Total consumed energy
- Current consumption/production per phase
- Voltage per phase
- Frequency monitoring

**Actor Roles:**
- Monitoring Appliance: Reads and reports measurement data
- Client/EMS: Receives and processes measurement data

### 3.2 Monitoring of Power Consumption (MPC)

**Purpose:** Device-level measurement data transfer for controllable consumers

**Target Devices:**
- Heat pumps
- EV charging points
- Other controllable loads

**Data Transfer Points:**
- Grid connection point (for DSO/ESP)
- Individual device level

### 3.3 SPINE Features & Functions

Based on EEBus specification, the relevant features for power monitoring include:

**Measurement Feature:**
- Electrical connection measurements
- Power values (active, reactive, apparent)
- Energy values (cumulative consumption/production)
- Voltage and current measurements
- Frequency data

**Device Classification Feature:**
- Device type identification
- Capability discovery

**Device Diagnosis Feature:**
- Connection status
- Heartbeat mechanism

## 4. ioBroker Adapter Architecture

### 4.1 High-Level Design

```
┌─────────────────────────────────────────────────────────┐
│                   ioBroker Core                         │
│              (Object & State Storage)                   │
└─────────────────┬───────────────────────────────────────┘
                  │
                  │ ioBroker API
                  │
┌─────────────────┴───────────────────────────────────────┐
│            ioBroker.eebus Adapter (Node.js)             │
│                                                          │
│  ┌──────────────┐    ┌──────────────┐    ┌───────────┐│
│  │  Adapter     │    │   EEBus      │    │  State    ││
│  │  Core Logic  │◄──►│   Bridge     │◄──►│  Manager  ││
│  └──────────────┘    └──────────────┘    └───────────┘│
│         │                    │                          │
│         │                    │                          │
│  ┌──────▼────────┐    ┌──────▼────────┐               │
│  │  Config       │    │  EEBus Go     │               │
│  │  Manager      │    │  Process      │               │
│  └───────────────┘    └───────────────┘               │
└─────────────────────────────┬───────────────────────────┘
                              │
                              │ SHIP/SPINE Protocol
                              │
                    ┌─────────┴────────────┐
                    │                      │
              ┌─────▼─────┐          ┌────▼──────┐
              │ Heat Pump │          │  Other    │
              │  Device   │          │  EEBus    │
              │  (EEBus)  │          │  Devices  │
              └───────────┘          └───────────┘
```

### 4.2 Component Responsibilities

**Adapter Core Logic:**
- Lifecycle management (start, stop, restart)
- Configuration validation
- Error handling and logging
- ioBroker state synchronization

**EEBus Bridge:**
- Spawn and manage eebus-go process
- Protocol translation (Go ↔ Node.js)
- Message queuing and buffering
- Connection state management

**State Manager:**
- Create and update ioBroker objects
- Map EEBus data to ioBroker states
- Handle state subscriptions
- Data type conversions

**Config Manager:**
- User settings (discovery mode, device selection)
- Connection parameters (certificates, SKI)
- Update intervals and thresholds

**EEBus Go Process:**
- SHIP protocol handling
- SPINE data model implementation
- Device discovery via mDNS
- Use case implementation (MGCP, MPC)

### 4.3 Data Flow

**Discovery Phase:**
1. User enables adapter with discovery mode
2. EEBus Go process starts mDNS service discovery
3. Discovered devices reported to adapter
4. Adapter creates device objects in ioBroker
5. User selects devices to monitor

**Operational Phase:**
1. EEBus Go establishes SHIP connection to device
2. Device announces capabilities via SPINE
3. Adapter subscribes to measurement features
4. Device sends measurement updates
5. Bridge forwards data to State Manager
6. State Manager updates ioBroker states
7. Users/scripts can read current values

## 5. ioBroker Object Structure

### 5.1 Proposed Object Tree

```
eebus.0
├── info
│   ├── connection (boolean)
│   └── discovery (boolean)
│
├── devices
│   ├── <device-ski-1>
│   │   ├── info
│   │   │   ├── name (string)
│   │   │   ├── type (string)
│   │   │   ├── manufacturer (string)
│   │   │   ├── model (string)
│   │   │   ├── serial (string)
│   │   │   └── connected (boolean)
│   │   │
│   │   ├── measurements
│   │   │   ├── power
│   │   │   │   ├── active (number, W)
│   │   │   │   ├── reactive (number, VAr)
│   │   │   │   └── apparent (number, VA)
│   │   │   │
│   │   │   ├── energy
│   │   │   │   ├── consumed (number, Wh)
│   │   │   │   └── produced (number, Wh)
│   │   │   │
│   │   │   ├── voltage
│   │   │   │   ├── L1 (number, V)
│   │   │   │   ├── L2 (number, V)
│   │   │   │   └── L3 (number, V)
│   │   │   │
│   │   │   ├── current
│   │   │   │   ├── L1 (number, A)
│   │   │   │   ├── L2 (number, A)
│   │   │   │   └── L3 (number, A)
│   │   │   │
│   │   │   └── frequency (number, Hz)
│   │   │
│   │   └── usecase
│   │       ├── mgcp (boolean)
│   │       └── mpc (boolean)
│   │
│   └── <device-ski-2>
│       └── ...
```

### 5.2 State Definitions

Key state attributes:
- **type**: 'number', 'string', 'boolean'
- **role**: e.g., 'value.power', 'value.energy', 'value.voltage'
- **read**: true (all measurements read-only initially)
- **write**: false (no control in first version)
- **unit**: 'W', 'Wh', 'V', 'A', 'Hz'

## 6. Implementation Steps

### Phase 1: Project Setup (Week 1-2)

**Step 1.1: Initialize Adapter Project**
- Run `npx @iobroker/create-adapter`
- Select TypeScript template
- Configure JSON UI for admin interface
- Set up Git repository and CI/CD

**Step 1.2: Set Up Development Environment**
- Install ioBroker development server
- Configure dev-server for local testing
- Set up debugging configuration
- Create basic project structure

**Step 1.3: Install EEBus Dependencies**
- Download/compile eebus-go binary
- Create wrapper scripts for process management
- Set up IPC mechanism (stdio/gRPC)
- Test basic process spawning

**Deliverables:**
- Working adapter skeleton
- Basic configuration UI
- EEBus Go process integration
- Development documentation

### Phase 2: EEBus Integration (Week 3-5)

**Step 2.1: Implement Device Discovery**
- Integrate mDNS discovery from eebus-go
- Parse discovered device information
- Create device objects in ioBroker
- Implement device selection UI

**Step 2.2: Establish SHIP Connection**
- Implement SKI (Subject Key Identifier) handling
- Certificate management (generate/import)
- Pairing mechanism with devices
- Connection state monitoring

**Step 2.3: Implement SPINE Communication**
- Parse device capability announcements
- Subscribe to measurement features
- Handle SPINE data model entities
- Implement error handling

**Deliverables:**
- Working device discovery
- SHIP connection establishment
- Basic SPINE message handling
- Device pairing functionality

### Phase 3: Heat Pump Monitoring (Week 6-8)

**Step 3.1: Implement MGCP Use Case**
- Subscribe to measurement features
- Parse power/energy measurements
- Extract voltage/current/frequency data
- Handle multi-phase measurements

**Step 3.2: Create State Management**
- Define ioBroker state structure
- Implement state creation/update logic
- Add data type conversions
- Implement update rate limiting

**Step 3.3: Data Validation & Processing**
- Validate measurement values
- Handle unit conversions
- Implement value caching
- Add timestamp handling

**Deliverables:**
- Working power consumption monitoring
- Complete state tree for heat pump
- Data validation and error handling
- Real-time updates in ioBroker

### Phase 4: User Interface & Configuration (Week 9-10)

**Step 4.1: Admin UI Development**
- Create device discovery interface
- Implement device selection controls
- Add connection status display
- Create configuration forms

**Step 4.2: Dashboard Widgets (Optional)**
- Create VIS widget for power display
- Add historical data visualization
- Implement real-time gauges
- Create energy consumption charts

**Step 4.3: Documentation**
- Write user manual
- Create installation guide
- Document configuration options
- Add troubleshooting section

**Deliverables:**
- Complete admin interface
- Optional VIS widgets
- Comprehensive documentation
- Configuration examples

### Phase 5: Testing & Release (Week 11-12)

**Step 5.1: Testing**
- Unit tests for core functionality
- Integration tests with real devices
- Performance testing
- Security review

**Step 5.2: Release Preparation**
- Package adapter for npm
- Submit to ioBroker repository
- Create release notes
- Set up issue tracking

**Step 5.3: Community Release**
- Beta testing with community
- Bug fixes and improvements
- Documentation updates
- Official release to stable

**Deliverables:**
- Tested and stable adapter
- Published to ioBroker repository
- Complete documentation
- Community support setup

## 7. Technical Considerations

### 7.1 Security

**Certificate Management:**
- Generate self-signed certificates for SHIP
- Secure storage of private keys
- Certificate validation
- Secure pairing mechanism

**Network Security:**
- mDNS only on local network
- Encrypted SHIP communication
- No internet exposure required
- Input validation for all data

### 7.2 Performance

**Resource Usage:**
- Go process memory footprint: ~20-50 MB
- Node.js adapter overhead: ~30-50 MB
- CPU usage: minimal (<5% on modern hardware)
- Network traffic: <1 KB/s per device

**Optimization Strategies:**
- Connection pooling
- Message batching
- Rate limiting for state updates
- Efficient IPC mechanism

### 7.3 Reliability

**Error Handling:**
- Graceful process crash recovery
- Automatic reconnection logic
- Connection timeout handling
- State synchronization after disconnect

**Monitoring:**
- Health check mechanisms
- Connection state tracking
- Error logging and alerting
- Performance metrics

### 7.4 Compatibility

**EEBus Protocol Versions:**
- Target SPINE 1.3.0 or later
- SHIP 1.0.0 compatibility
- Use case versions to support:
  - MGCP 1.0.0
  - MPC 1.0.0

**Device Support:**
- Heat pumps (primary focus)
- Future: EV chargers, PV inverters, batteries
- Manufacturer independence
- Standard compliance verification

## 8. Future Enhancements

### 8.1 Additional Use Cases

**Phase 2 Features:**
- Load management and control
- Dynamic power limitation
- EV charging coordination
- Demand response

**Phase 3 Features:**
- Incentive table (dynamic tariffs)
- Production forecasting
- Battery optimization
- Peer-to-peer energy trading

### 8.2 Advanced Features

**Control Capabilities:**
- Heat pump on/off control
- Temperature setpoint adjustment
- Operating mode selection
- Power limitation

**Integration:**
- Integration with ioBroker.energy
- Connection to external APIs
- MQTT bridge
- REST API endpoints

**Analytics:**
- Historical data analysis
- Energy cost calculation
- Efficiency monitoring
- Trend analysis

## 9. Resource Requirements

### 9.1 Development Resources

**Knowledge Required:**
- TypeScript/Node.js development
- ioBroker adapter API
- EEBus protocol understanding
- Network programming (mDNS, TLS)
- Go language basics (for integration)

**Tools:**
- Node.js 18+ with npm
- Go 1.21+ toolchain
- ioBroker development environment
- Git version control
- VS Code or similar IDE

**Hardware:**
- Development machine (Linux/Mac/Windows)
- EEBus-enabled heat pump or simulator
- Network infrastructure for testing

### 9.2 Testing Resources

**Test Devices:**
- At least one EEBus-enabled heat pump
- Alternative: EEBus simulator/emulator
- Network infrastructure (switch, router)

**Test Environment:**
- ioBroker test instance
- Isolated network segment (optional)
- Monitoring tools (Wireshark, etc.)

## 10. Risk Assessment

| Risk | Probability | Impact | Mitigation |
|------|------------|--------|------------|
| EEBus protocol complexity | Medium | High | Use existing eebus-go implementation |
| Device compatibility issues | Medium | Medium | Test with multiple manufacturers |
| Performance bottlenecks | Low | Medium | Implement efficient IPC and caching |
| Security vulnerabilities | Low | High | Follow BSI guidelines, regular audits |
| Go/Node.js integration issues | Medium | Medium | Use proven IPC mechanisms |
| Limited documentation | High | Medium | Rely on code examples, community support |
| Device availability for testing | Medium | Medium | Use simulators, community beta testing |

## 11. Success Criteria

The project will be considered successful when:

1. **Functional Requirements:**
   - Automatic discovery of EEBus devices on local network
   - Successful pairing with heat pump devices
   - Real-time power consumption monitoring (±5 second latency)
   - All measurement data correctly displayed in ioBroker
   - Stable operation for 7+ days without restart

2. **Quality Requirements:**
   - Code coverage >80% for critical functions
   - Zero critical security vulnerabilities
   - Memory usage <100 MB total
   - CPU usage <5% average
   - Response time <1 second for UI operations

3. **Documentation Requirements:**
   - Complete installation guide
   - User manual with screenshots
   - API documentation for developers
   - Troubleshooting guide

4. **Community Acceptance:**
   - Published in ioBroker repository
   - Positive reviews from beta testers
   - Active support and maintenance plan
   - At least 10 successful installations

## 12. Timeline Summary

| Phase | Duration | Milestone |
|-------|----------|-----------|
| Phase 1: Project Setup | 2 weeks | Adapter skeleton ready |
| Phase 2: EEBus Integration | 3 weeks | Device discovery working |
| Phase 3: Heat Pump Monitoring | 3 weeks | Power data visible in ioBroker |
| Phase 4: UI & Documentation | 2 weeks | Complete user experience |
| Phase 5: Testing & Release | 2 weeks | Published to stable repository |
| **Total** | **12 weeks** | **Production-ready adapter** |

## 13. Conclusion

Creating an ioBroker adapter for EEBus devices is a technically feasible project that will provide significant value to the smart home community. By leveraging the existing eebus-go implementation and following the ioBroker adapter development best practices, we can deliver a robust solution for integrating heat pumps and other energy devices.

The modular architecture allows for incremental development, starting with read-only power monitoring and expanding to control features and additional use cases in future versions. The use of industry-standard protocols and security practices ensures compatibility and safety.

With proper planning, testing, and community engagement, this adapter can become a valuable addition to the ioBroker ecosystem and contribute to more efficient energy management in smart homes.

---

**Document Version:** 1.0
**Last Updated:** 2026-03-29
**Author:** Project Planning Phase
**Status:** Initial Concept
