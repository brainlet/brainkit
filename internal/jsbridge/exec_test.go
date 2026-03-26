package jsbridge

import (
	"strings"
	"testing"
)

func TestExecSync(t *testing.T) {
	b := newTestBridge(t, Encoding(), Buffer(), Exec())
	val, err := b.Eval("test.js", `
		var result = globalThis.child_process.execSync("echo hello");
		// execSync returns Buffer on success
		var str = typeof result === "string" ? result : result.toString("utf8");
		JSON.stringify({ output: str.trim() });
	`)
	if err != nil {
		t.Fatalf("execSync: %v", err)
	}
	defer val.Free()
	s := val.String()
	if !strings.Contains(s, "hello") {
		t.Fatalf("expected 'hello' in output, got: %s", s)
	}
	t.Logf("execSync: %s", s)
}

func TestExecFileSync(t *testing.T) {
	b := newTestBridge(t, Encoding(), Buffer(), Exec())
	val, err := b.Eval("test.js", `
		var result = globalThis.child_process.execFileSync("echo", ["world"]);
		var str = typeof result === "string" ? result : result.toString("utf8");
		JSON.stringify({ output: str.trim() });
	`)
	if err != nil {
		t.Fatalf("execFileSync: %v", err)
	}
	defer val.Free()
	s := val.String()
	if !strings.Contains(s, "world") {
		t.Fatalf("expected 'world' in output, got: %s", s)
	}
	t.Logf("execFileSync: %s", s)
}

func TestSpawnSync(t *testing.T) {
	b := newTestBridge(t, Encoding(), Buffer(), Exec())
	val, err := b.Eval("test.js", `
		var result = globalThis.child_process.spawnSync("echo", ["sync-test"]);
		JSON.stringify({ stdout: result.stdout.trim(), status: result.status });
	`)
	if err != nil {
		t.Fatalf("spawnSync: %v", err)
	}
	defer val.Free()
	s := val.String()
	if !strings.Contains(s, "sync-test") {
		t.Fatalf("expected 'sync-test' in output, got: %s", s)
	}
	t.Logf("spawnSync: %s", s)
}

func TestExecSyncFailure(t *testing.T) {
	b := newTestBridge(t, Encoding(), Buffer(), Exec())
	val, err := b.Eval("test.js", `
		var caught = false;
		try {
			globalThis.child_process.execSync("exit 42");
		} catch(e) {
			caught = true;
		}
		JSON.stringify({ caught: caught });
	`)
	if err != nil {
		t.Fatalf("execSync failure: %v", err)
	}
	defer val.Free()
	if !strings.Contains(val.String(), `"caught":true`) {
		t.Fatalf("expected caught:true, got: %s", val.String())
	}
}
