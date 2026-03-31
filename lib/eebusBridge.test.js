'use strict';

const { expect } = require('chai');
const sinon = require('sinon');
const EventEmitter = require('events');
const childProcess = require('child_process');

describe('EEBusBridge', () => {
	let adapter;
	let bridge;
	let spawnStub;
	let mockProcess;
	let EEBusBridge;

	before(() => {
		// Stub spawn globally before loading the module
		spawnStub = sinon.stub(childProcess, 'spawn');
		// Now load the module with the stub in place
		EEBusBridge = require('./eebusBridge');
	});

	after(() => {
		if (spawnStub) {
			spawnStub.restore();
		}
	});

	beforeEach(() => {
		// Reset the stub for each test
		spawnStub.reset();

		// Create mock adapter
		adapter = {
			log: {
				info: sinon.stub(),
				warn: sinon.stub(),
				error: sinon.stub(),
				debug: sinon.stub(),
			},
		};

		// Create mock child process
		mockProcess = new EventEmitter();
		mockProcess.stdin = {
			write: sinon.stub().returns(true),
			end: sinon.stub(),
		};
		mockProcess.stdout = new EventEmitter();
		mockProcess.stderr = new EventEmitter();
		mockProcess.kill = sinon.stub().returns(true);
		mockProcess.killed = false;
		mockProcess.exitCode = null;
		mockProcess.pid = 12345;

		// Set the stub to return our mock process
		spawnStub.returns(mockProcess);
	});

	afterEach(() => {
		if (bridge && bridge.process) {
			bridge.process = null;
			bridge.connected = false;
		}
		spawnStub.restore();
	});

	describe('constructor', () => {
		it('should initialize with default options', () => {
			bridge = new EEBusBridge(adapter);

			expect(bridge.adapter).to.equal(adapter);
			expect(bridge.connected).to.be.false;
			expect(bridge.restartCount).to.equal(0);
			expect(bridge.messageQueue).to.be.an.instanceof(Map);
			expect(bridge.options.restartDelay).to.equal(5000);
			expect(bridge.options.maxRestarts).to.equal(5);
		});

		it('should accept custom options', () => {
			bridge = new EEBusBridge(adapter, {
				restartDelay: 10000,
				maxRestarts: 10,
				messageTimeout: 60000,
			});

			expect(bridge.options.restartDelay).to.equal(10000);
			expect(bridge.options.maxRestarts).to.equal(10);
			expect(bridge.options.messageTimeout).to.equal(60000);
		});

		it('should accept custom binary path', () => {
			bridge = new EEBusBridge(adapter, {
				binaryPath: '/custom/path/to/binary',
			});

			expect(bridge.options.binaryPath).to.equal('/custom/path/to/binary');
		});
	});

	describe('detectBinaryPath', () => {
		it('should detect binary path for current platform', () => {
			bridge = new EEBusBridge(adapter);

			const binaryPath = bridge.detectBinaryPath();

			expect(binaryPath).to.be.a('string');
			expect(binaryPath).to.include('bin');
			expect(binaryPath).to.include('eebus-bridge');
		});

		it('should include .exe extension on Windows', () => {
			const originalPlatform = process.platform;
			Object.defineProperty(process, 'platform', { value: 'win32' });

			bridge = new EEBusBridge(adapter);
			const binaryPath = bridge.detectBinaryPath();

			expect(binaryPath).to.include('.exe');

			Object.defineProperty(process, 'platform', { value: originalPlatform });
		});
	});

	describe('start', () => {
		beforeEach(() => {
			bridge = new EEBusBridge(adapter, {
				binaryPath: '/test/binary',
			});
		});

		it('should spawn the bridge process', async () => {
			const startPromise = bridge.start();

			// Simulate successful connection with proper event format
			setTimeout(() => {
				mockProcess.stdout.emit('data', JSON.stringify({ type: 'event', action: 'ready' }) + '\n');
			}, 10);

			await startPromise;

			expect(spawnStub.calledOnce).to.be.true;
			expect(spawnStub.firstCall.args[0]).to.equal('/test/binary');
			expect(bridge.connected).to.be.true;
		});

		it('should not start if already running', async () => {
			bridge.process = { fake: 'process' };

			await bridge.start();

			expect(spawnStub.called).to.be.false;
			expect(adapter.log.warn.calledWith('Bridge process already running')).to.be.true;
		});

		it('should reject if spawn fails', async () => {
			const startPromise = bridge.start();

			// Simulate spawn error
			setTimeout(() => {
				mockProcess.emit('error', new Error('Binary not found'));
			}, 10);

			try {
				await startPromise;
				expect.fail('Should have thrown error');
			} catch (error) {
				expect(error.message).to.equal('Binary not found');
			}
		});

		it('should reject if process exits immediately', async () => {
			const startPromise = bridge.start();

			// Simulate immediate exit
			setTimeout(() => {
				mockProcess.emit('exit', 1, null);
			}, 10);

			try {
				await startPromise;
				expect.fail('Should have thrown error');
			} catch (error) {
				expect(error.message).to.include('exited with code 1');
			}
		});

		it('should set up connection on ready message', async () => {
			const startPromise = bridge.start();

			// Simulate ready message with proper event format
			setTimeout(() => {
				mockProcess.stdout.emit('data', JSON.stringify({ type: 'event', action: 'ready' }) + '\n');
			}, 10);

			await startPromise;

			expect(bridge.connected).to.be.true;
			expect(bridge.process).to.equal(mockProcess);
		});
	});

	describe('handleMessage', () => {
		beforeEach(() => {
			bridge = new EEBusBridge(adapter, {
				binaryPath: '/test/binary',
			});
		});

		it('should handle response messages', () => {
			const message = {
				type: 'response',
				id: 1,
				payload: { success: true },
			};

			// Add pending request
			const resolve = sinon.stub();
			bridge.messageQueue.set(1, {
				resolve,
				reject: sinon.stub(),
				timeout: setTimeout(() => {}, 1000),
			});

			bridge.handleMessage(JSON.stringify(message));

			expect(resolve.calledOnce).to.be.true;
			expect(resolve.firstCall.args[0]).to.deep.equal({ success: true });
			expect(bridge.messageQueue.has(1)).to.be.false;
		});

		it('should handle event messages', (done) => {
			const message = {
				type: 'event',
				action: 'testEvent',
				payload: { data: 'test' },
			};

			bridge.on('testEvent', (payload) => {
				expect(payload).to.deep.equal({ data: 'test' });
				done();
			});

			bridge.handleMessage(JSON.stringify(message));
		});

		it('should handle deviceConnected event', (done) => {
			const message = {
				type: 'event',
				action: 'deviceConnected',
				payload: { ski: '69898d83b85363ab75428da04c4c31c52cf929f1' },
			};

			bridge.on('deviceConnected', (payload) => {
				expect(payload.ski).to.equal('69898d83b85363ab75428da04c4c31c52cf929f1');
				done();
			});

			bridge.handleMessage(JSON.stringify(message));
		});

		it('should handle invalid JSON gracefully', () => {
			bridge.handleMessage('invalid json {');

			expect(adapter.log.error.calledWith(sinon.match(/Failed to parse/))).to.be.true;
		});
	});

	describe('handleResponse', () => {
		beforeEach(() => {
			bridge = new EEBusBridge(adapter, {
				binaryPath: '/test/binary',
			});
		});

		it('should resolve pending request with result', () => {
			const resolve = sinon.stub();
			const reject = sinon.stub();
			const timeout = setTimeout(() => {}, 1000);

			bridge.messageQueue.set(1, { resolve, reject, timeout });

			bridge.handleResponse({
				id: 1,
				payload: { data: 'test' },
			});

			expect(resolve.calledOnce).to.be.true;
			expect(resolve.firstCall.args[0]).to.deep.equal({ data: 'test' });
			expect(reject.called).to.be.false;
			expect(bridge.messageQueue.has(1)).to.be.false;
		});

		it('should reject pending request with error', () => {
			const resolve = sinon.stub();
			const reject = sinon.stub();
			const timeout = setTimeout(() => {}, 1000);

			bridge.messageQueue.set(1, { resolve, reject, timeout });

			bridge.handleResponse({
				id: 1,
				error: 'Test error',
			});

			expect(reject.calledOnce).to.be.true;
			expect(reject.firstCall.args[0]).to.be.an.instanceof(Error);
			expect(reject.firstCall.args[0].message).to.equal('Test error');
			expect(resolve.called).to.be.false;
		});

		it('should ignore responses for non-existent requests', () => {
			bridge.handleResponse({
				id: 999,
				payload: { data: 'test' },
			});

			expect(adapter.log.warn.calledWith(sinon.match(/unknown message ID/))).to.be.true;
		});
	});

	describe('sendCommand', () => {
		beforeEach(async () => {
			bridge = new EEBusBridge(adapter, {
				binaryPath: '/test/binary',
				messageTimeout: 1000,
			});

			// Start and connect
			const startPromise = bridge.start();
			setTimeout(() => {
				mockProcess.stdout.emit('data', JSON.stringify({ type: 'event', action: 'ready' }) + '\n');
			}, 10);
			await startPromise;
		});

		it('should send command and wait for response', async () => {
			const commandPromise = bridge.sendCommand('testAction', { data: 'test' });

			// Verify command was written
			expect(mockProcess.stdin.write.called).to.be.true;
			const writtenData = JSON.parse(mockProcess.stdin.write.firstCall.args[0].trim());
			expect(writtenData.action).to.equal('testAction');
			expect(writtenData.payload).to.deep.equal({ data: 'test' });

			// Simulate response with proper payload field
			setTimeout(() => {
				mockProcess.stdout.emit(
					'data',
					JSON.stringify({
						type: 'response',
						id: writtenData.id,
						payload: { success: true },
					}) + '\n',
				);
			}, 10);

			const result = await commandPromise;
			expect(result).to.deep.equal({ success: true });
		});

		it('should reject if not connected', async () => {
			bridge.connected = false;

			try {
				await bridge.sendCommand('testAction');
				expect.fail('Should have thrown error');
			} catch (error) {
				expect(error.message).to.include('not connected');
			}
		});

		it('should timeout if no response', async () => {
			try {
				await bridge.sendCommand('testAction');
				expect.fail('Should have thrown error');
			} catch (error) {
				expect(error.message).to.include('timeout');
			}
		});

		it('should handle command errors', async () => {
			const commandPromise = bridge.sendCommand('testAction');

			const writtenData = JSON.parse(mockProcess.stdin.write.firstCall.args[0].trim());

			// Simulate error response
			setTimeout(() => {
				mockProcess.stdout.emit(
					'data',
					JSON.stringify({
						type: 'response',
						id: writtenData.id,
						error: 'Command failed',
					}) + '\n',
				);
			}, 10);

			try {
				await commandPromise;
				expect.fail('Should have thrown error');
			} catch (error) {
				expect(error.message).to.equal('Command failed');
			}
		});
	});

	describe('stop', () => {
		beforeEach(async () => {
			bridge = new EEBusBridge(adapter, {
				binaryPath: '/test/binary',
			});

			// Start and connect
			const startPromise = bridge.start();
			setTimeout(() => {
				mockProcess.stdout.emit('data', JSON.stringify({ type: 'event', action: 'ready' }) + '\n');
			}, 10);
			await startPromise;
		});

		it('should stop the bridge process gracefully', async () => {
			const stopPromise = bridge.stop();

			// Simulate process exit
			setTimeout(() => {
				mockProcess.exitCode = 0;
				mockProcess.emit('exit', 0, null);
			}, 10);

			await stopPromise;

			// stop() calls kill('SIGTERM'), not stdin.end()
			expect(mockProcess.kill.called).to.be.true;
			expect(mockProcess.kill.firstCall.args[0]).to.equal('SIGTERM');
			expect(bridge.connected).to.be.false;
			expect(bridge.process).to.be.null;
		});

		it('should do nothing if already stopped', async () => {
			bridge.process = null;

			await bridge.stop();

			// stop() just returns early when there's no process, no log message
			expect(adapter.log.info.calledWith('Stopping EEBus bridge process')).to.be.true;
		});

		it('should force kill if process does not exit', async () => {
			const stopPromise = bridge.stop();

			// First call is SIGTERM immediately
			expect(mockProcess.kill.callCount).to.be.greaterThan(0);
			expect(mockProcess.kill.firstCall.args[0]).to.equal('SIGTERM');

			// Simulate process not exiting gracefully - wait for force kill timeout (3000ms)
			setTimeout(() => {
				// After 3s timeout, should have called kill('SIGKILL')
				expect(mockProcess.kill.callCount).to.equal(2);
				expect(mockProcess.kill.secondCall.args[0]).to.equal('SIGKILL');

				// Now emit exit
				mockProcess.exitCode = 1;
				mockProcess.killed = true;
				mockProcess.emit('exit', 1, 'SIGKILL');
			}, 3100);

			await stopPromise;
		}).timeout(5000);
	});

	describe('handleStdout', () => {
		beforeEach(() => {
			bridge = new EEBusBridge(adapter, {
				binaryPath: '/test/binary',
			});
		});

		it('should handle complete JSON messages', () => {
			const handleMessageSpy = sinon.spy(bridge, 'handleMessage');

			bridge.handleStdout(Buffer.from(JSON.stringify({ type: 'ready' }) + '\n'));

			expect(handleMessageSpy.calledOnce).to.be.true;
			expect(handleMessageSpy.firstCall.args[0]).to.equal(JSON.stringify({ type: 'ready' }));
		});

		it('should buffer incomplete messages', () => {
			const handleMessageSpy = sinon.spy(bridge, 'handleMessage');

			// Send partial message
			bridge.handleStdout(Buffer.from('{"type":"re'));
			expect(handleMessageSpy.called).to.be.false;

			// Complete the message
			bridge.handleStdout(Buffer.from('ady"}\n'));
			expect(handleMessageSpy.calledOnce).to.be.true;
		});

		it('should handle multiple messages in one chunk', () => {
			const handleMessageSpy = sinon.spy(bridge, 'handleMessage');

			const messages = JSON.stringify({ type: 'ready' }) + '\n' + JSON.stringify({ type: 'event' }) + '\n';

			bridge.handleStdout(Buffer.from(messages));

			expect(handleMessageSpy.calledTwice).to.be.true;
		});
	});

	describe('isConnected', () => {
		it('should return connection status', () => {
			bridge = new EEBusBridge(adapter, {
				binaryPath: '/test/binary',
			});

			expect(bridge.isConnected()).to.be.false;

			// isConnected() checks both connected flag AND process existence
			bridge.connected = true;
			bridge.process = mockProcess;
			expect(bridge.isConnected()).to.be.true;
		});
	});
});
