'use strict';

/*
 * Created with @iobroker/create-adapter v3.1.2
 */

// The adapter-core module gives you access to the core ioBroker functions
// you need to create an adapter
const utils = require('@iobroker/adapter-core');

// Load EEBus bridge and state manager
const EEBusBridge = require('./lib/eebusBridge');
const StateManager = require('./lib/stateManager');

class Eebus extends utils.Adapter {
	/**
	 * @param {Partial<utils.AdapterOptions>} [options] - Adapter options
	 */
	constructor(options) {
		super({
			...options,
			name: 'eebus',
		});
		this.on('ready', this.onReady.bind(this));
		this.on('stateChange', this.onStateChange.bind(this));
		// this.on('objectChange', this.onObjectChange.bind(this));
		// this.on('message', this.onMessage.bind(this));
		this.on('unload', this.onUnload.bind(this));

		// Initialize EEBus components
		this.bridge = null;
		this.stateManager = null;
	}

	/**
	 * Is called when databases are connected and adapter received configuration.
	 */
	async onReady() {
		// Initialize your adapter here
		this.log.info('Starting EEBus adapter');

		// Create info states
		await this.createInfoStates();

		// Initialize state manager
		this.stateManager = new StateManager(this);

		// Initialize and start EEBus bridge
		try {
			this.bridge = new EEBusBridge(this, {
				// Bridge options can be added here
			});

			// Set up bridge event handlers
			this.setupBridgeHandlers();

			// Start the bridge
			await this.bridge.start();

			this.log.info('EEBus adapter started successfully');
		} catch (error) {
			this.log.error(`Failed to start EEBus bridge: ${error.message}`);
			this.log.info('The adapter will continue running but EEBus functionality will be unavailable');
			this.log.info('Please ensure the EEBus bridge binary is available');

			// Set connection state to false
			await this.setState('info.connection', false, true);
		}
	}

	/**
	 * Create info states for adapter status
	 */
	async createInfoStates() {
		await this.setObjectNotExistsAsync('info', {
			type: 'channel',
			common: {
				name: 'Information',
			},
			native: {},
		});

		await this.setObjectNotExistsAsync('info.connection', {
			type: 'state',
			common: {
				name: 'Bridge Connection',
				type: 'boolean',
				role: 'indicator.connected',
				read: true,
				write: false,
			},
			native: {},
		});

		await this.setObjectNotExistsAsync('info.discovery', {
			type: 'state',
			common: {
				name: 'Discovery Active',
				type: 'boolean',
				role: 'indicator',
				read: true,
				write: false,
			},
			native: {},
		});

		// Initialize with default values
		await this.setState('info.connection', false, true);
		await this.setState('info.discovery', false, true);
	}

	/**
	 * Set up event handlers for the EEBus bridge
	 */
	setupBridgeHandlers() {
		// Bridge connected
		this.bridge.on('connected', () => {
			this.log.info('Bridge connected');
			this.setState('info.connection', true, true);
		});

		// Bridge disconnected
		this.bridge.on('disconnected', () => {
			this.log.warn('Bridge disconnected');
			this.setState('info.connection', false, true);
		});

		// Bridge failed permanently
		this.bridge.on('failed', () => {
			this.log.error('Bridge failed permanently after multiple restart attempts');
			this.setState('info.connection', false, true);
		});

		// Device discovered
		this.bridge.on('deviceDiscovered', async (device) => {
			this.log.info(`Device discovered: ${device.name} (${device.ski})`);
			await this.stateManager.createDevice(device);
		});

		// Device connected
		this.bridge.on('deviceConnected', async (payload) => {
			this.log.info(`Device connected: ${payload.ski}`);
			await this.stateManager.updateConnectionState(payload.ski, true);
		});

		// Device disconnected
		this.bridge.on('deviceDisconnected', async (payload) => {
			this.log.info(`Device disconnected: ${payload.ski}`);
			await this.stateManager.updateConnectionState(payload.ski, false);
		});

		// Measurement update
		this.bridge.on('measurementUpdate', async (payload) => {
			this.log.debug(`Measurement update for ${payload.ski}`);
			await this.stateManager.updateMeasurements(payload.ski, payload.measurements);
		});

		// Generic event handler for debugging
		this.bridge.on('event', (action, payload) => {
			this.log.debug(`Bridge event: ${action}, payload: ${JSON.stringify(payload)}`);
		});

		// Error handler
		this.bridge.on('error', (error) => {
			this.log.error(`Bridge error: ${error.message}`);
		});
	}

	/**
	 * Is called when adapter shuts down - callback has to be called under any circumstances!
	 *
	 * @param {() => void} callback - Callback function
	 */
	async onUnload(callback) {
		try {
			this.log.info('Shutting down EEBus adapter');

			// Stop the bridge if running
			if (this.bridge) {
				await this.bridge.stop();
			}

			// Update connection state
			await this.setState('info.connection', false, true);
			await this.setState('info.discovery', false, true);

			this.log.info('EEBus adapter stopped');
			callback();
		} catch (error) {
			this.log.error(`Error during unloading: ${error.message}`);
			callback();
		}
	}

	// If you need to react to object changes, uncomment the following block and the corresponding line in the constructor.
	// You also need to subscribe to the objects with `this.subscribeObjects`, similar to `this.subscribeStates`.
	// /**
	//  * Is called if a subscribed object changes
	//  * @param {string} id
	//  * @param {ioBroker.Object | null | undefined} obj
	//  */
	// onObjectChange(id, obj) {
	// 	if (obj) {
	// 		// The object was changed
	// 		this.log.info(`object ${id} changed: ${JSON.stringify(obj)}`);
	// 	} else {
	// 		// The object was deleted
	// 		this.log.info(`object ${id} deleted`);
	// 	}
	// }

	/**
	 * Is called if a subscribed state changes
	 *
	 * @param {string} id - State ID
	 * @param {ioBroker.State | null | undefined} state - State object
	 */
	onStateChange(id, state) {
		if (state) {
			// The state was changed
			this.log.info(`state ${id} changed: ${state.val} (ack = ${state.ack})`);

			if (state.ack === false) {
				// This is a command from the user (e.g., from the UI or other adapter)
				// and should be processed by the adapter
				this.log.info(`User command received for ${id}: ${state.val}`);

				// TODO: Add your control logic here
			}
		} else {
			// The object was deleted or the state value has expired
			this.log.info(`state ${id} deleted`);
		}
	}
	// If you need to accept messages in your adapter, uncomment the following block and the corresponding line in the constructor.
	// /**
	//  * Some message was sent to this instance over message box. Used by email, pushover, text2speech, ...
	//  * Using this method requires "common.messagebox" property to be set to true in io-package.json
	//  * @param {ioBroker.Message} obj
	//  */
	// onMessage(obj) {
	// 	if (typeof obj === 'object' && obj.message) {
	// 		if (obj.command === 'send') {
	// 			// e.g. send email or pushover or whatever
	// 			this.log.info('send command');

	// 			// Send response in callback if required
	// 			if (obj.callback) this.sendTo(obj.from, obj.command, 'Message received', obj.callback);
	// 		}
	// 	}
	// }
}

if (require.main !== module) {
	// Export the constructor in compact mode
	/**
	 * @param {Partial<utils.AdapterOptions>} [options] - Adapter options
	 */
	module.exports = options => new Eebus(options);
} else {
	// otherwise start the instance directly
	new Eebus();
}
