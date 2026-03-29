package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/darkv0id/iobroker.eebus/bridge/internal/bridge"
)

func main() {
	// Set up logging to stderr (stdout is used for JSON messages)
	log.SetOutput(os.Stderr)
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	log.Println("EEBus Bridge starting...")

	// Create bridge instance
	b := bridge.New(os.Stdin, os.Stdout)

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
