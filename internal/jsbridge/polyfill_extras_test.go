package jsbridge

import (
	"strings"
	"testing"
)

func TestEventEmitter_SetMaxListeners(t *testing.T) {
	b := newTestBridge(t, Events())
	val, err := b.Eval("test.js", `
		var ee = new EventEmitter();
		ee.setMaxListeners(50);
		JSON.stringify({
			max: ee.getMaxListeners(),
			captureRejections: EventEmitter.captureRejections,
			defaultMax: EventEmitter.defaultMaxListeners,
		});
	`)
	if err != nil {
		t.Fatalf("EventEmitter statics: %v", err)
	}
	defer val.Free()
	s := val.String()
	if !strings.Contains(s, `"max":50`) {
		t.Fatalf("expected max:50, got: %s", s)
	}
	if !strings.Contains(s, `"captureRejections":false`) {
		t.Fatalf("expected captureRejections:false, got: %s", s)
	}
	t.Logf("EventEmitter statics: %s", s)
}

func TestEventEmitter_PrependListener(t *testing.T) {
	b := newTestBridge(t, Events())
	val, err := b.Eval("test.js", `
		var ee = new EventEmitter();
		var order = [];
		ee.on("test", function() { order.push("normal"); });
		ee.prependListener("test", function() { order.push("prepended"); });
		ee.emit("test");
		JSON.stringify({ order: order });
	`)
	if err != nil {
		t.Fatalf("prependListener: %v", err)
	}
	defer val.Free()
	s := val.String()
	if !strings.Contains(s, `["prepended","normal"]`) {
		t.Fatalf("expected prepended first, got: %s", s)
	}
}

func TestEventEmitter_EventNames(t *testing.T) {
	b := newTestBridge(t, Events())
	val, err := b.Eval("test.js", `
		var ee = new EventEmitter();
		ee.on("foo", function(){});
		ee.on("bar", function(){});
		JSON.stringify({ names: ee.eventNames().sort() });
	`)
	if err != nil {
		t.Fatalf("eventNames: %v", err)
	}
	defer val.Free()
	s := val.String()
	if !strings.Contains(s, `["bar","foo"]`) {
		t.Fatalf("expected [bar,foo], got: %s", s)
	}
}

func TestCrypto_GetFips(t *testing.T) {
	b := newTestBridge(t, Crypto())
	val, err := b.Eval("test.js", `
		JSON.stringify({
			fips: globalThis.crypto.getFips(),
			hasFn: typeof globalThis.crypto.getFips === "function",
		});
	`)
	if err != nil {
		t.Fatalf("getFips: %v", err)
	}
	defer val.Free()
	s := val.String()
	if !strings.Contains(s, `"fips":0`) {
		t.Fatalf("expected fips:0, got: %s", s)
	}
}

func TestProcess_EmitWarning(t *testing.T) {
	b := newTestBridge(t, Process())
	val, err := b.Eval("test.js", `
		// Should not throw
		globalThis.process.emitWarning("test warning");
		JSON.stringify({
			hasFn: typeof globalThis.process.emitWarning === "function",
			hasGetuid: typeof globalThis.process.getuid === "function",
			hasGetgid: typeof globalThis.process.getgid === "function",
			uid: globalThis.process.getuid(),
		});
	`)
	if err != nil {
		t.Fatalf("process.emitWarning: %v", err)
	}
	defer val.Free()
	s := val.String()
	if !strings.Contains(s, `"hasFn":true`) {
		t.Fatalf("expected hasFn:true, got: %s", s)
	}
	t.Logf("process extras: %s", s)
}

func TestOS_Release(t *testing.T) {
	b := newTestBridge(t, OS())
	val, err := b.Eval("test.js", `
		var r = globalThis.os.release();
		JSON.stringify({ release: r, notStub: r !== "0.0.0" });
	`)
	if err != nil {
		t.Fatalf("os.release: %v", err)
	}
	defer val.Free()
	s := val.String()
	if !strings.Contains(s, `"notStub":true`) {
		t.Fatalf("expected real release, got: %s", s)
	}
	t.Logf("os.release: %s", s)
}

func TestBuffer_PoolSize(t *testing.T) {
	b := newTestBridge(t, Encoding(), Buffer())
	val, err := b.Eval("test.js", `
		JSON.stringify({ poolSize: Buffer.poolSize });
	`)
	if err != nil {
		t.Fatalf("Buffer.poolSize: %v", err)
	}
	defer val.Free()
	s := val.String()
	if !strings.Contains(s, `"poolSize":8192`) {
		t.Fatalf("expected poolSize:8192, got: %s", s)
	}
}
