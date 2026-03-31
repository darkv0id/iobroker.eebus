package protocol

import (
	"encoding/json"
	"testing"
	"time"
)

// TestMessageTypes tests message type constants
func TestMessageTypes(t *testing.T) {
	tests := []struct {
		name     string
		msgType  MessageType
		expected string
	}{
		{"Command type", MessageTypeCommand, "command"},
		{"Response type", MessageTypeResponse, "response"},
		{"Event type", MessageTypeEvent, "event"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.msgType) != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, tt.msgType)
			}
		})
	}
}

// TestActionConstants tests action constant values
func TestActionConstants(t *testing.T) {
	tests := []struct {
		name     string
		action   string
		expected string
	}{
		// Command actions
		{"Start discovery action", ActionStartDiscovery, "startDiscovery"},
		{"Stop discovery action", ActionStopDiscovery, "stopDiscovery"},
		{"Register device action", ActionRegisterDevice, "registerDevice"},
		{"Connect device action", ActionConnectDevice, "connectDevice"},
		{"Disconnect device action", ActionDisconnectDevice, "disconnectDevice"},
		{"Subscribe measurements action", ActionSubscribeMeasurements, "subscribeMeasurements"},
		{"Unsubscribe measurements action", ActionUnsubscribeMeasurements, "unsubscribeMeasurements"},
		{"Get device info action", ActionGetDeviceInfo, "getDeviceInfo"},
		{"List devices action", ActionListDevices, "listDevices"},

		// Event actions
		{"Ready event", EventReady, "ready"},
		{"Device discovered event", EventDeviceDiscovered, "deviceDiscovered"},
		{"Device connected event", EventDeviceConnected, "deviceConnected"},
		{"Device disconnected event", EventDeviceDisconnected, "deviceDisconnected"},
		{"Measurement update event", EventMeasurementUpdate, "measurementUpdate"},
		{"Pairing state update event", EventPairingStateUpdate, "pairingStateUpdate"},
		{"SHIP handshake update event", EventShipHandshakeUpdate, "shipHandshakeUpdate"},
		{"Error event", EventError, "error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.action != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, tt.action)
			}
		})
	}
}

// TestNewResponse tests response message creation
func TestNewResponse(t *testing.T) {
	id := "test-123"
	action := "testAction"
	payload := map[string]interface{}{
		"result": "success",
		"count":  42, // Integer, not float
	}

	msg := NewResponse(id, action, payload)

	if msg == nil {
		t.Fatal("Expected message, got nil")
	}

	if msg.ID != id {
		t.Errorf("Expected ID %s, got %s", id, msg.ID)
	}

	if msg.Type != MessageTypeResponse {
		t.Errorf("Expected type %s, got %s", MessageTypeResponse, msg.Type)
	}

	if msg.Action != action {
		t.Errorf("Expected action %s, got %s", action, msg.Action)
	}

	if msg.Payload["result"] != "success" {
		t.Errorf("Expected payload result 'success', got %v", msg.Payload["result"])
	}

	// Before JSON serialization, integer values remain as int
	if count, ok := msg.Payload["count"].(int); !ok || count != 42 {
		t.Errorf("Expected payload count to be int(42), got %T(%v)", msg.Payload["count"], msg.Payload["count"])
	}

	if msg.Error != "" {
		t.Errorf("Expected empty error, got %s", msg.Error)
	}

	// Check timestamp is recent
	if time.Since(msg.Timestamp) > time.Second {
		t.Errorf("Timestamp is not recent: %v", msg.Timestamp)
	}
}

