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

func testProviderAddViaBus(t *testing.T, env *suite.TestEnv) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := sdk.Call[registrymsg.ProviderAddMsg, registrymsg.ProviderAddResp](env.Kit, ctx, registrymsg.ProviderAddMsg{
		Name:   "test-openai-add",
		Type:   "openai",
		Config: json.RawMessage(`{"APIKey":"test-key-123"}`),
	})
	if err != nil {
		t.Fatalf("provider add: %v", err)
	}
	if !resp.Added {
		t.Fatal("expected Added=true")
	}

	// Verify via registry.list
	listResp, err := sdk.Call[registrymsg.RegistryListMsg, registrymsg.RegistryListResp](env.Kit, ctx, registrymsg.RegistryListMsg{Category: "provider"})
	if err != nil {
		t.Fatalf("registry.list: %v", err)
	}
	if !strings.Contains(string(listResp.Items), "test-openai-add") {
		t.Fatalf("provider not in registry list: %s", listResp.Items)
	}
}

func testProviderAddInvalidName(t *testing.T, env *suite.TestEnv) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := sdk.Call[registrymsg.ProviderAddMsg, registrymsg.ProviderAddResp](env.Kit, ctx, registrymsg.ProviderAddMsg{
		Name: "", // invalid
		Type: "openai",
	})
	if err == nil {
		t.Fatal("expected validation error for empty name")
	}
}

func testProviderRemoveViaBus(t *testing.T, env *suite.TestEnv) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Add first
	addResp, err := sdk.Call[registrymsg.ProviderAddMsg, registrymsg.ProviderAddResp](env.Kit, ctx, registrymsg.ProviderAddMsg{
		Name: "test-remove-prov", Type: "openai", Config: json.RawMessage(`{"APIKey":"k"}`),
	})
	if err != nil {
		t.Fatalf("provider add: %v", err)
	}
	if !addResp.Added {
		t.Fatal("expected Added=true")
	}

	// Remove
	rmResp, err := sdk.Call[registrymsg.ProviderRemoveMsg, registrymsg.ProviderRemoveResp](env.Kit, ctx, registrymsg.ProviderRemoveMsg{Name: "test-remove-prov"})
	if err != nil {
		t.Fatalf("provider remove: %v", err)
	}
	if !rmResp.Removed {
		t.Fatal("expected Removed=true")
	}
}

// ── Real effect ──────────────────────────────────────────────────────────────

func testProviderAddThenResolveFromTS(t *testing.T, _ *suite.TestEnv) {
	env := suite.Full(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Add provider via bus
	resp, err := sdk.Call[registrymsg.ProviderAddMsg, registrymsg.ProviderAddResp](env.Kit, ctx, registrymsg.ProviderAddMsg{
		Name: "ts-resolve-test", Type: "openai", Config: json.RawMessage(`{"APIKey":"test-key"}`),
	})
	if err != nil {
		t.Fatalf("provider add: %v", err)
	}
	if !resp.Added {
		t.Fatal("expected Added=true")
	}

	// Deploy .ts that resolves the provider and reports back
	code := `
		const resolved = registry.resolve("provider", "ts-resolve-test");
		output(JSON.stringify(resolved));
	`
	testutil.Deploy(t, env.Kit, "resolve-prov-test.ts", code)
	defer testutil.Teardown(t, env.Kit, "resolve-prov-test.ts")

	result := testutil.EvalJS(t, env.Kit, "__check_resolve.ts", `
		const r = globalThis.__module_result;
		return r || "null";
	`)
	if result == "null" || result == "" {
		t.Fatal("expected provider to resolve, got null")
	}
	if !strings.Contains(result, "openai") {
		t.Fatalf("expected openai in resolved config, got: %s", result)
	}
}
