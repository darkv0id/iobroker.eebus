package eebus

import (
	"crypto/ecdsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"time"
	"unsafe"

	"github.com/enbility/eebus-go/api"
	"github.com/enbility/eebus-go/service"
	"github.com/enbility/eebus-go/usecases/ma/mgcp"
	"github.com/enbility/eebus-go/usecases/ma/mpc"
	shipapi "github.com/enbility/ship-go/api"
	shipcert "github.com/enbility/ship-go/cert"
	shipmodel "github.com/enbility/ship-go/model"
	spineapi "github.com/enbility/spine-go/api"
	"github.com/enbility/spine-go/model"
)

// Service wraps the EEBus service and provides integration with our bridge
type Service struct {
	service        api.ServiceInterface
	mgcpUC         *mgcp.MGCP
	mpcUC          *mpc.MPC
	config         *Config
	devices        map[string]*DeviceInfo
	devicesMux     sync.RWMutex
	pairingDevices map[string]bool // SKIs that are waiting for/approved for pairing
	pairingMux     sync.RWMutex
	eventCB        EventCallback
}

// Config holds configuration for the EEBus service
type Config struct {
	// Device identification
	VendorCode   string
	BrandName    string
	DeviceModel  string
	SerialNumber string

	// Network configuration
	Port          int
	Interfaces    []string
	CertPath      string
	AutoAcceptNew bool // Auto-accept pairing requests

	// Timeouts
	HeartbeatTimeout time.Duration
}

// DeviceInfo represents a discovered EEBus device
type DeviceInfo struct {
	SKI          string
	Name         string
	BrandName    string
	DeviceModel  string
	SerialNumber string
	DeviceType   string
	Connected    bool
	Entity       spineapi.EntityRemoteInterface
}

// EventCallback is called when EEBus events occur
type EventCallback func(event Event)

// Event represents an EEBus event
type Event struct {
	Type    string
	SKI     string
	Device  *DeviceInfo
	Payload map[string]interface{}
}

// NewService creates a new EEBus service instance
func NewService(config *Config, eventCB EventCallback) (*Service, error) {
	if config == nil {
		return nil, fmt.Errorf("config is required")
	}

	// Set defaults
	if config.Port == 0 {
		config.Port = 4712 // Default EEBus port
	}
	if config.HeartbeatTimeout == 0 {
		config.HeartbeatTimeout = 4 * time.Second
	}

	s := &Service{
		config:         config,
		devices:        make(map[string]*DeviceInfo),
		pairingDevices: make(map[string]bool),
		eventCB:        eventCB,
	}

	return s, nil
}

