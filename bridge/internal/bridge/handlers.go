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
	// Add EEBus service here later
	// eebusService *eebus.Service
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
}

// handleStartDiscovery starts device discovery
func (hm *HandlerManager) handleStartDiscovery(ctx context.Context, msg *protocol.Message) (*protocol.Message, error) {
	log.Println("Starting device discovery...")

	// TODO: Start mDNS discovery via EEBus service
	// For now, just return success

	return protocol.NewResponse(msg.ID, msg.Action, map[string]interface{}{
		"status": "started",
	}), nil
}

// handleStopDiscovery stops device discovery
func (hm *HandlerManager) handleStopDiscovery(ctx context.Context, msg *protocol.Message) (*protocol.Message, error) {
	log.Println("Stopping device discovery...")

	// TODO: Stop mDNS discovery via EEBus service

	return protocol.NewResponse(msg.ID, msg.Action, map[string]interface{}{
		"status": "stopped",
	}), nil
}

// handleConnectDevice connects to a specific device
func (hm *HandlerManager) handleConnectDevice(ctx context.Context, msg *protocol.Message) (*protocol.Message, error) {
	ski, ok := msg.Payload["ski"].(string)
	if !ok || ski == "" {
		return nil, fmt.Errorf("missing or invalid 'ski' in payload")
	}

	log.Printf("Connecting to device: %s", ski)

	// TODO: Connect to device via EEBus service

	return protocol.NewResponse(msg.ID, msg.Action, map[string]interface{}{
		"status": "connected",
		"ski":    ski,
	}), nil
}

// handleDisconnectDevice disconnects from a device
func (hm *HandlerManager) handleDisconnectDevice(ctx context.Context, msg *protocol.Message) (*protocol.Message, error) {
	ski, ok := msg.Payload["ski"].(string)
	if !ok || ski == "" {
		return nil, fmt.Errorf("missing or invalid 'ski' in payload")
	}

	log.Printf("Disconnecting from device: %s", ski)

	// TODO: Disconnect from device via EEBus service

	return protocol.NewResponse(msg.ID, msg.Action, map[string]interface{}{
		"status": "disconnected",
		"ski":    ski,
	}), nil
}

// handleSubscribeMeasurements subscribes to device measurements
func (hm *HandlerManager) handleSubscribeMeasurements(ctx context.Context, msg *protocol.Message) (*protocol.Message, error) {
	ski, ok := msg.Payload["ski"].(string)
	if !ok || ski == "" {
		return nil, fmt.Errorf("missing or invalid 'ski' in payload")
	}

	log.Printf("Subscribing to measurements for device: %s", ski)

	// TODO: Subscribe to measurements via EEBus service

	return protocol.NewResponse(msg.ID, msg.Action, map[string]interface{}{
		"status": "subscribed",
		"ski":    ski,
	}), nil
}

// handleUnsubscribeMeasurements unsubscribes from device measurements
func (hm *HandlerManager) handleUnsubscribeMeasurements(ctx context.Context, msg *protocol.Message) (*protocol.Message, error) {
	ski, ok := msg.Payload["ski"].(string)
	if !ok || ski == "" {
		return nil, fmt.Errorf("missing or invalid 'ski' in payload")
	}

	log.Printf("Unsubscribing from measurements for device: %s", ski)

	// TODO: Unsubscribe from measurements via EEBus service

	return protocol.NewResponse(msg.ID, msg.Action, map[string]interface{}{
		"status": "unsubscribed",
		"ski":    ski,
	}), nil
}

// handleGetDeviceInfo retrieves device information
func (hm *HandlerManager) handleGetDeviceInfo(ctx context.Context, msg *protocol.Message) (*protocol.Message, error) {
	ski, ok := msg.Payload["ski"].(string)
	if !ok || ski == "" {
		return nil, fmt.Errorf("missing or invalid 'ski' in payload")
	}

	log.Printf("Getting device info for: %s", ski)

	// TODO: Get device info via EEBus service

	return protocol.NewResponse(msg.ID, msg.Action, map[string]interface{}{
		"ski":  ski,
		"name": "Unknown Device",
		"type": "Unknown",
	}), nil
}
