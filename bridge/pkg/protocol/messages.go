package protocol

import "time"

// MessageType represents the type of message
type MessageType string

const (
	// MessageTypeCommand is sent from Node.js to Go (request)
	MessageTypeCommand MessageType = "command"
	// MessageTypeResponse is sent from Go to Node.js (reply to command)
	MessageTypeResponse MessageType = "response"
	// MessageTypeEvent is sent from Go to Node.js (unsolicited notification)
	MessageTypeEvent MessageType = "event"
)

// Message represents a message exchanged between Node.js and Go
type Message struct {
	ID        string                 `json:"id,omitempty"`
	Type      MessageType            `json:"type"`
	Action    string                 `json:"action"`
	Payload   map[string]interface{} `json:"payload,omitempty"`
	Error     string                 `json:"error,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
}

// Command Actions (Node.js -> Go)
const (
	ActionStartDiscovery      = "startDiscovery"
	ActionStopDiscovery       = "stopDiscovery"
	ActionConnectDevice       = "connectDevice"
	ActionDisconnectDevice    = "disconnectDevice"
	ActionSubscribeMeasurements = "subscribeMeasurements"
	ActionUnsubscribeMeasurements = "unsubscribeMeasurements"
	ActionGetDeviceInfo       = "getDeviceInfo"
)

// Event Actions (Go -> Node.js)
const (
	EventReady              = "ready"
	EventDeviceDiscovered   = "deviceDiscovered"
	EventDeviceConnected    = "deviceConnected"
	EventDeviceDisconnected = "deviceDisconnected"
	EventMeasurementUpdate  = "measurementUpdate"
	EventError              = "error"
)

// DeviceInfo represents discovered device information
type DeviceInfo struct {
	SKI     string `json:"ski"`
	Name    string `json:"name"`
	Type    string `json:"type"`
	Address string `json:"address"`
}

// MeasurementData represents power/energy measurements
type MeasurementData struct {
	SKI          string                 `json:"ski"`
	Measurements map[string]interface{} `json:"measurements"`
}

// PowerMeasurement represents power values
type PowerMeasurement struct {
	Active   *float64 `json:"active,omitempty"`
	Reactive *float64 `json:"reactive,omitempty"`
	Apparent *float64 `json:"apparent,omitempty"`
	Unit     string   `json:"unit"`
}

// EnergyMeasurement represents energy values
type EnergyMeasurement struct {
	Consumed *float64 `json:"consumed,omitempty"`
	Produced *float64 `json:"produced,omitempty"`
	Unit     string   `json:"unit"`
}

// VoltageMeasurement represents voltage per phase
type VoltageMeasurement struct {
	L1   *float64 `json:"L1,omitempty"`
	L2   *float64 `json:"L2,omitempty"`
	L3   *float64 `json:"L3,omitempty"`
	Unit string   `json:"unit"`
}

// CurrentMeasurement represents current per phase
type CurrentMeasurement struct {
	L1   *float64 `json:"L1,omitempty"`
	L2   *float64 `json:"L2,omitempty"`
	L3   *float64 `json:"L3,omitempty"`
	Unit string   `json:"unit"`
}

// NewResponse creates a new response message
func NewResponse(id, action string, payload map[string]interface{}) *Message {
	return &Message{
		ID:        id,
		Type:      MessageTypeResponse,
		Action:    action,
		Payload:   payload,
		Timestamp: time.Now(),
	}
}

// NewErrorResponse creates a new error response
func NewErrorResponse(id, action, errorMsg string) *Message {
	return &Message{
		ID:        id,
		Type:      MessageTypeResponse,
		Action:    action,
		Error:     errorMsg,
		Timestamp: time.Now(),
	}
}

// NewEvent creates a new event message
func NewEvent(action string, payload map[string]interface{}) *Message {
	return &Message{
		Type:      MessageTypeEvent,
		Action:    action,
		Payload:   payload,
		Timestamp: time.Now(),
	}
}