// Start initializes and starts the EEBus service
func (s *Service) Start() error {
	log.Println("Initializing EEBus service...")

	// Load or create certificate
	cert, err := s.loadOrCreateCertificate()
	if err != nil {
		return fmt.Errorf("failed to get certificate: %w", err)
	}

	// Create service configuration
	configuration, err := api.NewConfiguration(
		s.config.VendorCode,
		s.config.BrandName,
		s.config.DeviceModel,
		s.config.SerialNumber,
		model.DeviceTypeTypeEnergyManagementSystem,
		[]model.EntityTypeType{model.EntityTypeTypeCEM},
		s.config.Port,
		cert,
		s.config.HeartbeatTimeout,
	)
	if err != nil {
		return fmt.Errorf("failed to create configuration: %w", err)
	}

	// Set network interfaces
	// Only set interfaces if explicitly configured
	// If not configured, leave empty to use all interfaces (eebus-go default behavior)
	if len(s.config.Interfaces) > 0 {
		log.Printf("Using configured network interfaces: %v", s.config.Interfaces)
		configuration.SetInterfaces(s.config.Interfaces)
	} else {
		// Detect interfaces for logging purposes
		detectedInterfaces, err := s.detectNetworkInterfaces()
		if err != nil {
			log.Printf("Warning: Failed to detect network interfaces: %v", err)
		} else if len(detectedInterfaces) > 0 {
			log.Printf("Detected network interfaces, using all: %v", detectedInterfaces)
		}
		// Don't call SetInterfaces() - let eebus-go use all interfaces by default
	}

	// Create the EEBus service
	s.service = service.NewService(configuration, s)

	// Set auto-accept if configured
	if s.config.AutoAcceptNew {
		s.service.SetAutoAccept(true)
	}
	s.service.SetLogging(s)

	// Start the service
	log.Println("Starting EEBus service...")
	if err := s.service.Setup(); err != nil {
		return fmt.Errorf("failed to setup service: %w", err)
	}

	// After setup, we can access LocalDevice and initialize use cases
	localEntity := s.service.LocalDevice().EntityForType(model.EntityTypeTypeCEM)
	if localEntity == nil {
		return fmt.Errorf("failed to get local entity for CEM")
	}

	// Initialize use cases
	s.mgcpUC = mgcp.NewMGCP(localEntity, s.handleEntityEvent)
	s.mpcUC = mpc.NewMPC(localEntity, s.handleEntityEvent)

	// Add use cases to service
	s.service.AddUseCase(s.mgcpUC)
	s.service.AddUseCase(s.mpcUC)

	// Start doesn't return an error - any runtime errors come through callbacks
	s.service.Start()

	log.Printf("EEBus service started on port %d", s.config.Port)
	return nil
}

// Stop shuts down the EEBus service
func (s *Service) Stop() {
	if s.service != nil {
		log.Println("Stopping EEBus service...")
		s.service.Shutdown()
	}
}

// GetDevices returns all discovered devices
func (s *Service) GetDevices() []*DeviceInfo {
	s.devicesMux.RLock()
	defer s.devicesMux.RUnlock()

	devices := make([]*DeviceInfo, 0, len(s.devices))
	for _, device := range s.devices {
		devices = append(devices, device)
	}
	return devices
}

// GetDevice returns information about a specific device by SKI
func (s *Service) GetDevice(ski string) (*DeviceInfo, bool) {
	s.devicesMux.RLock()
	defer s.devicesMux.RUnlock()
	device, ok := s.devices[ski]
	return device, ok
}

// RegisterDevice manually registers a device with its SKI, IP address, and port
// This bypasses mDNS discovery and allows direct connection
func (s *Service) RegisterDevice(ski, ip string, port int) error {
	if s.service == nil {
		return fmt.Errorf("service not initialized")
	}

	log.Printf("Manually registering device - SKI: %s, IP: %s, Port: %d", ski, ip, port)

	// Parse and validate IP address
	ipAddr := net.ParseIP(ip)
	if ipAddr == nil {
		return fmt.Errorf("invalid IP address: %s", ip)
	}

	// Add device to pairing list so AllowWaitingForTrust returns true
	s.pairingMux.Lock()
	s.pairingDevices[ski] = true
	s.pairingMux.Unlock()
	log.Printf("Added device to pairing list: SKI=%s", ski)

	// Mark the device as trusted/paired in the service
	s.service.RegisterRemoteSKI(ski)

	// Access the private connectionsHub field using reflection
	// This is necessary because eebus-go doesn't expose the hub's ReportMdnsEntries method
	serviceValue := reflect.ValueOf(s.service)
	if serviceValue.Kind() == reflect.Ptr {
		serviceValue = serviceValue.Elem()
	}

	// Get the connectionsHub field
	hubField := serviceValue.FieldByName("connectionsHub")
	if !hubField.IsValid() {
		return fmt.Errorf("failed to access connectionsHub field")
	}

	// Make the unexported field accessible
	hubField = reflect.NewAt(hubField.Type(), unsafe.Pointer(hubField.UnsafeAddr())).Elem()

	// Get the interface value from the field
	hubInterface := hubField.Interface()

	// Type assert to MdnsReportInterface
	mdnsReporter, ok := hubInterface.(shipapi.MdnsReportInterface)
	if !ok {
		return fmt.Errorf("connectionsHub does not implement MdnsReportInterface")
	}

	// Create a manual mDNS entry
	entry := &shipapi.MdnsEntry{
		Name:       "Manual-" + ski[:8], // Short name for manual entry
		Ski:        ski,
		Identifier: "manual-device",
		Path:       "/ship/",           // Standard SHIP path
		Register:   false,              // Not registered via mDNS
		Brand:      "Unknown",
		Type:       "Unknown",
		Model:      "Manual Registration",
		Host:       ip,
		Port:       port,
		Addresses:  []net.IP{ipAddr},
	}

	// Report this entry to the hub, which will trigger connection attempt
	entries := map[string]*shipapi.MdnsEntry{
		ski: entry,
	}
	mdnsReporter.ReportMdnsEntries(entries, true)

	log.Printf("Device registered successfully - SKI: %s, will attempt connection to %s:%d", ski, ip, port)
	return nil
}

