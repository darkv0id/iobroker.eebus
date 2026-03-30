'use strict';

const { spawn } = require('child_process');
const path = require('path');
const EventEmitter = require('events');

/**
 * EEBus Bridge - Manages communication with the Go EEBus process
 *
 * This class handles:
 * - Spawning and managing the Go binary lifecycle
 * - JSON message communication over stdio
 * - Message queuing and response handling
 * - Automatic reconnection on failures
 */
class EEBusBridge extends EventEmitter {
	/**
	 * @param {object} adapter - ioBroker adapter instance for logging
	 * @param {object} options - Bridge options
	 */
	constructor(adapter, options = {}) {
		super();

		this.adapter = adapter;
		this.options = {
			binaryPath: options.binaryPath || this.detectBinaryPath(),
			restartDelay: options.restartDelay || 5000,
			maxRestarts: options.maxRestarts || 5,
			messageTimeout: options.messageTimeout || 30000,
			...options,
		};

		this.process = null;
		this.connected = false;
		this.restartCount = 0;
		this.messageQueue = new Map(); // id -> {resolve, reject, timeout}
		this.messageIdCounter = 0;
		this.stdoutBuffer = '';

		this.adapter.log.info('EEBus Bridge initialized');
	}

	/**
	 * Detect the correct Go binary path based on platform
	 * @returns {string} Path to the Go binary
	 */
	detectBinaryPath() {
		const platform = process.platform;
		const arch = process.arch;

		// Binary name based on platform
		let binaryName = 'eebus-bridge';
		if (platform === 'win32') {
			binaryName += '.exe';
		}

		// Construct path: bin/<platform>-<arch>/eebus-bridge
		const binaryPath = path.join(__dirname, '..', 'bin', `${platform}-${arch}`, binaryName);

		this.adapter.log.debug(`Detected binary path: ${binaryPath}`);
		return binaryPath;
	}

	/**
	 * Start the Go bridge process
	 * @returns {Promise<void>}
	 */
	async start() {
		if (this.process) {
			this.adapter.log.warn('Bridge process already running');
			return;
		}

		this.adapter.log.info(`Starting EEBus bridge process: ${this.options.binaryPath}`);

		return new Promise((resolve, reject) => {
			let rejected = false;

			try {
				// Spawn the Go binary
				this.process = spawn(this.options.binaryPath, [], {
					stdio: ['pipe', 'pipe', 'pipe'],
				});

				// Handle immediate spawn errors (e.g., binary not found)
				this.process.on('error', (error) => {
					if (!rejected) {
						rejected = true;
						this.process = null;
						this.adapter.log.error(`Failed to spawn bridge process: ${error.message}`);
						reject(error);
					}
				});

				// Handle immediate exit (e.g., binary crashes immediately)
				this.process.on('exit', (code, signal) => {
					if (!rejected && !this.connected) {
						rejected = true;
						this.process = null;
						this.adapter.log.error(`Bridge process exited immediately with code ${code}`);
						reject(new Error(`Bridge process exited with code ${code}`));
					}
				});

				// Set up event handlers (this will override the error/exit handlers above after connection)
				this.setupProcessHandlers();

				// Wait for initial connection
				this.waitForConnection()
					.then(() => {
						if (!rejected) {
							this.connected = true;
							this.restartCount = 0;
							this.adapter.log.info('EEBus bridge process started successfully');
							this.emit('connected');
							resolve();
						}
					})
					.catch((error) => {
						if (!rejected) {
							rejected = true;
							if (this.process) {
								this.process.kill();
								this.process = null;
							}
							reject(error);
						}
					});
			} catch (error) {
				rejected = true;
				this.adapter.log.error(`Failed to start bridge process: ${error.message}`);
				reject(error);
			}
		});
	}

