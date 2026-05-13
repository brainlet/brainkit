package jsbridge

import "testing"

func TestNodeReadable_PushAndData(t *testing.T) {
	b := newTestBridge(t, Console(), Encoding(), Events(), NodeStreams())
	val, err := b.Eval("test.js", `
		var chunks = [];
		var r = new globalThis.stream.Readable();
		r.on("data", function(chunk) { chunks.push(chunk); });
		r.push("hello");
		r.push("world");
		r.push(null);
		JSON.stringify(chunks);
	`)
	if err != nil {
		t.Fatal(err)
	}
	defer val.Free()
	if val.String() != `["hello","world"]` {
		t.Errorf("got %s", val.String())
	}
}

func TestNodeReadable_Pipe(t *testing.T) {
	b := newTestBridge(t, Console(), Encoding(), Events(), NodeStreams())
	val, err := b.Eval("test.js", `
		var NS = globalThis.stream;
		var output = [];
		var src = new NS.Readable();
		var transform = new NS.Transform({
			transform: function(chunk, enc, cb) { this.push(chunk + "!"); cb(); }
		});
		var dest = new NS.Writable({
			write: function(chunk, enc, cb) { output.push(chunk); cb(); }
		});
		src.pipe(transform).pipe(dest);
		src.push("a");
		src.push("b");
		src.push(null);
		JSON.stringify(output);
	`)
	if err != nil {
		t.Fatal(err)
	}
	defer val.Free()
	if val.String() != `["a!","b!"]` {
		t.Errorf("got %s", val.String())
	}
}

func TestNodeStream_ConsecutiveAsyncIterators(t *testing.T) {
	b := newTestBridge(t, Console(), Encoding(), Events(), Timers(), NodeStreams())
	// Simulates: command1 reads from transform via for-await, exits, command2 reads next message.
	// Data for command2 must NOT be lost when command1's iterator exits.
	setup, err := b.Eval("setup.js", `
		var NS = globalThis.stream;
		var src = new NS.Readable();
		var transform = new NS.Transform({
			transform: function(chunk, enc, cb) { this.push(chunk); cb(); }
		});
		src.pipe(transform);

		async function command() {
			for await (var msg of transform) {
				return msg; // exits after first message, like conn.command()
			}
		}
		globalThis.__test_src = src;
		globalThis.__test_command = command;
		"ok"
	`)
	if err != nil {
		t.Fatal(err)
	}
	setup.Free()

	// Push two messages synchronously (simulates TCP delivering hello + saslStart responses)
	val, err := b.EvalAsync("test.js", `(async () => {
		globalThis.__test_src.push("hello-response");
		globalThis.__test_src.push("saslStart-response");
		var r1 = await globalThis.__test_command();
		var r2 = await globalThis.__test_command();
		return JSON.stringify([r1, r2]);
	})()`)
	if err != nil {
		t.Fatal(err)
	}
	defer val.Free()
	if val.String() != `["hello-response","saslStart-response"]` {
		t.Fatalf("got %s — second command lost data (MongoDB SCRAM bug)", val.String())
	}
}

func TestNodeTransform_WriteThrough(t *testing.T) {
	b := newTestBridge(t, Console(), Encoding(), Events(), NodeStreams())
	val, err := b.Eval("test.js", `
		var NS = globalThis.stream;
		var output = [];
		var pt = new NS.PassThrough();
		pt.on("data", function(chunk) { output.push(chunk); });
		pt.write("x");
		pt.write("y");
		pt.end();
		JSON.stringify(output);
	`)
	if err != nil {
		t.Fatal(err)
	}
	defer val.Free()
	if val.String() != `["x","y"]` {
		t.Errorf("got %s", val.String())
	}
}

func TestNodeStreamPromises(t *testing.T) {
	b := newTestBridge(t, Console(), Encoding(), Events(), NodeStreams())
	val, err := b.Eval("stream-promises.js", `(function() {
		var NS = globalThis.stream;
		var pt = new NS.PassThrough();
		var finished = NS.promises.finished(pt);
		var pipeline = NS.promises.pipeline();
		pt.end("ok");
		return JSON.stringify({
			hasFinished: typeof NS.promises.finished === "function",
			hasPipeline: typeof NS.promises.pipeline === "function",
			finishedThen: !!finished && typeof finished.then === "function",
			pipelineThen: !!pipeline && typeof pipeline.then === "function"
		});
	})()`)
	if err != nil {
		t.Fatal(err)
	}
	defer val.Free()
	if val.String() != `{"hasFinished":true,"hasPipeline":true,"finishedThen":true,"pipelineThen":true}` {
		t.Fatalf("stream promises = %s", val.String())
	}
}

func TestNodeStream_EventEmitterAlias(t *testing.T) {
	b := newTestBridge(t, Console(), Encoding(), Events(), NodeStreams())
	val, err := b.Eval("stream-event-emitter.js", `(function() {
		var NS = globalThis.stream;
		var emitter = new NS.EventEmitter();
		var stream = new NS.Stream();
		var readable = new NS.Readable();
		return JSON.stringify({
			eventEmitter: NS.EventEmitter === globalThis.EventEmitter,
			streamIsEmitter: stream instanceof globalThis.EventEmitter,
			readableIsStream: readable instanceof NS.Stream
		});
	})()`)
	if err != nil {
		t.Fatal(err)
	}
	defer val.Free()
	if val.String() != `{"eventEmitter":true,"streamIsEmitter":true,"readableIsStream":true}` {
		t.Fatalf("stream EventEmitter alias = %s", val.String())
	}
}

func TestNodeReadable_BufferWhenPaused(t *testing.T) {
	b := newTestBridge(t, Console(), Encoding(), Events(), NodeStreams())
	val, err := b.Eval("test.js", `
		var NS = globalThis.stream;
		var r = new NS.Readable();
		// Push data BEFORE adding listener — should buffer
		r.push("buffered1");
		r.push("buffered2");
		// Now add listener — should flush buffered data
		var chunks = [];
		r.on("data", function(chunk) { chunks.push(chunk); });
		r.push("live");
		r.push(null);
		JSON.stringify(chunks);
	`)
	if err != nil {
		t.Fatal(err)
	}
	defer val.Free()
	if val.String() != `["buffered1","buffered2","live"]` {
		t.Errorf("got %s", val.String())
	}
}