// detectNetworkInterfaces returns all available non-loopback network interfaces
func (s *Service) detectNetworkInterfaces() ([]string, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("failed to list network interfaces: %w", err)
	}

	var result []string
	for _, iface := range interfaces {
		// Skip loopback and interfaces that are down
		if iface.Flags&net.FlagLoopback != 0 || iface.Flags&net.FlagUp == 0 {
			continue
		}

		// Check if interface has IP addresses
		addrs, err := iface.Addrs()
		if err != nil {
			log.Printf("Warning: Failed to get addresses for interface %s: %v", iface.Name, err)
			continue
		}

		// Only include interfaces with at least one IP address
		if len(addrs) > 0 {
			result = append(result, iface.Name)
		}
	}

	return result, nil
}

// loadOrCreateCertificate loads an existing certificate or creates a new one
func (s *Service) loadOrCreateCertificate() (tls.Certificate, error) {
	certPath := s.config.CertPath
	if certPath == "" {
		certPath = "cert"
	}

	certFile := filepath.Join(certPath, "cert.pem")
	keyFile := filepath.Join(certPath, "key.pem")

	// Try to load existing certificate
	if _, certErr := os.Stat(certFile); certErr == nil {
		if _, keyErr := os.Stat(keyFile); keyErr == nil {
			log.Printf("Loading existing certificate from %s...", certPath)
			cert, err := tls.LoadX509KeyPair(certFile, keyFile)
			if err != nil {
				log.Printf("WARNING: Certificate files exist but failed to load: %v", err)
				log.Printf("WARNING: This may indicate corrupted certificate files")
				log.Printf("WARNING: Attempting to create new certificate...")
			} else {
				log.Printf("Successfully loaded existing certificate")
				return cert, nil
			}
		} else {
			log.Printf("Certificate found but key file missing, creating new pair...")
		}
	} else {
		log.Printf("No existing certificate found at %s, creating new one...", certPath)
	}

	// Create new certificate using ship-go helper
	log.Println("Creating new certificate...")

	// Create cert directory if it doesn't exist
	if err := os.MkdirAll(certPath, 0755); err != nil {
		return tls.Certificate{}, fmt.Errorf("failed to create cert directory: %w", err)
	}

	// Create certificate with proper SKI using ship-go
	// CommonName format: deviceModel-serialNumber
	cert, err := shipcert.CreateCertificate(
		s.config.BrandName,     // Organizational Unit
		s.config.BrandName,     // Organization
		"DE",                   // Country
		fmt.Sprintf("%s-%s", s.config.DeviceModel, s.config.SerialNumber), // Common Name
	)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("failed to create certificate: %w", err)
	}

	// Save certificate
	certOut, err := os.Create(certFile)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("failed to create cert file: %w", err)
	}
	defer certOut.Close()

	if err := pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: cert.Certificate[0]}); err != nil {
		return tls.Certificate{}, fmt.Errorf("failed to write cert: %w", err)
	}

	// Save private key
	keyOut, err := os.Create(keyFile)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("failed to create key file: %w", err)
	}
	defer keyOut.Close()

	privKey, ok := cert.PrivateKey.(*ecdsa.PrivateKey)
	if !ok {
		return tls.Certificate{}, fmt.Errorf("certificate private key is not ECDSA")
	}

	privBytes, err := x509.MarshalECPrivateKey(privKey)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("failed to marshal private key: %w", err)
	}

	if err := pem.Encode(keyOut, &pem.Block{Type: "EC PRIVATE KEY", Bytes: privBytes}); err != nil {
		return tls.Certificate{}, fmt.Errorf("failed to write key: %w", err)
	}

	log.Printf("Certificate created and saved to %s", certPath)
	return cert, nil
}