	/**
	 * Set up handlers for process events
	 */
	setupProcessHandlers() {
		if (!this.process) {
			return;
		}

		// Handle stdout data (JSON messages from Go)
		this.process.stdout.on('data', (data) => {
			this.handleStdout(data);
		});

		// Handle stderr (Go logging/errors)
		this.process.stderr.on('data', (data) => {
			const message = data.toString().trim();
			this.adapter.log.debug(`[Go] ${message}`);
		});

		// Handle process exit (only if already connected)
		this.process.on('exit', (code, signal) => {
			if (this.connected) {
				this.adapter.log.warn(`Bridge process exited with code ${code}, signal ${signal}`);
				this.handleProcessExit(code, signal);
			}
		});

		// Handle process errors (only if already connected)
		this.process.on('error', (error) => {
			if (this.connected) {
				this.adapter.log.error(`Bridge process error: ${error.message}`);
				this.emit('error', error);
			}
		});
	}

	/**
	 * Handle stdout data from Go process
	 * @param {Buffer} data - Raw data from stdout
	 */
	handleStdout(data) {
		// Append to buffer
		this.stdoutBuffer += data.toString();

		// Process complete JSON messages (newline-delimited)
		let newlineIndex;
		while ((newlineIndex = this.stdoutBuffer.indexOf('\n')) !== -1) {
			const line = this.stdoutBuffer.slice(0, newlineIndex).trim();
			this.stdoutBuffer = this.stdoutBuffer.slice(newlineIndex + 1);

			if (line.length > 0) {
				this.handleMessage(line);
			}
		}
	}

	/**
	 * Handle a complete JSON message from Go
	 * @param {string} line - JSON string
	 */
	handleMessage(line) {
		try {
			const message = JSON.parse(line);
			this.adapter.log.debug(`Received message: ${JSON.stringify(message)}`);

			if (message.type === 'response' && message.id) {
				// This is a response to a command we sent
				this.handleResponse(message);
			} else if (message.type === 'event') {
				// This is an unsolicited event from Go
				this.handleEvent(message);
			} else {
				this.adapter.log.warn(`Unknown message type: ${JSON.stringify(message)}`);
			}
		} catch (error) {
			this.adapter.log.error(`Failed to parse JSON message: ${error.message}`);
			this.adapter.log.debug(`Problematic line: ${line}`);
		}
	}

	/**
	 * Handle a response message
	 * @param {object} message - Parsed response message
	 */
	handleResponse(message) {
		const pending = this.messageQueue.get(message.id);
		if (!pending) {
			this.adapter.log.warn(`Received response for unknown message ID: ${message.id}`);
			return;
		}

		// Clear timeout
		clearTimeout(pending.timeout);
		this.messageQueue.delete(message.id);

		// Resolve or reject based on response
		if (message.error) {
			pending.reject(new Error(message.error));
		} else {
			pending.resolve(message.payload);
		}
	}

	/**
	 * Handle an event message
	 * @param {object} message - Parsed event message
	 */
	handleEvent(message) {
		// Emit event with action name and payload
		this.emit(message.action, message.payload);
		this.emit('event', message.action, message.payload);
	}

	/**
	 * Handle process exit and potentially restart
	 * @param {number} code - Exit code
	 * @param {string} signal - Exit signal
	 */
	async handleProcessExit(code, signal) {
		this.connected = false;
		this.process = null;

		// Reject all pending messages
		for (const [id, pending] of this.messageQueue.entries()) {
			clearTimeout(pending.timeout);
			pending.reject(new Error('Bridge process exited'));
		}
		this.messageQueue.clear();

		this.emit('disconnected');

		// Attempt restart if within limits
		if (this.restartCount < this.options.maxRestarts) {
			this.restartCount++;
			this.adapter.log.info(
				`Attempting to restart bridge (${this.restartCount}/${this.options.maxRestarts})`,
			);

			setTimeout(() => {
				this.start().catch((error) => {
					this.adapter.log.error(`Failed to restart bridge: ${error.message}`);
				});
			}, this.options.restartDelay);
		} else {
			this.adapter.log.error(
				`Bridge process failed after ${this.options.maxRestarts} restart attempts`,
			);
			this.emit('failed');
		}
	}

