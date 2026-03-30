#!/usr/bin/env node

const { spawn } = require('child_process');
const readline = require('readline');
require('dotenv').config();

// Configuration from environment variables
const DEVICE_SKI = process.env.EEBUS_DEVICE_SKI || '';
const DEVICE_IP = process.env.EEBUS_DEVICE_IP || '';
const DEVICE_PORT = parseInt(process.env.EEBUS_DEVICE_PORT || '4712', 10);

if (!DEVICE_SKI || !DEVICE_IP) {
  console.error('Error: EEBUS_DEVICE_SKI and EEBUS_DEVICE_IP must be set in .env file');
  console.error('Copy .env.example to .env and fill in your device details');
  process.exit(1);
}

console.log('===== EEBus Manual Device Registration Test =====');
console.log(`Device SKI: ${DEVICE_SKI}`);
console.log(`Device IP:  ${DEVICE_IP}`);
console.log(`Device Port: ${DEVICE_PORT}`);
console.log('=================================================\n');

// Start the bridge
console.log('Starting EEBus bridge...');
const bridge = spawn('./bin/eebus-bridge', [], {
  cwd: __dirname,
  stdio: ['pipe', 'pipe', 'pipe']
});

// Create readline interfaces for stdout and stderr
const stdoutRL = readline.createInterface({
  input: bridge.stdout,
  crlfDelay: Infinity
});

const stderrRL = readline.createInterface({
  input: bridge.stderr,
  crlfDelay: Infinity
});

let bridgeReady = false;

// Handle stdout (JSON messages)
stdoutRL.on('line', (line) => {
  try {
    const msg = JSON.parse(line);
    console.log(`[BRIDGE EVENT] ${msg.action}:`, JSON.stringify(msg.payload || {}, null, 2));

    // When bridge is ready, send the register command
    if (msg.action === 'ready' && !bridgeReady) {
      bridgeReady = true;
      console.log('\n✓ Bridge is ready!\n');

      // Wait a moment for everything to initialize
      setTimeout(() => {
        console.log('Sending manual device registration command...');
        const registerCommand = {
          id: 'test-register-1',
          type: 'command',
          action: 'registerDevice',
          payload: {
            ski: DEVICE_SKI,
            ip: DEVICE_IP,
            port: DEVICE_PORT
          },
          timestamp: new Date().toISOString()
        };

        console.log('Command:', JSON.stringify(registerCommand, null, 2));
        bridge.stdin.write(JSON.stringify(registerCommand) + '\n');
      }, 1000);
    }

    // Check for device events
    if (msg.action === 'deviceDiscovered') {
      console.log('\n✓✓✓ DEVICE DISCOVERED! ✓✓✓');
      console.log('Device Info:', msg.payload);
    }

    if (msg.action === 'deviceConnected') {
      console.log('\n✓✓✓ DEVICE CONNECTED! ✓✓✓');
      console.log('Device SKI:', msg.payload.ski);
    }

    if (msg.action === 'measurementUpdate') {
      console.log('\n✓✓✓ MEASUREMENT DATA RECEIVED! ✓✓✓');
      console.log('Measurements:', JSON.stringify(msg.payload, null, 2));
    }
  } catch (e) {
    console.log(`[BRIDGE STDOUT] ${line}`);
  }
});

// Handle stderr (logs)
stderrRL.on('line', (line) => {
  console.log(`[BRIDGE LOG] ${line}`);

  // Look for important log messages
  if (line.includes('registered successfully')) {
    console.log('\n✓ Device registration successful!\n');
  }

  if (line.includes('trying to connect')) {
    console.log('\n→ Attempting to connect to device...\n');
  }

  if (line.includes('connection to') && line.includes('failed')) {
    console.log('\n✗ Connection attempt failed (see details above)\n');
  }
});

// Handle bridge errors
bridge.on('error', (err) => {
  console.error('Failed to start bridge:', err);
  process.exit(1);
});

bridge.on('close', (code) => {
  console.log(`\nBridge process exited with code ${code}`);
  process.exit(code);
});

// Handle script termination
process.on('SIGINT', () => {
  console.log('\n\nStopping test...');
  bridge.kill();
  process.exit(0);
});

process.on('SIGTERM', () => {
  bridge.kill();
  process.exit(0);
});

console.log('Waiting for bridge to start...\n');

// Set a timeout to prevent hanging forever
setTimeout(() => {
  if (!bridgeReady) {
    console.error('\n✗ Timeout: Bridge did not become ready within 10 seconds');
    bridge.kill();
    process.exit(1);
  }
}, 10000);

// Keep the script running to monitor events
console.log('Monitoring for events (Press Ctrl+C to stop)...\n');
