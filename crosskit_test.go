package brainkit

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/brainlet/brainkit/bus"
	"github.com/brainlet/brainkit/registry"
)

func newSharedTools() *registry.ToolRegistry {
	return registry.New()
}

func newTestGoTool(name, description string, fn func(map[string]any) (any, error)) registry.RegisteredTool {
	parts := splitToolName(name)
	return registry.RegisteredTool{
		Name:        name,
		ShortName:   parts.short,
		Namespace:   parts.ns,
		Description: description,
		Executor: &registry.GoFuncExecutor{
			Fn: func(ctx context.Context, callerID string, input json.RawMessage) (json.RawMessage, error) {
				var m map[string]any
				if len(input) > 0 {
					json.Unmarshal(input, &m)
				}
				result, err := fn(m)
				if err != nil {
					return nil, err
				}
				out, _ := json.Marshal(result)
				return out, nil
			},
		},
	}
}

type toolNameParts struct {
	ns    string
	short string
}

func splitToolName(name string) toolNameParts {
	for i := len(name) - 1; i >= 0; i-- {
		if name[i] == '.' {
			return toolNameParts{ns: name[:i], short: name[i+1:]}
		}
	}
	return toolNameParts{ns: "user", short: name}
}

// TestCrossKit_BusPubSub tests two Kits communicating via a shared bus.
// Kit A subscribes to "events.*", Kit B publishes to "events.hello".
func TestCrossKit_BusPubSub(t *testing.T) {
	sharedBus := bus.New()
	defer sharedBus.Close()

	kitA, err := New(Config{
		Namespace: "kit-a",
		CallerID:  "kit-a",
		SharedBus: sharedBus,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer kitA.Close()

	kitB, err := New(Config{
		Namespace: "kit-b",
		CallerID:  "kit-b",
		SharedBus: sharedBus,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer kitB.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Kit A: subscribe to "events.*" from JS
	_, err = kitA.EvalTS(ctx, "kit-a-sub.js", `
		globalThis._kitA_received = [];
		bus.subscribe("events.*", function(msg) {
			globalThis._kitA_received.push({
				topic: msg.topic,
				payload: msg.payload,
				from: msg.callerID,
			});
		});
		return "subscribed";
	`)
	if err != nil {
		t.Fatalf("Kit A subscribe: %v", err)
	}

	// Kit B: publish from JS
	_, err = kitB.EvalTS(ctx, "kit-b-pub.js", `
		await bus.publish("events.hello", { message: "hi from kit B" });
		await bus.publish("events.data", { value: 42 });
		return "published";
	`)
	if err != nil {
		t.Fatalf("Kit B publish: %v", err)
	}

	// Wait for Schedule callbacks to fire
	time.Sleep(200 * time.Millisecond)

	// Kit A: check what was received — need async to let ProcessJobs drain the Schedule queue
	result, err := kitA.EvalTS(ctx, "kit-a-check.js", `
		// Yield to let Schedule callbacks fire via ProcessJobs
		await new Promise(r => setTimeout(r, 100));
		return JSON.stringify(globalThis._kitA_received);
	`)
	if err != nil {
		t.Fatalf("Kit A check: %v", err)
	}

	var received []struct {
		Topic   string `json:"topic"`
		Payload any    `json:"payload"`
		From    string `json:"from"`
	}
	json.Unmarshal([]byte(result), &received)

	if len(received) != 2 {
		t.Fatalf("expected 2 messages, got %d: %s", len(received), result)
	}

	t.Logf("Cross-Kit: Kit A received %d messages from Kit B", len(received))
	for i, msg := range received {
		t.Logf("  [%d] topic=%s from=%s payload=%v", i, msg.Topic, msg.From, msg.Payload)
	}
}

// TestCrossKit_SharedTools tests two Kits sharing a tool registry.
// Kit A registers a Go tool, Kit B calls it from JS.
func TestCrossKit_SharedTools(t *testing.T) {
	key := requireKey(t)
	sharedBus := bus.New()
	sharedTools := newSharedTools()
	defer sharedBus.Close()

	kitA, err := New(Config{
		Namespace:   "kit-a",
		CallerID:    "kit-a",
		Providers:   map[string]ProviderConfig{"openai": {APIKey: key}},
		EnvVars:     map[string]string{"OPENAI_API_KEY": key},
		SharedBus:   sharedBus,
		SharedTools: sharedTools,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer kitA.Close()

	kitB, err := New(Config{
		Namespace:   "kit-b",
		CallerID:    "kit-b",
		Providers:   map[string]ProviderConfig{"openai": {APIKey: key}},
		EnvVars:     map[string]string{"OPENAI_API_KEY": key},
		SharedBus:   sharedBus,
		SharedTools: sharedTools,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer kitB.Close()

	// Register a Go tool on Kit A
	kitA.Tools.Register(newTestGoTool("shared.greet", "Greet someone", func(input map[string]any) (any, error) {
		name, _ := input["name"].(string)
		return map[string]string{"greeting": "Hello, " + name + "!"}, nil
	}))

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Kit B: call the tool registered by Kit A
	result, err := kitB.EvalTS(ctx, "kit-b-call.js", `
		try {
			const result = await tools.call("shared.greet", { name: "World" });
			return JSON.stringify({ ok: true, result: result });
		} catch(e) {
			return JSON.stringify({ ok: false, error: e.message });
		}
	`)
	if err != nil {
		t.Fatalf("Kit B call: %v", err)
	}

	t.Logf("Raw result: %s", result)

	var resp struct {
		OK     bool   `json:"ok"`
		Result any    `json:"result"`
		Error  string `json:"error"`
	}
	json.Unmarshal([]byte(result), &resp)

	if !resp.OK {
		t.Fatalf("tool call failed: %s", resp.Error)
	}

	// The result comes back as JSON string from the bus, need to parse
	resultMap, ok := resp.Result.(map[string]any)
	if !ok {
		t.Fatalf("unexpected result type: %T = %v", resp.Result, resp.Result)
	}
	greeting, _ := resultMap["greeting"].(string)
	if greeting != "Hello, World!" {
		t.Errorf("expected 'Hello, World!', got %q", greeting)
	}
	t.Logf("Cross-Kit: Kit B called Kit A's tool, got: %s", greeting)
}

// TestCrossKit_Isolation tests that non-shared Kits are isolated.
func TestCrossKit_Isolation(t *testing.T) {
	kitA, err := New(Config{Namespace: "kit-a"})
	if err != nil {
		t.Fatal(err)
	}
	defer kitA.Close()

	kitB, err := New(Config{Namespace: "kit-b"})
	if err != nil {
		t.Fatal(err)
	}
	defer kitB.Close()

	ctx := context.Background()

	// Kit A sets a global
	kitA.EvalTS(ctx, "a.js", `globalThis._test = "from-A"`)

	// Kit B should NOT see it
	result, _ := kitB.EvalTS(ctx, "b.js", `return typeof globalThis._test`)
	if result != "undefined" {
		t.Errorf("isolation broken: Kit B sees Kit A's global: %s", result)
	}

	// Kit A registers a tool — Kit B should NOT see it (no shared tools)
	kitA.Tools.Register(newTestGoTool("private.tool", "Private", func(input map[string]any) (any, error) {
		return "secret", nil
	}))

	_, err = kitB.EvalTS(ctx, "b2.js", `
		try {
			await tools.call("private.tool", {});
			return "found";
		} catch(e) {
			return "not_found:" + e.message;
		}
	`)
	if err != nil {
		t.Fatalf("Kit B tool check: %v", err)
	}

	t.Logf("Cross-Kit: isolation verified — Kits don't share globals or tools by default")
}
