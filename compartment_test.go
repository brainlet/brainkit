package brainkit

import (
	"context"
	"testing"
)

func TestDeploy_CreatesResources(t *testing.T) {
	kit := newTestKitNoKey(t)
	ctx := context.Background()

	resources, err := kit.Deploy(ctx, "team.ts", `
		agent({ name: "deploy-leader", model: "openai/gpt-4o-mini", instructions: "lead" });
	`)
	if err != nil {
		t.Fatal(err)
	}
	if len(resources) < 1 {
		t.Fatalf("expected at least 1 resource, got %d", len(resources))
	}

	found := false
	for _, r := range resources {
		if r.Name == "deploy-leader" {
			found = true
		}
	}
	if !found {
		t.Fatalf("deploy-leader not found in resources: %+v", resources)
	}
}

func TestTeardown_RemovesResources(t *testing.T) {
	kit := newTestKitNoKey(t)
	ctx := context.Background()

	kit.Deploy(ctx, "teardown-test.ts", `
		agent({ name: "tear-agent", model: "openai/gpt-4o-mini", instructions: "test" });
	`)

	removed, err := kit.Teardown(ctx, "teardown-test.ts")
	if err != nil {
		t.Fatal(err)
	}
	if removed < 1 {
		t.Fatalf("expected at least 1 removed, got %d", removed)
	}

	resources, _ := kit.ListResources("agent")
	for _, r := range resources {
		if r.Name == "tear-agent" {
			t.Fatal("agent should be gone after teardown")
		}
	}
}

func TestDeploy_Isolation(t *testing.T) {
	kit := newTestKitNoKey(t)
	ctx := context.Background()

	kit.Deploy(ctx, "file-a.ts", `var config = "A";`)
	kit.Deploy(ctx, "file-b.ts", `var config = "B";`)

	deployments := kit.ListDeployments()
	if len(deployments) < 2 {
		t.Fatalf("expected 2 deployments, got %d", len(deployments))
	}
}

func TestDeploy_DoubleDeploy_Errors(t *testing.T) {
	kit := newTestKitNoKey(t)
	ctx := context.Background()

	kit.Deploy(ctx, "dup.ts", `var x = 1;`)
	_, err := kit.Deploy(ctx, "dup.ts", `var x = 2;`)
	if err == nil {
		t.Fatal("expected error on double deploy")
	}
}

func TestRedeploy_AtomicSwap(t *testing.T) {
	kit := newTestKitNoKey(t)
	ctx := context.Background()

	kit.Deploy(ctx, "swap.ts", `
		agent({ name: "swap-v1", model: "openai/gpt-4o-mini", instructions: "v1" });
	`)

	_, err := kit.Redeploy(ctx, "swap.ts", `
		agent({ name: "swap-v2", model: "openai/gpt-4o-mini", instructions: "v2" });
	`)
	if err != nil {
		t.Fatal(err)
	}

	allResources, _ := kit.ListResources("agent")
	foundV1, foundV2 := false, false
	for _, r := range allResources {
		if r.Name == "swap-v1" {
			foundV1 = true
		}
		if r.Name == "swap-v2" {
			foundV2 = true
		}
	}
	if foundV1 {
		t.Fatal("v1 should be gone")
	}
	if !foundV2 {
		t.Fatal("v2 should exist")
	}
}

func TestTeardown_Idempotent(t *testing.T) {
	kit := newTestKitNoKey(t)
	ctx := context.Background()

	removed, err := kit.Teardown(ctx, "nonexistent.ts")
	if err != nil {
		t.Fatal(err)
	}
	if removed != 0 {
		t.Fatalf("expected 0, got %d", removed)
	}
}

func TestDeploy_DeployError_CleansUp(t *testing.T) {
	kit := newTestKitNoKey(t)
	ctx := context.Background()

	_, err := kit.Deploy(ctx, "bad.ts", `throw new Error("intentional crash");`)
	if err == nil {
		t.Fatal("expected error from bad code")
	}

	// Should not be in deployments
	deployments := kit.ListDeployments()
	for _, d := range deployments {
		if d.Source == "bad.ts" {
			t.Fatal("failed deploy should not be listed")
		}
	}
}
