package bridge

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/darkv0id/iobroker.eebus/bridge/internal/eebus"
	"github.com/darkv0id/iobroker.eebus/bridge/pkg/protocol"
)

// mockEEBusService implements EEBusServiceInterface for testing
type mockEEBusService struct {
	devices        map[string]*eebus.DeviceInfo
	registeredSKIs []string // Track registered devices
}

func newMockEEBusService() *mockEEBusService {
	return &mockEEBusService{
		devices:        make(map[string]*eebus.DeviceInfo),
		registeredSKIs: make([]string, 0),
	}
}

func (m *mockEEBusService) GetDevices() []*eebus.DeviceInfo {
	devices := make([]*eebus.DeviceInfo, 0, len(m.devices))
	for _, device := range m.devices {
		devices = append(devices, device)
	}
	return devices
}

func (m *mockEEBusService) GetDevice(ski string) (*eebus.DeviceInfo, bool) {
	device, exists := m.devices[ski]
	return device, exists
}

func (m *mockEEBusService) RegisterDevice(ski, ip string, port int) error {
	m.registeredSKIs = append(m.registeredSKIs, ski)
	// Add to devices for testing
	m.devices[ski] = &eebus.DeviceInfo{
		SKI:       ski,
		Name:      "Manually Registered Device",
		Connected: false,
	}
	return nil
}

func (m *mockEEBusService) Stop() {
	// Mock implementation - no-op for testing
}

func (m *mockEEBusService) AddDevice(device *eebus.DeviceInfo) {
	m.devices[device.SKI] = device
}

// TestNewHandlerManager tests handler manager creation
func TestNewHandlerManager(t *testing.T) {
	b := New(strings.NewReader(""), &bytes.Buffer{})
	hm := NewHandlerManager(b)

	if hm == nil {
		t.Fatal("Expected handler manager, got nil")
	}

	if hm.bridge != b {
		t.Error("Expected handler manager to reference bridge")
	}
}

// TestRegisterAll tests that all handlers are registered
func TestRegisterAll(t *testing.T) {
	b := New(strings.NewReader(""), &bytes.Buffer{})
	hm := NewHandlerManager(b)

	// Before registration
	if len(b.commandHandlers) != 0 {
		t.Errorf("Expected 0 handlers before registration, got %d", len(b.commandHandlers))
	}

	hm.RegisterAll()

	// After registration - should have 9 handlers
	expectedHandlers := []string{
		protocol.ActionStartDiscovery,
		protocol.ActionStopDiscovery,
		protocol.ActionRegisterDevice,
		protocol.ActionConnectDevice,
		protocol.ActionDisconnectDevice,
		protocol.ActionSubscribeMeasurements,
		protocol.ActionUnsubscribeMeasurements,
		protocol.ActionGetDeviceInfo,
		protocol.ActionListDevices,
	}

	if len(b.commandHandlers) != len(expectedHandlers) {
		t.Errorf("Expected %d handlers, got %d", len(expectedHandlers), len(b.commandHandlers))
	}

	for _, action := range expectedHandlers {
		if _, exists := b.commandHandlers[action]; !exists {
			t.Errorf("Expected handler for '%s' to be registered", action)
		}
	}
}

// TestHandleStartDiscovery tests start discovery handler
func TestHandleStartDiscovery(t *testing.T) {
	b := New(strings.NewReader(""), &bytes.Buffer{})
	mockService := newMockEEBusService()
	b.SetEEBusService(mockService)

	hm := NewHandlerManager(b)

	msg := &protocol.Message{
		ID:     "test-1",
		Type:   protocol.MessageTypeCommand,
		Action: protocol.ActionStartDiscovery,
	}

	response, err := hm.handleStartDiscovery(context.Background(), msg)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if response == nil {
		t.Fatal("Expected response, got nil")
	}

	if response.ID != "test-1" {
		t.Errorf("Expected ID 'test-1', got %s", response.ID)
	}

	if response.Type != protocol.MessageTypeResponse {
		t.Errorf("Expected type 'response', got %s", response.Type)
	}

	if response.Payload["status"] != "started" {
		t.Errorf("Expected status 'started', got %v", response.Payload["status"])
	}
}

