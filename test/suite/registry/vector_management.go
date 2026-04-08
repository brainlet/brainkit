package registry

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/brainlet/brainkit/internal/testutil"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/messages"
	"github.com/brainlet/brainkit/test/suite"
)

// ── Bus plumbing ─────────────────────────────────────────────────────────────

func testVectorAddViaBus(t *testing.T, env *suite.TestEnv) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pr, _ := sdk.PublishVectorAdd(env.Kit, ctx, messages.VectorAddMsg{
		Name:   "test-vec-add",
		Type:   "sqlite",
		Config: json.RawMessage(`{}`),
	})
	respCh := make(chan messages.VectorAddResp, 1)
	unsub, _ := sdk.SubscribeVectorAddResp(env.Kit, ctx, pr.ReplyTo,
		func(resp messages.VectorAddResp, msg messages.Message) { respCh <- resp })
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
	pr2, _ := sdk.Publish(env.Kit, ctx, messages.RegistryListMsg{Category: "vectorStore"})
	listCh := make(chan messages.RegistryListResp, 1)
	unsub2, _ := sdk.SubscribeTo[messages.RegistryListResp](env.Kit, ctx, pr2.ReplyTo,
		func(resp messages.RegistryListResp, msg messages.Message) { listCh <- resp })
	defer unsub2()

	select {
	case resp := <-listCh:
		if !strings.Contains(string(resp.Items), "test-vec-add") {
			t.Fatalf("vector store not in list: %s", resp.Items)
		}
	case <-ctx.Done():
		t.Fatal("timeout on list")
	}
}

func testVectorRemoveViaBus(t *testing.T, env *suite.TestEnv) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Add
	pr, _ := sdk.PublishVectorAdd(env.Kit, ctx, messages.VectorAddMsg{
		Name: "test-vec-rm", Type: "sqlite", Config: json.RawMessage(`{}`),
	})
	ch := make(chan messages.VectorAddResp, 1)
	unsub, _ := sdk.SubscribeVectorAddResp(env.Kit, ctx, pr.ReplyTo,
		func(resp messages.VectorAddResp, msg messages.Message) { ch <- resp })
	<-ch
	unsub()

	// Remove
	pr2, _ := sdk.PublishVectorRemove(env.Kit, ctx, messages.VectorRemoveMsg{Name: "test-vec-rm"})
	rmCh := make(chan messages.VectorRemoveResp, 1)
	unsub2, _ := sdk.SubscribeVectorRemoveResp(env.Kit, ctx, pr2.ReplyTo,
		func(resp messages.VectorRemoveResp, msg messages.Message) { rmCh <- resp })
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

func testVectorAddThenResolveFromTS(t *testing.T, _ *suite.TestEnv) {
	env := suite.Full(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Add vector store via bus
	pr, _ := sdk.PublishVectorAdd(env.Kit, ctx, messages.VectorAddMsg{
		Name: "ts-vec-resolve", Type: "sqlite", Config: json.RawMessage(`{}`),
	})
	ch := make(chan messages.VectorAddResp, 1)
	unsub, _ := sdk.SubscribeVectorAddResp(env.Kit, ctx, pr.ReplyTo,
		func(resp messages.VectorAddResp, msg messages.Message) { ch <- resp })
	<-ch
	unsub()

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
	if !strings.Contains(result, "sqlite") {
		t.Fatalf("expected 'sqlite' in resolved config, got: %s", result)
	}
}
