package bridge

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/darkv0id/iobroker.eebus/bridge/pkg/protocol"
)

// mockReader provides controlled input for testing
type mockReader struct {
	data   []byte
	offset int
	closed bool
	mu     sync.Mutex
}

func newMockReader(data string) *mockReader {
	return &mockReader{
		data: []byte(data),
	}
}

func (m *mockReader) Read(p []byte) (n int, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return 0, io.EOF
	}

	if m.offset >= len(m.data) {
		// Return EOF when all data is read
		return 0, io.EOF
	}

	n = copy(p, m.data[m.offset:])
	m.offset += n
	return n, nil
}

func (m *mockReader) Close() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closed = true
}

// TestNew tests bridge creation
func TestNew(t *testing.T) {
	input := strings.NewReader("")
	output := &bytes.Buffer{}

	b := New(input, output)

	if b == nil {
		t.Fatal("Expected bridge to be created, got nil")
	}

	if b.input == nil {
		t.Error("Expected input to be set")
	}

	if b.output == nil {
		t.Error("Expected output to be set")
	}

	if b.ctx == nil {
		t.Error("Expected context to be set")
	}

	if b.cancel == nil {
		t.Error("Expected cancel function to be set")
	}

	if b.commandHandlers == nil {
		t.Error("Expected command handlers map to be initialized")
	}

	if len(b.commandHandlers) != 0 {
		t.Errorf("Expected no handlers registered, got %d", len(b.commandHandlers))
	}
}

// TestRegisterHandler tests handler registration
func TestRegisterHandler(t *testing.T) {
	b := New(strings.NewReader(""), &bytes.Buffer{})

	handlerCalled := false
	testHandler := func(ctx context.Context, msg *protocol.Message) (*protocol.Message, error) {
		handlerCalled = true
		return protocol.NewResponse(msg.ID, msg.Action, map[string]interface{}{"status": "ok"}), nil
	}

	b.RegisterHandler("testAction", testHandler)

	handler, exists := b.commandHandlers["testAction"]
	if !exists {
		t.Fatal("Expected handler to be registered")
	}

	if handler == nil {
		t.Fatal("Expected handler function to be set")
	}

	// Test calling the handler
	msg := &protocol.Message{
		ID:     "test-1",
		Type:   protocol.MessageTypeCommand,
		Action: "testAction",
	}

	response, err := handler(context.Background(), msg)
	if err != nil {
		t.Errorf("Handler returned error: %v", err)
	}

	if !handlerCalled {
		t.Error("Handler was not called")
	}

	if response == nil {
		t.Fatal("Expected response, got nil")
	}

	if response.Payload["status"] != "ok" {
		t.Errorf("Expected status 'ok', got %v", response.Payload["status"])
	}
}

// TestSendMessage tests sending messages to output
func TestSendMessage(t *testing.T) {
	output := &bytes.Buffer{}
	b := New(strings.NewReader(""), output)

	msg := protocol.NewResponse("test-123", "testAction", map[string]interface{}{
		"result": "success",
		"count":  42,
	})

	err := b.SendMessage(msg)
	if err != nil {
		t.Fatalf("SendMessage failed: %v", err)
	}

	// Check output contains valid JSON with newline
	outputStr := output.String()
	if !strings.HasSuffix(outputStr, "\n") {
		t.Error("Expected output to end with newline")
	}

	// Parse the JSON
	var decoded protocol.Message
	if err := json.Unmarshal([]byte(strings.TrimSpace(outputStr)), &decoded); err != nil {
		t.Fatalf("Failed to parse output JSON: %v", err)
	}

	if decoded.ID != "test-123" {
		t.Errorf("Expected ID 'test-123', got %s", decoded.ID)
	}

	if decoded.Type != protocol.MessageTypeResponse {
		t.Errorf("Expected type 'response', got %s", decoded.Type)
	}

	if decoded.Action != "testAction" {
		t.Errorf("Expected action 'testAction', got %s", decoded.Action)
	}
}

