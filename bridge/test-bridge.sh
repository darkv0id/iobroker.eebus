#!/bin/bash
# Simple test script for the EEBus bridge

echo "Testing EEBus Bridge communication..."
echo ""

# Create a named pipe for communication
FIFO=$(mktemp -u)
mkfifo "$FIFO"

# Start the bridge with the fifo as input
../bin/linux-x64/eebus-bridge < "$FIFO" 2>&1 &
BRIDGE_PID=$!

echo "Bridge started with PID: $BRIDGE_PID"

# Open the fifo for writing (keeps it open)
exec 3>"$FIFO"

# Give it a moment to start
sleep 0.5

# Send a startDiscovery command
echo '{"id":"1","type":"command","action":"startDiscovery","payload":{},"timestamp":"2026-03-29T12:00:00Z"}' >&3

# Wait for response
sleep 0.5

# Send a getDeviceInfo command
echo '{"id":"2","type":"command","action":"getDeviceInfo","payload":{"ski":"test-device-123"},"timestamp":"2026-03-29T12:00:00Z"}' >&3

# Wait for response
sleep 0.5

# Send unknown command to test error handling
echo '{"id":"3","type":"command","action":"unknownAction","payload":{},"timestamp":"2026-03-29T12:00:00Z"}' >&3

# Wait a bit for processing
sleep 1

# Terminate the bridge gracefully
echo ""
echo "Terminating bridge..."
kill -TERM $BRIDGE_PID

# Wait for it to finish
wait $BRIDGE_PID 2>/dev/null

# Cleanup
exec 3>&-
rm -f "$FIFO"

echo ""
echo "Test complete"
