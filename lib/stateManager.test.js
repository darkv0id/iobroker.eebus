'use strict';

const { expect } = require('chai');
const sinon = require('sinon');
const StateManager = require('./stateManager');

describe('StateManager', () => {
	let adapter;
	let stateManager;

	beforeEach(() => {
		// Create mock adapter
		adapter = {
			log: {
				info: sinon.stub(),
				warn: sinon.stub(),
				error: sinon.stub(),
				debug: sinon.stub(),
			},
			setObjectNotExistsAsync: sinon.stub().resolves(),
			setState: sinon.stub().resolves(),
			delObjectAsync: sinon.stub().resolves(),
		};

		stateManager = new StateManager(adapter);
	});

	describe('createDevice', () => {
		it('should create device structure with valid SKI', async () => {
			const device = {
				ski: '69898d83b85363ab75428da04c4c31c52cf929f1',
				name: 'Test Device',
				address: '192.168.1.100',
				type: 'HeatPump',
			};

			await stateManager.createDevice(device);

			expect(adapter.setObjectNotExistsAsync.callCount).to.be.greaterThan(0);
			expect(stateManager.devices.has(device.ski)).to.be.true;
		});

		it('should reject device with invalid SKI', async () => {
			const device = {
				ski: 'invalid-ski',
				name: 'Test Device',
				address: '192.168.1.100',
				type: 'HeatPump',
			};

			try {
				await stateManager.createDevice(device);
				expect.fail('Should have thrown error');
			} catch (error) {
				expect(error.message).to.match(/Invalid SKI format/);
			}
		});

		it('should reject device with uppercase SKI', async () => {
			const device = {
				ski: '69898D83B85363AB75428DA04C4C31C52CF929F1',
				name: 'Test Device',
				address: '192.168.1.100',
				type: 'HeatPump',
			};

			try {
				await stateManager.createDevice(device);
				expect.fail('Should have thrown error');
			} catch (error) {
				expect(error.message).to.match(/Invalid SKI format/);
			}
		});

		it('should create device root folder', async () => {
			const device = {
				ski: '69898d83b85363ab75428da04c4c31c52cf929f1',
				name: 'Test Device',
				address: '192.168.1.100',
				type: 'HeatPump',
			};

			await stateManager.createDevice(device);

			const deviceCall = adapter.setObjectNotExistsAsync
				.getCalls()
				.find((call) => call.args[0] === `devices.${device.ski}`);

			expect(deviceCall).to.exist;
			expect(deviceCall.args[1].type).to.equal('device');
			expect(deviceCall.args[1].common.name).to.equal('Test Device');
			expect(deviceCall.args[1].native.ski).to.equal(device.ski);
		});

		it('should create info, measurements, and usecases channels', async () => {
			const device = {
				ski: '69898d83b85363ab75428da04c4c31c52cf929f1',
				name: 'Test Device',
				address: '192.168.1.100',
				type: 'HeatPump',
			};

			await stateManager.createDevice(device);

			const channels = adapter.setObjectNotExistsAsync
				.getCalls()
				.filter((call) => call.args[1].type === 'channel')
				.map((call) => call.args[0]);

			expect(channels).to.include(`devices.${device.ski}.info`);
			expect(channels).to.include(`devices.${device.ski}.measurements`);
			expect(channels).to.include(`devices.${device.ski}.usecases`);
		});
	});

	describe('ensureDeviceExists', () => {
		it('should not recreate existing device', async () => {
			const ski = '69898d83b85363ab75428da04c4c31c52cf929f1';
			const device = {
				ski,
				name: 'Test Device',
				address: '192.168.1.100',
				type: 'HeatPump',
			};

			await stateManager.createDevice(device);
			adapter.setObjectNotExistsAsync.resetHistory();

			await stateManager.ensureDeviceExists(ski);

			expect(adapter.setObjectNotExistsAsync.called).to.be.false;
		});

		it('should create minimal device structure for new device', async () => {
			const ski = '69898d83b85363ab75428da04c4c31c52cf929f1';

			await stateManager.ensureDeviceExists(ski);

			expect(stateManager.devices.has(ski)).to.be.true;
			expect(adapter.setObjectNotExistsAsync.callCount).to.be.greaterThan(0);
		});

		it('should reject invalid SKI', async () => {
			try {
				await stateManager.ensureDeviceExists('invalid');
				expect.fail('Should have thrown error');
			} catch (error) {
				expect(error.message).to.match(/Invalid SKI format/);
			}
		});
	});

	describe('updateMeasurements', () => {
		beforeEach(async () => {
			const device = {
				ski: '69898d83b85363ab75428da04c4c31c52cf929f1',
				name: 'Test Device',
				address: '192.168.1.100',
				type: 'HeatPump',
			};
			await stateManager.createDevice(device);
			adapter.setState.resetHistory();
		});

		it('should update power measurement from Go bridge format', async () => {
			const measurement = {
				type: 'power',
				value: 1390,
				unit: 'W',
				usecase: 'mpc',
			};

			await stateManager.updateMeasurements('69898d83b85363ab75428da04c4c31c52cf929f1', measurement);

			const powerCall = adapter.setState
				.getCalls()
				.find((call) =>
					call.args[0].includes('measurements.power.active'),
				);

			expect(powerCall).to.exist;
			expect(powerCall.args[1].val).to.equal(1390);
			expect(powerCall.args[1].ack).to.be.true;
		});

		it('should update energy measurement', async () => {
			const measurement = {
				type: 'energy',
				value: 5000,
				unit: 'Wh',
				usecase: 'mpc',
			};

			await stateManager.updateMeasurements('69898d83b85363ab75428da04c4c31c52cf929f1', measurement);

			const energyCall = adapter.setState
				.getCalls()
				.find((call) =>
					call.args[0].includes('measurements.energy.consumed'),
				);

			expect(energyCall).to.exist;
			expect(energyCall.args[1].val).to.equal(5000);
		});

		it('should update voltage measurement', async () => {
			const measurement = {
				type: 'voltage',
				value: 230,
				unit: 'V',
			};

			await stateManager.updateMeasurements('69898d83b85363ab75428da04c4c31c52cf929f1', measurement);

			const voltageCall = adapter.setState
				.getCalls()
				.find((call) =>
					call.args[0].includes('measurements.voltage.L1'),
				);

			expect(voltageCall).to.exist;
			expect(voltageCall.args[1].val).to.equal(230);
		});

		it('should update current measurement', async () => {
			const measurement = {
				type: 'current',
				value: 6.5,
				unit: 'A',
			};

			await stateManager.updateMeasurements('69898d83b85363ab75428da04c4c31c52cf929f1', measurement);

			const currentCall = adapter.setState
				.getCalls()
				.find((call) =>
					call.args[0].includes('measurements.current.L1'),
				);

			expect(currentCall).to.exist;
			expect(currentCall.args[1].val).to.equal(6.5);
		});

		it('should update frequency measurement', async () => {
			const measurement = {
				type: 'frequency',
				value: 50,
				unit: 'Hz',
			};

			await stateManager.updateMeasurements('69898d83b85363ab75428da04c4c31c52cf929f1', measurement);

			const frequencyCall = adapter.setState
				.getCalls()
				.find((call) => call.args[0].includes('measurements.frequency'));

			expect(frequencyCall).to.exist;
			expect(frequencyCall.args[1].val).to.equal(50);
		});

		it('should handle nested measurement format', async () => {
			const measurement = {
				power: {
					active: 1390,
					reactive: 200,
					apparent: 1400,
				},
			};

			await stateManager.updateMeasurements('69898d83b85363ab75428da04c4c31c52cf929f1', measurement);

			const activePowerCall = adapter.setState
				.getCalls()
				.find((call) =>
					call.args[0].includes('measurements.power.active'),
				);
			const reactivePowerCall = adapter.setState
				.getCalls()
				.find((call) =>
					call.args[0].includes('measurements.power.reactive'),
				);

			expect(activePowerCall).to.exist;
			expect(activePowerCall.args[1].val).to.equal(1390);
			expect(reactivePowerCall).to.exist;
			expect(reactivePowerCall.args[1].val).to.equal(200);
		});

		it('should warn for unknown device', async () => {
			const measurement = {
				type: 'power',
				value: 1390,
			};

			await stateManager.updateMeasurements('0123456789abcdef0123456789abcdef01234567', measurement);

			expect(adapter.log.warn.calledWith(sinon.match(/unknown device/))).to.be.true;
		});

		it('should ignore null values', async () => {
			const measurement = {
				type: 'power',
				value: null,
			};

			await stateManager.updateMeasurements('69898d83b85363ab75428da04c4c31c52cf929f1', measurement);

			expect(adapter.setState.called).to.be.false;
		});

		it('should ignore undefined values', async () => {
			const measurement = {
				type: 'power',
				value: undefined,
			};

			await stateManager.updateMeasurements('69898d83b85363ab75428da04c4c31c52cf929f1', measurement);

			expect(adapter.setState.called).to.be.false;
		});
	});

	describe('updateConnectionState', () => {
		beforeEach(async () => {
			const device = {
				ski: '69898d83b85363ab75428da04c4c31c52cf929f1',
				name: 'Test Device',
				address: '192.168.1.100',
				type: 'HeatPump',
			};
			await stateManager.createDevice(device);
			adapter.setState.resetHistory();
		});

		it('should update connection state to true', async () => {
			await stateManager.updateConnectionState('69898d83b85363ab75428da04c4c31c52cf929f1', true);

			const connectionCall = adapter.setState
				.getCalls()
				.find((call) => call.args[0].includes('info.connected'));

			expect(connectionCall).to.exist;
			expect(connectionCall.args[1].val).to.be.true;
			expect(connectionCall.args[1].ack).to.be.true;
		});

		it('should update connection state to false', async () => {
			await stateManager.updateConnectionState('69898d83b85363ab75428da04c4c31c52cf929f1', false);

			const connectionCall = adapter.setState
				.getCalls()
				.find((call) => call.args[0].includes('info.connected'));

			expect(connectionCall).to.exist;
			expect(connectionCall.args[1].val).to.be.false;
		});

		it('should reject invalid SKI', async () => {
			try {
				await stateManager.updateConnectionState('invalid', true);
				expect.fail('Should have thrown error');
			} catch (error) {
				expect(error.message).to.match(/Invalid SKI format/);
			}
		});
	});

	describe('removeDevice', () => {
		beforeEach(async () => {
			const device = {
				ski: '69898d83b85363ab75428da04c4c31c52cf929f1',
				name: 'Test Device',
				address: '192.168.1.100',
				type: 'HeatPump',
			};
			await stateManager.createDevice(device);
		});

		it('should remove device and delete objects', async () => {
			await stateManager.removeDevice('69898d83b85363ab75428da04c4c31c52cf929f1');

			expect(stateManager.devices.has('69898d83b85363ab75428da04c4c31c52cf929f1')).to.be.false;
			expect(
				adapter.delObjectAsync.calledWith('devices.69898d83b85363ab75428da04c4c31c52cf929f1', {
					recursive: true,
				}),
			).to.be.true;
		});

		it('should reject invalid SKI', async () => {
			try {
				await stateManager.removeDevice('invalid');
				expect.fail('Should have thrown error');
			} catch (error) {
				expect(error.message).to.match(/Invalid SKI format/);
			}
		});

		it('should handle deletion errors gracefully', async () => {
			adapter.delObjectAsync.rejects(new Error('Deletion failed'));

			await stateManager.removeDevice('69898d83b85363ab75428da04c4c31c52cf929f1');

			expect(adapter.log.error.calledWith(sinon.match(/Failed to remove device/))).to.be.true;
			expect(stateManager.devices.has('69898d83b85363ab75428da04c4c31c52cf929f1')).to.be.false;
		});
	});

	describe('getDevices', () => {
		it('should return empty array when no devices', () => {
			const devices = stateManager.getDevices();
			expect(devices).to.be.an('array').that.is.empty;
		});

		it('should return all tracked devices', async () => {
			const device1 = {
				ski: '69898d83b85363ab75428da04c4c31c52cf929f1',
				name: 'Device 1',
				address: '192.168.1.100',
				type: 'HeatPump',
			};
			const device2 = {
				ski: '0123456789abcdef0123456789abcdef01234567',
				name: 'Device 2',
				address: '192.168.1.101',
				type: 'WallBox',
			};

			await stateManager.createDevice(device1);
			await stateManager.createDevice(device2);

			const devices = stateManager.getDevices();
			expect(devices).to.have.lengthOf(2);
			expect(devices).to.deep.include(device1);
			expect(devices).to.deep.include(device2);
		});
	});

	describe('hasDevice', () => {
		beforeEach(async () => {
			const device = {
				ski: '69898d83b85363ab75428da04c4c31c52cf929f1',
				name: 'Test Device',
				address: '192.168.1.100',
				type: 'HeatPump',
			};
			await stateManager.createDevice(device);
		});

		it('should return true for existing device', () => {
			expect(stateManager.hasDevice('69898d83b85363ab75428da04c4c31c52cf929f1')).to.be.true;
		});

		it('should return false for non-existing device', () => {
			expect(stateManager.hasDevice('0123456789abcdef0123456789abcdef01234567')).to.be.false;
		});

		it('should reject invalid SKI', () => {
			expect(() => stateManager.hasDevice('invalid')).to.throw(Error, /Invalid SKI format/);
		});
	});
});