// TestSendEvent tests sending event messages
func TestSendEvent(t *testing.T) {
	output := &bytes.Buffer{}
	b := New(strings.NewReader(""), output)

	err := b.SendEvent("deviceConnected", map[string]interface{}{
		"ski":  "69898d83b85363ab75428da04c4c31c52cf929f1",
		"name": "Test Device",
	})

	if err != nil {
		t.Fatalf("SendEvent failed: %v", err)
	}

	var decoded protocol.Message
	if err := json.Unmarshal([]byte(strings.TrimSpace(output.String())), &decoded); err != nil {
		t.Fatalf("Failed to parse output: %v", err)
	}

	if decoded.Type != protocol.MessageTypeEvent {
		t.Errorf("Expected type 'event', got %s", decoded.Type)
	}

	if decoded.Action != "deviceConnected" {
		t.Errorf("Expected action 'deviceConnected', got %s", decoded.Action)
	}

	if decoded.ID != "" {
		t.Errorf("Expected empty ID for event, got %s", decoded.ID)
	}

	if decoded.Payload["ski"] != "69898d83b85363ab75428da04c4c31c52cf929f1" {
		t.Errorf("Expected SKI in payload, got %v", decoded.Payload["ski"])
	}
}

// TestSendMessageConcurrent tests concurrent message sending
func TestSendMessageConcurrent(t *testing.T) {
	output := &bytes.Buffer{}
	b := New(strings.NewReader(""), output)

	const numMessages = 100
	var wg sync.WaitGroup
	wg.Add(numMessages)

	// Send messages concurrently
	for i := 0; i < numMessages; i++ {
		go func(id int) {
			defer wg.Done()
			msg := protocol.NewEvent("test", map[string]interface{}{
				"id": id,
			})
			if err := b.SendMessage(msg); err != nil {
				t.Errorf("Failed to send message %d: %v", id, err)
			}
		}(i)
	}

	wg.Wait()

	// All messages should be in output
	// Count newlines to verify all messages were written
	lines := strings.Count(output.String(), "\n")
	if lines != numMessages {
		t.Errorf("Expected %d messages in output, got %d", numMessages, lines)
	}
}

// TestHandleMessageCommand tests command message handling
func TestHandleMessageCommand(t *testing.T) {
	output := &bytes.Buffer{}
	b := New(strings.NewReader(""), output)

	// Register a test handler
	b.RegisterHandler("testCommand", func(ctx context.Context, msg *protocol.Message) (*protocol.Message, error) {
		return protocol.NewResponse(msg.ID, msg.Action, map[string]interface{}{
			"received": msg.Payload["data"],
		}), nil
	})

	// Handle a command message
	msg := &protocol.Message{
		ID:      "cmd-1",
		Type:    protocol.MessageTypeCommand,
		Action:  "testCommand",
		Payload: map[string]interface{}{"data": "test"},
	}

	b.handleMessage(msg)

	// Wait a bit for goroutine to complete
	time.Sleep(100 * time.Millisecond)

	// Check response was sent
	var decoded protocol.Message
	if err := json.Unmarshal([]byte(strings.TrimSpace(output.String())), &decoded); err != nil {
		t.Fatalf("Failed to parse output: %v", err)
	}

	if decoded.ID != "cmd-1" {
		t.Errorf("Expected ID 'cmd-1', got %s", decoded.ID)
	}

	if decoded.Type != protocol.MessageTypeResponse {
		t.Errorf("Expected type 'response', got %s", decoded.Type)
	}

	if decoded.Payload["received"] != "test" {
		t.Errorf("Expected received 'test', got %v", decoded.Payload["received"])
	}
}

