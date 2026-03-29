package bridge

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"sync"

	"github.com/darkv0id/iobroker.eebus/bridge/pkg/protocol"
)

// Bridge manages the communication between Node.js and EEBus
type Bridge struct {
	input           io.Reader
	output          io.Writer
	outputMutex     sync.Mutex
	ctx             context.Context
	cancel          context.CancelFunc
	commandHandlers map[string]CommandHandler
	wg              sync.WaitGroup
}

// CommandHandler is a function that handles a command
type CommandHandler func(ctx context.Context, msg *protocol.Message) (*protocol.Message, error)

// New creates a new Bridge instance
func New(input io.Reader, output io.Writer) *Bridge {
	ctx, cancel := context.WithCancel(context.Background())
	return &Bridge{
		input:           input,
		output:          output,
		ctx:             ctx,
		cancel:          cancel,
		commandHandlers: make(map[string]CommandHandler),
	}
}

// RegisterHandler registers a command handler
func (b *Bridge) RegisterHandler(action string, handler CommandHandler) {
	b.commandHandlers[action] = handler
}

// Start starts the bridge and begins processing messages
func (b *Bridge) Start() error {
	log.Println("Bridge starting...")

	// Send ready event to Node.js
	if err := b.SendEvent(protocol.EventReady, map[string]interface{}{
		"status": "ready",
	}); err != nil {
		return fmt.Errorf("failed to send ready event: %w", err)
	}

	log.Println("Bridge ready, waiting for commands...")

	// Start reading from stdin
	// Note: scanner.Scan() is blocking and will exit naturally when stdin is closed
	// (i.e., when Node.js closes the pipe or terminates the process)
	scanner := bufio.NewScanner(b.input)

	// Increase buffer size for large messages
	const maxCapacity = 1024 * 1024 // 1MB
	buf := make([]byte, maxCapacity)
	scanner.Buffer(buf, maxCapacity)

	for scanner.Scan() {
		// Check if we should stop (for cleanup coordination)
		select {
		case <-b.ctx.Done():
			log.Println("Bridge context cancelled, stopping scanner...")
			return nil
		default:
			// Continue processing
		}

		line := scanner.Text()
		if line == "" {
			continue
		}

		// Parse message
		var msg protocol.Message
		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			log.Printf("Failed to parse message: %v", err)
			continue
		}

		// Handle message in goroutine
		b.wg.Add(1)
		go func(m protocol.Message) {
			defer b.wg.Done()
			b.handleMessage(&m)
		}(msg)
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading input: %w", err)
	}

	// Wait for all message handlers to complete
	log.Println("Waiting for message handlers to complete...")
	b.wg.Wait()

	log.Println("Bridge scanner loop exited")
	return nil
}

// handleMessage processes an incoming message
func (b *Bridge) handleMessage(msg *protocol.Message) {
	log.Printf("Received %s: %s", msg.Type, msg.Action)

	switch msg.Type {
	case protocol.MessageTypeCommand:
		b.handleCommand(msg)
	default:
		log.Printf("Unknown message type: %s", msg.Type)
	}
}

// handleCommand processes a command message
func (b *Bridge) handleCommand(msg *protocol.Message) {
	handler, exists := b.commandHandlers[msg.Action]
	if !exists {
		log.Printf("No handler for action: %s", msg.Action)
		response := protocol.NewErrorResponse(msg.ID, msg.Action, "Unknown action")
		if err := b.SendMessage(response); err != nil {
			log.Printf("Failed to send error response: %v", err)
		}
		return
	}

	// Execute handler with context
	response, err := handler(b.ctx, msg)
	if err != nil {
		log.Printf("Handler error for %s: %v", msg.Action, err)
		response = protocol.NewErrorResponse(msg.ID, msg.Action, err.Error())
	}

	// Send response
	if err := b.SendMessage(response); err != nil {
		log.Printf("Failed to send response: %v", err)
	}
}

// SendMessage sends a message to Node.js
func (b *Bridge) SendMessage(msg *protocol.Message) error {
	b.outputMutex.Lock()
	defer b.outputMutex.Unlock()

	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	// Write message followed by newline
	if _, err := b.output.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("failed to write message: %w", err)
	}

	return nil
}

// SendEvent sends an event to Node.js
func (b *Bridge) SendEvent(action string, payload map[string]interface{}) error {
	event := protocol.NewEvent(action, payload)
	return b.SendMessage(event)
}

// Stop gracefully stops the bridge
// Note: The scanner loop will exit naturally when stdin is closed by Node.js.
// This method is for coordinating shutdown of other components (like EEBus service).
func (b *Bridge) Stop() {
	log.Println("Bridge stopping...")
	b.cancel() // Cancel context to signal all components to stop
	b.wg.Wait() // Wait for all goroutines to finish
	log.Println("Bridge stopped")
}

// Context returns the bridge's context for use by other components
func (b *Bridge) Context() context.Context {
	return b.ctx
}