// TestHandleStartDiscoveryNoService tests error when service not initialized
func TestHandleStartDiscoveryNoService(t *testing.T) {
	b := New(strings.NewReader(""), &bytes.Buffer{})
	// Don't set service - should error
	hm := NewHandlerManager(b)

	msg := &protocol.Message{
		ID:     "test-1",
		Type:   protocol.MessageTypeCommand,
		Action: protocol.ActionStartDiscovery,
	}

	response, err := hm.handleStartDiscovery(context.Background(), msg)

	if err == nil {
		t.Fatal("Expected error when service not initialized")
	}

	if response != nil {
		t.Errorf("Expected nil response when error, got %v", response)
	}

	if !strings.Contains(err.Error(), "not initialized") {
		t.Errorf("Expected error about service not initialized, got: %s", err.Error())
	}
}

// TestHandleStopDiscovery tests stop discovery handler
func TestHandleStopDiscovery(t *testing.T) {
	b := New(strings.NewReader(""), &bytes.Buffer{})
	hm := NewHandlerManager(b)

	msg := &protocol.Message{
		ID:     "test-1",
		Type:   protocol.MessageTypeCommand,
		Action: protocol.ActionStopDiscovery,
	}

	response, err := hm.handleStopDiscovery(context.Background(), msg)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if response.Payload["status"] != "active" {
		t.Errorf("Expected status 'active', got %v", response.Payload["status"])
	}
}

// TestHandleRegisterDevice tests manual device registration
func TestHandleRegisterDevice(t *testing.T) {
	b := New(strings.NewReader(""), &bytes.Buffer{})
	mockService := newMockEEBusService()
	b.SetEEBusService(mockService)

	hm := NewHandlerManager(b)

	msg := &protocol.Message{
		ID:     "test-1",
		Type:   protocol.MessageTypeCommand,
		Action: protocol.ActionRegisterDevice,
		Payload: map[string]interface{}{
			"ski":  "69898d83b85363ab75428da04c4c31c52cf929f1",
			"ip":   "192.168.1.100",
			"port": 4712,
		},
	}

	response, err := hm.handleRegisterDevice(context.Background(), msg)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if response.Payload["status"] != "registered" {
		t.Errorf("Expected status 'registered', got %v", response.Payload["status"])
	}

	if response.Payload["ski"] != "69898d83b85363ab75428da04c4c31c52cf929f1" {
		t.Errorf("Expected SKI in response, got %v", response.Payload["ski"])
	}

	// Verify device was registered
	if len(mockService.registeredSKIs) != 1 {
		t.Errorf("Expected 1 registered SKI, got %d", len(mockService.registeredSKIs))
	}

	if mockService.registeredSKIs[0] != "69898d83b85363ab75428da04c4c31c52cf929f1" {
		t.Errorf("Expected registered SKI, got %s", mockService.registeredSKIs[0])
	}
}

// TestHandleRegisterDeviceWithFloatPort tests port as float64 (from JSON)
func TestHandleRegisterDeviceWithFloatPort(t *testing.T) {
	b := New(strings.NewReader(""), &bytes.Buffer{})
	mockService := newMockEEBusService()
	b.SetEEBusService(mockService)

	hm := NewHandlerManager(b)

	msg := &protocol.Message{
		ID:     "test-1",
		Type:   protocol.MessageTypeCommand,
		Action: protocol.ActionRegisterDevice,
		Payload: map[string]interface{}{
			"ski":  "69898d83b85363ab75428da04c4c31c52cf929f1",
			"ip":   "192.168.1.100",
			"port": float64(4712), // JSON unmarshals numbers as float64
		},
	}

	response, err := hm.handleRegisterDevice(context.Background(), msg)

	if err != nil {
		t.Fatalf("Expected no error with float64 port, got: %v", err)
	}

	// Port should be converted to int correctly
	if portFloat, ok := response.Payload["port"].(int); !ok || portFloat != 4712 {
		t.Errorf("Expected port 4712 (int), got %T(%v)", response.Payload["port"], response.Payload["port"])
	}
}

