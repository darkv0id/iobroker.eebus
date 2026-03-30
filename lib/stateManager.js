'use strict';

/**
 * State Manager - Manages ioBroker objects and states for EEBus devices
 *
 * This class handles:
 * - Creating device object hierarchies
 * - Updating state values from EEBus data
 * - Mapping EEBus data model to ioBroker structure
 * - Data type conversions and validations
 */
class StateManager {
	/**
	 * @param {object} adapter - ioBroker adapter instance
	 */
	constructor(adapter) {
		this.adapter = adapter;
		this.devices = new Map(); // SKI -> device info
		this.adapter.log.info('State Manager initialized');
	}

	/**
	 * Create device objects for a discovered device
	 * @param {object} device - Device information from EEBus
	 * @returns {Promise<void>}
	 */
	async createDevice(device) {
		const deviceId = this.sanitizeSKI(device.ski);

		this.adapter.log.info(`Creating device structure for ${device.name} (${deviceId})`);

		// Store device info
		this.devices.set(deviceId, device);

		// Create device root folder
		await this.adapter.setObjectNotExistsAsync(`devices.${deviceId}`, {
			type: 'device',
			common: {
				name: device.name || 'Unknown Device',
			},
			native: {
				ski: device.ski,
				address: device.address,
				type: device.type,
			},
		});

		// Create info channel
		await this.createDeviceInfo(deviceId, device);

		// Create measurements channel
		await this.createMeasurementsStructure(deviceId);

		// Create use cases channel
		await this.createUseCasesStructure(deviceId);

		this.adapter.log.info(`Device structure created for ${deviceId}`);
	}

	/**
	 * Create device info states
	 * @param {string} deviceId - Sanitized device ID
	 * @param {object} device - Device information
	 * @returns {Promise<void>}
	 */
	async createDeviceInfo(deviceId, device) {
		const channelId = `devices.${deviceId}.info`;

		await this.adapter.setObjectNotExistsAsync(channelId, {
			type: 'channel',
			common: {
				name: 'Device Information',
			},
			native: {},
		});

		// Device name
		await this.createState(channelId, 'name', {
			name: 'Device Name',
			type: 'string',
			role: 'info.name',
			read: true,
			write: false,
		});
		await this.adapter.setState(`${channelId}.name`, device.name || 'Unknown', true);

		// Device type
		await this.createState(channelId, 'type', {
			name: 'Device Type',
			type: 'string',
			role: 'info.type',
			read: true,
			write: false,
		});
		await this.adapter.setState(`${channelId}.type`, device.type || 'Unknown', true);

		// Manufacturer (placeholder)
		await this.createState(channelId, 'manufacturer', {
			name: 'Manufacturer',
			type: 'string',
			role: 'info.manufacturer',
			read: true,
			write: false,
		});

		// Model (placeholder)
		await this.createState(channelId, 'model', {
			name: 'Model',
			type: 'string',
			role: 'info.model',
			read: true,
			write: false,
		});

		// Serial number (placeholder)
		await this.createState(channelId, 'serial', {
			name: 'Serial Number',
			type: 'string',
			role: 'info.serial',
			read: true,
			write: false,
		});

		// Connection state
		await this.createState(channelId, 'connected', {
			name: 'Connected',
			type: 'boolean',
			role: 'indicator.connected',
			read: true,
			write: false,
		});
		await this.adapter.setState(`${channelId}.connected`, false, true);
	}

	/**
	 * Create measurements channel structure
	 * @param {string} deviceId - Sanitized device ID
	 * @returns {Promise<void>}
	 */
	async createMeasurementsStructure(deviceId) {
		const channelId = `devices.${deviceId}.measurements`;

		await this.adapter.setObjectNotExistsAsync(channelId, {
			type: 'channel',
			common: {
				name: 'Measurements',
			},
			native: {},
		});

		// Power measurements
		await this.createPowerStates(deviceId);

		// Energy measurements
		await this.createEnergyStates(deviceId);

		// Voltage measurements
		await this.createVoltageStates(deviceId);

		// Current measurements
		await this.createCurrentStates(deviceId);

		// Frequency
		await this.createState(`${channelId}.frequency`, 'frequency', {
			name: 'Frequency',
			type: 'number',
			role: 'value.frequency',
			read: true,
			write: false,
			unit: 'Hz',
		});
	}

