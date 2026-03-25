package jsbridge

import (
	"context"
	"testing"
	"time"

	quickjs "github.com/buke/quickjs-go"
)

// TestAsyncIteratorPipeSchedule reproduces the MongoDB driver hang:
// GoSocket → pipe → Transform._transform → push → emit('data') →
// async iterator eventHandler → resolve Promise → for await advances.
//
// The critical path: data delivery happens inside a ctx.Schedule callback
// (simulating GoSocket's read loop), and the for-await loop needs multiple
// rounds of JS_ExecutePendingJob to propagate through nested async generators.
func TestAsyncIteratorPipeSchedule(t *testing.T) {
	b := newTestBridge(t, Console(), Encoding(), Streams(), Timers(), Events())

	// Set up the MongoDB-like pipe + async iterator pattern.
	// This mirrors connection.ts: socket.pipe(SizedMessageTransform) + onData() + readMany()
	setup, err := b.Eval("setup.js", `
		// onData async iterator (mirrors node-mongodb-native/src/cmap/wire_protocol/on_data.ts)
		function onData(emitter) {
			var unconsumedEvents = [];
			var unconsumedPromises = [];
			var finished = false;

			function eventHandler(value) {
				if (unconsumedPromises.length > 0) {
					var p = unconsumedPromises.shift();
					p.resolve({ value: value, done: false });
				} else {
					unconsumedEvents.push(value);
				}
			}

			emitter.on("data", eventHandler);

			return {
				next: function() {
					if (unconsumedEvents.length > 0) {
						return Promise.resolve({ value: unconsumedEvents.shift(), done: false });
					}
					if (finished) {
						return Promise.resolve({ value: undefined, done: true });
					}
					var resolve, reject;
					var promise = new Promise(function(res, rej) { resolve = res; reject = rej; });
					unconsumedPromises.push({ resolve: resolve, reject: reject });
					return promise;
				},
				return: function() {
					finished = true;
					emitter.removeListener("data", eventHandler);
					return Promise.resolve({ value: undefined, done: true });
				},
				[Symbol.asyncIterator]: function() { return this; }
			};
		}

		// EventEmitter for the transform output (simulates SizedMessageTransform)
		var transform = new EventEmitter();
		transform._paused = true;
		transform._buffer = [];
		transform.write = function(chunk) {
			this.push(chunk);
			return true;
		};
		transform.push = function(data) {
			if (this._paused) { this._buffer.push(data); return true; }
			this.emit("data", data);
			return true;
		};
		transform.pause = function() { this._paused = true; };
		transform.resume = function() {
			this._paused = false;
			while (this._buffer.length) this.emit("data", this._buffer.shift());
		};
		transform.removeListener = function(ev, fn) {
			var a = this._e[ev];
			if (a) this._e[ev] = a.filter(function(f) { return f !== fn; });
			return this;
		};

		// Source emitter (simulates GoSocket)
		var source = new EventEmitter();
		// pipe: source → transform (mirrors GoSocket.pipe())
		source.on("data", function(chunk) {
			transform.write(chunk);
		});

		// readMany equivalent (mirrors Connection.readMany - async generator)
		async function* readMany() {
			try {
				var dataEvents = onData(transform);
				transform.resume();
				for await (var message of dataEvents) {
					yield message;
					return; // like !response.moreToCome
				}
			} finally {
				transform.pause();
			}
		}

		// sendWire equivalent (mirrors Connection.sendWire - async generator wrapping readMany)
		async function* sendWire() {
			for await (var response of readMany()) {
				yield response;
			}
		}

		// command equivalent (mirrors Connection.command - consumes sendWire)
		async function command() {
			for await (var document of sendWire()) {
				return document;
			}
			throw new Error("no response");
		}

		globalThis.__test_source = source;
		globalThis.__test_command = command;
		"ok"
	`)
	if err != nil {
		t.Fatalf("setup: %v", err)
	}
	setup.Free()

	// Schedule data delivery from a goroutine after a delay.
	// This simulates GoSocket's read loop: Go reads TCP data → ctx.Schedule → emit('data')
	b.Go(func(goCtx context.Context) {
		time.Sleep(100 * time.Millisecond)
		b.ctx.Schedule(func(qctx *quickjs.Context) {
			qctx.Eval(`globalThis.__test_source.emit("data", "hello-from-schedule")`)
		})
	})

	// Run the async command — should await data from Schedule
	done := make(chan string, 1)
	go func() {
		val, err := b.EvalAsync("command.js", `(async () => {
			var result = await globalThis.__test_command();
			return String(result);
		})()`)
		if err != nil {
			done <- "ERROR: " + err.Error()
			return
		}
		r := val.String()
		val.Free()
		done <- r
	}()

	select {
	case result := <-done:
		if result == "hello-from-schedule" {
			t.Logf("SUCCESS: for-await + Schedule works through pipe chain")
		} else {
			t.Fatalf("unexpected result: %s", result)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("TIMEOUT: for-await with Schedule-delivered data hung (reproduces MongoDB issue)")
	}
}

// TestAsyncIteratorWithDecompress adds an async decompressResponse call
// in the sendWire layer (like MongoDB driver does), to test whether
// the additional async step in the generator chain causes a hang.
func TestAsyncIteratorWithDecompress(t *testing.T) {
	b := newTestBridge(t, Console(), Encoding(), Streams(), Timers(), Events())

	setup, err := b.Eval("setup.js", `
		function onData(emitter) {
			var unconsumedEvents = [];
			var unconsumedPromises = [];
			var finished = false;
			function eventHandler(value) {
				if (unconsumedPromises.length > 0) {
					unconsumedPromises.shift().resolve({ value: value, done: false });
				} else {
					unconsumedEvents.push(value);
				}
			}
			emitter.on("data", eventHandler);
			return {
				next: function() {
					if (unconsumedEvents.length > 0) return Promise.resolve({ value: unconsumedEvents.shift(), done: false });
					if (finished) return Promise.resolve({ value: undefined, done: true });
					var resolve, reject;
					var promise = new Promise(function(res, rej) { resolve = res; reject = rej; });
					unconsumedPromises.push({ resolve: resolve, reject: reject });
					return promise;
				},
				return: function() { finished = true; emitter.removeListener("data", eventHandler); return Promise.resolve({value:undefined,done:true}); },
				[Symbol.asyncIterator]: function() { return this; }
			};
		}

		var transform = new EventEmitter();
		transform._paused = true;
		transform._buffer = [];
		transform.write = function(chunk) { this.push(chunk); return true; };
		transform.push = function(data) {
			if (this._paused) { this._buffer.push(data); return true; }
			this.emit("data", data);
			return true;
		};
		transform.pause = function() { this._paused = true; };
		transform.resume = function() { this._paused = false; while (this._buffer.length) this.emit("data", this._buffer.shift()); };
		transform.removeListener = function(ev, fn) { var a = this._e[ev]; if (a) this._e[ev] = a.filter(function(f) { return f !== fn; }); return this; };

		var source = new EventEmitter();
		source.on("data", function(chunk) { transform.write(chunk); });

		// Add async decompressResponse (like MongoDB's decompressResponse)
		async function decompressResponse(message) {
			// In the non-compressed case, just return the message with moreToCome=false
			return { data: message, moreToCome: false };
		}

		async function* readMany() {
			try {
				var dataEvents = onData(transform);
				transform.resume();
				for await (var message of dataEvents) {
					yield message;
					return;
				}
			} finally {
				transform.pause();
			}
		}

		// sendWire adds the async decompressResponse call
		async function* sendWire() {
			for await (var response of readMany()) {
				var decompressed = await decompressResponse(response);
				yield decompressed;
				if (!decompressed.moreToCome) return;
			}
		}

		// sendCommand adds session handling (another layer)
		async function* sendCommand() {
			for await (var document of sendWire()) {
				yield document;
			}
		}

		async function command() {
			for await (var document of sendCommand()) {
				return document;
			}
			throw new Error("no response");
		}

		globalThis.__test_source = source;
		globalThis.__test_command = command;
		"ok"
	`)
	if err != nil {
		t.Fatalf("setup: %v", err)
	}
	setup.Free()

	b.Go(func(goCtx context.Context) {
		time.Sleep(100 * time.Millisecond)
		b.ctx.Schedule(func(qctx *quickjs.Context) {
			qctx.Eval(`globalThis.__test_source.emit("data", "mongo-response-bytes")`)
		})
	})

	done := make(chan string, 1)
	go func() {
		val, err := b.EvalAsync("command.js", `(async () => {
			var result = await globalThis.__test_command();
			return JSON.stringify(result);
		})()`)
		if err != nil {
			done <- "ERROR: " + err.Error()
			return
		}
		r := val.String()
		val.Free()
		done <- r
	}()

	select {
	case result := <-done:
		t.Logf("SUCCESS: result = %s", result)
	case <-time.After(5 * time.Second):
		t.Fatal("TIMEOUT: nested async generators with decompressResponse hung")
	}
}