// TestHandleRegisterDeviceDefaultPort tests default port when not specified
func TestHandleRegisterDeviceDefaultPort(t *testing.T) {
	b := New(strings.NewReader(""), &bytes.Buffer{})
	mockService := newMockEEBusService()
	b.SetEEBusService(mockService)

	hm := NewHandlerManager(b)

	msg := &protocol.Message{
		ID:     "test-1",
		Type:   protocol.MessageTypeCommand,
		Action: protocol.ActionRegisterDevice,
		Payload: map[string]interface{}{
			"ski": "69898d83b85363ab75428da04c4c31c52cf929f1",
			"ip":  "192.168.1.100",
			// No port - should default to 4712
		},
	}

	response, err := hm.handleRegisterDevice(context.Background(), msg)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if response.Payload["port"] != 4712 {
		t.Errorf("Expected default port 4712, got %v", response.Payload["port"])
	}
}

// TestHandleRegisterDeviceMissingSKI tests error when SKI is missing
func TestHandleRegisterDeviceMissingSKI(t *testing.T) {
	b := New(strings.NewReader(""), &bytes.Buffer{})
	mockService := newMockEEBusService()
	b.SetEEBusService(mockService)

	hm := NewHandlerManager(b)

	msg := &protocol.Message{
		ID:     "test-1",
		Type:   protocol.MessageTypeCommand,
		Action: protocol.ActionRegisterDevice,
		Payload: map[string]interface{}{
			"ip":   "192.168.1.100",
			"port": 4712,
			// Missing SKI
		},
	}

	response, err := hm.handleRegisterDevice(context.Background(), msg)

	if err == nil {
		t.Fatal("Expected error when SKI is missing")
	}

	if response != nil {
		t.Errorf("Expected nil response when error, got %v", response)
	}

	if !strings.Contains(err.Error(), "ski") {
		t.Errorf("Expected error about missing SKI, got: %s", err.Error())
	}
}

// TestHandleConnectDevice tests connecting to a device
func TestHandleConnectDevice(t *testing.T) {
	b := New(strings.NewReader(""), &bytes.Buffer{})
	mockService := newMockEEBusService()

	// Add a test device
	mockService.AddDevice(&eebus.DeviceInfo{
		SKI:       "69898d83b85363ab75428da04c4c31c52cf929f1",
		Name:      "Test Device",
		Connected: true,
	})

	b.SetEEBusService(mockService)
	hm := NewHandlerManager(b)

	msg := &protocol.Message{
		ID:     "test-1",
		Type:   protocol.MessageTypeCommand,
		Action: protocol.ActionConnectDevice,
		Payload: map[string]interface{}{
			"ski": "69898d83b85363ab75428da04c4c31c52cf929f1",
		},
	}

	response, err := hm.handleConnectDevice(context.Background(), msg)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if response.Payload["status"] != "connected" {
		t.Errorf("Expected status 'connected', got %v", response.Payload["status"])
	}

	if response.Payload["connected"] != true {
		t.Errorf("Expected connected true, got %v", response.Payload["connected"])
	}
}

