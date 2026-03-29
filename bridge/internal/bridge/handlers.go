package bridge

import (
	"context"
	"fmt"
	"log"

	"github.com/darkv0id/iobroker.eebus/bridge/pkg/protocol"
)

// HandlerManager manages command handlers and their dependencies
type HandlerManager struct {
	bridge *Bridge
}

// NewHandlerManager creates a new handler manager
func NewHandlerManager(bridge *Bridge) *HandlerManager {
	return &HandlerManager{
		bridge: bridge,
	}
}

// RegisterAll registers all command handlers
func (hm *HandlerManager) RegisterAll() {
	hm.bridge.RegisterHandler(protocol.ActionStartDiscovery, hm.handleStartDiscovery)
	hm.bridge.RegisterHandler(protocol.ActionStopDiscovery, hm.handleStopDiscovery)
	hm.bridge.RegisterHandler(protocol.ActionConnectDevice, hm.handleConnectDevice)
	hm.bridge.RegisterHandler(protocol.ActionDisconnectDevice, hm.handleDisconnectDevice)
	hm.bridge.RegisterHandler(protocol.ActionSubscribeMeasurements, hm.handleSubscribeMeasurements)
	hm.bridge.RegisterHandler(protocol.ActionUnsubscribeMeasurements, hm.handleUnsubscribeMeasurements)
	hm.bridge.RegisterHandler(protocol.ActionGetDeviceInfo, hm.handleGetDeviceInfo)
	hm.bridge.RegisterHandler(protocol.ActionListDevices, hm.handleListDevices)
}

// handleStartDiscovery starts device discovery
func (hm *HandlerManager) handleStartDiscovery(ctx context.Context, msg *protocol.Message) (*protocol.Message, error) {
	log.Println("Starting device discovery...")

	service := hm.bridge.GetEEBusService()
	if service == nil {
		return nil, fmt.Errorf("EEBus service not initialized")
	}

	// Note: EEBus service automatically starts mDNS discovery when it starts
	// Discovery is always active while the service is running

	return protocol.NewResponse(msg.ID, msg.Action, map[string]interface{}{
		"status": "started",
		"note":   "Discovery is automatically active",
	}), nil
}

// handleStopDiscovery stops device discovery
func (hm *HandlerManager) handleStopDiscovery(ctx context.Context, msg *protocol.Message) (*protocol.Message, error) {
	log.Println("Stopping device discovery...")

	// Note: Discovery cannot be stopped without stopping the entire service
	// It's always active while the service is running

	return protocol.NewResponse(msg.ID, msg.Action, map[string]interface{}{
		"status": "active",
		"note":   "Discovery cannot be stopped individually",
	}), nil
}

// handleConnectDevice connects to a specific device
func (hm *HandlerManager) handleConnectDevice(ctx context.Context, msg *protocol.Message) (*protocol.Message, error) {
	ski, ok := msg.Payload["ski"].(string)
	if !ok || ski == "" {
		return nil, fmt.Errorf("missing or invalid 'ski' in payload")
	}

	log.Printf("Connecting to device: %s", ski)

	service := hm.bridge.GetEEBusService()
	if service == nil {
		return nil, fmt.Errorf("EEBus service not initialized")
	}

	// Check if device exists
	device, exists := service.GetDevice(ski)
	if !exists {
		return nil, fmt.Errorf("device not found: %s", ski)
	}

	// Note: EEBus service automatically manages connections when devices are discovered
	// We just verify the device is known

	return protocol.NewResponse(msg.ID, msg.Action, map[string]interface{}{
		"status":    "connected",
		"ski":       ski,
		"connected": device.Connected,
	}), nil
}

// handleDisconnectDevice disconnects from a device
func (hm *HandlerManager) handleDisconnectDevice(ctx context.Context, msg *protocol.Message) (*protocol.Message, error) {
	ski, ok := msg.Payload["ski"].(string)
	if !ok || ski == "" {
		return nil, fmt.Errorf("missing or invalid 'ski' in payload")
	}

	log.Printf("Disconnecting from device: %s", ski)

	// Note: EEBus service manages connections automatically
	// Explicit disconnect is not supported in the current implementation

	return protocol.NewResponse(msg.ID, msg.Action, map[string]interface{}{
		"status": "noted",
		"ski":    ski,
		"note":   "Connection is managed automatically by EEBus service",
	}), nil
}

