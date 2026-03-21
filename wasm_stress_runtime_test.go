//go:build stress

package brainkit

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"testing"

	"github.com/brainlet/brainkit/bus"
)

func TestWASMStress_StateIsolation(t *testing.T) {
	kit := newTestKitNoKey(t)
	ctx := context.Background()

	source := hostTestSource(`
export function run(): i32 {
  const existing = host_get_state("persist-test");
  if (existing != "") {
    return 1;
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

	for i := 0; i < 5; i++ {
		result, err := kit.EvalTS(ctx, fmt.Sprintf("run%d.ts", i), `
			var r = await wasm.run("isolation");
			return JSON.stringify(r);
		`)
		if err != nil {
			t.Fatalf("run %d: %v", i, err)
		}
		var rr struct {
			ExitCode int `json:"exitCode"`
		}
		json.Unmarshal([]byte(result), &rr)
		if rr.ExitCode != 0 {
			t.Fatalf("run %d: state leaked (exitCode=%d)", i, rr.ExitCode)
		}
	}
}

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
}

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
  host_send("stress.order.1", '{}');
  host_send("stress.order.2", '{}');
  host_send("stress.order.3", '{}');
  host_send("stress.order.4", '{}');
  host_send("stress.order.5", '{}');
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
	var rr struct {
		ExitCode int `json:"exitCode"`
	}
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

	for i := 0; i < 10; i++ {
		result, err := kit.EvalTS(ctx, fmt.Sprintf("run%d.ts", i), `
			var r = await wasm.run("rapid");
			return JSON.stringify(r);
		`)
		if err != nil {
			t.Fatalf("run %d: %v", i, err)
		}
		var rr struct {
			ExitCode int `json:"exitCode"`
		}
		json.Unmarshal([]byte(result), &rr)
		if rr.ExitCode != 1 {
			t.Errorf("run %d: exitCode=%d, want 1 (fresh instance each time)", i, rr.ExitCode)
		}
	}
}

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

func TestWASMStress_HostCallLoop(t *testing.T) {
	kit := newTestKitNoKey(t)
	ctx := context.Background()

	source := hostTestSource(`
export function run(): i32 {
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
	var rr struct {
		ExitCode int `json:"exitCode"`
	}
	json.Unmarshal([]byte(result), &rr)
	if rr.ExitCode != 0 {
		t.Fatalf("exitCode=%d (host call loop failed)", rr.ExitCode)
	}
}