// TestHandleConnectDeviceNotFound tests error when device not found
func TestHandleConnectDeviceNotFound(t *testing.T) {
	b := New(strings.NewReader(""), &bytes.Buffer{})
	mockService := newMockEEBusService()
	b.SetEEBusService(mockService)

	hm := NewHandlerManager(b)

	msg := &protocol.Message{
		ID:     "test-1",
		Type:   protocol.MessageTypeCommand,
		Action: protocol.ActionConnectDevice,
		Payload: map[string]interface{}{
			"ski": "nonexistent",
		},
	}

	_, err := hm.handleConnectDevice(context.Background(), msg)

	if err == nil {
		t.Fatal("Expected error when device not found")
	}

	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("Expected error about device not found, got: %s", err.Error())
	}
}

// TestHandleGetDeviceInfo tests getting device information
func TestHandleGetDeviceInfo(t *testing.T) {
	b := New(strings.NewReader(""), &bytes.Buffer{})
	mockService := newMockEEBusService()

	mockService.AddDevice(&eebus.DeviceInfo{
		SKI:          "69898d83b85363ab75428da04c4c31c52cf929f1",
		Name:         "Test Heat Pump",
		BrandName:    "TestBrand",
		DeviceModel:  "HeatPump-2000",
		SerialNumber: "SN123456",
		DeviceType:   "HeatPump",
		Connected:    true,
	})

	b.SetEEBusService(mockService)
	hm := NewHandlerManager(b)

	msg := &protocol.Message{
		ID:     "test-1",
		Type:   protocol.MessageTypeCommand,
		Action: protocol.ActionGetDeviceInfo,
		Payload: map[string]interface{}{
			"ski": "69898d83b85363ab75428da04c4c31c52cf929f1",
		},
	}

	response, err := hm.handleGetDeviceInfo(context.Background(), msg)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if response.Payload["ski"] != "69898d83b85363ab75428da04c4c31c52cf929f1" {
		t.Errorf("Expected SKI in response, got %v", response.Payload["ski"])
	}

	if response.Payload["name"] != "Test Heat Pump" {
		t.Errorf("Expected name 'Test Heat Pump', got %v", response.Payload["name"])
	}

	if response.Payload["brandName"] != "TestBrand" {
		t.Errorf("Expected brandName 'TestBrand', got %v", response.Payload["brandName"])
	}

	if response.Payload["connected"] != true {
		t.Errorf("Expected connected true, got %v", response.Payload["connected"])
	}
}

// TestHandleListDevices tests listing all devices
func TestHandleListDevices(t *testing.T) {
	b := New(strings.NewReader(""), &bytes.Buffer{})
	mockService := newMockEEBusService()

	// Add multiple devices
	mockService.AddDevice(&eebus.DeviceInfo{
		SKI:       "device1",
		Name:      "Device 1",
		Connected: true,
	})
	mockService.AddDevice(&eebus.DeviceInfo{
		SKI:       "device2",
		Name:      "Device 2",
		Connected: false,
	})

	b.SetEEBusService(mockService)
	hm := NewHandlerManager(b)

	msg := &protocol.Message{
		ID:     "test-1",
		Type:   protocol.MessageTypeCommand,
		Action: protocol.ActionListDevices,
	}

	response, err := hm.handleListDevices(context.Background(), msg)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Check count
	count, ok := response.Payload["count"].(int)
	if !ok {
		t.Fatalf("Expected count to be int, got %T", response.Payload["count"])
	}

	if count != 2 {
		t.Errorf("Expected count 2, got %d", count)
	}

	// Check devices array
	devices, ok := response.Payload["devices"].([]map[string]interface{})
	if !ok {
		t.Fatalf("Expected devices to be array, got %T", response.Payload["devices"])
	}

	if len(devices) != 2 {
		t.Errorf("Expected 2 devices, got %d", len(devices))
	}

	// Verify device data
	foundDevice1 := false
	foundDevice2 := false
	for _, device := range devices {
		if device["ski"] == "device1" {
			foundDevice1 = true
			if device["name"] != "Device 1" {
				t.Errorf("Expected Device 1 name, got %v", device["name"])
			}
			if device["connected"] != true {
				t.Errorf("Expected Device 1 connected true, got %v", device["connected"])
			}
		}
		if device["ski"] == "device2" {
			foundDevice2 = true
			if device["connected"] != false {
				t.Errorf("Expected Device 2 connected false, got %v", device["connected"])
			}
		}
	}

	if !foundDevice1 || !foundDevice2 {
		t.Error("Expected to find both devices in response")
	}
}

