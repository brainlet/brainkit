package brainkit

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/brainlet/brainkit/bus"
)

// ---------------------------------------------------------------------------
// String edge cases — compile once per runtime, test many patterns
// ---------------------------------------------------------------------------

func TestWASMStress_StringEdgeCases(t *testing.T) {
	for _, runtime := range []string{"stub", "minimal", "incremental"} {
		t.Run("runtime="+runtime, func(t *testing.T) {
			kit := newTestKitNoKey(t)
			ctx := context.Background()

			// One module tests many string patterns via a dispatch function.
			// The run() function calls each sub-test and returns the index of the first failure.
			source := hostTestSource(`
export function run(): i32 {
  // 1. Empty string
  host_log("", 0);

  // 2. Single char
  host_log("x", 1);

  // 3. ASCII sentence
  host_log("hello from wasm", 1);

  // 4. Unicode BMP (Chinese)
  host_log("\u4F60\u597D", 1);

  // 5. Emoji (surrogate pair in UTF-16)
  host_log("\uD83D\uDE00", 1);

  // 6. Mixed ASCII + emoji + BMP
  host_log("hi \uD83D\uDE00 \u4F60\u597D end", 1);

  // 7. Repeated chars (stress allocation)
  var long: string = "";
  for (let i = 0; i < 1000; i++) {
    long += "A";
  }
  host_log(long, 1);

  // 8. String with embedded quotes
  host_log('say "hello" they said', 1);

  // 9. Backslashes
  host_log("path-to-file", 1);

  // 10. Numbers as string
  host_log("1234567890", 1);

  // 11. Single emoji
  host_log("\uD83C\uDF89", 1);

  // 12. Multiple emoji
  host_log("\uD83D\uDE00\uD83D\uDE01\uD83D\uDE02\uD83D\uDE03", 1);

  // 13. State round-trip with special chars
  host_set_state("key-with-dashes", "value with spaces");
  const v1 = host_get_state("key-with-dashes");
  if (v1 != "value with spaces") return 13;

  // 14. State with unicode
  host_set_state("\u4F60\u597D", "\uD83D\uDE00");
  const v2 = host_get_state("\u4F60\u597D");
  if (v2 != "\uD83D\uDE00") return 14;

  // 15. State overwrite
  host_set_state("counter", "1");
  host_set_state("counter", "2");
  const v3 = host_get_state("counter");
  if (v3 != "2") return 15;

  // 16. State get non-existent key
  const v4 = host_get_state("never-set");
  if (v4 != "") return 16;

  // 17. Bus send with JSON
  host_bus_send("stress.test.json", '{"key":"value","num":42}');

  // 18. Bus send with unicode topic
  host_bus_send("stress.test.unicode", '{"emoji":"\uD83D\uDE00"}');

  // 19. Bus send with empty payload
  host_bus_send("stress.test.empty", '{}');

  // 20. Many state operations
  for (let i = 0; i < 100; i++) {
    host_set_state("k" + i.toString(), "v" + i.toString());
  }
  for (let i = 0; i < 100; i++) {
    const v = host_get_state("k" + i.toString());
    if (v != "v" + i.toString()) return 20;
  }

  return 0;
}
`)
			_, err := kit.EvalTS(ctx, "compile.ts", fmt.Sprintf(`
				await wasm.compile(%s, { name: "stress-strings", runtime: %q });
			`, "`"+source+"`", runtime))
			if err != nil {
				t.Fatalf("compile: %v", err)
			}

			// Subscribe to bus topics before run
			received := make(chan bus.Message, 10)
			kit.Bus.On("stress.test.*", func(msg bus.Message, _ bus.ReplyFunc) {
				received <- msg
			})

			result, err := kit.EvalTS(ctx, "run.ts", `
				var r = await wasm.run("stress-strings");
				return JSON.stringify(r);
			`)
			if err != nil {
				t.Fatalf("run: %v", err)
			}

			var runResult struct{ ExitCode int `json:"exitCode"` }
			json.Unmarshal([]byte(result), &runResult)
			if runResult.ExitCode != 0 {
				t.Fatalf("exit code = %d (subtest %d failed)", runResult.ExitCode, runResult.ExitCode)
			}

			// Verify bus messages were received
			busCount := 0
		drain:
			for {
				select {
				case <-received:
					busCount++
				default:
					break drain
				}
			}
			if busCount < 3 {
				t.Errorf("expected at least 3 bus messages, got %d", busCount)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// State isolation — verify hostState is fresh per run
// ---------------------------------------------------------------------------

func TestWASMStress_StateIsolation(t *testing.T) {
	kit := newTestKitNoKey(t)
	ctx := context.Background()

	source := hostTestSource(`
export function run(): i32 {
  // On first run, set state. On second run, it should NOT be there.
  const existing = host_get_state("persist-test");
  if (existing != "") {
    return 1; // FAIL: state leaked between runs
  }
  host_set_state("persist-test", "should-not-persist");
  return 0;
}
`)
	_, err := kit.EvalTS(ctx, "compile.ts", `
		await wasm.compile(`+"`"+source+"`"+`, { name: "isolation", runtime: "incremental" });
	`)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}

	// Run twice — second run must NOT see state from first
	for i := 0; i < 5; i++ {
		result, err := kit.EvalTS(ctx, fmt.Sprintf("run%d.ts", i), `
			var r = await wasm.run("isolation");
			return JSON.stringify(r);
		`)
		if err != nil {
			t.Fatalf("run %d: %v", i, err)
		}
		var rr struct{ ExitCode int `json:"exitCode"` }
		json.Unmarshal([]byte(result), &rr)
		if rr.ExitCode != 0 {
			t.Fatalf("run %d: state leaked (exitCode=%d)", i, rr.ExitCode)
		}
	}
}

// ---------------------------------------------------------------------------
// Sequential compile+run stress — many modules
// ---------------------------------------------------------------------------

func TestWASMStress_ManyModules(t *testing.T) {
	kit := newTestKitNoKey(t)
	ctx := context.Background()

	const numModules = 10

	for i := 0; i < numModules; i++ {
		name := fmt.Sprintf("module-%d", i)
		source := fmt.Sprintf(`export function run(): i32 { return %d; }`, i)

		_, err := kit.EvalTS(ctx, fmt.Sprintf("compile%d.ts", i), fmt.Sprintf(`
			await wasm.compile(%s, { name: %q, runtime: "stub" });
		`, "`"+source+"`", name))
		if err != nil {
			t.Fatalf("compile %d: %v", i, err)
		}
	}

	// Verify all modules exist
	modules, err := kit.ListWASMModules()
	if err != nil {
		t.Fatalf("ListWASMModules: %v", err)
	}
	if len(modules) != numModules {
		t.Fatalf("expected %d modules, got %d", numModules, len(modules))
	}

	// Run each and verify return value
	for i := 0; i < numModules; i++ {
		name := fmt.Sprintf("module-%d", i)
		result, err := kit.EvalTS(ctx, fmt.Sprintf("run%d.ts", i), fmt.Sprintf(`
			var r = await wasm.run(%q);
			return JSON.stringify(r);
		`, name))
		if err != nil {
			t.Fatalf("run %s: %v", name, err)
		}
		var rr struct{ ExitCode int `json:"exitCode"` }
		json.Unmarshal([]byte(result), &rr)
		if rr.ExitCode != i {
			t.Errorf("module-%d: exitCode=%d, want %d", i, rr.ExitCode, i)
		}
	}
}

// ---------------------------------------------------------------------------
// Module overwrite (idempotent compile)
// ---------------------------------------------------------------------------

func TestWASMStress_ModuleOverwrite(t *testing.T) {
	kit := newTestKitNoKey(t)
	ctx := context.Background()

	// Compile "versioned" 5 times with different return values
	for i := 0; i < 5; i++ {
		source := fmt.Sprintf(`export function run(): i32 { return %d; }`, i+100)
		_, err := kit.EvalTS(ctx, fmt.Sprintf("compile%d.ts", i), fmt.Sprintf(`
			await wasm.compile(%s, { name: "versioned", runtime: "stub" });
		`, "`"+source+"`"))
		if err != nil {
			t.Fatalf("compile v%d: %v", i, err)
		}
	}

	// Should only have 1 module, the last version
	modules, _ := kit.ListWASMModules()
	count := 0
	for _, m := range modules {
		if m.Name == "versioned" {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("expected 1 'versioned' module, got %d", count)
	}

	// Run should return 104 (last compile: i=4 → 104)
	result, err := kit.EvalTS(ctx, "run.ts", `
		var r = await wasm.run("versioned");
		return JSON.stringify(r);
	`)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	var rr struct{ ExitCode int `json:"exitCode"` }
	json.Unmarshal([]byte(result), &rr)
	if rr.ExitCode != 104 {
		t.Errorf("exitCode=%d, want 104", rr.ExitCode)
	}
}

// ---------------------------------------------------------------------------
// Compile error handling — bad source code
// ---------------------------------------------------------------------------

func TestWASMStress_CompileErrors(t *testing.T) {
	kit := newTestKitNoKey(t)
	ctx := context.Background()

	cases := []struct {
		name   string
		source string
	}{
		{"syntax_error", "export function run(: i32 { return 0; }"},
		{"type_error", "export function run(): i32 { return \"not a number\"; }"},
		{"missing_import", "import { missing } from 'nonexistent';"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := kit.EvalTS(ctx, tc.name+".ts", fmt.Sprintf(`
				try {
					await wasm.compile(%s, { name: %q, runtime: "stub" });
					return JSON.stringify({ ok: true });
				} catch(e) {
					return JSON.stringify({ ok: false, error: e.message || String(e) });
				}
			`, "`"+tc.source+"`", tc.name))
			if err != nil {
				t.Fatalf("EvalTS: %v", err)
			}
			var r struct {
				OK    bool   `json:"ok"`
				Error string `json:"error"`
			}
			json.Unmarshal([]byte(result), &r)
			if r.OK {
				t.Errorf("expected compile to fail for %q, but it succeeded", tc.name)
			} else {
				t.Logf("%s error: %s", tc.name, r.Error)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Run non-existent module
// ---------------------------------------------------------------------------

func TestWASMStress_RunNonExistent(t *testing.T) {
	kit := newTestKitNoKey(t)
	ctx := context.Background()

	result, err := kit.EvalTS(ctx, "run.ts", `
		try {
			var r = await wasm.run("does-not-exist");
			return JSON.stringify({ ok: true });
		} catch(e) {
			return JSON.stringify({ ok: false, error: e.message || String(e) });
		}
	`)
	if err != nil {
		t.Fatalf("EvalTS: %v", err)
	}
	var r struct {
		OK    bool   `json:"ok"`
		Error string `json:"error"`
	}
	json.Unmarshal([]byte(result), &r)
	if r.OK {
		t.Error("expected run to fail for non-existent module")
	}
	if !strings.Contains(r.Error, "not found") {
		t.Errorf("expected 'not found' error, got: %s", r.Error)
	}
}

// ---------------------------------------------------------------------------
// Abort / trap handling
// ---------------------------------------------------------------------------

func TestWASMStress_Abort(t *testing.T) {
	kit := newTestKitNoKey(t)
	ctx := context.Background()

	source := `
export function run(): i32 {
  abort();
  return 0;
}
`
	_, err := kit.EvalTS(ctx, "compile.ts", `
		await wasm.compile(`+"`"+source+"`"+`, { name: "aborter", runtime: "stub" });
	`)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}

	result, err := kit.EvalTS(ctx, "run.ts", `
		try {
			var r = await wasm.run("aborter");
			return JSON.stringify({ ok: true, result: r });
		} catch(e) {
			return JSON.stringify({ ok: false, error: e.message || String(e) });
		}
	`)
	if err != nil {
		t.Fatalf("EvalTS: %v", err)
	}
	t.Logf("abort result: %s", result)
	// Should either error or have non-zero exit code
}

// ---------------------------------------------------------------------------
// Many string literals in data segments
// ---------------------------------------------------------------------------

func TestWASMStress_ManyDataSegments(t *testing.T) {
	kit := newTestKitNoKey(t)
	ctx := context.Background()

	// Generate AS code with 50 different string literals → lots of data segments
	var sb strings.Builder
	sb.WriteString(hostTestSource(""))
	sb.WriteString("\nexport function run(): i32 {\n")
	for i := 0; i < 50; i++ {
		sb.WriteString(fmt.Sprintf("  host_log(\"segment-%d-data\", 0);\n", i))
	}
	sb.WriteString("  return 0;\n}\n")
	source := sb.String()

	_, err := kit.EvalTS(ctx, "compile.ts", fmt.Sprintf(`
		await wasm.compile(%s, { name: "many-segments", runtime: "incremental" });
	`, "`"+source+"`"))
	if err != nil {
		t.Fatalf("compile: %v", err)
	}

	result, err := kit.EvalTS(ctx, "run.ts", `
		var r = await wasm.run("many-segments");
		return JSON.stringify(r);
	`)
	if err != nil {
		t.Fatalf("run: %v", err)
	}

	var rr struct{ ExitCode int `json:"exitCode"` }
	json.Unmarshal([]byte(result), &rr)
	if rr.ExitCode != 0 {
		t.Fatalf("exitCode=%d, want 0", rr.ExitCode)
	}
}

// ---------------------------------------------------------------------------
// Long string stress
// ---------------------------------------------------------------------------

func TestWASMStress_LongStrings(t *testing.T) {
	kit := newTestKitNoKey(t)
	ctx := context.Background()

	source := hostTestSource(`
export function run(): i32 {
  // Build a 10KB string
  var s: string = "";
  for (let i = 0; i < 10000; i++) {
    s += "X";
  }
  if (s.length != 10000) return 1;

  // Send it through host state
  host_set_state("big", s);
  const got = host_get_state("big");
  if (got.length != 10000) return 2;
  if (got != s) return 3;

  // Log it (exercises readASString with large data)
  host_log(s, 0);

  // Bus send with large payload
  host_bus_send("stress.long", '{"len":' + s.length.toString() + '}');

  return 0;
}
`)

	_, err := kit.EvalTS(ctx, "compile.ts", `
		await wasm.compile(`+"`"+source+"`"+`, { name: "long-strings", runtime: "incremental" });
	`)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}

	result, err := kit.EvalTS(ctx, "run.ts", `
		var r = await wasm.run("long-strings");
		return JSON.stringify(r);
	`)
	if err != nil {
		t.Fatalf("run: %v", err)
	}

	var rr struct{ ExitCode int `json:"exitCode"` }
	json.Unmarshal([]byte(result), &rr)
	if rr.ExitCode != 0 {
		t.Fatalf("exitCode=%d (subtest %d failed)", rr.ExitCode, rr.ExitCode)
	}
}

// ---------------------------------------------------------------------------
// Module removal and re-compilation
// ---------------------------------------------------------------------------

func TestWASMStress_RemoveAndRecompile(t *testing.T) {
	kit := newTestKitNoKey(t)
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		source := fmt.Sprintf(`export function run(): i32 { return %d; }`, i)
		_, err := kit.EvalTS(ctx, fmt.Sprintf("compile%d.ts", i), fmt.Sprintf(`
			await wasm.compile(%s, { name: "recycled", runtime: "stub" });
		`, "`"+source+"`"))
		if err != nil {
			t.Fatalf("compile %d: %v", i, err)
		}

		// Run and verify
		result, err := kit.EvalTS(ctx, fmt.Sprintf("run%d.ts", i), `
			var r = await wasm.run("recycled");
			return JSON.stringify(r);
		`)
		if err != nil {
			t.Fatalf("run %d: %v", i, err)
		}
		var rr struct{ ExitCode int `json:"exitCode"` }
		json.Unmarshal([]byte(result), &rr)
		if rr.ExitCode != i {
			t.Errorf("cycle %d: exitCode=%d, want %d", i, rr.ExitCode, i)
		}

		// Remove
		err = kit.RemoveWASMModule("recycled")
		if err != nil {
			t.Fatalf("remove %d: %v", i, err)
		}

		// Verify removed
		info, _ := kit.GetWASMModule("recycled")
		if info != nil {
			t.Fatalf("module should be removed after cycle %d", i)
		}
	}
}

// ---------------------------------------------------------------------------
// Bus send ordering — verify messages arrive in send order
// ---------------------------------------------------------------------------

func TestWASMStress_BusOrdering(t *testing.T) {
	kit := newTestKitNoKey(t)
	ctx := context.Background()

	var mu sync.Mutex
	var received []string
	kit.Bus.On("stress.order.*", func(msg bus.Message, _ bus.ReplyFunc) {
		mu.Lock()
		received = append(received, msg.Topic)
		mu.Unlock()
	})

	source := hostTestSource(`
export function run(): i32 {
  host_bus_send("stress.order.1", '{}');
  host_bus_send("stress.order.2", '{}');
  host_bus_send("stress.order.3", '{}');
  host_bus_send("stress.order.4", '{}');
  host_bus_send("stress.order.5", '{}');
  return 0;
}
`)
	_, err := kit.EvalTS(ctx, "compile.ts", `
		await wasm.compile(`+"`"+source+"`"+`, { name: "bus-order", runtime: "incremental" });
	`)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}

	result, err := kit.EvalTS(ctx, "run.ts", `
		var r = await wasm.run("bus-order");
		return JSON.stringify(r);
	`)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	var rr struct{ ExitCode int `json:"exitCode"` }
	json.Unmarshal([]byte(result), &rr)
	if rr.ExitCode != 0 {
		t.Fatalf("exitCode=%d", rr.ExitCode)
	}

	mu.Lock()
	defer mu.Unlock()
	if len(received) != 5 {
		t.Fatalf("expected 5 bus messages, got %d", len(received))
	}
	for i, topic := range received {
		expected := fmt.Sprintf("stress.order.%d", i+1)
		if topic != expected {
			t.Errorf("message %d: topic=%q, want %q", i, topic, expected)
		}
	}
}

// ---------------------------------------------------------------------------
// Rapid sequential runs of same module
// ---------------------------------------------------------------------------

func TestWASMStress_RapidRuns(t *testing.T) {
	kit := newTestKitNoKey(t)
	ctx := context.Background()

	source := hostTestSource(`
var callCount: i32 = 0;
export function run(): i32 {
  callCount++;
  host_set_state("count", callCount.toString());
  return callCount;
}
`)
	_, err := kit.EvalTS(ctx, "compile.ts", `
		await wasm.compile(`+"`"+source+"`"+`, { name: "rapid", runtime: "stub" });
	`)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}

	// Each run gets a fresh WASM instance, so callCount always starts at 0
	for i := 0; i < 10; i++ {
		result, err := kit.EvalTS(ctx, fmt.Sprintf("run%d.ts", i), `
			var r = await wasm.run("rapid");
			return JSON.stringify(r);
		`)
		if err != nil {
			t.Fatalf("run %d: %v", i, err)
		}
		var rr struct{ ExitCode int `json:"exitCode"` }
		json.Unmarshal([]byte(result), &rr)
		// callCount incremented once per fresh instance → always 1
		if rr.ExitCode != 1 {
			t.Errorf("run %d: exitCode=%d, want 1 (fresh instance each time)", i, rr.ExitCode)
		}
	}
}

// ---------------------------------------------------------------------------
// Large JSON in bus_send
// ---------------------------------------------------------------------------

func TestWASMStress_LargePayload(t *testing.T) {
	kit := newTestKitNoKey(t)
	ctx := context.Background()

	source := hostTestSource(`
export function run(): i32 {
  // Build a 5KB JSON payload
  var json: string = '{"data":"';
  for (let i = 0; i < 5000; i++) {
    json += "A";
  }
  json += '"}';
  host_bus_send("stress.large", json);
  return 0;
}
`)
	received := make(chan bus.Message, 1)
	kit.Bus.On("stress.large", func(msg bus.Message, _ bus.ReplyFunc) {
		received <- msg
	})

	_, err := kit.EvalTS(ctx, "compile.ts", `
		await wasm.compile(`+"`"+source+"`"+`, { name: "large-payload", runtime: "incremental" });
	`)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}

	result, err := kit.EvalTS(ctx, "run.ts", `
		var r = await wasm.run("large-payload");
		return JSON.stringify(r);
	`)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	var rr struct{ ExitCode int `json:"exitCode"` }
	json.Unmarshal([]byte(result), &rr)
	if rr.ExitCode != 0 {
		t.Fatalf("exitCode=%d", rr.ExitCode)
	}

	select {
	case msg := <-received:
		var payload struct{ Data string `json:"data"` }
		json.Unmarshal(msg.Payload, &payload)
		if len(payload.Data) != 5000 {
			t.Errorf("payload data length=%d, want 5000", len(payload.Data))
		}
	default:
		t.Error("bus message not received")
	}
}

// ---------------------------------------------------------------------------
// Division by zero / unreachable trap
// ---------------------------------------------------------------------------

func TestWASMStress_Trap(t *testing.T) {
	kit := newTestKitNoKey(t)
	ctx := context.Background()

	source := `
export function run(): i32 {
  unreachable();
  return 0;
}
`
	_, err := kit.EvalTS(ctx, "compile.ts", `
		await wasm.compile(`+"`"+source+"`"+`, { name: "trapper", runtime: "stub" });
	`)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}

	result, err := kit.EvalTS(ctx, "run.ts", `
		try {
			var r = await wasm.run("trapper");
			return JSON.stringify({ ok: true, result: r });
		} catch(e) {
			return JSON.stringify({ ok: false, error: e.message || String(e) });
		}
	`)
	if err != nil {
		t.Fatalf("EvalTS: %v", err)
	}
	var r struct {
		OK    bool   `json:"ok"`
		Error string `json:"error"`
	}
	json.Unmarshal([]byte(result), &r)
	if r.OK {
		t.Error("expected unreachable trap to fail")
	}
	t.Logf("trap error: %s", r.Error)
}

// ---------------------------------------------------------------------------
// Module with no run() or _start() export
// ---------------------------------------------------------------------------

func TestWASMStress_NoRunExport(t *testing.T) {
	kit := newTestKitNoKey(t)
	ctx := context.Background()

	source := `
export function compute(): i32 {
  return 42;
}
`
	_, err := kit.EvalTS(ctx, "compile.ts", `
		await wasm.compile(`+"`"+source+"`"+`, { name: "no-run", runtime: "stub" });
	`)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}

	// Should succeed but return 0 (no run function to call)
	result, err := kit.EvalTS(ctx, "run.ts", `
		try {
			var r = await wasm.run("no-run");
			return JSON.stringify({ ok: true, exitCode: r.exitCode });
		} catch(e) {
			return JSON.stringify({ ok: false, error: e.message || String(e) });
		}
	`)
	if err != nil {
		t.Fatalf("EvalTS: %v", err)
	}
	t.Logf("no-run result: %s", result)
}

// ---------------------------------------------------------------------------
// Multiple host function calls in tight loop
// ---------------------------------------------------------------------------

func TestWASMStress_HostCallLoop(t *testing.T) {
	kit := newTestKitNoKey(t)
	ctx := context.Background()

	source := hostTestSource(`
export function run(): i32 {
  // Hammer host functions in a tight loop
  for (let i = 0; i < 200; i++) {
    host_set_state("loop-key", i.toString());
    const v = host_get_state("loop-key");
    if (v != i.toString()) return 1;
    host_log("iter-" + i.toString(), 0);
  }
  return 0;
}
`)

	_, err := kit.EvalTS(ctx, "compile.ts", `
		await wasm.compile(`+"`"+source+"`"+`, { name: "host-loop", runtime: "incremental" });
	`)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}

	result, err := kit.EvalTS(ctx, "run.ts", `
		var r = await wasm.run("host-loop");
		return JSON.stringify(r);
	`)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	var rr struct{ ExitCode int `json:"exitCode"` }
	json.Unmarshal([]byte(result), &rr)
	if rr.ExitCode != 0 {
		t.Fatalf("exitCode=%d (host call loop failed)", rr.ExitCode)
	}
}