	/**
	 * Create power measurement states
	 * @param {string} deviceId - Sanitized device ID
	 * @returns {Promise<void>}
	 */
	async createPowerStates(deviceId) {
		const channelId = `devices.${deviceId}.measurements.power`;

		await this.adapter.setObjectNotExistsAsync(channelId, {
			type: 'channel',
			common: {
				name: 'Power',
			},
			native: {},
		});

		await this.createState(channelId, 'active', {
			name: 'Active Power',
			type: 'number',
			role: 'value.power',
			read: true,
			write: false,
			unit: 'W',
		});

		await this.createState(channelId, 'reactive', {
			name: 'Reactive Power',
			type: 'number',
			role: 'value.power.reactive',
			read: true,
			write: false,
			unit: 'VAr',
		});

		await this.createState(channelId, 'apparent', {
			name: 'Apparent Power',
			type: 'number',
			role: 'value.power.apparent',
			read: true,
			write: false,
			unit: 'VA',
		});
	}

	/**
	 * Create energy measurement states
	 * @param {string} deviceId - Sanitized device ID
	 * @returns {Promise<void>}
	 */
	async createEnergyStates(deviceId) {
		const channelId = `devices.${deviceId}.measurements.energy`;

		await this.adapter.setObjectNotExistsAsync(channelId, {
			type: 'channel',
			common: {
				name: 'Energy',
			},
			native: {},
		});

		await this.createState(channelId, 'consumed', {
			name: 'Energy Consumed',
			type: 'number',
			role: 'value.energy',
			read: true,
			write: false,
			unit: 'Wh',
		});

		await this.createState(channelId, 'produced', {
			name: 'Energy Produced',
			type: 'number',
			role: 'value.energy.produced',
			read: true,
			write: false,
			unit: 'Wh',
		});
	}

	/**
	 * Create voltage measurement states
	 * @param {string} deviceId - Sanitized device ID
	 * @returns {Promise<void>}
	 */
	async createVoltageStates(deviceId) {
		const channelId = `devices.${deviceId}.measurements.voltage`;

		await this.adapter.setObjectNotExistsAsync(channelId, {
			type: 'channel',
			common: {
				name: 'Voltage',
			},
			native: {},
		});

		for (const phase of ['L1', 'L2', 'L3']) {
			await this.createState(channelId, phase, {
				name: `Voltage ${phase}`,
				type: 'number',
				role: 'value.voltage',
				read: true,
				write: false,
				unit: 'V',
			});
		}
	}

	/**
	 * Create current measurement states
	 * @param {string} deviceId - Sanitized device ID
	 * @returns {Promise<void>}
	 */
	async createCurrentStates(deviceId) {
		const channelId = `devices.${deviceId}.measurements.current`;

		await this.adapter.setObjectNotExistsAsync(channelId, {
			type: 'channel',
			common: {
				name: 'Current',
			},
			native: {},
		});

		for (const phase of ['L1', 'L2', 'L3']) {
			await this.createState(channelId, phase, {
				name: `Current ${phase}`,
				type: 'number',
				role: 'value.current',
				read: true,
				write: false,
				unit: 'A',
			});
		}
	}

	/**
	 * Create use cases channel structure
	 * @param {string} deviceId - Sanitized device ID
	 * @returns {Promise<void>}
	 */
	async createUseCasesStructure(deviceId) {
		const channelId = `devices.${deviceId}.usecases`;

		await this.adapter.setObjectNotExistsAsync(channelId, {
			type: 'channel',
			common: {
				name: 'Use Cases',
			},
			native: {},
		});

		await this.createState(channelId, 'mgcp', {
			name: 'Monitoring of Grid Connection Point',
			type: 'boolean',
			role: 'indicator',
			read: true,
			write: false,
		});

		await this.createState(channelId, 'mpc', {
			name: 'Monitoring of Power Consumption',
			type: 'boolean',
			role: 'indicator',
			read: true,
			write: false,
		});
	}

	/**
	 * Create a state object
	 * @param {string} channelId - Channel ID
	 * @param {string} stateId - State ID
	 * @param {object} common - Common state properties
	 * @returns {Promise<void>}
	 */
	async createState(channelId, stateId, common) {
		await this.adapter.setObjectNotExistsAsync(`${channelId}.${stateId}`, {
			type: 'state',
			common,
			native: {},
		});
	}