	/**
	 * Wait for initial connection confirmation
	 * @returns {Promise<void>}
	 */
	waitForConnection() {
		return new Promise((resolve, reject) => {
			// For now, we expect the Go process to send a "ready" event
			// If we don't receive it within the timeout, we fail
			const timeout = setTimeout(() => {
				reject(new Error('Bridge process did not send ready signal within timeout'));
			}, 5000);

			// Listen for first message from Go process indicating it's ready
			const onReady = () => {
				clearTimeout(timeout);
				this.removeListener('ready', onReady);
				resolve();
			};

			this.once('ready', onReady);

			// TODO: For now, until Go binary sends "ready" event, we accept any event as "ready"
			// This is temporary - the Go binary should send a proper ready message
			const onAnyEvent = () => {
				clearTimeout(timeout);
				this.removeListener('event', onAnyEvent);
				resolve();
			};

			this.once('event', onAnyEvent);
		});
	}

	/**
	 * Send a command to the Go process
	 * @param {string} action - Command action
	 * @param {object} payload - Command payload
	 * @returns {Promise<object>} Response payload
	 */
	async sendCommand(action, payload = {}) {
		if (!this.connected || !this.process) {
			throw new Error('Bridge not connected');
		}

		const messageId = String(++this.messageIdCounter);

		const message = {
			id: messageId,
			type: 'command',
			action,
			payload,
			timestamp: new Date().toISOString(),
		};

		return new Promise((resolve, reject) => {
			// Set up timeout
			const timeout = setTimeout(() => {
				this.messageQueue.delete(messageId);
				reject(new Error(`Command timeout: ${action}`));
			}, this.options.messageTimeout);

			// Store in queue
			this.messageQueue.set(messageId, { resolve, reject, timeout });

			// Send message
			const json = JSON.stringify(message) + '\n';
			this.adapter.log.debug(`Sending command: ${json.trim()}`);

			this.process.stdin.write(json, (error) => {
				if (error) {
					clearTimeout(timeout);
					this.messageQueue.delete(messageId);
					reject(error);
				}
			});
		});
	}

	/**
	 * Stop the bridge process
	 * @returns {Promise<void>}
	 */
	async stop() {
		this.adapter.log.info('Stopping EEBus bridge process');

		if (!this.process) {
			return;
		}

		// Prevent auto-restart
		this.restartCount = this.options.maxRestarts;

		// Save process reference locally to ensure we can kill it even if this.process becomes null
		const processToKill = this.process;
		const pid = processToKill.pid;

		return new Promise((resolve) => {
			const cleanup = () => {
				this.process = null;
				this.connected = false;
				resolve();
			};

			// Set up exit handler
			processToKill.once('exit', cleanup);

			// Try graceful shutdown first
			this.adapter.log.debug(`Sending SIGTERM to bridge process (PID: ${pid})`);
			processToKill.kill('SIGTERM');

			// Force kill after timeout if process doesn't exit
			const forceKillTimeout = setTimeout(() => {
				// Check if process is still running by checking if it still exists
				if (processToKill.exitCode === null && !processToKill.killed) {
					this.adapter.log.warn(`Force killing bridge process (PID: ${pid})`);
					try {
						processToKill.kill('SIGKILL');
					} catch (error) {
						this.adapter.log.error(`Failed to force kill process: ${error.message}`);
					}
				}
			}, 3000); // Reduced from 5000 to 3000ms

			// Clean up the timeout when process exits
			processToKill.once('exit', () => {
				clearTimeout(forceKillTimeout);
			});
		});
	}

	/**
	 * Check if bridge is connected
	 * @returns {boolean}
	 */
	isConnected() {
		return this.connected && this.process !== null;
	}
}

module.exports = EEBusBridge;