// TestHandleMessageUnknownAction tests handling of unknown action
func TestHandleMessageUnknownAction(t *testing.T) {
	output := &bytes.Buffer{}
	b := New(strings.NewReader(""), output)

	msg := &protocol.Message{
		ID:     "cmd-unknown",
		Type:   protocol.MessageTypeCommand,
		Action: "unknownAction",
	}

	b.handleMessage(msg)

	// Wait for goroutine to complete
	time.Sleep(100 * time.Millisecond)

	// Check error response was sent
	var decoded protocol.Message
	if err := json.Unmarshal([]byte(strings.TrimSpace(output.String())), &decoded); err != nil {
		t.Fatalf("Failed to parse output: %v", err)
	}

	if decoded.ID != "cmd-unknown" {
		t.Errorf("Expected ID 'cmd-unknown', got %s", decoded.ID)
	}

	if decoded.Error == "" {
		t.Error("Expected error message, got empty string")
	}

	if !strings.Contains(decoded.Error, "Unknown action") {
		t.Errorf("Expected error about unknown action, got: %s", decoded.Error)
	}
}

// TestHandleCommandWithError tests command handler that returns error
func TestHandleCommandWithError(t *testing.T) {
	output := &bytes.Buffer{}
	b := New(strings.NewReader(""), output)

	// Register handler that returns error
	b.RegisterHandler("failingCommand", func(ctx context.Context, msg *protocol.Message) (*protocol.Message, error) {
		return nil, fmt.Errorf("intentional test error")
	})

	msg := &protocol.Message{
		ID:     "cmd-fail",
		Type:   protocol.MessageTypeCommand,
		Action: "failingCommand",
	}

	b.handleCommand(msg)

	// Check error response was sent
	var decoded protocol.Message
	if err := json.Unmarshal([]byte(strings.TrimSpace(output.String())), &decoded); err != nil {
		t.Fatalf("Failed to parse output: %v", err)
	}

	if decoded.ID != "cmd-fail" {
		t.Errorf("Expected ID 'cmd-fail', got %s", decoded.ID)
	}

	if decoded.Error != "intentional test error" {
		t.Errorf("Expected error message 'intentional test error', got: %s", decoded.Error)
	}
}

// TestStartSendsReadyEvent tests that Start() sends ready event
func TestStartSendsReadyEvent(t *testing.T) {
	// Create input that will close immediately (EOF)
	input := strings.NewReader("")
	output := &bytes.Buffer{}
	b := New(input, output)

	// Start bridge (will exit immediately due to EOF)
	err := b.Start()
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Check ready event was sent
	outputStr := output.String()
	lines := strings.Split(strings.TrimSpace(outputStr), "\n")

	if len(lines) < 1 {
		t.Fatal("Expected at least one line of output")
	}

	var readyEvent protocol.Message
	if err := json.Unmarshal([]byte(lines[0]), &readyEvent); err != nil {
		t.Fatalf("Failed to parse ready event: %v", err)
	}

	if readyEvent.Type != protocol.MessageTypeEvent {
		t.Errorf("Expected type 'event', got %s", readyEvent.Type)
	}

	if readyEvent.Action != protocol.EventReady {
		t.Errorf("Expected action 'ready', got %s", readyEvent.Action)
	}

	if readyEvent.Payload["status"] != "ready" {
		t.Errorf("Expected status 'ready', got %v", readyEvent.Payload["status"])
	}
}

