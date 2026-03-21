//go:build stress

package brainkit

import (
	"context"
	"fmt"
	"testing"
)

func TestDeploy_Stress30Cycles(t *testing.T) {
	kit := newTestKitNoKey(t)
	ctx := context.Background()

	for i := range 30 {
		source := fmt.Sprintf("stress-%d.ts", i)
		agentName := fmt.Sprintf("stress-%d", i)

		_, err := kit.Deploy(ctx, source, fmt.Sprintf(
			`agent({ name: %q, model: "openai/gpt-4o-mini", instructions: "cycle %d" });`, agentName, i))
		if err != nil {
			t.Fatalf("cycle %d deploy: %v", i, err)
		}

		removed, err := kit.Teardown(ctx, source)
		if err != nil {
			t.Fatalf("cycle %d teardown: %v", i, err)
		}
		if removed < 1 {
			t.Fatalf("cycle %d: expected removal", i)
		}
	}

	if len(kit.ListDeployments()) != 0 {
		t.Fatal("deployments should be empty after stress")
	}
	t.Log("PASS: 30 deploy/teardown cycles with SES Compartments")
}

func TestDeploy_IsolationStress(t *testing.T) {
	kit := newTestKitNoKey(t)
	ctx := context.Background()

	for i := range 10 {
		source := fmt.Sprintf("iso-%d.ts", i)
		kit.Deploy(ctx, source, `var config = "file-specific"; var counter = 0;`)
	}

	if len(kit.ListDeployments()) != 10 {
		t.Fatalf("expected 10 deployments, got %d", len(kit.ListDeployments()))
	}

	for i := range 5 {
		kit.Teardown(ctx, fmt.Sprintf("iso-%d.ts", i))
	}

	if len(kit.ListDeployments()) != 5 {
		t.Fatalf("expected 5 deployments after partial teardown, got %d", len(kit.ListDeployments()))
	}

	for i := 5; i < 10; i++ {
		kit.Teardown(ctx, fmt.Sprintf("iso-%d.ts", i))
	}

	if len(kit.ListDeployments()) != 0 {
		t.Fatal("all deployments should be gone")
	}
	t.Log("PASS: 10 isolated deployments, partial teardown, full teardown")
}
