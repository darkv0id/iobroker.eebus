package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/darkv0id/iobroker.eebus/bridge/internal/bridge"
	"github.com/darkv0id/iobroker.eebus/bridge/internal/eebus"
	"github.com/darkv0id/iobroker.eebus/bridge/pkg/protocol"
)

func main() {
	// Set up logging to stderr (stdout is used for JSON messages)
	log.SetOutput(os.Stderr)
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	log.Println("EEBus Bridge starting...")

	// Create bridge instance
	b := bridge.New(os.Stdin, os.Stdout)

	// Create EEBus service configuration
	eebusConfig := &eebus.Config{
		VendorCode:       "IOBROKER",
		BrandName:        "ioBroker",
		DeviceModel:      "EEBus Adapter",
		SerialNumber:     "IOB-EEBUS-001",
		Port:             4712,
		Interfaces:       []string{}, // Empty = use all interfaces
		CertPath:         "cert",
		AutoAcceptNew:    false, // Require manual pairing
		HeartbeatTimeout: 4 * time.Second,
	}

	// Create event callback that forwards events to the bridge
	eventCallback := func(event eebus.Event) {
		log.Printf("EEBus event: %s for SKI=%s", event.Type, event.SKI)

		// Forward event to bridge (to Node.js)
		var action string
		payload := make(map[string]interface{})

		switch event.Type {
		case "deviceDiscovered":
			action = protocol.EventDeviceDiscovered
			if event.Device != nil {
				payload["ski"] = event.Device.SKI
				payload["name"] = event.Device.Name
				payload["brandName"] = event.Device.BrandName
				payload["deviceModel"] = event.Device.DeviceModel
				payload["serialNumber"] = event.Device.SerialNumber
				payload["deviceType"] = event.Device.DeviceType
				payload["connected"] = event.Device.Connected
			}

		case "deviceConnected":
			action = protocol.EventDeviceConnected
			payload["ski"] = event.SKI

		case "deviceDisconnected":
			action = protocol.EventDeviceDisconnected
			payload["ski"] = event.SKI

		case "measurementUpdate":
			action = protocol.EventMeasurementUpdate
			payload["ski"] = event.SKI
			// Copy all payload fields from event
			for k, v := range event.Payload {
				payload[k] = v
			}

		case "pairingStateUpdate":
			action = protocol.EventPairingStateUpdate
			payload["ski"] = event.SKI
			// Copy all payload fields from event
			for k, v := range event.Payload {
				payload[k] = v
			}

		case "shipHandshakeUpdate":
			action = protocol.EventShipHandshakeUpdate
			payload["ski"] = event.SKI
			// Copy all payload fields from event
			for k, v := range event.Payload {
				payload[k] = v
			}

		default:
			log.Printf("Unknown event type: %s", event.Type)
			return
		}

		// Send event via bridge
		if err := b.SendEvent(action, payload); err != nil {
			log.Printf("Failed to send event to bridge: %v", err)
		}
	}

	// Create EEBus service
	eebusService, err := eebus.NewService(eebusConfig, eventCallback)
	if err != nil {
		log.Fatalf("Failed to create EEBus service: %v", err)
	}

	// Start EEBus service
	log.Println("Starting EEBus service...")
	if err := eebusService.Start(); err != nil {
		log.Fatalf("Failed to start EEBus service: %v", err)
	}

	// Set EEBus service in bridge so handlers can use it
	b.SetEEBusService(eebusService)

	// Create and register handlers
	handlerManager := bridge.NewHandlerManager(b)
	handlerManager.RegisterAll()

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Start bridge in goroutine
	errChan := make(chan error, 1)
	go func() {
		if err := b.Start(); err != nil {
			errChan <- err
		}
	}()

	// Wait for shutdown signal or error
	select {
	case sig := <-sigChan:
		log.Printf("Received signal: %v, shutting down...", sig)
		b.Stop()
	case err := <-errChan:
		log.Printf("Bridge error: %v", err)
		b.Stop()
		os.Exit(1)
	}

	log.Println("EEBus Bridge stopped")
}