// TestNewErrorResponse tests error response message creation
func TestNewErrorResponse(t *testing.T) {
	id := "test-456"
	action := "failedAction"
	errorMsg := "something went wrong"

	msg := NewErrorResponse(id, action, errorMsg)

	if msg == nil {
		t.Fatal("Expected message, got nil")
	}

	if msg.ID != id {
		t.Errorf("Expected ID %s, got %s", id, msg.ID)
	}

	if msg.Type != MessageTypeResponse {
		t.Errorf("Expected type %s, got %s", MessageTypeResponse, msg.Type)
	}

	if msg.Action != action {
		t.Errorf("Expected action %s, got %s", action, msg.Action)
	}

	if msg.Error != errorMsg {
		t.Errorf("Expected error %s, got %s", errorMsg, msg.Error)
	}

	if msg.Payload != nil {
		t.Errorf("Expected nil payload, got %v", msg.Payload)
	}

	// Check timestamp is recent
	if time.Since(msg.Timestamp) > time.Second {
		t.Errorf("Timestamp is not recent: %v", msg.Timestamp)
	}
}

// TestNewEvent tests event message creation
func TestNewEvent(t *testing.T) {
	action := "deviceDiscovered"
	payload := map[string]interface{}{
		"ski":  "69898d83b85363ab75428da04c4c31c52cf929f1",
		"name": "Test Device",
		"port": 4712, // Integer
	}

	msg := NewEvent(action, payload)

	if msg == nil {
		t.Fatal("Expected message, got nil")
	}

	if msg.ID != "" {
		t.Errorf("Expected empty ID for event, got %s", msg.ID)
	}

	if msg.Type != MessageTypeEvent {
		t.Errorf("Expected type %s, got %s", MessageTypeEvent, msg.Type)
	}

	if msg.Action != action {
		t.Errorf("Expected action %s, got %s", action, msg.Action)
	}

	if msg.Payload["ski"] != "69898d83b85363ab75428da04c4c31c52cf929f1" {
		t.Errorf("Expected SKI in payload, got %v", msg.Payload["ski"])
	}

	if msg.Payload["name"] != "Test Device" {
		t.Errorf("Expected name in payload, got %v", msg.Payload["name"])
	}

	// Port should be integer before JSON serialization
	if port, ok := msg.Payload["port"].(int); !ok || port != 4712 {
		t.Errorf("Expected port to be int(4712), got %T(%v)", msg.Payload["port"], msg.Payload["port"])
	}

	if msg.Error != "" {
		t.Errorf("Expected empty error, got %s", msg.Error)
	}

	// Check timestamp is recent
	if time.Since(msg.Timestamp) > time.Second {
		t.Errorf("Timestamp is not recent: %v", msg.Timestamp)
	}
}