	/**
	 * Update measurements for a device
	 * @param {string} ski - Device SKI
	 * @param {object} measurements - Measurement data from EEBus
	 * @returns {Promise<void>}
	 */
	async updateMeasurements(ski, measurements) {
		const deviceId = this.sanitizeSKI(ski);

		if (!this.devices.has(deviceId)) {
			this.adapter.log.warn(`Received measurements for unknown device: ${deviceId}`);
			return;
		}

		this.adapter.log.debug(`Updating measurements for ${deviceId}`);

		// Handle individual measurement format from Go bridge
		// Format: { type: "power", value: 1390, unit: "W", usecase: "mpc" }
		if (measurements.type && measurements.value !== undefined) {
			const { type, value } = measurements;

			switch (type) {
				case 'power':
					// Map power measurement to active power
					await this.updateValue(deviceId, 'measurements.power.active', value);
					this.adapter.log.debug(`Updated power: ${value}W`);
					break;

				case 'energy':
					// Map energy measurement to consumed energy
					await this.updateValue(deviceId, 'measurements.energy.consumed', value);
					this.adapter.log.debug(`Updated energy: ${value}Wh`);
					break;

				case 'voltage':
					// Map voltage measurement to L1 (for now, phase info might come later)
					await this.updateValue(deviceId, 'measurements.voltage.L1', value);
					this.adapter.log.debug(`Updated voltage: ${value}V`);
					break;

				case 'current':
					// Map current measurement to L1 (for now, phase info might come later)
					await this.updateValue(deviceId, 'measurements.current.L1', value);
					this.adapter.log.debug(`Updated current: ${value}A`);
					break;

				case 'frequency':
					await this.updateValue(deviceId, 'measurements.frequency', value);
					this.adapter.log.debug(`Updated frequency: ${value}Hz`);
					break;

				default:
					this.adapter.log.warn(`Unknown measurement type: ${type}`);
			}
			return;
		}

		// Handle nested format (for backwards compatibility)
		// Update power measurements
		if (measurements.power) {
			await this.updateValue(deviceId, 'measurements.power.active', measurements.power.active);
			await this.updateValue(deviceId, 'measurements.power.reactive', measurements.power.reactive);
			await this.updateValue(deviceId, 'measurements.power.apparent', measurements.power.apparent);
		}

		// Update energy measurements
		if (measurements.energy) {
			await this.updateValue(deviceId, 'measurements.energy.consumed', measurements.energy.consumed);
			await this.updateValue(deviceId, 'measurements.energy.produced', measurements.energy.produced);
		}

		// Update voltage measurements
		if (measurements.voltage) {
			await this.updateValue(deviceId, 'measurements.voltage.L1', measurements.voltage.L1);
			await this.updateValue(deviceId, 'measurements.voltage.L2', measurements.voltage.L2);
			await this.updateValue(deviceId, 'measurements.voltage.L3', measurements.voltage.L3);
		}

		// Update current measurements
		if (measurements.current) {
			await this.updateValue(deviceId, 'measurements.current.L1', measurements.current.L1);
			await this.updateValue(deviceId, 'measurements.current.L2', measurements.current.L2);
			await this.updateValue(deviceId, 'measurements.current.L3', measurements.current.L3);
		}

		// Update frequency
		if (measurements.frequency !== undefined) {
			await this.updateValue(deviceId, 'measurements.frequency', measurements.frequency);
		}
	}

	/**
	 * Update device connection state
	 * @param {string} ski - Device SKI
	 * @param {boolean} connected - Connection state
	 * @returns {Promise<void>}
	 */
	async updateConnectionState(ski, connected) {
		const deviceId = this.sanitizeSKI(ski);
		await this.updateValue(deviceId, 'info.connected', connected);
		this.adapter.log.info(`Device ${deviceId} connection state: ${connected}`);
	}

	/**
	 * Update a single state value
	 * @param {string} deviceId - Sanitized device ID
	 * @param {string} path - State path relative to device
	 * @param {any} value - New value
	 * @returns {Promise<void>}
	 */
	async updateValue(deviceId, path, value) {
		if (value === undefined || value === null) {
			return;
		}

		const stateId = `devices.${deviceId}.${path}`;
		await this.adapter.setState(stateId, { val: value, ack: true });
	}

	/**
	 * Remove a device and all its states
	 * @param {string} ski - Device SKI
	 * @returns {Promise<void>}
	 */
	async removeDevice(ski) {
		const deviceId = this.sanitizeSKI(ski);

		this.adapter.log.info(`Removing device ${deviceId}`);

		// Remove from tracked devices
		this.devices.delete(deviceId);

		// Delete all objects for this device
		try {
			await this.adapter.delObjectAsync(`devices.${deviceId}`, { recursive: true });
			this.adapter.log.info(`Device ${deviceId} removed successfully`);
		} catch (error) {
			this.adapter.log.error(`Failed to remove device ${deviceId}: ${error.message}`);
		}
	}

	/**
	 * Sanitize SKI for use as ioBroker ID
	 * @param {string} ski - Original SKI
	 * @returns {string} Sanitized ID
	 */
	sanitizeSKI(ski) {
		// Replace any characters that aren't alphanumeric, underscore, or hyphen
		return ski.replace(/[^a-zA-Z0-9_-]/g, '_');
	}

	/**
	 * Get list of all tracked devices
	 * @returns {Array<object>} Array of device information
	 */
	getDevices() {
		return Array.from(this.devices.values());
	}

	/**
	 * Check if a device exists
	 * @param {string} ski - Device SKI
	 * @returns {boolean}
	 */
	hasDevice(ski) {
		const deviceId = this.sanitizeSKI(ski);
		return this.devices.has(deviceId);
	}
}

module.exports = StateManager;