// TestHandleListDevicesEmpty tests listing when no devices
func TestHandleListDevicesEmpty(t *testing.T) {
	b := New(strings.NewReader(""), &bytes.Buffer{})
	mockService := newMockEEBusService() // Empty service
	b.SetEEBusService(mockService)

	hm := NewHandlerManager(b)

	msg := &protocol.Message{
		ID:     "test-1",
		Type:   protocol.MessageTypeCommand,
		Action: protocol.ActionListDevices,
	}

	response, err := hm.handleListDevices(context.Background(), msg)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if response.Payload["count"] != 0 {
		t.Errorf("Expected count 0, got %v", response.Payload["count"])
	}

	devices, ok := response.Payload["devices"].([]map[string]interface{})
	if !ok {
		t.Fatalf("Expected devices array, got %T", response.Payload["devices"])
	}

	if len(devices) != 0 {
		t.Errorf("Expected empty devices array, got %d items", len(devices))
	}
}

// TestHandleSubscribeMeasurements tests measurement subscription
func TestHandleSubscribeMeasurements(t *testing.T) {
	b := New(strings.NewReader(""), &bytes.Buffer{})
	mockService := newMockEEBusService()

	mockService.AddDevice(&eebus.DeviceInfo{
		SKI:  "69898d83b85363ab75428da04c4c31c52cf929f1",
		Name: "Test Device",
	})

	b.SetEEBusService(mockService)
	hm := NewHandlerManager(b)

	msg := &protocol.Message{
		ID:     "test-1",
		Type:   protocol.MessageTypeCommand,
		Action: protocol.ActionSubscribeMeasurements,
		Payload: map[string]interface{}{
			"ski": "69898d83b85363ab75428da04c4c31c52cf929f1",
		},
	}

	response, err := hm.handleSubscribeMeasurements(context.Background(), msg)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if response.Payload["status"] != "subscribed" {
		t.Errorf("Expected status 'subscribed', got %v", response.Payload["status"])
	}
}

// TestHandleUnsubscribeMeasurements tests measurement unsubscription
func TestHandleUnsubscribeMeasurements(t *testing.T) {
	b := New(strings.NewReader(""), &bytes.Buffer{})
	hm := NewHandlerManager(b)

	msg := &protocol.Message{
		ID:     "test-1",
		Type:   protocol.MessageTypeCommand,
		Action: protocol.ActionUnsubscribeMeasurements,
		Payload: map[string]interface{}{
			"ski": "69898d83b85363ab75428da04c4c31c52cf929f1",
		},
	}

	response, err := hm.handleUnsubscribeMeasurements(context.Background(), msg)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if response.Payload["status"] != "noted" {
		t.Errorf("Expected status 'noted', got %v", response.Payload["status"])
	}
}

// TestHandleDisconnectDevice tests device disconnection
func TestHandleDisconnectDevice(t *testing.T) {
	b := New(strings.NewReader(""), &bytes.Buffer{})
	hm := NewHandlerManager(b)

	msg := &protocol.Message{
		ID:     "test-1",
		Type:   protocol.MessageTypeCommand,
		Action: protocol.ActionDisconnectDevice,
		Payload: map[string]interface{}{
			"ski": "69898d83b85363ab75428da04c4c31c52cf929f1",
		},
	}

	response, err := hm.handleDisconnectDevice(context.Background(), msg)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if response.Payload["status"] != "noted" {
		t.Errorf("Expected status 'noted', got %v", response.Payload["status"])
	}
}
