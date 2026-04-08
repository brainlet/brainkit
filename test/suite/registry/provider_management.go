package registry

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/brainlet/brainkit/internal/testutil"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/test/suite"
)

// ── Bus plumbing ─────────────────────────────────────────────────────────────

func testProviderAddViaBus(t *testing.T, env *suite.TestEnv) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pr, _ := sdk.PublishProviderAdd(env.Kit, ctx, sdk.ProviderAddMsg{
		Name:   "test-openai-add",
		Type:   "openai",
		Config: json.RawMessage(`{"APIKey":"test-key-123"}`),
	})
	respCh := make(chan sdk.ProviderAddResp, 1)
	unsub, _ := sdk.SubscribeProviderAddResp(env.Kit, ctx, pr.ReplyTo,
		func(resp sdk.ProviderAddResp, msg sdk.Message) { respCh <- resp })
	defer unsub()

	select {
	case resp := <-respCh:
		if resp.Error != "" {
			t.Fatalf("error: %s", resp.Error)
		}
		if !resp.Added {
			t.Fatal("expected Added=true")
		}
	case <-ctx.Done():
		t.Fatal("timeout")
	}

	// Verify via registry.list
	pr2, _ := sdk.Publish(env.Kit, ctx, sdk.RegistryListMsg{Category: "provider"})
	listCh := make(chan sdk.RegistryListResp, 1)
	unsub2, _ := sdk.SubscribeTo[sdk.RegistryListResp](env.Kit, ctx, pr2.ReplyTo,
		func(resp sdk.RegistryListResp, msg sdk.Message) { listCh <- resp })
	defer unsub2()

	select {
	case resp := <-listCh:
		if !strings.Contains(string(resp.Items), "test-openai-add") {
			t.Fatalf("provider not in registry list: %s", resp.Items)
		}
	case <-ctx.Done():
		t.Fatal("timeout on list")
	}
}

func testProviderAddInvalidName(t *testing.T, env *suite.TestEnv) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pr, _ := sdk.PublishProviderAdd(env.Kit, ctx, sdk.ProviderAddMsg{
		Name: "", // invalid
		Type: "openai",
	})
	respCh := make(chan sdk.ProviderAddResp, 1)
	unsub, _ := sdk.SubscribeProviderAddResp(env.Kit, ctx, pr.ReplyTo,
		func(resp sdk.ProviderAddResp, msg sdk.Message) { respCh <- resp })
	defer unsub()

	select {
	case resp := <-respCh:
		if resp.Error == "" {
			t.Fatal("expected validation error for empty name")
		}
	case <-ctx.Done():
		t.Fatal("timeout")
	}
}

func testProviderRemoveViaBus(t *testing.T, env *suite.TestEnv) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Add first
	pr, _ := sdk.PublishProviderAdd(env.Kit, ctx, sdk.ProviderAddMsg{
		Name: "test-remove-prov", Type: "openai", Config: json.RawMessage(`{"APIKey":"k"}`),
	})
	ch := make(chan sdk.ProviderAddResp, 1)
	unsub, _ := sdk.SubscribeProviderAddResp(env.Kit, ctx, pr.ReplyTo,
		func(resp sdk.ProviderAddResp, msg sdk.Message) { ch <- resp })
	<-ch
	unsub()

	// Remove
	pr2, _ := sdk.PublishProviderRemove(env.Kit, ctx, sdk.ProviderRemoveMsg{Name: "test-remove-prov"})
	rmCh := make(chan sdk.ProviderRemoveResp, 1)
	unsub2, _ := sdk.SubscribeProviderRemoveResp(env.Kit, ctx, pr2.ReplyTo,
		func(resp sdk.ProviderRemoveResp, msg sdk.Message) { rmCh <- resp })
	defer unsub2()

	select {
	case resp := <-rmCh:
		if !resp.Removed {
			t.Fatal("expected Removed=true")
		}
	case <-ctx.Done():
		t.Fatal("timeout")
	}
}

// ── Real effect ──────────────────────────────────────────────────────────────

func testProviderAddThenResolveFromTS(t *testing.T, _ *suite.TestEnv) {
	env := suite.Full(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Add provider via bus
	pr, _ := sdk.PublishProviderAdd(env.Kit, ctx, sdk.ProviderAddMsg{
		Name: "ts-resolve-test", Type: "openai", Config: json.RawMessage(`{"APIKey":"test-key"}`),
	})
	ch := make(chan sdk.ProviderAddResp, 1)
	unsub, _ := sdk.SubscribeProviderAddResp(env.Kit, ctx, pr.ReplyTo,
		func(resp sdk.ProviderAddResp, msg sdk.Message) { ch <- resp })
	<-ch
	unsub()

	// Deploy .ts that resolves the provider and reports back
	code := `
		const resolved = registry.resolve("provider", "ts-resolve-test");
		output(JSON.stringify(resolved));
	`
	testutil.Deploy(t, env.Kit, "resolve-prov-test.ts", code)
	defer testutil.Teardown(t, env.Kit, "resolve-prov-test.ts")

	result := testutil.EvalTS(t, env.Kit, "__check_resolve.ts", `
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