// handleEntityEvent is called when entity events occur (from use cases)
func (s *Service) handleEntityEvent(ski string, device spineapi.DeviceRemoteInterface, entity spineapi.EntityRemoteInterface, event api.EventType) {
	log.Printf("Entity event: SKI=%s, Event=%v", ski, event)

	// Update device info
	s.updateDeviceInfo(ski, device, entity)

	// Handle specific events based on use case
	// Events like DataUpdatePower, DataUpdateEnergyConsumed, etc.
	s.handleUseCaseEvent(ski, entity, event)
}

// updateDeviceInfo updates the cached device information
func (s *Service) updateDeviceInfo(ski string, device spineapi.DeviceRemoteInterface, entity spineapi.EntityRemoteInterface) {
	s.devicesMux.Lock()
	defer s.devicesMux.Unlock()

	if _, exists := s.devices[ski]; !exists {
		// New device discovered
		deviceInfo := &DeviceInfo{
			SKI:       ski,
			Connected: true,
			Entity:    entity,
		}

		// Get device classification info if available
		if device != nil {
			if address := device.Address(); address != nil {
				deviceInfo.Name = string(*address)
			}
		}

		s.devices[ski] = deviceInfo
		log.Printf("New device discovered: SKI=%s", ski)

		// Notify via callback
		if s.eventCB != nil {
			s.eventCB(Event{
				Type:   "deviceDiscovered",
				SKI:    ski,
				Device: deviceInfo,
			})
		}
	}
}

// handleUseCaseEvent handles specific use case events and extracts data
func (s *Service) handleUseCaseEvent(ski string, entity spineapi.EntityRemoteInterface, event api.EventType) {
	switch event {
	case mgcp.DataUpdatePower:
		s.handlePowerUpdate(ski, entity, "mgcp")
	case mgcp.DataUpdateEnergyConsumed:
		s.handleEnergyUpdate(ski, entity, "mgcp")
	case mgcp.DataUpdateCurrentPerPhase:
		s.handleCurrentUpdate(ski, entity, "mgcp")
	case mgcp.DataUpdateVoltagePerPhase:
		s.handleVoltageUpdate(ski, entity, "mgcp")
	case mgcp.DataUpdateFrequency:
		s.handleFrequencyUpdate(ski, entity, "mgcp")
	case mpc.DataUpdatePower:
		s.handlePowerUpdate(ski, entity, "mpc")
	case mpc.DataUpdateEnergyConsumed:
		s.handleEnergyUpdate(ski, entity, "mpc")
	}
}

// handlePowerUpdate extracts and sends power measurement updates
func (s *Service) handlePowerUpdate(ski string, entity spineapi.EntityRemoteInterface, usecase string) {
	var power float64
	var err error

	if usecase == "mgcp" && s.mgcpUC != nil {
		power, err = s.mgcpUC.Power(entity)
	} else if usecase == "mpc" && s.mpcUC != nil {
		power, err = s.mpcUC.Power(entity)
	}

	if err != nil {
		log.Printf("Failed to get power for SKI=%s: %v", ski, err)
		return
	}

	log.Printf("Power update: SKI=%s, Power=%.2fW", ski, power)

	if s.eventCB != nil {
		s.eventCB(Event{
			Type: "measurementUpdate",
			SKI:  ski,
			Payload: map[string]interface{}{
				"type":    "power",
				"usecase": usecase,
				"value":   power,
				"unit":    "W",
			},
		})
	}
}

