package brainkit

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/brainlet/brainkit/bus"
)

func TestDeployHandler_DeployViaBus(t *testing.T) {
	kit := newTestKitNoKey(t)
	ctx := context.Background()

	resp, err := bus.AskSync(kit.Bus, ctx, bus.Message{
		Topic:   "kit.deploy",
		Payload: json.RawMessage(`{"source":"bus-deploy.ts","code":"agent({ name: 'bus-agent', model: 'openai/gpt-4o-mini', instructions: 'test' });"}`),
	})
	if err != nil {
		t.Fatal(err)
	}

	var result struct {
		Deployed  bool           `json:"deployed"`
		Resources []ResourceInfo `json:"resources"`
	}
	json.Unmarshal(resp.Payload, &result)
	if !result.Deployed {
		t.Fatalf("expected deployed=true, got: %s", resp.Payload)
	}
	t.Logf("deployed via bus: %d resources", len(result.Resources))
}

func TestDeployHandler_TeardownViaBus(t *testing.T) {
	kit := newTestKitNoKey(t)
	ctx := context.Background()

	bus.AskSync(kit.Bus, ctx, bus.Message{
		Topic:   "kit.deploy",
		Payload: json.RawMessage(`{"source":"bus-td.ts","code":"var x = 1;"}`),
	})

	resp, err := bus.AskSync(kit.Bus, ctx, bus.Message{
		Topic:   "kit.teardown",
		Payload: json.RawMessage(`{"source":"bus-td.ts"}`),
	})
	if err != nil {
		t.Fatal(err)
	}

	var result struct{ Removed int `json:"removed"` }
	json.Unmarshal(resp.Payload, &result)
	t.Logf("removed via bus: %d", result.Removed)
}

func TestDeployHandler_ListViaBus(t *testing.T) {
	kit := newTestKitNoKey(t)
	ctx := context.Background()

	kit.Deploy(ctx, "list-a.ts", `var a = 1;`)
	kit.Deploy(ctx, "list-b.ts", `var b = 2;`)

	resp, err := bus.AskSync(kit.Bus, ctx, bus.Message{
		Topic:   "kit.list",
		Payload: json.RawMessage(`{}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("deployments: %s", resp.Payload)

	var deployments []deploymentInfo
	json.Unmarshal(resp.Payload, &deployments)
	if len(deployments) < 2 {
		t.Fatalf("expected 2 deployments, got %d", len(deployments))
	}
}

func TestDeployHandler_UnknownTopic(t *testing.T) {
	kit := newTestKitNoKey(t)
	ctx := context.Background()

	resp, err := bus.AskSync(kit.Bus, ctx, bus.Message{
		Topic:   "kit.bogus",
		Payload: json.RawMessage(`{}`),
	})
	if err != nil {
		t.Fatal(err)
	}

	var errResult struct{ Error string `json:"error"` }
	json.Unmarshal(resp.Payload, &errResult)
	if errResult.Error == "" {
		t.Fatal("expected error for unknown kit topic")
	}
}