// TestMessageJSONSerialization tests JSON marshaling and unmarshaling
// NOTE: JSON unmarshaling converts all numbers to float64 when using map[string]interface{}
// This is a Go JSON limitation, not a bug. Our handlers must handle both int and float64.
func TestMessageJSONSerialization(t *testing.T) {
	tests := []struct {
		name string
		msg  *Message
	}{
		{
			name: "Command message with integer port",
			msg: &Message{
				ID:     "cmd-1",
				Type:   MessageTypeCommand,
				Action: ActionRegisterDevice,
				Payload: map[string]interface{}{
					"ski":  "test-ski",
					"ip":   "192.168.1.100",
					"port": 4712, // Integer before JSON
				},
				Timestamp: time.Now().Truncate(time.Millisecond),
			},
		},
		{
			name: "Response message with integer count",
			msg: &Message{
				ID:     "resp-1",
				Type:   MessageTypeResponse,
				Action: ActionListDevices,
				Payload: map[string]interface{}{
					"devices": []map[string]interface{}{
						{"ski": "device1"},
						{"ski": "device2"},
					},
					"count": 2, // Integer before JSON
				},
				Timestamp: time.Now().Truncate(time.Millisecond),
			},
		},
		{
			name: "Error response message",
			msg: &Message{
				ID:        "err-1",
				Type:      MessageTypeResponse,
				Action:    ActionConnectDevice,
				Error:     "device not found",
				Timestamp: time.Now().Truncate(time.Millisecond),
			},
		},
		{
			name: "Event message",
			msg: &Message{
				Type:   MessageTypeEvent,
				Action: EventDeviceConnected,
				Payload: map[string]interface{}{
					"ski": "69898d83b85363ab75428da04c4c31c52cf929f1",
				},
				Timestamp: time.Now().Truncate(time.Millisecond),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Marshal to JSON
			data, err := json.Marshal(tt.msg)
			if err != nil {
				t.Fatalf("Failed to marshal message: %v", err)
			}

			// Unmarshal back
			var decoded Message
			err = json.Unmarshal(data, &decoded)
			if err != nil {
				t.Fatalf("Failed to unmarshal message: %v", err)
			}

			// Compare fields
			if decoded.ID != tt.msg.ID {
				t.Errorf("ID mismatch: expected %s, got %s", tt.msg.ID, decoded.ID)
			}

			if decoded.Type != tt.msg.Type {
				t.Errorf("Type mismatch: expected %s, got %s", tt.msg.Type, decoded.Type)
			}

			if decoded.Action != tt.msg.Action {
				t.Errorf("Action mismatch: expected %s, got %s", tt.msg.Action, decoded.Action)
			}

			if decoded.Error != tt.msg.Error {
				t.Errorf("Error mismatch: expected %s, got %s", tt.msg.Error, decoded.Error)
			}

			// Timestamp should be close (allowing for JSON precision loss)
			if !decoded.Timestamp.Equal(tt.msg.Timestamp) {
				t.Errorf("Timestamp mismatch: expected %v, got %v", tt.msg.Timestamp, decoded.Timestamp)
			}

			// IMPORTANT: After JSON round-trip, integer values become float64
			// This is a limitation of JSON unmarshaling with map[string]interface{}
			// Handlers must handle both int and float64 types
			if tt.msg.Payload != nil {
				if port, exists := tt.msg.Payload["port"]; exists {
					// After JSON round-trip, should be float64
					if decodedPort, ok := decoded.Payload["port"].(float64); !ok {
						t.Errorf("Expected port to be float64 after JSON round-trip, got %T", decoded.Payload["port"])
					} else {
						// Value should match (42 becomes 42.0)
						var expectedFloat float64
						switch v := port.(type) {
						case int:
							expectedFloat = float64(v)
						case float64:
							expectedFloat = v
						}
						if decodedPort != expectedFloat {
							t.Errorf("Port value mismatch: expected %v, got %v", expectedFloat, decodedPort)
						}
					}
				}

				if count, exists := tt.msg.Payload["count"]; exists {
					// After JSON round-trip, should be float64
					if decodedCount, ok := decoded.Payload["count"].(float64); !ok {
						t.Errorf("Expected count to be float64 after JSON round-trip, got %T", decoded.Payload["count"])
					} else {
						var expectedFloat float64
						switch v := count.(type) {
						case int:
							expectedFloat = float64(v)
						case float64:
							expectedFloat = v
						}
						if decodedCount != expectedFloat {
							t.Errorf("Count value mismatch: expected %v, got %v", expectedFloat, decodedCount)
						}
					}
				}
			}
		})
	}
}

// TestMessageJSONOmitEmpty tests that omitempty works correctly
func TestMessageJSONOmitEmpty(t *testing.T) {
	// Event message (no ID, no error, no payload)
	msg := &Message{
		Type:      MessageTypeEvent,
		Action:    EventReady,
		Timestamp: time.Now(),
	}

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Failed to marshal message: %v", err)
	}

	// Parse JSON to check what fields are present
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("Failed to unmarshal to map: %v", err)
	}

	// ID should be omitted
	if _, hasID := raw["id"]; hasID {
		t.Error("Expected ID to be omitted when empty")
	}

	// Error should be omitted
	if _, hasError := raw["error"]; hasError {
		t.Error("Expected error to be omitted when empty")
	}

	// Payload should be omitted
	if _, hasPayload := raw["payload"]; hasPayload {
		t.Error("Expected payload to be omitted when nil")
	}

	// Type and action should be present
	if _, hasType := raw["type"]; !hasType {
		t.Error("Expected type to be present")
	}

	if _, hasAction := raw["action"]; !hasAction {
		t.Error("Expected action to be present")
	}

	if _, hasTimestamp := raw["timestamp"]; !hasTimestamp {
		t.Error("Expected timestamp to be present")
	}
}