// handleEnergyUpdate extracts and sends energy measurement updates
func (s *Service) handleEnergyUpdate(ski string, entity spineapi.EntityRemoteInterface, usecase string) {
	var energy float64
	var err error

	if usecase == "mgcp" && s.mgcpUC != nil {
		energy, err = s.mgcpUC.EnergyConsumed(entity)
	} else if usecase == "mpc" && s.mpcUC != nil {
		energy, err = s.mpcUC.EnergyConsumed(entity)
	}

	if err != nil {
		log.Printf("Failed to get energy for SKI=%s: %v", ski, err)
		return
	}

	log.Printf("Energy update: SKI=%s, Energy=%.2fWh", ski, energy)

	if s.eventCB != nil {
		s.eventCB(Event{
			Type: "measurementUpdate",
			SKI:  ski,
			Payload: map[string]interface{}{
				"type":    "energy",
				"usecase": usecase,
				"value":   energy,
				"unit":    "Wh",
			},
		})
	}
}

// handleCurrentUpdate extracts and sends current measurement updates
func (s *Service) handleCurrentUpdate(ski string, entity spineapi.EntityRemoteInterface, usecase string) {
	var currents []float64
	var err error

	if usecase == "mgcp" && s.mgcpUC != nil {
		currents, err = s.mgcpUC.CurrentPerPhase(entity)
	} else if usecase == "mpc" && s.mpcUC != nil {
		currents, err = s.mpcUC.CurrentPerPhase(entity)
	}

	if err != nil {
		log.Printf("Failed to get currents for SKI=%s: %v", ski, err)
		return
	}

	log.Printf("Current update: SKI=%s, Currents=%v", ski, currents)

	if s.eventCB != nil {
		s.eventCB(Event{
			Type: "measurementUpdate",
			SKI:  ski,
			Payload: map[string]interface{}{
				"type":    "current",
				"usecase": usecase,
				"values":  currents,
				"unit":    "A",
			},
		})
	}
}

// handleVoltageUpdate extracts and sends voltage measurement updates
func (s *Service) handleVoltageUpdate(ski string, entity spineapi.EntityRemoteInterface, usecase string) {
	var voltages []float64
	var err error

	if usecase == "mgcp" && s.mgcpUC != nil {
		voltages, err = s.mgcpUC.VoltagePerPhase(entity)
	} else if usecase == "mpc" && s.mpcUC != nil {
		voltages, err = s.mpcUC.VoltagePerPhase(entity)
	}

	if err != nil {
		log.Printf("Failed to get voltages for SKI=%s: %v", ski, err)
		return
	}

	log.Printf("Voltage update: SKI=%s, Voltages=%v", ski, voltages)

	if s.eventCB != nil {
		s.eventCB(Event{
			Type: "measurementUpdate",
			SKI:  ski,
			Payload: map[string]interface{}{
				"type":    "voltage",
				"usecase": usecase,
				"values":  voltages,
				"unit":    "V",
			},
		})
	}
}

// handleFrequencyUpdate extracts and sends frequency measurement updates
func (s *Service) handleFrequencyUpdate(ski string, entity spineapi.EntityRemoteInterface, usecase string) {
	var frequency float64
	var err error

	if usecase == "mgcp" && s.mgcpUC != nil {
		frequency, err = s.mgcpUC.Frequency(entity)
	} else if usecase == "mpc" && s.mpcUC != nil {
		frequency, err = s.mpcUC.Frequency(entity)
	}

	if err != nil {
		log.Printf("Failed to get frequency for SKI=%s: %v", ski, err)
		return
	}

	log.Printf("Frequency update: SKI=%s, Frequency=%.2fHz", ski, frequency)

	if s.eventCB != nil {
		s.eventCB(Event{
			Type: "measurementUpdate",
			SKI:  ski,
			Payload: map[string]interface{}{
				"type":    "frequency",
				"usecase": usecase,
				"value":   frequency,
				"unit":    "Hz",
			},
		})
	}
}