// TestStartProcessesMessages tests that Start() processes incoming messages
func TestStartProcessesMessages(t *testing.T) {
	// Create command message
	cmd := protocol.Message{
		ID:      "test-cmd",
		Type:    protocol.MessageTypeCommand,
		Action:  "echo",
		Payload: map[string]interface{}{"message": "hello"},
	}
	cmdJSON, _ := json.Marshal(cmd)

	// Create input with command followed by EOF
	input := newMockReader(string(cmdJSON) + "\n")
	output := &bytes.Buffer{}
	b := New(input, output)

	// Register echo handler
	b.RegisterHandler("echo", func(ctx context.Context, msg *protocol.Message) (*protocol.Message, error) {
		return protocol.NewResponse(msg.ID, msg.Action, msg.Payload), nil
	})

	// Start bridge in goroutine
	errChan := make(chan error, 1)
	go func() {
		errChan <- b.Start()
	}()

	// Wait for processing
	time.Sleep(200 * time.Millisecond)

	// Close input to trigger EOF
	input.Close()

	// Wait for Start to return
	select {
	case err := <-errChan:
		if err != nil {
			t.Fatalf("Start failed: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Start did not return in time")
	}

	// Check output contains ready event and response
	outputStr := output.String()
	lines := strings.Split(strings.TrimSpace(outputStr), "\n")

	if len(lines) < 2 {
		t.Fatalf("Expected at least 2 lines (ready + response), got %d", len(lines))
	}

	// Parse response (should be second line after ready event)
	var response protocol.Message
	if err := json.Unmarshal([]byte(lines[1]), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if response.ID != "test-cmd" {
		t.Errorf("Expected ID 'test-cmd', got %s", response.ID)
	}

	if response.Type != protocol.MessageTypeResponse {
		t.Errorf("Expected type 'response', got %s", response.Type)
	}

	if response.Payload["message"] != "hello" {
		t.Errorf("Expected message 'hello', got %v", response.Payload["message"])
	}
}

// TestContextCancellation tests that context cancellation stops message processing
func TestContextCancellation(t *testing.T) {
	// Create input that blocks forever
	input := newMockReader("")
	output := &bytes.Buffer{}
	b := New(input, output)

	// Start bridge in goroutine
	done := make(chan struct{})
	go func() {
		b.Start()
		close(done)
	}()

	// Wait a bit for Start to begin
	time.Sleep(100 * time.Millisecond)

	// Cancel context
	b.cancel()

	// Close input to unblock scanner
	input.Close()

	// Wait for Start to return
	select {
	case <-done:
		// Success - Start returned after cancel
	case <-time.After(2 * time.Second):
		t.Fatal("Start did not return after context cancellation")
	}
}

// TestStop tests graceful bridge shutdown
func TestStop(t *testing.T) {
	input := newMockReader("")
	output := &bytes.Buffer{}
	b := New(input, output)

	// Start bridge in goroutine
	go b.Start()

	// Wait a bit for Start to begin
	time.Sleep(100 * time.Millisecond)

	// Call Stop
	b.Stop()

	// Close input to unblock scanner
	input.Close()

	// Wait a bit for shutdown
	time.Sleep(100 * time.Millisecond)

	// Context should be cancelled
	select {
	case <-b.ctx.Done():
		// Success - context was cancelled
	default:
		t.Error("Expected context to be cancelled after Stop()")
	}
}

// TestContextMethod tests Context() method
func TestContextMethod(t *testing.T) {
	b := New(strings.NewReader(""), &bytes.Buffer{})

	ctx := b.Context()
	if ctx == nil {
		t.Fatal("Expected context, got nil")
	}

	if ctx != b.ctx {
		t.Error("Expected Context() to return bridge's context")
	}

	// Verify context is usable
	select {
	case <-ctx.Done():
		t.Error("Context should not be done initially")
	default:
		// Good - context not done
	}

	// Cancel context
	b.cancel()

	// Now context should be done
	select {
	case <-ctx.Done():
		// Good - context is done
	case <-time.After(100 * time.Millisecond):
		t.Error("Context should be done after cancel")
	}
}

// TestEEBusServiceAccessors tests Set/GetEEBusService methods
func TestEEBusServiceAccessors(t *testing.T) {
	b := New(strings.NewReader(""), &bytes.Buffer{})

	// Initially should be nil
	if b.GetEEBusService() != nil {
		t.Error("Expected nil service initially")
	}

	// Set a mock service (nil is fine for this test)
	b.SetEEBusService(nil)

	// Get should return the same
	if b.GetEEBusService() != nil {
		t.Error("Expected nil service after setting nil")
	}
}
