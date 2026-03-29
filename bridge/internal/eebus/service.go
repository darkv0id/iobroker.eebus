package eebus

import (
	"crypto/ecdsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/enbility/eebus-go/api"
	"github.com/enbility/eebus-go/service"
	"github.com/enbility/eebus-go/usecases/ma/mgcp"
	"github.com/enbility/eebus-go/usecases/ma/mpc"
	shipapi "github.com/enbility/ship-go/api"
	shipcert "github.com/enbility/ship-go/cert"
	spineapi "github.com/enbility/spine-go/api"
	"github.com/enbility/spine-go/model"
)

// Service wraps the EEBus service and provides integration with our bridge
type Service struct {
	service    api.ServiceInterface
	mgcpUC     *mgcp.MGCP
	mpcUC      *mpc.MPC
	config     *Config
	devices    map[string]*DeviceInfo
	devicesMux sync.RWMutex
	eventCB    EventCallback
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
		config:  config,
		devices: make(map[string]*DeviceInfo),
		eventCB: eventCB,
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

	// Set network interfaces if specified
	if len(s.config.Interfaces) > 0 {
		configuration.SetInterfaces(s.config.Interfaces)
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

// loadOrCreateCertificate loads an existing certificate or creates a new one
func (s *Service) loadOrCreateCertificate() (tls.Certificate, error) {
	certPath := s.config.CertPath
	if certPath == "" {
		certPath = "cert"
	}

	certFile := filepath.Join(certPath, "cert.pem")
	keyFile := filepath.Join(certPath, "key.pem")

	// Try to load existing certificate
	if _, err := os.Stat(certFile); err == nil {
		log.Println("Loading existing certificate...")
		cert, err := tls.LoadX509KeyPair(certFile, keyFile)
		if err != nil {
			log.Printf("Failed to load certificate, creating new one: %v", err)
		} else {
			return cert, nil
		}
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
	log.Printf("Remote SKI connected: %s", ski)

	s.devicesMux.Lock()
	if device, exists := s.devices[ski]; exists {
		device.Connected = true
	}
	s.devicesMux.Unlock()

	if s.eventCB != nil {
		s.eventCB(Event{
			Type: "deviceConnected",
			SKI:  ski,
		})
	}
}

// RemoteSKIDisconnected is called when a remote SKI disconnects
func (s *Service) RemoteSKIDisconnected(service api.ServiceInterface, ski string) {
	log.Printf("Remote SKI disconnected: %s", ski)

	s.devicesMux.Lock()
	if device, exists := s.devices[ski]; exists {
		device.Connected = false
	}
	s.devicesMux.Unlock()

	if s.eventCB != nil {
		s.eventCB(Event{
			Type: "deviceDisconnected",
			SKI:  ski,
		})
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

// ServicePairingDetailUpdate is called when pairing details are updated
func (s *Service) ServicePairingDetailUpdate(ski string, detail *shipapi.ConnectionStateDetail) {
	state := "unknown"
	if detail != nil {
		state = string(detail.State())
	}
	log.Printf("Service pairing update: SKI=%s, State=%s", ski, state)
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