// ServiceReaderInterface implementation

// RemoteSKIConnected is called when a remote SKI connects
func (s *Service) RemoteSKIConnected(service api.ServiceInterface, ski string) {
	log.Printf("========== REMOTE SKI CONNECTED ==========")
	log.Printf("SKI: %s", ski)
	log.Printf("Service: %v", service)

	s.devicesMux.Lock()
	deviceExists := false
	if device, exists := s.devices[ski]; exists {
		deviceExists = true
		device.Connected = true
		log.Printf("Device found in devices map, marking as connected")
		log.Printf("Device info: Name=%s, Brand=%s, Model=%s", device.Name, device.BrandName, device.DeviceModel)
	} else {
		log.Printf("WARNING: Device %s not found in devices map!", ski)
	}
	s.devicesMux.Unlock()

	log.Printf("Device exists in map: %v", deviceExists)
	log.Printf("Emitting deviceConnected event for SKI: %s", ski)

	if s.eventCB != nil {
		s.eventCB(Event{
			Type: "deviceConnected",
			SKI:  ski,
		})
		log.Printf("deviceConnected event emitted successfully")
	} else {
		log.Printf("WARNING: eventCB is nil, cannot emit deviceConnected event!")
	}
	log.Printf("==========================================")
}

// RemoteSKIDisconnected is called when a remote SKI disconnects
func (s *Service) RemoteSKIDisconnected(service api.ServiceInterface, ski string) {
	log.Printf("Remote SKI disconnected: %s", ski)

	// Only emit disconnect event if device was previously connected
	// This avoids emitting disconnects during the pairing handshake
	wasConnected := false
	s.devicesMux.Lock()
	if device, exists := s.devices[ski]; exists {
		wasConnected = device.Connected
		device.Connected = false
	}
	s.devicesMux.Unlock()

	// Only emit event if the device was actually connected before
	if wasConnected && s.eventCB != nil {
		s.eventCB(Event{
			Type: "deviceDisconnected",
			SKI:  ski,
		})
	} else {
		log.Printf("Skipping disconnect event for %s (was not connected)", ski)
	}
}

// VisibleRemoteServicesUpdated is called when the list of visible services changes
func (s *Service) VisibleRemoteServicesUpdated(service api.ServiceInterface, entries []shipapi.RemoteService) {
	log.Printf("Visible remote services updated: %d services", len(entries))

	for _, entry := range entries {
		log.Printf("  - Service: SKI=%s, Name=%s", entry.Ski, entry.Name)
	}
}

// ServiceShipIDUpdate is called when the service's SHIP ID is updated
func (s *Service) ServiceShipIDUpdate(ski string, shipdID string) {
	log.Printf("Service SHIP ID update: SKI=%s, SHIP ID=%s", ski, shipdID)
}

// connectionStateToString converts ConnectionState enum to string name
func connectionStateToString(state shipapi.ConnectionState) string {
	switch state {
	case shipapi.ConnectionStateNone:
		return "none"
	case shipapi.ConnectionStateQueued:
		return "queued"
	case shipapi.ConnectionStateInitiated:
		return "initiated"
	case shipapi.ConnectionStateReceivedPairingRequest:
		return "waiting_for_approval"
	case shipapi.ConnectionStateInProgress:
		return "in_progress"
	case shipapi.ConnectionStateTrusted:
		return "trusted"
	case shipapi.ConnectionStatePin:
		return "pin"
	case shipapi.ConnectionStateCompleted:
		return "approved"
	case shipapi.ConnectionStateRemoteDeniedTrust:
		return "denied"
	case shipapi.ConnectionStateError:
		return "error"
	default:
		return "unknown"
	}
}