// handleSubscribeMeasurements subscribes to device measurements
func (hm *HandlerManager) handleSubscribeMeasurements(ctx context.Context, msg *protocol.Message) (*protocol.Message, error) {
	ski, ok := msg.Payload["ski"].(string)
	if !ok || ski == "" {
		return nil, fmt.Errorf("missing or invalid 'ski' in payload")
	}

	log.Printf("Subscribing to measurements for device: %s", ski)

	service := hm.bridge.GetEEBusService()
	if service == nil {
		return nil, fmt.Errorf("EEBus service not initialized")
	}

	// Check if device exists
	_, exists := service.GetDevice(ski)
	if !exists {
		return nil, fmt.Errorf("device not found: %s", ski)
	}

	// Note: MGCP/MPC use cases automatically receive measurement updates
	// when devices support those use cases. No explicit subscription is needed.

	return protocol.NewResponse(msg.ID, msg.Action, map[string]interface{}{
		"status": "subscribed",
		"ski":    ski,
		"note":   "Measurements are automatically received via MGCP/MPC use cases",
	}), nil
}

// handleUnsubscribeMeasurements unsubscribes from device measurements
func (hm *HandlerManager) handleUnsubscribeMeasurements(ctx context.Context, msg *protocol.Message) (*protocol.Message, error) {
	ski, ok := msg.Payload["ski"].(string)
	if !ok || ski == "" {
		return nil, fmt.Errorf("missing or invalid 'ski' in payload")
	}

	log.Printf("Unsubscribing from measurements for device: %s", ski)

	// Note: Measurements are always active for supported use cases
	// Cannot be unsubscribed individually

	return protocol.NewResponse(msg.ID, msg.Action, map[string]interface{}{
		"status": "noted",
		"ski":    ski,
		"note":   "Measurements cannot be unsubscribed individually",
	}), nil
}

// handleGetDeviceInfo retrieves device information
func (hm *HandlerManager) handleGetDeviceInfo(ctx context.Context, msg *protocol.Message) (*protocol.Message, error) {
	ski, ok := msg.Payload["ski"].(string)
	if !ok || ski == "" {
		return nil, fmt.Errorf("missing or invalid 'ski' in payload")
	}

	log.Printf("Getting device info for: %s", ski)

	service := hm.bridge.GetEEBusService()
	if service == nil {
		return nil, fmt.Errorf("EEBus service not initialized")
	}

	device, exists := service.GetDevice(ski)
	if !exists {
		return nil, fmt.Errorf("device not found: %s", ski)
	}

	return protocol.NewResponse(msg.ID, msg.Action, map[string]interface{}{
		"ski":          device.SKI,
		"name":         device.Name,
		"brandName":    device.BrandName,
		"deviceModel":  device.DeviceModel,
		"serialNumber": device.SerialNumber,
		"deviceType":   device.DeviceType,
		"connected":    device.Connected,
	}), nil
}

// handleListDevices lists all discovered devices
func (hm *HandlerManager) handleListDevices(ctx context.Context, msg *protocol.Message) (*protocol.Message, error) {
	log.Println("Listing all devices...")

	service := hm.bridge.GetEEBusService()
	if service == nil {
		return nil, fmt.Errorf("EEBus service not initialized")
	}

	devices := service.GetDevices()
	deviceList := make([]map[string]interface{}, len(devices))

	for i, device := range devices {
		deviceList[i] = map[string]interface{}{
			"ski":          device.SKI,
			"name":         device.Name,
			"brandName":    device.BrandName,
			"deviceModel":  device.DeviceModel,
			"serialNumber": device.SerialNumber,
			"deviceType":   device.DeviceType,
			"connected":    device.Connected,
		}
	}

	return protocol.NewResponse(msg.ID, msg.Action, map[string]interface{}{
		"devices": deviceList,
		"count":   len(deviceList),
	}), nil
}
