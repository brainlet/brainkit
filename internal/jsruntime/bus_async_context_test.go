package jsruntime

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/brainlet/brainkit/internal/jsbridge"
)

func TestBusRuntimeBindsAsyncContextForStreamCallbacks(t *testing.T) {
	b, err := jsbridge.New(jsbridge.Config{},
		jsbridge.Timers(),
		jsbridge.Scheduling(),
		jsbridge.NodeCompat(),
	)
	if err != nil {
		t.Fatalf("bridge: %v", err)
	}
	defer b.Close()

	bootstrap, err := b.Eval("bus-async-context-bootstrap.js", `
		globalThis.__kit_bridgeControl = {};
		globalThis.__go_resource_register = function() {};
		globalThis.__go_brainkit_bus_emit = function() {};
		globalThis.__go_brainkit_bus_reply = function() {};
		globalThis.__go_brainkit_subscribe = function(topic) { return "sub:" + topic; };
		globalThis.__go_brainkit_unsubscribe = function() {};
		globalThis.__go_brainkit_bus_call_stream = function(topic, payload, targetNamespace, timeoutMs, streamID) {
			return new Promise(function(resolve) {
				setTimeout(function() {
					var entry = globalThis.__kit_bus_stream_handlers[streamID];
					if (entry && typeof entry.onChunk === "function") {
						entry.onChunk({ value: "chunk" }, { topic: topic });
					}
					resolve(JSON.stringify({ value: "final" }));
				}, 1);
			});
		};
	`)
	if err != nil {
		t.Fatalf("bootstrap: %v", err)
	}
	bootstrap.Free()

	loaded, err := b.Eval("bus.js", busJS)
	if err != nil {
		t.Fatalf("load bus.js: %v", err)
	}
	loaded.Free()

	val, err := b.EvalAsync("bus-async-context.js", `(async function() {
		var als = new async_hooks.AsyncLocalStorage();
		var seen = [];
		await new Promise(function(resolve, reject) {
			als.run("bus-store", function() {
				__kit_bus.callStream("topic.stream", {}, {
					timeoutMs: 100,
					onChunk: function(chunk, msg) {
						seen.push("chunk:" + als.getStore() + ":" + chunk.value + ":" + msg.topic);
					},
				}).then(function(final) {
					seen.push("final:" + als.getStore() + ":" + final.value);
					resolve();
				}, reject);
			});
		});

		var subID = "";
		als.run("sub-store", function() {
			subID = __kit_bus.subscribe("topic.subscribe", function(msg) {
				seen.push("sub:" + als.getStore() + ":" + msg.payload.value);
			});
		});
		__bus_subs[subID]({ payload: { value: "message" }, topic: "topic.subscribe" });

		return JSON.stringify({ seen: seen, after: typeof als.getStore() });
	})()`)
	if err != nil {
		t.Fatalf("EvalAsync: %v", err)
	}
	defer val.Free()

	var parsed struct {
		Seen  []string `json:"seen"`
		After string   `json:"after"`
	}
	if err := json.Unmarshal([]byte(val.String()), &parsed); err != nil {
		t.Fatalf("json: %v\n%s", err, val.String())
	}
	want := []string{
		"chunk:bus-store:chunk:topic.stream",
		"final:bus-store:final",
		"sub:sub-store:message",
	}
	if fmt.Sprint(parsed.Seen) != fmt.Sprint(want) || parsed.After != "undefined" {
		t.Fatalf("bus async context propagation failed: got %+v want seen %v", parsed, want)
	}
}
