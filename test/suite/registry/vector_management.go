package registry

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/brainlet/brainkit/internal/testutil"
	"github.com/brainlet/brainkit/modules/registry/registrymsg"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/test/suite"
)

// ── Bus plumbing ─────────────────────────────────────────────────────────────

func testVectorAddViaBus(t *testing.T, env *suite.TestEnv) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := sdk.Call[registrymsg.VectorAddMsg, registrymsg.VectorAddResp](env.Kit, ctx, registrymsg.VectorAddMsg{
		Name:   "test-vec-add",
		Type:   "sqlite",
		Config: json.RawMessage(`{}`),
	})
	if err != nil {
		t.Fatalf("vector add: %v", err)
	}
	if !resp.Added {
		t.Fatal("expected Added=true")
	}

	// Verify via registry.list
	listResp, err := sdk.Call[registrymsg.RegistryListMsg, registrymsg.RegistryListResp](env.Kit, ctx, registrymsg.RegistryListMsg{Category: "vectorStore"})
	if err != nil {
		t.Fatalf("registry.list: %v", err)
	}
	if !strings.Contains(string(listResp.Items), "test-vec-add") {
		t.Fatalf("vector store not in list: %s", listResp.Items)
	}
}

func testVectorRemoveViaBus(t *testing.T, env *suite.TestEnv) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Add
	addResp, err := sdk.Call[registrymsg.VectorAddMsg, registrymsg.VectorAddResp](env.Kit, ctx, registrymsg.VectorAddMsg{
		Name: "test-vec-rm", Type: "sqlite", Config: json.RawMessage(`{}`),
	})
	if err != nil {
		t.Fatalf("vector add: %v", err)
	}
	if !addResp.Added {
		t.Fatal("expected Added=true")
	}

	// Remove
	rmResp, err := sdk.Call[registrymsg.VectorRemoveMsg, registrymsg.VectorRemoveResp](env.Kit, ctx, registrymsg.VectorRemoveMsg{Name: "test-vec-rm"})
	if err != nil {
		t.Fatalf("vector remove: %v", err)
	}
	if !rmResp.Removed {
		t.Fatal("expected Removed=true")
	}
}

// ── Real effect ──────────────────────────────────────────────────────────────

func testVectorAddThenResolveFromTS(t *testing.T, _ *suite.TestEnv) {
	env := suite.Full(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Add vector store via bus
	resp, err := sdk.Call[registrymsg.VectorAddMsg, registrymsg.VectorAddResp](env.Kit, ctx, registrymsg.VectorAddMsg{
		Name: "ts-vec-resolve", Type: "sqlite", Config: json.RawMessage(`{}`),
	})
	if err != nil {
		t.Fatalf("vector add: %v", err)
	}
	if !resp.Added {
		t.Fatal("expected Added=true")
	}

	// Deploy .ts that resolves the vector store
	code := `
		const resolved = registry.resolve("vectorStore", "ts-vec-resolve");
		output(JSON.stringify(resolved));
	`
	testutil.Deploy(t, env.Kit, "vec-resolve-test.ts", code)
	defer testutil.Teardown(t, env.Kit, "vec-resolve-test.ts")

	result := testutil.EvalTS(t, env.Kit, "__check_vec.ts", `
		return globalThis.__module_result || "null";
	`)
	if result == "null" || result == "" {
		t.Fatal("expected vector store to resolve, got null")
	}
	if !strings.Contains(result, "libsql") {
		t.Fatalf("expected 'libsql' in resolved config, got: %s", result)
	}
	if !strings.Contains(result, `"URL":"http://127.0.0.1:`) {
		t.Fatalf("expected live bridge URL in resolved config, got: %s", result)
	}
}