// ServicePairingDetailUpdate is called when pairing details are updated
func (s *Service) ServicePairingDetailUpdate(ski string, detail *shipapi.ConnectionStateDetail) {
	state := "unknown"
	if detail != nil {
		state = connectionStateToString(detail.State())
		log.Printf("========== PAIRING STATE UPDATE ==========")
		log.Printf("SKI: %s", ski)
		log.Printf("State: %s (raw: %v)", state, detail.State())
		log.Printf("Detail: %+v", detail)
	} else {
		log.Printf("========== PAIRING STATE UPDATE (nil detail) ==========")
		log.Printf("SKI: %s, State: unknown (detail is nil)", ski)
	}

	// Emit pairing state event
	if s.eventCB != nil {
		s.eventCB(Event{
			Type: "pairingStateUpdate",
			SKI:  ski,
			Payload: map[string]interface{}{
				"state": state,
			},
		})
		log.Printf("Pairing state event emitted successfully")
	} else {
		log.Printf("WARNING: eventCB is nil, cannot emit pairing state event!")
	}
	log.Printf("==========================================")
}

// AllowWaitingForTrust determines if we should wait for pairing approval for a device
func (s *Service) AllowWaitingForTrust(ski string) bool {
	s.pairingMux.RLock()
	defer s.pairingMux.RUnlock()

	// If AutoAcceptNew is enabled, allow all devices
	if s.config.AutoAcceptNew {
		log.Printf("AllowWaitingForTrust: SKI=%s, allowing (auto-accept enabled)", ski)
		return true
	}

	// Check if this device is in our pairing list
	allowed := s.pairingDevices[ski]
	log.Printf("AllowWaitingForTrust: SKI=%s, allowed=%v", ski, allowed)
	return allowed
}

// HandleShipHandshakeStateUpdate is called during the SHIP handshake process
func (s *Service) HandleShipHandshakeStateUpdate(ski string, state shipmodel.ShipState) {
	stateStr := fmt.Sprintf("%v", state.State)
	errStr := ""
	if state.Error != nil {
		errStr = state.Error.Error()
	}

	log.Printf("========== SHIP HANDSHAKE UPDATE ==========")
	log.Printf("SKI: %s", ski)
	log.Printf("State: %s", stateStr)
	log.Printf("Error: %s", errStr)
	log.Printf("Full state struct: %+v", state)
	log.Printf("==========================================")

	// Emit handshake state event
	if s.eventCB != nil {
		s.eventCB(Event{
			Type: "shipHandshakeUpdate",
			SKI:  ski,
			Payload: map[string]interface{}{
				"state": stateStr,
				"error": errStr,
			},
		})
		log.Printf("SHIP handshake event emitted successfully")
	}
}

// LoggingInterface implementation

// Trace logs trace messages
func (s *Service) Trace(args ...interface{}) {
	// Trace logging is very verbose, skip it
}

// Tracef logs formatted trace messages
func (s *Service) Tracef(format string, args ...interface{}) {
	// Trace logging is very verbose, skip it
}

// Debug logs debug messages
func (s *Service) Debug(args ...interface{}) {
	log.Print(args...)
}

// Debugf logs formatted debug messages
func (s *Service) Debugf(format string, args ...interface{}) {
	log.Printf(format, args...)
}

// Info logs info messages
func (s *Service) Info(args ...interface{}) {
	log.Print(args...)
}

// Infof logs formatted info messages
func (s *Service) Infof(format string, args ...interface{}) {
	log.Printf(format, args...)
}

// Error logs error messages
func (s *Service) Error(args ...interface{}) {
	log.Print(args...)
}

// Errorf logs formatted error messages
func (s *Service) Errorf(format string, args ...interface{}) {
	log.Printf(format, args...)
}