// TestMessageWithNumberTypes tests that numbers in payload work correctly
// Tests both integer and float types, documenting JSON behavior
func TestMessageWithNumberTypes(t *testing.T) {
	msg := &Message{
		ID:     "num-test",
		Type:   MessageTypeResponse,
		Action: "testNumbers",
		Payload: map[string]interface{}{
			"intValue":    42,     // int before JSON
			"floatValue":  3.14,   // float64 before and after JSON
			"largeValue":  int64(9223372036854775807), // int64 before JSON
			"stringValue": "hello",
		},
		Timestamp: time.Now(),
	}

	// Before JSON: integers are integers
	if v, ok := msg.Payload["intValue"].(int); !ok || v != 42 {
		t.Errorf("Before JSON: Expected intValue to be int(42), got %T(%v)", msg.Payload["intValue"], msg.Payload["intValue"])
	}

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Failed to marshal message: %v", err)
	}

	var decoded Message
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal message: %v", err)
	}

	// After JSON: all numbers become float64 in map[string]interface{}
	if v, ok := decoded.Payload["intValue"].(float64); !ok || v != 42.0 {
		t.Errorf("After JSON: Expected intValue to be float64(42.0), got %T(%v)", decoded.Payload["intValue"], decoded.Payload["intValue"])
	}

	if v, ok := decoded.Payload["floatValue"].(float64); !ok || v != 3.14 {
		t.Errorf("Expected floatValue to be float64(3.14), got %T(%v)", decoded.Payload["floatValue"], decoded.Payload["floatValue"])
	}

	// Large int64 values may lose precision in JSON (JavaScript number limit)
	// But should still be float64 type
	if _, ok := decoded.Payload["largeValue"].(float64); !ok {
		t.Errorf("Expected largeValue to be float64, got %T", decoded.Payload["largeValue"])
	}

	if v, ok := decoded.Payload["stringValue"].(string); !ok || v != "hello" {
		t.Errorf("Expected stringValue to be string('hello'), got %T(%v)", decoded.Payload["stringValue"], decoded.Payload["stringValue"])
	}
}

// TestEmptyPayload tests messages with empty but non-nil payload
// NOTE: JSON omitempty will omit empty maps, so empty payload becomes nil after JSON round-trip
func TestEmptyPayload(t *testing.T) {
	msg := NewResponse("test-id", "testAction", map[string]interface{}{})

	// Before JSON: should be empty map (not nil)
	if msg.Payload == nil {
		t.Error("Before JSON: Expected empty payload map, got nil")
	}
	if len(msg.Payload) != 0 {
		t.Errorf("Before JSON: Expected empty payload, got %d items", len(msg.Payload))
	}

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Failed to marshal message: %v", err)
	}

	var decoded Message
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal message: %v", err)
	}

	// After JSON: empty payload is omitted due to omitempty, so it becomes nil
	// This is expected JSON behavior with omitempty tag
	if decoded.Payload != nil {
		t.Errorf("After JSON: Expected payload to be omitted (nil), got %v", decoded.Payload)
	}
}

// TestIntegerPreservation tests that we use integers in Go code
// even though JSON will convert them to float64
func TestIntegerPreservation(t *testing.T) {
	// Create message with integer port (as our code does)
	msg := NewResponse("test", ActionRegisterDevice, map[string]interface{}{
		"port": 4712, // integer
	})

	// Before JSON serialization, should be int
	if port, ok := msg.Payload["port"].(int); !ok {
		t.Errorf("Expected port to be int before JSON, got %T", msg.Payload["port"])
	} else if port != 4712 {
		t.Errorf("Expected port value 4712, got %d", port)
	}

	// After JSON round-trip, becomes float64
	data, _ := json.Marshal(msg)
	var decoded Message
	json.Unmarshal(data, &decoded)

	if port, ok := decoded.Payload["port"].(float64); !ok {
		t.Errorf("Expected port to be float64 after JSON, got %T", decoded.Payload["port"])
	} else if port != 4712.0 {
		t.Errorf("Expected port value 4712.0, got %f", port)
	}

	// Handlers must accept both types (see handlers.go lines 82-87)
	// This tests that our test reflects real-world usage
}
