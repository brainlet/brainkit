package brainkit

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/brainlet/brainkit/bus"
)

// hostTestSource prepends the @external declarations to AS user code.
// AS passes strings natively — no manual encoding needed.
func hostTestSource(userCode string) string {
	return `
@external("host", "log")
declare function host_log(msg: string, level: i32): void;

@external("host", "call_tool")
declare function host_call_tool(name: string, argsJSON: string): string;

@external("host", "call_agent")
declare function host_call_agent(name: string, prompt: string): string;

@external("host", "get_state")
declare function host_get_state(key: string): string;

@external("host", "set_state")
declare function host_set_state(key: string, value: string): void;

@external("host", "bus_send")
declare function host_bus_send(topic: string, payloadJSON: string): void;

` + userCode
}

func TestWASMHost_Log(t *testing.T) {
	kit := newTestKitNoKey(t)
	ctx := context.Background()

	source := hostTestSource(`
export function run(): i32 {
  host_log("hello from wasm", 1);
  return 0;
}
`)

	_, err := kit.EvalTS(ctx, "compile.ts", `
		await wasm.compile(`+"`"+source+"`"+`, { name: "log-test", runtime: "incremental" });
	`)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}

	result, err := kit.EvalTS(ctx, "run.ts", `
		var r = await wasm.run("log-test");
		return JSON.stringify(r);
	`)
	if err != nil {
		t.Fatalf("run: %v", err)
	}

	var runResult struct{ ExitCode int `json:"exitCode"` }
	json.Unmarshal([]byte(result), &runResult)
	if runResult.ExitCode != 0 {
		t.Errorf("exit code = %d, want 0", runResult.ExitCode)
	}
	t.Logf("WASM host.log: exit=%d", runResult.ExitCode)
}

func TestWASMHost_State(t *testing.T) {
	kit := newTestKitNoKey(t)
	ctx := context.Background()

	source := hostTestSource(`
export function run(): i32 {
  host_set_state("counter", "42");

  const val = host_get_state("counter");
  if (val == "42") return 0; // success
  if (val == "") return 1;   // not found
  return 2;                  // wrong value
}
`)

	_, err := kit.EvalTS(ctx, "compile.ts", `
		await wasm.compile(`+"`"+source+"`"+`, { name: "state-test", runtime: "incremental" });
	`)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}

	result, err := kit.EvalTS(ctx, "run.ts", `
		var r = await wasm.run("state-test");
		return JSON.stringify(r);
	`)
	if err != nil {
		t.Fatalf("run: %v", err)
	}

	var runResult struct{ ExitCode int `json:"exitCode"` }
	json.Unmarshal([]byte(result), &runResult)
	if runResult.ExitCode != 0 {
		t.Errorf("exit code = %d, want 0 (state round-trip failed)", runResult.ExitCode)
	}
	t.Logf("WASM host state: exit=%d", runResult.ExitCode)
}

func TestWASMHost_BusSend(t *testing.T) {
	kit := newTestKitNoKey(t)
	ctx := context.Background()

	received := make(chan string, 1)
	kit.Bus.Subscribe("wasm.test.*", func(msg bus.Message) {
		received <- string(msg.Payload)
	})

	source := hostTestSource(`
export function run(): i32 {
  host_bus_send("wasm.test.ping", '{"message":"hello"}');
  return 0;
}
`)

	_, err := kit.EvalTS(ctx, "compile.ts", `
		await wasm.compile(`+"`"+source+"`"+`, { name: "bus-test", runtime: "incremental" });
	`)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}

	result, err := kit.EvalTS(ctx, "run.ts", `
		var r = await wasm.run("bus-test");
		return JSON.stringify(r);
	`)
	if err != nil {
		t.Fatalf("run: %v", err)
	}

	var runResult struct{ ExitCode int `json:"exitCode"` }
	json.Unmarshal([]byte(result), &runResult)
	if runResult.ExitCode != 0 {
		t.Errorf("exit code = %d, want 0", runResult.ExitCode)
	}

	select {
	case msg := <-received:
		t.Logf("Bus received: %s", msg)
	default:
		t.Log("Bus message not received synchronously")
	}

	t.Logf("WASM host bus_send: exit=%d", runResult.ExitCode)
}

func TestWASMImport_Resolution(t *testing.T) {
	kit := newTestKitNoKey(t)
	ctx := context.Background()

	// This source imports from "wasm" — should resolve via ~lib/wasm injection
	source := `
import { wasmLibVersion } from "wasm";

export function run(): i32 {
  return wasmLibVersion();
}
`
	_, err := kit.EvalTS(ctx, "compile.ts", `
		await wasm.compile(`+"`"+source+"`"+`, { name: "import-test", runtime: "stub" });
	`)
	if err != nil {
		t.Fatalf("compile with import: %v", err)
	}

	result, err := kit.EvalTS(ctx, "run.ts", `
		var r = await wasm.run("import-test");
		return JSON.stringify(r);
	`)
	if err != nil {
		t.Fatalf("run: %v", err)
	}

	var rr struct{ ExitCode int `json:"exitCode"` }
	json.Unmarshal([]byte(result), &rr)
	if rr.ExitCode != 1 {
		t.Errorf("exitCode = %d, want 1 (wasmLibVersion)", rr.ExitCode)
	}
	t.Logf("import resolution: exitCode=%d", rr.ExitCode)
}

func TestWASMHost_HasState(t *testing.T) {
	kit := newTestKitNoKey(t)
	ctx := context.Background()

	source := hostTestSource(`
@external("host", "has_state")
declare function host_has_state(key: string): i32;

export function run(): i32 {
  // 1. has_state for non-existent key -> 0
  if (host_has_state("missing") != 0) return 1;

  // 2. Set a key, then has_state -> 1
  host_set_state("exists", "val");
  if (host_has_state("exists") != 1) return 2;

  // 3. Set to empty string, has_state still -> 1
  host_set_state("empty", "");
  if (host_has_state("empty") != 1) return 3;

  return 0;
}
`)
	_, err := kit.EvalTS(ctx, "compile.ts", `
		await wasm.compile(`+"`"+source+"`"+`, { name: "has-state-test", runtime: "incremental" });
	`)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}

	result, err := kit.EvalTS(ctx, "run.ts", `
		var r = await wasm.run("has-state-test");
		return JSON.stringify(r);
	`)
	if err != nil {
		t.Fatalf("run: %v", err)
	}

	var runResult struct{ ExitCode int `json:"exitCode"` }
	json.Unmarshal([]byte(result), &runResult)
	if runResult.ExitCode != 0 {
		t.Errorf("exitCode = %d, want 0 (subtest %d failed)", runResult.ExitCode, runResult.ExitCode)
	}
}
